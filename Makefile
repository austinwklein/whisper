.PHONY: build run dev test clean fmt lint

# Build the application
build:
	go build -o whisper .

# Run in development mode
dev:
	go run main.go

# Run the application
run: build
	./whisper

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -cover ./...

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -f whisper
	rm -rf logs/*

# Setup dev environment
setup:
	go mod download
	go mod tidy