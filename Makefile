.PHONY: build build-dev run dev test clean fmt lint clean-db clean-db-dev reset reset-dev

# Build the application (production mode - uses ~/.whisper/whisper.db)
build:
	go build -o whisper .

# Build in dev mode (uses ./data/whisper.db in current directory)
build-dev:
	go build -ldflags "-X 'github.com/austinwklein/whisper/config.DefaultDBPath=./data/whisper.db'" -o whisper .
	@echo "Built in DEV mode - database will be at ./data/whisper.db"

# Run in development mode with go run
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

# Clean database (production)
clean-db:
	rm -rf ~/.whisper
	@echo "Database cleaned"

# Clean database (dev mode)
clean-db-dev:
	rm -rf ./data
	@echo "Dev database cleaned"

# Reset everything (build artifacts + database)
reset: clean clean-db
	@echo "Full reset complete"

# Reset for dev mode
reset-dev: clean clean-db-dev
	@echo "Dev reset complete"

# Setup dev environment
setup:
	go mod download
	go mod tidy
