package nifireceiver

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"

	"github.com/tvaintrob/otel-collector-nifi-receiver/internal/metadata"
)

// NewFactory creates a factory for DataDog receiver.
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithTraces(createTracesReceiver, metadata.TracesStability))
}

func createDefaultConfig() component.Config {
	return &Config{
		ServerConfig: confighttp.ServerConfig{
			Endpoint: "localhost:8200",
		},
	}
}

func createTracesReceiver(_ context.Context, params receiver.CreateSettings, cfg component.Config, consumer consumer.Traces) (receiver.Traces, error) {
	rcfg := cfg.(*Config)
	return newNifiReceiver(rcfg, consumer, params)
}
