package translator

import (
	"strings"

	"go.opentelemetry.io/otel/propagation"
)

// caseInsensitiveMapCarrier is a TextMapCarrier that uses a map held in memory as a storage
// access to the map is case insensitive
type caseInsensitiveMapCarrier map[string]string

// Ensure that CaseInsensitiveMapCarrier is a TextMapCarrier
var _ propagation.TextMapCarrier = caseInsensitiveMapCarrier{}

// Get returns the value associated with the passed key.
func (c caseInsensitiveMapCarrier) Get(key string) string {
	val, ok := c[key]
	if ok {
		return val
	}

	for k, v := range c {
		if strings.EqualFold(key, k) {
			return v
		}
	}

	return ""
}

// Set stores the key-value pair.
func (c caseInsensitiveMapCarrier) Set(key string, value string) {
	c[key] = value
}

// Keys lists the keys stored in this carrier.
func (c caseInsensitiveMapCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}
