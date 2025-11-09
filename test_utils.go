package main

import (
	"context"
	"testing"

	"github.com/austinwklein/whisper/config"
	"github.com/austinwklein/whisper/p2p"
	"github.com/austinwklein/whisper/storage"
)

// NewTestApp TestApp creates a test app instance
func NewTestApp(t *testing.T) *App {
	// Create test storage (would need a mock implementation)
	// For now, we'll use SQLite with a temp file
	store, _ := storage.NewSQLiteStorage(":memory:")

	// Create test P2P host
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	p2pHost, _ := p2p.NewP2PHost(ctx, 0, nil) // Port 0 = random free port

	return &App{
		config: &config.Config{
			Port:     9999,
			LogLevel: "debug",
		},
		storage: store,
		p2p:     p2pHost,
	}
}
