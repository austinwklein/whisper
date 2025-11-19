.PHONY: build build-dev build-gui build-gui-frontend run dev test clean fmt lint clean-db clean-db-dev reset reset-dev clean-all setup check-webkit

# Detect wails binary location
WAILS := $(shell which wails 2>/dev/null || echo "$$HOME/go/bin/wails")

# Detect OS and architecture
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

# Build the CLI application (production mode - uses ~/.whisper/whisper.db)
build:
	go build -tags '!gui' -o whisper .

# Build frontend for GUI (run before build-gui)
build-gui-frontend:
	@echo "Building frontend assets..."
	@mkdir -p frontend/dist
	cd frontend && npm install --legacy-peer-deps && npm run build

# Check and setup webkit compatibility for Linux
check-webkit:
ifeq ($(UNAME_S),Linux)
	@echo "Checking webkit2gtk compatibility on Linux..."
	@if ! pkg-config --exists webkit2gtk-4.0 2>/dev/null; then \
		if pkg-config --exists webkit2gtk-4.1 2>/dev/null; then \
			echo "webkit2gtk-4.0 not found, but webkit2gtk-4.1 is available."; \
			echo "Creating compatibility symlinks..."; \
			PKGCONFIG_DIR=$$(pkg-config --variable pc_path pkg-config | tr ':' '\n' | grep -m1 "/usr"); \
			if [ -w "$$PKGCONFIG_DIR" ]; then \
				sudo ln -sf webkit2gtk-4.1.pc $$PKGCONFIG_DIR/webkit2gtk-4.0.pc 2>/dev/null || true; \
				sudo ln -sf webkit2gtk-web-extension-4.1.pc $$PKGCONFIG_DIR/webkit2gtk-web-extension-4.0.pc 2>/dev/null || true; \
				echo "Symlinks created successfully."; \
			else \
				echo "NOTE: You may need to create symlinks manually with:"; \
				echo "  sudo ln -sf /usr/lib64/pkgconfig/webkit2gtk-4.1.pc /usr/lib64/pkgconfig/webkit2gtk-4.0.pc"; \
				echo "  sudo ln -sf /usr/lib64/pkgconfig/webkit2gtk-web-extension-4.1.pc /usr/lib64/pkgconfig/webkit2gtk-web-extension-4.0.pc"; \
			fi; \
		else \
			echo "ERROR: Neither webkit2gtk-4.0 nor webkit2gtk-4.1 found."; \
			echo "Please install webkit2gtk development packages:"; \
			echo "  Fedora: sudo dnf install webkit2gtk4.1-devel"; \
			echo "  Ubuntu/Debian: sudo apt-get install libwebkit2gtk-4.1-dev"; \
			exit 1; \
		fi; \
	else \
		echo "webkit2gtk-4.0 found - no compatibility layer needed."; \
	fi
else
	@echo "macOS detected - skipping webkit check (using native WebKit)"
endif

# Build the GUI application (with frontend pre-build and webkit check)
build-gui: check-webkit build-gui-frontend
	@echo "Building GUI application for $(UNAME_S)/$(UNAME_M)..."
ifeq ($(UNAME_S),Darwin)
	@# On macOS, build for the native architecture
	@if [ "$(UNAME_M)" = "arm64" ]; then \
		echo "Building for macOS ARM64 (M1/M2/M3)..."; \
		$(WAILS) build -platform darwin/arm64 -tags gui; \
	else \
		echo "Building for macOS AMD64 (Intel)..."; \
		$(WAILS) build -platform darwin/amd64 -tags gui; \
	fi
else
	@# On Linux, build for Linux AMD64
	@echo "Building for Linux AMD64..."
	$(WAILS) build -platform linux/amd64 -tags gui
endif

# Build CLI in dev mode (uses ./data/whisper.db in current directory)
build-dev:
	go build -tags '!gui' -ldflags "-X 'github.com/austinwklein/whisper/config.DefaultDBPath=./data/whisper.db'" -o whisper .
	@echo "Built in DEV mode - database will be at ./data/whisper.db"

# Run GUI in development mode
dev-gui: check-webkit
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
