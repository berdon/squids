.PHONY: build test coverage-check ape-build

COVERAGE_MIN ?= 90.0

coverage-check:
	./scripts/coverage-gate.sh $(COVERAGE_MIN)

test:
	go test ./...

build: coverage-check
	go build -o bin/sq ./cmd/sq

ape-build:
	./scripts/build-ape.sh
