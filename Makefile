.PHONY: help
help:		## Show this help
	@sed -ne '/@sed/!s/## //p' $(MAKEFILE_LIST)

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
