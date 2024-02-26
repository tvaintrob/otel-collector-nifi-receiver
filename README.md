# OpenTelemetry Collector Nifi Receiver

[OpenTelemetry Collector](https://opentelemetry.io/docs/collector/) receiver that allows consuming [Apache Nifi](https://nifi.apache.org/) provenance data.

## Open Issues

- Receive bulletin data to add errors to the traces
- Handle context propagation from listeners (expect `tracecontext` attribute)
- Add links to `JOIN` events

## Building the distribution

To build the distribution use the `make otelcol-nifi` target

```
make otelcol-nifi
./cmd/otelcol-nifi/otelcol-nifi --config=./cmd/otelcol-nifi/default-config.yaml
```
