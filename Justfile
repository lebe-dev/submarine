version := `cat VERSION`
imageName := 'tinyops/submarine'

# ldflags injecting the version from the VERSION file into main.version
ldflags := '-X main.version=' + version

# Always produce static binaries on every platform (no libc/cgo linkage).
# Exported so every recipe — build, test, run, release — inherits it.
export CGO_ENABLED := '0'

# Install git hooks (run gofmt + go vet on every commit)
install-hooks:
    git config core.hooksPath .githooks
    @echo "git hooks installed (.githooks)"

# Remove build artifacts
cleanup:
    rm -f sm coverage.out coverage.html

# Upgrade dependencies and tidy go.mod
bump-deps:
    go get -u ./...
    go mod tidy

# Build the sm binary
build: format
    go build -ldflags="{{ ldflags }}" -o sm ./cmd/sm

# Check formatting and run go vet
lint: format
    test -z "$(gofmt -l cmd internal pkg examples)"
    go vet ./...

# Run all tests, or focus on a pattern: `just test TestParseTimestamp`
test pattern="":
    #!/usr/bin/env sh
    if [ -n "{{ pattern }}" ]; then go test ./... -run "{{ pattern }}" -v; else go test ./...; fi

# Run the full test suite
test-all:
    go test ./...

# Run tests with an HTML coverage report
coverage:
    go test ./... -coverprofile=coverage.out
    go tool cover -func=coverage.out
    go tool cover -html=coverage.out -o coverage.html
    @echo "coverage report generated at coverage.html"

# Remove coverage artifacts
coverage-clean:
    rm -f coverage.out coverage.html

# Format the code
format:
    go fmt ./...

# Run the CLI: `just run get test-data/valid/complex.srt 1`
run *args:
    go run ./cmd/sm {{ args }}

# Build the Docker image
build-image: test && lint
    docker build --progress=plain --platform linux/amd64 --build-arg VERSION={{ version }} \
        -t {{ imageName }}:{{ version }} -t {{ imageName }}:latest .

# Push the Docker image
push-image:
    docker push {{ imageName }}:{{ version }}
    docker push {{ imageName }}:latest

# Build and push the Docker image
release-image: build-image && push-image

# Build a compressed linux/amd64 release archive
release-linux: test && lint
    just cleanup
    GOOS=linux GOARCH=amd64 go build -ldflags="-w -s {{ ldflags }}" -o sm ./cmd/sm
    zip -9 sm-{{ version }}-linux-amd64.zip sm
    rm -f sm

# Build a compressed macos/arm64 release archive
release-macos: test && lint
    just cleanup
    GOOS=darwin GOARCH=arm64 go build -ldflags="-w -s {{ ldflags }}" -o sm ./cmd/sm
    zip -9 sm-{{ version }}-macos-arm64.zip sm
    rm -f sm

release: release-linux release-macos release-image
