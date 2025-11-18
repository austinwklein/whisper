package app

import (
	"context"
	"fmt"

	"github.com/austinwklein/whisper/auth"
	"github.com/austinwklein/whisper/conference"
	"github.com/austinwklein/whisper/config"
	"github.com/austinwklein/whisper/friends"
	"github.com/austinwklein/whisper/messages"
	"github.com/austinwklein/whisper/p2p"
	"github.com/austinwklein/whisper/storage"
)

// App struct holds the application state
type App struct {
	ctx               context.Context
	config            *config.Config
	storage           storage.Storage
	p2p               *p2p.P2PHost
	auth              *auth.AuthService
	friendManager     *friends.Manager
	messageManager    *messages.Manager
	conferenceManager *conference.Manager
	currentUser       *storage.User
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) Startup(ctx context.Context) error {
	a.ctx = ctx

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	a.config = cfg

	// Initialize storage
	store, err := storage.NewSQLiteStorage(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	a.storage = store

	// Initialize auth service
	a.auth = auth.NewAuthService(a.storage)

	fmt.Println("Whisper GUI initialized")

	return nil
}

// Shutdown is called at application termination
func (a *App) Shutdown(ctx context.Context) error {
	// Close P2P host if running
	if a.p2p != nil {
		a.p2p.Close()
	}

	// Close storage
	if a.storage != nil {
		a.storage.Close()
	}

	return nil
}

// GetPeerInfo returns the current peer information
func (a *App) GetPeerInfo() string {
	if a.p2p == nil {
		return "Not connected"
	}
	return a.p2p.Host().ID().String()
}

// GetMultiaddr returns the full multiaddress of the current peer
func (a *App) GetMultiaddr() string {
	if a.p2p == nil {
		return "Not connected"
	}
	addrs := a.p2p.GetFullAddrs()
	if len(addrs) > 0 {
		return addrs[0]
	}
	return "No addresses available"
}

// Register creates a new user account
func (a *App) Register(username, password, fullName string) error {
	// Create a temporary P2P host to get a peer ID
	tempCtx := context.Background()
	tempP2P, err := p2p.NewP2PHost(tempCtx, a.config.Port, nil)
	if err != nil {
		return fmt.Errorf("failed to create temporary P2P host: %w", err)
	}
	peerID := tempP2P.Host().ID().String()
	tempP2P.Close()

	return a.auth.Register(a.ctx, username, password, fullName, peerID)
}

// Login authenticates a user
func (a *App) Login(username, password string) error {
	user, err := a.auth.Login(a.ctx, username, password)
	if err != nil {
		return err
	}

	a.currentUser = user

	// Initialize P2P host after successful login
	p2pHost, err := p2p.NewP2PHost(a.ctx, a.config.Port, nil)
	if err != nil {
		return fmt.Errorf("failed to initialize P2P host: %w", err)
	}
	a.p2p = p2pHost

	// Update user's peer ID in database
	user.PeerID = p2pHost.Host().ID().String()
	err = a.storage.UpdateUser(a.ctx, user)
	if err != nil {
		return fmt.Errorf("failed to update peer ID: %w", err)
	}

	// Initialize managers
	a.friendManager = friends.NewManager(a.storage, p2pHost.Host())
	a.friendManager.SetCurrentUser(user.ID)

	a.messageManager = messages.NewManager(a.storage, p2pHost.Host())
	a.messageManager.SetCurrentUser(user.ID)

	a.conferenceManager = conference.NewManager(a.storage, p2pHost.Host(), p2pHost.PubSub())
	a.conferenceManager.SetCurrentUser(user.ID)

	return nil
}

// Logout logs out the current user
func (a *App) Logout() error {
	// Close P2P connection
	if a.p2p != nil {
		a.p2p.Close()
		a.p2p = nil
	}

	// Clear managers
	a.friendManager = nil
	a.messageManager = nil
	a.conferenceManager = nil
	a.currentUser = nil

	return nil
}

// GetCurrentUser returns the currently logged in user info
func (a *App) GetCurrentUser() map[string]interface{} {
	if a.currentUser == nil {
		return nil
	}

	return map[string]interface{}{
		"id":       a.currentUser.ID,
		"username": a.currentUser.Username,
		"fullName": a.currentUser.FullName,
		"peerID":   a.currentUser.PeerID,
	}
}

// IsLoggedIn returns whether a user is currently logged in
func (a *App) IsLoggedIn() bool {
	return a.currentUser != nil
}
