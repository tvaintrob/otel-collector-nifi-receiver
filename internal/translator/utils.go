package translator

import (
	"context"
	"encoding/binary"

	"github.com/google/uuid"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

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

func extractTraceContext(attrs map[string]string) trace.SpanContext {
	tc := propagation.TraceContext{}
	ctx := tc.Extract(context.Background(), caseInsensitiveMapCarrier(attrs))
	return trace.SpanContextFromContext(ctx)
}
