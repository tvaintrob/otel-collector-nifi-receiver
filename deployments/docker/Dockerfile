FROM alpine:latest as prep
RUN apk --update add ca-certificates

RUN mkdir -p /tmp

FROM golang:1.21-alpine as builder
RUN apk add --no-cache make
WORKDIR /app
COPY . .
RUN make otelcol-nifi

FROM scratch

ARG USER_UID=10001
USER ${USER_UID}

COPY --from=prep /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app/cmd/otelcol-nifi/otelcol-nifi /otelcol-nifi
COPY --from=builder /app/cmd/otelcol-nifi/default-config.yaml /etc/otel/config.yaml

EXPOSE 4317 55680 55679
ENTRYPOINT ["/otelcol-nifi"]
CMD ["--config", "/etc/otel/config.yaml"]
