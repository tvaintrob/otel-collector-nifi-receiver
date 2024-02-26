package nifireceiver

import (
	"github.com/tvaintrob/otel-collector-nifi-receiver/internal/translator"
	"go.opentelemetry.io/collector/config/confighttp"
)

type Config struct {
	confighttp.ServerConfig `mapstructure:",squash"`

	IgnoredEventTypes []translator.ProvenanceEventType `mapstructure:"ignored_events,omitempty"`
}
