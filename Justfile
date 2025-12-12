set shell := ["bash", "-c"]

# Build the CLI binary into bin/doc2x using a local Go cache
build:
	mkdir -p bin
	go build -o bin/doc2x ./cmd/doc2x

# Format Go sources
fmt:
	gofmt -w cmd/doc2x/*.go *.go

# Tidy go.mod/go.sum with a local Go cache
tidy:
    go mod tidy

# Remove built binaries
clean-bin:
	rm -rf bin/doc2x
