# Wails GUI Setup

This document explains the Wails GUI integration for Whisper.

## Overview

Whisper now has two interfaces:
1. **CLI** - Command-line interface (original)
2. **GUI** - Desktop application with Wails framework

Both share the same backend code (p2p, storage, auth, friends, messages, conference).

## Architecture

### Build Tags

The project uses Go build tags to separate CLI and GUI builds:

- **CLI**: Uses `!gui` tag, builds `main_cli.go`
- **GUI**: Uses `gui` tag, builds `main_gui.go`

### Directory Structure

```
whisper/
├── main_cli.go           # CLI entry point (build tag: !gui)
├── main_gui.go           # GUI entry point (build tag: gui)
├── app/
│   └── app.go           # GUI application logic
├── frontend/
│   ├── src/
│   │   ├── App.svelte   # Main Svelte component
│   │   ├── main.ts      # Frontend entry point
│   │   └── wailsjs/     # Generated Wails bindings
│   ├── dist/            # Built frontend assets
│   ├── package.json
│   ├── vite.config.js
│   └── index.html
├── build/
│   └── bin/             # Built GUI application
└── wails.json           # Wails configuration
```

## Building

### Prerequisites

1. **Go** 1.24+
2. **Node.js** 25+ and npm
3. **Wails CLI**: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
4. **Xcode Command Line Tools** (macOS)

### Build Commands

```bash
# Build CLI (default)
make build

# Build GUI application
make build-gui

# Build CLI in dev mode (local database)
make build-dev
```

### Manual Build Commands

```bash
# CLI
go build -tags '!gui' -o whisper .

# GUI
wails build -tags gui
```

## Development

### CLI Development

```bash
# Run CLI in dev mode
make dev

# Or directly
go run -tags '!gui' main_cli.go
```

### GUI Development

```bash
# Run GUI with hot reload
make dev-gui

# Or directly
wails dev -tags gui
```

The GUI dev mode provides:
- Hot reload for frontend changes
- Auto-rebuild for backend changes
- Development tools in the browser

## Running

### CLI

```bash
# Run built CLI
./whisper

# Or
make run
```

### GUI

```bash
# macOS
open build/bin/whisper.app

# Or double-click the app in Finder
```

## Frontend Stack

- **Framework**: Svelte 5
- **Build Tool**: Vite 7
- **Language**: TypeScript
- **Styling**: CSS (scoped in Svelte components)

## Backend Integration

The GUI communicates with the Go backend through Wails bindings:

### Go Methods (app/app.go)

```go
// Authentication
Register(username, password, fullName string) error
Login(username, password string) error
Logout() error
GetCurrentUser() map[string]interface{}
IsLoggedIn() bool

// P2P Info
GetPeerInfo() string
GetMultiaddr() string
```

### Frontend Usage (TypeScript)

```typescript
import { Register, Login, GetPeerInfo } from './wailsjs/go/app/App'

// Register user
await Register("alice", "password123", "Alice Smith")

// Login
await Login("alice", "password123")

// Get peer ID
const peerID = await GetPeerInfo()
```

## Current GUI Features (Phase 6.1)

### Implemented
- ✅ Basic UI layout
- ✅ User registration
- ✅ User login/logout
- ✅ Peer ID display
- ✅ Connection status indicator

### TODO (Phase 6.2+)
- ⏭️ Friend list with online status
- ⏭️ Friend requests (send/accept/reject)
- ⏭️ Direct messaging
- ⏭️ Message history
- ⏭️ Conference creation and management
- ⏭️ Conference messaging
- ⏭️ Peer connection interface
- ⏭️ Notifications
- ⏭️ Settings panel

## Configuration

### wails.json

```json
{
  "name": "whisper",
  "outputfilename": "whisper",
  "frontend:install": "npm install",
  "frontend:build": "npm run build",
  "frontend:dev:watcher": "npm run dev",
  "frontend:dev:serverUrl": "http://localhost:5173",
  "wailsjsdir": "./frontend/src",
  "version": "2",
  "outputType": "desktop"
}
```

## Troubleshooting

### Issue: "main redeclared"

**Cause**: Both `main_cli.go` and `main_gui.go` are being compiled

**Solution**: Use build tags:
```bash
# CLI
go build -tags '!gui' -o whisper .

# GUI
wails build -tags gui
```

### Issue: Wails bindings not found

**Cause**: Bindings not generated

**Solution**:
```bash
wails generate module -tags gui
```

### Issue: Frontend build fails

**Cause**: Dependencies not installed

**Solution**:
```bash
cd frontend
npm install
npm run build
```

### Issue: Application won't start

**Cause**: Database initialization error

**Solution**:
```bash
# Check database path
echo $WHISPER_DB

# Reset database
rm -rf ~/.whisper
```

## Makefile Targets

```bash
make build          # Build CLI
make build-gui      # Build GUI
make build-dev      # Build CLI (dev mode)
make dev            # Run CLI dev mode
make dev-gui        # Run GUI dev mode
make clean          # Clean all build artifacts
make test           # Run tests
```

## Known Limitations

1. **Platform Support**: Currently only tested on macOS
2. **Single Instance**: GUI doesn't support multiple simultaneous users
3. **No Message Encryption**: Messages are plaintext (planned for v0.2)
4. **Ephemeral Peer IDs**: New peer ID on each restart

## Future Enhancements (Phase 6.2+)

1. **Windows Support**: Build for Windows with `.exe`
2. **Linux Support**: Build for Linux with AppImage
3. **System Tray**: Minimize to tray, show notifications
4. **Auto-update**: Check for updates on startup
5. **Themes**: Light/dark mode toggle
6. **Emoji Picker**: Rich message formatting
7. **File Sharing**: Send files to friends
8. **Voice/Video**: WebRTC integration

## Contributing

When adding new backend methods for GUI:

1. Add method to `app/app.go`
2. Regenerate bindings: `wails generate module -tags gui`
3. Use method in frontend: `import { MethodName } from './wailsjs/go/app/App'`
4. Test in dev mode: `make dev-gui`

---

**Last Updated**: 2025-11-18
**Phase**: 6.1 - Basic GUI Setup Complete
**Status**: CLI and GUI both functional, ready for feature expansion
