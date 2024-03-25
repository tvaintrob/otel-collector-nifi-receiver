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
	"go.uber.org/zap"
)

type EventTranslator interface {
	// TranslateProvenanceEvents translates a slice of ProvenanceEvent into a ptrace.Traces
	TranslateProvenanceEvents(events []ProvenanceEvent) ptrace.Traces

	// TranslateBulletinEvents translates a slice of BulletinEvent into a ptrace.Traces
	TranslateBulletinEvents(events []BulletinEvent) ptrace.Traces

	// Cleanup cleans up the translator
	Cleanup()
}

type spanContextTracking struct {
	spanContext trace.SpanContext
	ttl         time.Time
}

type eventTranslator struct {
	logger            *zap.Logger
	ignoredEventTypes map[ProvenanceEventType]bool

	// Keep track of the span context for each event.EntityId
	spanContextTracking map[string]spanContextTracking
}

func NewEventTranslator(logger *zap.Logger, ignoredEventTypes []ProvenanceEventType) EventTranslator {
	ignoredEventsMap := make(map[ProvenanceEventType]bool)
	for _, eventType := range ignoredEventTypes {
		ignoredEventsMap[eventType] = true
	}

	return &eventTranslator{
		logger:              logger,
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

		kind := t.getSpanKind(event)
		serviceName := t.getServiceName(event)
		slice, exist := groupByService[serviceName]
		if !exist {
			slice = ptrace.NewSpanSlice()
			groupByService[serviceName] = slice
		}

		spanCtx := t.getSpanContext(event)
		newSpan := slice.AppendEmpty()
		newSpan.SetKind(ptrace.SpanKind(kind))
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

// TranslateBulletinEvents translates a slice of BulletinEvent into a ptrace.Traces
func (t *eventTranslator) TranslateBulletinEvents(events []BulletinEvent) ptrace.Traces {
	groupByService := make(map[string]ptrace.SpanSlice)
	for _, event := range events {
		serviceName := event.BulletinGroupName
		slice, exist := groupByService[serviceName]
		if !exist {
			slice = ptrace.NewSpanSlice()
			groupByService[serviceName] = slice
		}

		newSpan := slice.AppendEmpty()
		newSpan.SetKind(ptrace.SpanKindInternal)
		newSpan.SetSpanID(uuidToSpanID(event.ObjectId))
        newSpan.SetTraceID(uuidToTraceID(event.BulletinGroupId))

		newSpan.SetName(fmt.Sprintf("%s %s", event.BulletinSourceName, event.BulletinLevel))

		ts, err := time.Parse("2006-01-02T15:04:05.999Z", event.BulletinTimestamp)
		if err != nil {
			t.logger.Error("failed to parse timestamp for event",
				zap.String("object.id", event.ObjectId),
				zap.Int64("bulletin.id", event.BulletinId))
			continue
		}

		newSpan.SetStartTimestamp(pcommon.Timestamp(ts.UnixMilli() * 1000000))
		newSpan.SetEndTimestamp(pcommon.Timestamp(ts.UnixMilli() * 1000000)) // Bulletin events dont have any duration

		newSpan.Attributes().PutStr("nifi.object.id", event.ObjectId)
		newSpan.Attributes().PutStr("nifi.platform", event.Platform)
		newSpan.Attributes().PutInt("nifi.bulletin.id", event.BulletinId)
		newSpan.Attributes().PutStr("nifi.bulletin.category", event.BulletinCategory)
		newSpan.Attributes().PutStr("nifi.bulletin.group.id", event.BulletinGroupId)
		newSpan.Attributes().PutStr("nifi.bulletin.group.name", event.BulletinGroupName)
		newSpan.Attributes().PutStr("nifi.bulletin.group.path", event.BulletinGroupPath)
		newSpan.Attributes().PutStr("nifi.bulletin.level", event.BulletinLevel)
		newSpan.Attributes().PutStr("nifi.bulletin.message", event.BulletinMessage)
		newSpan.Attributes().PutStr("nifi.bulletin.node.address", event.BulletinNodeAddress)
		newSpan.Attributes().PutStr("nifi.bulletin.node.id", event.BulletinNodeId)
		newSpan.Attributes().PutStr("nifi.bulletin.source.id", event.BulletinSourceId)
		newSpan.Attributes().PutStr("nifi.bulletin.source.name", event.BulletinSourceName)
		newSpan.Attributes().PutStr("nifi.bulletin.source.type", event.BulletinSourceType)
		newSpan.Attributes().PutStr("nifi.bulletin.flowfile.id", event.BulletinFlowFileUuid)
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

// getSpanKind returns the span kind for the event
func (t *eventTranslator) getSpanKind(event ProvenanceEvent) trace.SpanKind {
	switch event.EventType {
	case ProvenanceEventTypeSend:
		return trace.SpanKindClient
	case ProvenanceEventTypeReceive:
		return trace.SpanKindServer
	default:
		return trace.SpanKindInternal
	}
}

// getSpanContext returns the span context for the event
func (t *eventTranslator) getSpanContext(event ProvenanceEvent) trace.SpanContext {
	// try to extract the span context from the event
	if spanCtx := extractTraceContext(event.UpdatedAttributes); spanCtx.IsValid() {
		t.spanContextTracking[event.EntityId] = spanContextTracking{spanContext: spanCtx, ttl: time.Now().Add(5 * time.Minute)}
		return spanCtx
	}

	defaultSpanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID(uuidToTraceID(event.EntityId)),
	})

	// Fork events create a new span context, keep track of it,
	// and connect it to all the childIds of the fork event
	if event.EventType == ProvenanceEventTypeFork {
		traceID := uuidToTraceID(event.EntityId)
		parentSpanID := uuidToSpanID(event.EventId)
		if ctx, ok := t.spanContextTracking[event.EntityId]; ok {
			traceID = pcommon.TraceID(ctx.spanContext.TraceID())
		}

		childSpanCtx := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    trace.TraceID(traceID),
			SpanID:     trace.SpanID(parentSpanID),
			TraceFlags: 0,
		})

		for _, childId := range event.ChildIds {
			t.spanContextTracking[childId] = spanContextTracking{spanContext: childSpanCtx, ttl: time.Now().Add(5 * time.Minute)}
		}

		if ctx, ok := t.spanContextTracking[event.EntityId]; ok {
			return ctx.spanContext
		}

		return defaultSpanCtx
	}

	if ctx, ok := t.spanContextTracking[event.EntityId]; ok {
		return ctx.spanContext
	}

	return defaultSpanCtx
}
