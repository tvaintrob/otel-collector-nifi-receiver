package translator

import (
	"encoding/binary"

	"github.com/google/uuid"
	"go.opentelemetry.io/collector/pdata/pcommon"
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
