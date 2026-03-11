.PHONY: build test coverage-check ape-build

COVERAGE_MIN ?= 90.0

coverage-check:
	go test ./... -coverpkg=./... -coverprofile=coverage.out
	@total=$$(go tool cover -func=coverage.out | awk '/^total:/ {gsub("%", "", $$3); print $$3}'); \
	echo "Total coverage: $${total}%"; \
	awk -v c="$$total" -v min="$(COVERAGE_MIN)" 'BEGIN { if (c+0 < min+0) { printf("Coverage gate failed: %.1f%% < %.1f%%\n", c, min); exit 1 } }'

test:
	go test ./...

build: coverage-check
	go build -o bin/sq ./cmd/sq

ape-build:
	./scripts/build-ape.sh
