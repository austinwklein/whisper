# Build Guide

## Quick Reference

### Common Commands

```bash
# Clean the database (fixes "already exists" errors)
make clean-db

# Full reset (build artifacts + database)
make reset

# Build GUI application
make build-gui

# Build CLI application
make build

# Run GUI in dev mode (with hot reload)
make dev-gui
```

### Cross-Platform Building

When switching between machines (e.g., from ARM to x86):

```bash
# On the old machine (or before pushing to git)
make clean-all    # Removes node_modules and all build artifacts

# On the new machine (or after pulling from git)
make setup        # Installs all dependencies
make build-gui    # Build the GUI
```

### Database Management

```bash
# Clean production database (~/.whisper)
make clean-db

# Clean dev database (./data)
make clean-db-dev

# Full reset with database
make reset        # Production
make reset-dev    # Dev mode
```

### Development Workflow

**For GUI Development:**
```bash
make dev-gui      # Starts dev server with hot reload
```

**For CLI Development:**
```bash
make build-dev    # Uses local ./data/whisper.db
./whisper
```

**For Testing Multiple Instances:**
```bash
# Terminal 1 - Alice
make clean-db
make build-gui
./build/bin/whisper.app/Contents/MacOS/whisper

# Terminal 2 - Bob  
# (Uses different port automatically)
./build/bin/whisper.app/Contents/MacOS/whisper
```

## Troubleshooting

### Error: "pattern all:frontend/dist: no matching files found"

**Cause:** Frontend hasn't been built yet

**Solution:**
```bash
make clean
make build-gui    # Now includes frontend pre-build
```

### Error: "database is locked" or "username already exists"

**Cause:** Multiple instances using same database or stale data

**Solution:**
```bash
make clean-db     # Clean database
# Restart application
```

### Build fails on different machine/architecture

**Cause:** node_modules compiled for different architecture

**Solution:**
```bash
make clean-all    # Remove everything
make setup        # Reinstall dependencies
make build-gui    # Build fresh
```

### Frontend changes not showing up

**Cause:** Vite cache or stale dist files

**Solution:**
```bash
make clean        # Removes frontend/dist and cache
make build-gui    # Rebuild
```

## Build Targets

| Command | Description |
|---------|-------------|
| `make build` | Build CLI (production mode) |
| `make build-gui` | Build GUI with .app bundle |
| `make build-dev` | Build CLI (dev mode, local DB) |
| `make build-gui-frontend` | Build only frontend assets |
| `make dev-gui` | Run GUI with hot reload |
| `make clean` | Remove build artifacts |
| `make clean-all` | Remove everything including node_modules |
| `make clean-db` | Delete production database |
| `make clean-db-dev` | Delete dev database |
| `make reset` | Full reset (clean + clean-db) |
| `make setup` | Install all dependencies |

## Architecture Support

The Makefile automatically detects your system architecture and builds accordingly:
- **macOS ARM (M1/M2/M3)**: darwin/arm64
- **macOS Intel**: darwin/amd64
- **Linux**: linux/amd64 or linux/arm64
- **Windows**: windows/amd64

Wails handles the platform detection automatically.

## CI/CD Notes

For automated builds:

```bash
# Clean build from scratch
make clean-all
make setup
make build-gui

# Or use reset for database cleanup too
make reset
make setup
make build-gui
```
