package nifireceiver

import (
	"encoding/binary"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// ProvenanceEvent is a struct that represents a single provenance event
type ProvenanceEvent struct {
	EventId             string            `json:"eventId"`
	EventOrdinal        int64             `json:"eventOrdinal"`
	EventType           string            `json:"eventType"`
	TimestampMillis     int64             `json:"timestampMillis"`
	DurationMillis      int64             `json:"durationMillis"`
	LineageStart        int64             `json:"lineageStart"`
	Details             string            `json:"details"`
	ComponentId         string            `json:"componentId"`
	ComponentType       string            `json:"componentType"`
	ComponentName       string            `json:"componentName"`
	ProcessGroupId      string            `json:"processGroupId"`
	ProcessGroupName    string            `json:"processGroupName"`
	EntityId            string            `json:"entityId"`
	EntityType          string            `json:"entityType"`
	EntitySize          int64             `json:"entitySize"`
	PreviousEntitySize  int64             `json:"previousEntitySize"`
	UpdatedAttributes   map[string]string `json:"updatedAttributes"`
	PreviousAttributes  map[string]string `json:"previousAttributes"`
	ActorHostname       string            `json:"actorHostname"`
	ContentURI          string            `json:"contentURI"`
	PreviousContentURI  string            `json:"previousContentURI"`
	ParentIds           []string          `json:"parentIds"`
	ChildIds            []string          `json:"childIds"`
	Platform            string            `json:"platform"`
	Application         string            `json:"application"`
	RemoteIdentifier    string            `json:"remoteIdentifier"`
	AlternateIdentifier string            `json:"alternateIdentifier"`
	TransitUri          string            `json:"transitUri"`
}

// ProvenanceEventBatch is a struct that represents a batch of provenance events
type ProvenanceEventBatch []ProvenanceEvent

// toTraceData converts a ProvenanceEventBatch to a pdata.Traces
//
// Each event in the batch contains a unique EventId, which can be used as the span id,
// additionally the EntityId can be used as the trace id, special attention should be given
// to cases where the flow splits, in that case there will be an even that signals the split,
// that event should contain the child ids, each one of the child ids mark a new entity id
// that should be connected to the original entity id.
func toTraceData(pb ProvenanceEventBatch) ptrace.Traces {
	groupByService := make(map[string]ptrace.SpanSlice)
	for _, event := range pb {
		slice, exist := groupByService[event.ProcessGroupName]
		if !exist {
			slice = ptrace.NewSpanSlice()
			groupByService[event.ProcessGroupName] = slice
		}

		newSpan := slice.AppendEmpty()
		newSpan.SetTraceID(uuidToTraceID(event.EntityId))
		newSpan.SetSpanID(uuidToSpanID(event.EventId))
		newSpan.SetStartTimestamp(pcommon.Timestamp(event.TimestampMillis * 1000000))
		newSpan.SetEndTimestamp(pcommon.Timestamp((event.TimestampMillis + event.DurationMillis) * 1000000))
		newSpan.SetName(fmt.Sprintf("%s %s", event.ComponentName, event.EventType))
		newSpan.Status().SetCode(ptrace.StatusCodeOk)
		newSpan.SetKind(ptrace.SpanKindInternal)
		newSpan.Attributes().PutStr("nifi.event.id", event.EventId)
		newSpan.Attributes().PutStr("nifi.event.type", event.EventType)
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
			newSpan.Attributes().PutStr(fmt.Sprintf("nifi.%s", key), val)
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

func uuidToTraceID(uuidStr string) pcommon.TraceID {
	var traceID [16]byte
	u := uuid.MustParse(uuidStr)
	binary.BigEndian.PutUint64(traceID[:], uint64(u.ID()))
	binary.BigEndian.PutUint64(traceID[8:], uint64(u.ID()))

	return traceID
}

func uuidToSpanID(uuidStr string) pcommon.SpanID {
	var traceID [8]byte
	u := uuid.MustParse(uuidStr)
	binary.BigEndian.PutUint64(traceID[:], uint64(u.ID()))

	return traceID
}
