.PHONY: build build-dev build-gui build-gui-frontend run dev test clean fmt lint clean-db clean-db-dev reset reset-dev clean-all setup

# Detect wails binary location
WAILS := $(shell which wails 2>/dev/null || echo "$$HOME/go/bin/wails")

# Build the CLI application (production mode - uses ~/.whisper/whisper.db)
build:
	go build -tags '!gui' -o whisper .

# Build frontend for GUI (run before build-gui)
build-gui-frontend:
	@echo "Building frontend assets..."
	@mkdir -p frontend/dist
	cd frontend && npm install --legacy-peer-deps && npm run build

# Build the GUI application (with frontend pre-build)
build-gui: build-gui-frontend
	@echo "Building GUI application..."
	$(WAILS) build -tags gui

# Build CLI in dev mode (uses ./data/whisper.db in current directory)
build-dev:
	go build -tags '!gui' -ldflags "-X 'github.com/austinwklein/whisper/config.DefaultDBPath=./data/whisper.db'" -o whisper .
	@echo "Built in DEV mode - database will be at ./data/whisper.db"

# Run GUI in development mode
dev-gui:
	$(WAILS) dev -tags gui

# Run CLI in development mode with go run
dev:
	go run -tags '!gui' main_cli.go

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
	rm -rf build/bin/*
	rm -rf frontend/dist
	rm -rf frontend/node_modules/.vite
	@echo "Build artifacts cleaned"

# Clean everything including node_modules (use before switching machines)
clean-all: clean clean-db
	rm -rf frontend/node_modules
	@echo "All artifacts and dependencies cleaned"

# Clean database (production)
clean-db:
	rm -rf ~/.whisper
	@echo "Production database cleaned (~/.whisper)"

# Clean database (dev mode)
clean-db-dev:
	rm -rf ./data
	@echo "Dev database cleaned (./data)"

# Reset everything (build artifacts + database)
reset: clean clean-db
	@echo "Full reset complete - ready for fresh build"

# Reset for dev mode
reset-dev: clean clean-db-dev
	@echo "Dev reset complete - ready for fresh build"

# Setup dev environment
setup:
	@echo "Setting up development environment..."
	go mod download
	go mod tidy
	cd frontend && npm install --legacy-peer-deps
	@echo "Setup complete!"
