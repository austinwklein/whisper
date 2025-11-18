package main

import (
	"context"
	"fmt"
	"log"

	"github.com/austinwklein/whisper/auth"
	"github.com/austinwklein/whisper/conference"
	"github.com/austinwklein/whisper/config"
	"github.com/austinwklein/whisper/friends"
	"github.com/austinwklein/whisper/messages"
	"github.com/austinwklein/whisper/p2p"
	"github.com/austinwklein/whisper/storage"
)

// App struct
type App struct {
	ctx               context.Context
	config            *config.Config
	storage           storage.Storage
	p2p               *p2p.P2PHost
	auth              *auth.AuthService
	friendManager     *friends.Manager
	messageManager    *messages.Manager
	conferenceManager *conference.Manager
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Initialize Whisper components
	var err error

	// Load config
	a.config, err = config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize storage
	a.storage, err = storage.NewSQLiteStorage(a.config.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Initialize P2P host
	a.p2p, err = p2p.NewP2PHost(ctx, a.config.Port, nil)
	if err != nil {
		log.Fatalf("Failed to initialize P2P host: %v", err)
	}

	// Initialize auth service
	a.auth = auth.NewAuthService(a.storage)

	// Initialize managers
	a.friendManager = friends.NewManager(a.storage, a.p2p.Host())
	a.messageManager = messages.NewManager(a.storage, a.p2p.Host())
	a.conferenceManager = conference.NewManager(a.storage, a.p2p.Host(), a.p2p.PubSub())

	log.Println("Whisper GUI initialized")
}

// Register creates a new user account
func (a *App) Register(username, password, fullName string) error {
	peerID := a.p2p.Host().ID().String()
	return a.auth.Register(a.ctx, username, password, fullName, peerID)
}

// Login authenticates a user
func (a *App) Login(username, password string) error {
	user, err := a.auth.Login(a.ctx, username, password)
	if err != nil {
		return err
	}

	// Update peer ID in database
	peerID := a.p2p.Host().ID().String()
	user.PeerID = peerID
	if err := a.storage.UpdateUser(a.ctx, user); err != nil {
		return fmt.Errorf("failed to update peer ID: %w", err)
	}

	return nil
}

// IsLoggedIn checks if a user is currently logged in
func (a *App) IsLoggedIn() bool {
	return a.auth.IsAuthenticated()
}

// GetCurrentUser returns the currently logged in user
func (a *App) GetCurrentUser() map[string]interface{} {
	user, err := a.auth.CurrentUser()
	if err != nil || user == nil {
		return nil
	}

	return map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"fullName": user.FullName,
		"peerID":   user.PeerID,
	}
}

// GetPeerInfo returns the peer ID
func (a *App) GetPeerInfo() string {
	return a.p2p.Host().ID().String()
}

// GetMultiaddr returns the multiaddress for this peer
func (a *App) GetMultiaddr() string {
	addrs := a.p2p.Host().Addrs()
	if len(addrs) > 0 {
		peerID := a.p2p.Host().ID()
		return fmt.Sprintf("%s/p2p/%s", addrs[0], peerID)
	}
	return ""
}
