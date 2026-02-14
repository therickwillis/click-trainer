.PHONY: test test-unit test-integration lint build dev coverage clean

test:
	go test -v -race -count=1 -timeout 60s ./...

test-unit:
	go test -v -race -count=1 -timeout 30s -short ./...

test-integration:
	@if [ -z "$$TEST_DATABASE_URL" ]; then \
		echo "ERROR: TEST_DATABASE_URL is not set"; \
		exit 1; \
	fi
	go test -v -race -count=1 -timeout 60s ./...

lint:
	golangci-lint run ./...

build:
	go build -o ./tmp/main ./cmd/web

dev:
	air

coverage:
	go test -race -count=1 -timeout 60s -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

clean:
	rm -rf tmp/ coverage.out
