package nifireceiver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type nifiReceiver struct {
	address      string
	config       *Config
	params       receiver.CreateSettings
	nextConsumer consumer.Traces
	server       *http.Server
	tReceiver    *receiverhelper.ObsReport
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
	defer func(spanCount *int) {
		r.tReceiver.EndTracesOp(obsCtx, "datadog", *spanCount, err)
	}(&spanCount)

	body, err := io.ReadAll(req.Body)
	if err != nil {
		r.params.Logger.Error("Error reading request body", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	r.params.Logger.Log(zapcore.InfoLevel, "Handling nifi provenance data",
		zap.String("method", req.Method),
		zap.String("path", req.URL.Path),
		zap.String("body", string(body)),
	)
}
