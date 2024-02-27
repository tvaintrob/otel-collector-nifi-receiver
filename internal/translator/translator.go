package translator

import (
	"fmt"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

type forkEvent struct {
	traceId      string
	parentSpanId string
	ttl          time.Time
}

type ProvEventsTranslator struct {
	ignoredEventTypes map[ProvenanceEventType]bool
	forkTracking      map[string]*forkEvent
}

func New(ignoredEventTypes []ProvenanceEventType) *ProvEventsTranslator {
	ignored := make(map[ProvenanceEventType]bool)
	for _, eventType := range ignoredEventTypes {
		ignored[eventType] = true
	}

	return &ProvEventsTranslator{
		forkTracking:      make(map[string]*forkEvent),
		ignoredEventTypes: ignored,
	}
}

func (t *ProvEventsTranslator) shouldIgnore(eventType ProvenanceEventType) bool {
	_, ok := t.ignoredEventTypes[eventType]
	return ok
}

func (t *ProvEventsTranslator) TranslateProvenanceEvents(events []ProvenanceEvent) ptrace.Traces {
	groupByService := make(map[string]ptrace.SpanSlice)
	for _, event := range events {
		if t.shouldIgnore(event.EventType) {
			continue
		}

		slice, exist := groupByService[event.ProcessGroupName]
		if !exist {
			slice = ptrace.NewSpanSlice()
			groupByService[event.ProcessGroupName] = slice
		}

		newSpan := slice.AppendEmpty()
		newSpan.Status().SetCode(ptrace.StatusCodeUnset)
		newSpan.SetName(fmt.Sprintf("%s %s", event.ComponentName, event.EventType))

		switch event.EventType {
		case ProvenanceEventTypeFetch:
			newSpan.SetKind(ptrace.SpanKindServer)
		case ProvenanceEventTypeReceive:
			newSpan.SetKind(ptrace.SpanKindServer)
		case ProvenanceEventTypeRemoteInvocation:
			newSpan.SetKind(ptrace.SpanKindClient)
		case ProvenanceEventTypeSend:
			newSpan.SetKind(ptrace.SpanKindClient)
		default:
			newSpan.SetKind(ptrace.SpanKindInternal)
		}

		if event.EventType == ProvenanceEventTypeFork && len(event.ChildIds) > 0 {
			fe := &forkEvent{
				traceId:      event.EntityId,
				parentSpanId: event.EventId,
				ttl:          time.Now().Add(5 * time.Minute),
			}

			for _, childId := range event.ChildIds {
				parent, ok := t.forkTracking[event.EntityId]
				if ok {
					fe.traceId = parent.traceId
				}

				t.forkTracking[childId] = fe
			}
		}

		spanID := event.EventId
		traceID := event.EntityId

		rootEntity, ok := t.forkTracking[event.EntityId]
		if ok {
			traceID = rootEntity.traceId
			newSpan.SetParentSpanID(uuidToSpanID(rootEntity.parentSpanId))
		}

		newSpan.SetSpanID(uuidToSpanID(spanID))
		newSpan.SetTraceID(uuidToTraceID(traceID))
		newSpan.SetEndTimestamp(pcommon.Timestamp((event.TimestampMillis + event.DurationMillis) * 1000000))
		newSpan.SetStartTimestamp(pcommon.Timestamp(event.TimestampMillis * 1000000))

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
			newSpan.Attributes().PutStr(fmt.Sprintf("nifi.attributes.%s", key), val)
		}
	}

	results := ptrace.NewTraces()
	for service, spans := range groupByService {
		rs := results.ResourceSpans().AppendEmpty()
		rs.SetSchemaUrl(semconv.SchemaURL)
		rs.Resource().Attributes().PutStr(string(semconv.ServiceNameKey), service)

		in := rs.ScopeSpans().AppendEmpty()
		in.Scope().SetName("nifi.provenance.receiver")
		in.Scope().SetVersion("0.1.0")
		spans.CopyTo(in.Spans())
	}

	return results
}

// Cleanup removes old fork tracking entries
func (t *ProvEventsTranslator) Cleanup() {
	for key, value := range t.forkTracking {
		if time.Now().After(value.ttl) {
			delete(t.forkTracking, key)
		}
	}
}
