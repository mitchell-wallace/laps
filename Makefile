.PHONY: build test lint clean

build:
	go build -ldflags "-X main.version=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)" -o bin/laps ./cmd/laps

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -rf bin/
