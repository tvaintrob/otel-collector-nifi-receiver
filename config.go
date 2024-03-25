package nifireceiver

import (
	"github.com/tvaintrob/otel-collector-nifi-receiver/internal/translator"
	"go.opentelemetry.io/collector/config/confighttp"
)

type Config struct {
	confighttp.ServerConfig `mapstructure:",squash"`

	IgnoredEventTypes         []translator.ProvenanceEventType `mapstructure:"ignored_events,omitempty"`
	ContextPropagationAliases map[string]string                `mapstructure:"context_propagation_aliases,omitempty"`
	BulletinURLPath           string                           `mapstructure:"bulletin_url_path,omitempty"`
	ProvenanceURLPath         string                           `mapstructure:"provenance_url_path,omitempty"`
}
