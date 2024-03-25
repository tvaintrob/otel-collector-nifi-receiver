package translator

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func uuidToTraceID(uuidStr string) pcommon.TraceID {
	var traceID [16]byte
	u := uuid.MustParse(uuidStr)
	b, _ := u.MarshalBinary()
	for i := 0; i < len(traceID); i++ {
		traceID[i] = b[i]
	}
	return traceID
}

func uuidToSpanID(uuidStr string) pcommon.SpanID {
	var traceID [8]byte
	u := uuid.MustParse(uuidStr)
	b, _ := u.MarshalBinary()
	for i := 0; i < len(traceID); i++ {
		traceID[i] = b[i]
	}
	return traceID
}

func extractTraceContext(attrs, aliases map[string]string) trace.SpanContext {
	tc := propagation.TraceContext{}
	ctx := tc.Extract(context.Background(), newCaseInsensitiveMapCarrier(attrs, aliases))
	return trace.SpanContextFromContext(ctx)
}
