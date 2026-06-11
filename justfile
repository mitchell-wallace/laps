# Laps - a lightweight task tracker for AI coding agents
# See SPEC.md and README.md for project details.
# Use `just --list` to see available commands.

# Build the laps binary with version metadata
build:
    go build -ldflags "-X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" -o bin/laps ./cmd/laps

# Run all tests
test:
    go test ./...

# Run golangci-lint
lint:
    golangci-lint run ./...

# Remove build output
clean:
    rm -rf bin/

# Build and run the dev binary (passes through additional arguments)
run *args:
    go run ./cmd/laps {{args}}

# Tidy go modules
tidy:
    go mod tidy

# Format Go source code
fmt:
    go fmt ./...

# Run go vet
vet:
    go vet ./...

# Full quality check: format, vet, lint, test
audit: fmt vet lint test

# Watch Go files and re-run tests on change (requires entr)
watch:
    find . -name '*.go' | entr -r go test ./...

# Install the laps binary to $GOPATH/bin
install:
    go install ./cmd/laps

# Run a snapshot release locally (dry run, no publish)
release-dry:
    goreleaser release --snapshot --clean
