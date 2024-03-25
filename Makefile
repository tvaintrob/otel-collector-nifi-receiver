.PHONY: help
help: 		## Print this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[$$()% a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


.PHONY: format
format:		## Format source files
	go fmt ./...

.PHONY: test
test:		## Run tests
	go test -v ./...

.PHONY: lint
lint:		## Run linter
	go vet ./...
	golangci-lint run

.PHONY: otelcol-nifi
otelcol-nifi:	## Build otelcol-nifi distribution
	@cd ./cmd/otelcol-nifi && go run go.opentelemetry.io/collector/cmd/builder@v0.95.0 --config=otelcol-builder.yaml

.PHONY: otelcol-dev
otelcol-dev: otelcol-nifi 	## Run otelcol-nifi in dev mode
	@go run github.com/cosmtrek/air@v1.43.0 \
		--build.cmd "make otelcol-nifi" --build.bin "./cmd/otelcol-nifi/otelcol-nifi" --build.delay "100" \
		--build.args_bin "--config=./cmd/otelcol-nifi/default-config.yaml" \
		--build.exclude_dir "cmd" \
		--build.include_ext "go" \
		--misc.clean_on_exit "true"
