package translator

import (
	"strings"

	"go.opentelemetry.io/otel/propagation"
)

// caseInsensitiveMapCarrier is a TextMapCarrier that uses a map held in memory as a storage
// access to the map is case insensitive
type caseInsensitiveMapCarrier struct {
	internalMap map[string]string
	aliases     map[string]string
}

// Ensure that CaseInsensitiveMapCarrier is a TextMapCarrier
var _ propagation.TextMapCarrier = caseInsensitiveMapCarrier{}

// newCaseInsensitiveMapCarrier creates a new carrier
func newCaseInsensitiveMapCarrier(attrs, aliases map[string]string) caseInsensitiveMapCarrier {
	return caseInsensitiveMapCarrier{internalMap: attrs, aliases: aliases}
}

// Get returns the value associated with the passed key.
func (c caseInsensitiveMapCarrier) Get(key string) string {
	alias, ok := c.aliases[key]
	if !ok {
		alias = key
	}

	val, ok := c.internalMap[alias]
	if ok {
		return val
	}

	for k, v := range c.internalMap {
		if strings.EqualFold(key, k) {
			return v
		}
	}

	return ""
}

// Set stores the key-value pair.
func (c caseInsensitiveMapCarrier) Set(key string, value string) {
	c.internalMap[key] = value
}

// Keys lists the keys stored in this carrier.
func (c caseInsensitiveMapCarrier) Keys() []string {
	keys := make([]string, 0, len(c.internalMap))
	for k := range c.internalMap {
		keys = append(keys, k)
	}
	return keys
}
