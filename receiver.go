package nifireceiver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/tvaintrob/otel-collector-nifi-receiver/internal/metadata"
	"github.com/tvaintrob/otel-collector-nifi-receiver/internal/translator"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.uber.org/zap"
)

type nifiReceiver struct {
	address      string
	config       *Config
	params       receiver.CreateSettings
	nextConsumer consumer.Traces
	server       *http.Server
	tReceiver    *receiverhelper.ObsReport
	translator   *translator.ProvEventsTranslator
}

func newNifiReceiver(config *Config, nextConsumer consumer.Traces, params receiver.CreateSettings) (receiver.Traces, error) {
	if nextConsumer == nil {
		return nil, component.ErrNilNextConsumer
	}

	instance, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{LongLivedCtx: false, ReceiverID: params.ID, Transport: "http", ReceiverCreateSettings: params})
	if err != nil {
		return nil, err
	}

	return &nifiReceiver{
		params:       params,
		config:       config,
		nextConsumer: nextConsumer,
		server:       &http.Server{},
		tReceiver:    instance,
		translator:   translator.New(config.IgnoredEventTypes),
	}, nil
}

// Start the receiver and listen for traces
func (r *nifiReceiver) Start(_ context.Context, host component.Host) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/provenance", r.handleTraces)

	var err error
	r.server, err = r.config.ServerConfig.ToServer(host, r.params.TelemetrySettings, mux)
	if err != nil {
		return fmt.Errorf("failed to create server definition: %w", err)
	}

	hln, err := r.config.ServerConfig.ToListener()
	if err != nil {
		return fmt.Errorf("failed to create nifi listener: %w", err)
	}

	r.address = hln.Addr().String()

	go func() {
		if err := r.server.Serve(hln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			r.params.TelemetrySettings.ReportStatus(component.NewFatalErrorEvent(fmt.Errorf("error starting nifi receiver: %w", err)))
		}
	}()
	return nil
}

// Shutdown the receiver
func (r *nifiReceiver) Shutdown(ctx context.Context) (err error) {
	return r.server.Shutdown(ctx)
}

func (r *nifiReceiver) handleTraces(w http.ResponseWriter, req *http.Request) {
	obsCtx := r.tReceiver.StartTracesOp(req.Context())
	var err error
	var spanCount int
	var provenanceEvents []translator.ProvenanceEvent
	defer func(spanCount *int) {
		r.tReceiver.EndTracesOp(obsCtx, metadata.Type.String(), *spanCount, err)
	}(&spanCount)

	jsonDecoder := json.NewDecoder(req.Body)
	err = jsonDecoder.Decode(&provenanceEvents)
	if err != nil {
		http.Error(w, "Failed to decode JSON", http.StatusBadRequest)
		r.params.Logger.Error("Failed to decode JSON", zap.Error(err))
		return
	}

	traces := r.translator.TranslateProvenanceEvents(provenanceEvents)
	spanCount = traces.SpanCount()
	err = r.nextConsumer.ConsumeTraces(obsCtx, traces)
	if err != nil {
		http.Error(w, "Failed to consume traces", http.StatusInternalServerError)
		r.params.Logger.Error("Failed to consume traces", zap.Error(err))
		return
	}

	r.translator.Cleanup()
	_, _ = w.Write([]byte("OK"))
}
