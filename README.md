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

### provenance_url_path (Optional)

The URL path to receive traces on.

Default: `/v1/provenance`

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

## Deployment

### Docker

#### Create a config file

Initially create a config file to run the collector, for an example you can take a look at the [default-config.yaml](./cmd/otelcol-nifi/default-config.yaml) file.

#### Run the image

```bash
docker run --rm -v ./collector-nifi-config.yaml:/etc/otel/config.yaml -t ghcr.io/tvaintrob/otel-collector-nifi-receiver:latest
```

### Kubernetes

A helm chart is provided for easier deployments in K8s environments, currently it is unpublished, to use it do the following:

#### Create a `values.yaml` file

Create a `values.yaml` file with the wanted configuration.

```bash
git clone https://github.com/tvaintrob/otel-collector-nifi-receiver.git

# Build chart dependencies
helm dependency update otel-collector-nifi-receiver/deployments/helm

# Install the chart using your values file
helm install -f values.yaml otelcol-nifi otel-collector-nifi-receiver/deployments/helm
```
