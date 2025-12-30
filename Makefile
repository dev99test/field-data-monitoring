.PHONY: build test lint

build:
go build ./cmd/loganalyzer

test:
go test ./...

lint:
gofmt -w $(shell find . -name '*.go' -not -path './vendor/*')
