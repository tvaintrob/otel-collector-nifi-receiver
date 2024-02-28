# OpenTelemetry Collector Nifi Receiver

[OpenTelemetry Collector](https://opentelemetry.io/docs/collector/) receiver that allows consuming [Apache Nifi](https://nifi.apache.org/) provenance data.

## Open Issues

- Receive bulletin data to add errors to the traces

## Building the distribution

To build the distribution use the `make otelcol-nifi` target

```bash
# Build the distribution
make otelcol-nifi

# Run the built binary with the default config file
./cmd/otelcol-nifi/otelcol-nifi --config=./cmd/otelcol-nifi/default-config.yaml
```

## Configuration

Example: 

```yaml
receivers:
  nifi:
    endpoint: localhost:8200
```

### ignored_events (Optional)

A list of event types to ignore, for a list of possible values see: [./internal/translator/models.go](./internal/translator/models.go)

Default: 

```yaml
ignored_events:
  - DOWNLOAD
```

### HTTP Service Config

All config params here are valid as well

<https://github.com/open-telemetry/opentelemetry-collector/tree/main/config/confighttp#server-configuration>
