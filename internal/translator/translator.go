package translator

import (
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

type EventTranslator interface {
	// TranslateProvenanceEvents translates a slice of ProvenanceEvent into a ptrace.Traces
	TranslateProvenanceEvents(events []ProvenanceEvent) ptrace.Traces

	// Cleanup cleans up the translator
	Cleanup()
}

type spanContextTracking struct {
	spanContext trace.SpanContext
	ttl         time.Time
}

type eventTranslator struct {
	ignoredEventTypes map[ProvenanceEventType]bool

	// Keep track of the span context for each event.EntityId
	spanContextTracking map[string]spanContextTracking
}

func New(ignoredEventTypes []ProvenanceEventType) EventTranslator {
	ignoredEventsMap := make(map[ProvenanceEventType]bool)
	for _, eventType := range ignoredEventTypes {
		ignoredEventsMap[eventType] = true
	}

	return &eventTranslator{
		ignoredEventTypes:   ignoredEventsMap,
		spanContextTracking: make(map[string]spanContextTracking),
	}
}

// TranslateProvenanceEvents translates a slice of ProvenanceEvent into a ptrace.Traces
func (t *eventTranslator) TranslateProvenanceEvents(events []ProvenanceEvent) ptrace.Traces {
	groupByService := make(map[string]ptrace.SpanSlice)
	for _, event := range events {
		if t.shouldIgnore(event) {
			continue
		}

		serviceName := t.getServiceName(event)
		slice, exist := groupByService[serviceName]
		if !exist {
			slice = ptrace.NewSpanSlice()
			groupByService[serviceName] = slice
		}

		spanCtx := t.getSpanContext(event)
		newSpan := slice.AppendEmpty()
		newSpan.SetTraceID(pcommon.TraceID(spanCtx.TraceID()))
		newSpan.SetParentSpanID(pcommon.SpanID(spanCtx.SpanID()))
		newSpan.SetSpanID(uuidToSpanID(event.EventId))

		newSpan.SetName(fmt.Sprintf("%s %s", event.ComponentName, event.EventType))
		newSpan.SetStartTimestamp(pcommon.Timestamp(event.TimestampMillis * 1000000))
		newSpan.SetEndTimestamp(pcommon.Timestamp((event.TimestampMillis + event.DurationMillis) * 1000000))

		newSpan.Attributes().PutStr("nifi.event.id", event.EventId)
		newSpan.Attributes().PutStr("nifi.event.type", string(event.EventType))
		newSpan.Attributes().PutStr("nifi.event.details", event.Details)
		newSpan.Attributes().PutStr("nifi.component.id", event.ComponentId)
		newSpan.Attributes().PutStr("nifi.component.type", event.ComponentType)
		newSpan.Attributes().PutStr("nifi.component.name", event.ComponentName)
		newSpan.Attributes().PutStr("nifi.process.group.id", event.ProcessGroupId)
		newSpan.Attributes().PutStr("nifi.process.group.name", event.ProcessGroupName)
		newSpan.Attributes().PutStr("nifi.entity.id", event.EntityId)
		newSpan.Attributes().PutStr("nifi.entity.type", event.EntityType)
		newSpan.Attributes().PutInt("nifi.entity.size", event.EntitySize)
		newSpan.Attributes().PutStr("nifi.hostname", event.ActorHostname)
		newSpan.Attributes().PutStr("nifi.platform", event.Platform)
		newSpan.Attributes().PutStr("nifi.application", event.Application)

		for key, val := range event.UpdatedAttributes {
			newSpan.Attributes().PutStr(fmt.Sprintf("nifi.attributes.%s", strings.ToLower(key)), val)
		}

		if event.EventType == ProvenanceEventTypeJoin {
			// Add links to the parent spans, only unique links
			spanCtxs := make(map[trace.TraceID]trace.SpanContext)
			for _, parent := range event.ParentIds {
				spanCtx := t.spanContextTracking[parent].spanContext
				spanCtxs[spanCtx.TraceID()] = spanCtx
			}

			for _, spanCtx := range spanCtxs {
				ln := newSpan.Links().AppendEmpty()
				ln.SetTraceID(pcommon.TraceID(spanCtx.TraceID()))
			}
		}
	}

	results := ptrace.NewTraces()
	for service, spans := range groupByService {
		rs := results.ResourceSpans().AppendEmpty()
		rs.SetSchemaUrl(semconv.SchemaURL)
		rs.Resource().Attributes().PutStr(string(semconv.ServiceNameKey), service)

		in := rs.ScopeSpans().AppendEmpty()
		in.Scope().SetName("nifi.provenance.receiver")

		info, ok := debug.ReadBuildInfo()
		if ok {
			in.Scope().SetVersion(info.Main.Version)
		} else {
			in.Scope().SetVersion("unknown")
		}

		spans.CopyTo(in.Spans())
	}

	return results
}

func (t *eventTranslator) Cleanup() {
	for k := range t.spanContextTracking {
		if time.Now().After(t.spanContextTracking[k].ttl) {
			delete(t.spanContextTracking, k)
		}
	}
}

// shouldIgnore returns true if the event should be ignored
func (t *eventTranslator) shouldIgnore(event ProvenanceEvent) bool {
	_, ok := t.ignoredEventTypes[event.EventType]
	return ok
}

// getServiceName returns the service name for the event
func (t *eventTranslator) getServiceName(event ProvenanceEvent) string {
	return event.ProcessGroupName
}

// getSpanContext returns the span context for the event
func (t *eventTranslator) getSpanContext(event ProvenanceEvent) trace.SpanContext {
	// try to extract the span context from the event
	spanCtx := extractTraceContext(event.UpdatedAttributes)
	if spanCtx.IsValid() {
		t.spanContextTracking[event.EntityId] = spanContextTracking{spanContext: spanCtx, ttl: time.Now().Add(5 * time.Minute)}
		return spanCtx
	}

	defaultSpanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID(uuidToTraceID(event.EntityId)),
	})

	// Fork events create a new span context, keep track of it,
	// and connect it to all the childIds of the fork event
	if event.EventType == ProvenanceEventTypeFork {
		var traceID [16]byte
		parentSpanID := uuidToSpanID(event.EventId)
		spanCtx, ok := t.spanContextTracking[event.EntityId]
		if ok {
			traceID = spanCtx.spanContext.TraceID()
		} else {
			traceID = uuidToTraceID(event.EntityId)
		}

		childSpanCtx := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    trace.TraceID(traceID),
			SpanID:     trace.SpanID(parentSpanID),
			TraceFlags: 0,
		})

		for _, childId := range event.ChildIds {
			t.spanContextTracking[childId] = spanContextTracking{spanContext: childSpanCtx, ttl: time.Now().Add(5 * time.Minute)}
		}

		if ok {
			return spanCtx.spanContext
		}

		return defaultSpanCtx
	}

	if ctx, ok := t.spanContextTracking[event.EntityId]; ok {
		return ctx.spanContext
	}

	return defaultSpanCtx
}
