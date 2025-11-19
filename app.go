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

	// Set current user in managers
	fmt.Printf("DEBUG Login: Setting current user in managers: ID=%d, Username=%s\n", user.ID, user.Username)
	a.friendManager.SetCurrentUser(user.ID)
	a.messageManager.SetCurrentUser(user.ID)

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

// Logout logs out the current user
func (a *App) Logout() error {
	a.auth.Logout()
	// Clear current user from managers
	a.friendManager.SetCurrentUser(0)
	a.messageManager.SetCurrentUser(0)
	return nil
}

// GetFriends returns the list of friends for the current user
func (a *App) GetFriends() ([]map[string]interface{}, error) {
	user, err := a.auth.CurrentUser()
	if err != nil {
		return nil, fmt.Errorf("not logged in: %w", err)
	}

	fmt.Printf("DEBUG GetFriends: Current user = %s (ID: %d)\n", user.Username, user.ID)

	friendsList, err := a.storage.GetFriends(a.ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get friends: %w", err)
	}

	fmt.Printf("DEBUG GetFriends: Found %d friends\n", len(friendsList))

	result := make([]map[string]interface{}, len(friendsList))
	for i, friend := range friendsList {
		fmt.Printf("DEBUG GetFriends: Friend %d: ID=%d, Username=%s, FullName=%s, Status=%s\n",
			i, friend.ID, friend.Username, friend.FullName, friend.Status)
		// Check if friend is online (connected to P2P network)
		isOnline := false
		if friend.PeerID != "" {
			// Check if peer is in the peerstore and has open connections
			peerID, err := p2p.ParsePeerID(friend.PeerID)
			if err == nil {
				conns := a.p2p.Host().Network().ConnsToPeer(peerID)
				isOnline = len(conns) > 0
			}
		}

		result[i] = map[string]interface{}{
			"id":       friend.ID,
			"username": friend.Username,
			"fullName": friend.FullName,
			"peerID":   friend.PeerID,
			"status":   friend.Status,
			"online":   isOnline,
		}
	}

	return result, nil
}

// ConnectToPeer connects to a peer via their multiaddress
func (a *App) ConnectToPeer(multiaddr string) error {
	return a.p2p.ConnectToPeer(a.ctx, multiaddr)
}

// SendFriendRequest sends a friend request to a peer
func (a *App) SendFriendRequest(multiaddr, username string) error {
	user, err := a.auth.CurrentUser()
	if err != nil {
		return fmt.Errorf("not logged in: %w", err)
	}

	// First connect to the peer
	if err := a.p2p.ConnectToPeer(a.ctx, multiaddr); err != nil {
		return fmt.Errorf("failed to connect to peer: %w", err)
	}

	// Extract peer ID from multiaddress
	peerID, err := p2p.ExtractPeerIDFromMultiaddr(multiaddr)
	if err != nil {
		return fmt.Errorf("invalid multiaddress: %w", err)
	}

	// Send friend request
	if err := a.friendManager.SendFriendRequest(a.ctx, user, peerID); err != nil {
		return fmt.Errorf("failed to send friend request: %w", err)
	}

	return nil
}

// GetFriendRequests returns pending friend requests
func (a *App) GetFriendRequests() ([]map[string]interface{}, error) {
	user, err := a.auth.CurrentUser()
	if err != nil {
		return nil, fmt.Errorf("not logged in: %w", err)
	}

	fmt.Printf("DEBUG GetFriendRequests: Current user = %s (ID: %d)\n", user.Username, user.ID)

	requests, err := a.storage.GetPendingFriendRequests(a.ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get friend requests: %w", err)
	}

	fmt.Printf("DEBUG GetFriendRequests: Found %d pending requests\n", len(requests))

	result := make([]map[string]interface{}, len(requests))
	for i, req := range requests {
		fmt.Printf("DEBUG GetFriendRequests: Request %d: ID=%d, UserID=%d, FriendID=%d, Username=%s, FullName=%s\n",
			i, req.ID, req.UserID, req.FriendID, req.Username, req.FullName)

		result[i] = map[string]interface{}{
			"id":       req.ID,
			"username": req.Username,
			"fullName": req.FullName,
			"peerID":   req.PeerID,
			"status":   req.Status,
		}
	}

	return result, nil
}

// AcceptFriendRequest accepts a friend request
func (a *App) AcceptFriendRequest(username string) error {
	user, err := a.auth.CurrentUser()
	if err != nil {
		return fmt.Errorf("not logged in: %w", err)
	}

	return a.friendManager.AcceptFriendRequest(a.ctx, user, username)
}

// RejectFriendRequest rejects a friend request
func (a *App) RejectFriendRequest(username string) error {
	user, err := a.auth.CurrentUser()
	if err != nil {
		return fmt.Errorf("not logged in: %w", err)
	}

	return a.friendManager.RejectFriendRequest(a.ctx, user, username)
}

// SendMessage sends a direct message to a friend
func (a *App) SendMessage(username, content string) error {
	user, err := a.auth.CurrentUser()
	if err != nil {
		return fmt.Errorf("not logged in: %w", err)
	}

	return a.messageManager.SendMessage(a.ctx, user, username, content)
}

// GetMessages returns message history with a friend
func (a *App) GetMessages(username string, limit int) ([]map[string]interface{}, error) {
	user, err := a.auth.CurrentUser()
	if err != nil {
		return nil, fmt.Errorf("not logged in: %w", err)
	}

	if limit <= 0 {
		limit = 50
	}

	fmt.Printf("DEBUG GetMessages: Current user = %s (ID: %d), looking up friend '%s'\n", user.Username, user.ID, username)

	// Get friend info - first try direct lookup
	friend, err := a.storage.GetUserByUsername(a.ctx, username)
	if err != nil {
		fmt.Printf("DEBUG GetMessages: GetUserByUsername error: %v\n", err)
		return nil, fmt.Errorf("failed to get friend: %w", err)
	}
	if friend == nil {
		fmt.Printf("DEBUG GetMessages: Friend '%s' not found by username, checking friends table\n", username)
		// Try to find via friend record
		friends, err := a.storage.GetFriends(a.ctx, user.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get friends: %w", err)
		}

		// Find the friend with matching username
		var friendUserID int64
		for _, f := range friends {
			fmt.Printf("DEBUG GetMessages: Checking friend: FriendID=%d, Username=%s\n", f.FriendID, f.Username)
			if f.Username == username {
				friendUserID = f.FriendID
				break
			}
		}

		if friendUserID == 0 {
			return nil, fmt.Errorf("friend '%s' not found", username)
		}

		// Look up by ID instead
		fmt.Printf("DEBUG GetMessages: Found friend record, looking up user by ID: %d\n", friendUserID)
		friend, err = a.storage.GetUserByID(a.ctx, friendUserID)
		if err != nil || friend == nil {
			return nil, fmt.Errorf("failed to get user by ID: %w", err)
		}
		fmt.Printf("DEBUG GetMessages: Found user by ID: Username=%s, ID=%d\n", friend.Username, friend.ID)
	} else {
		fmt.Printf("DEBUG GetMessages: Found friend '%s' (ID: %d)\n", friend.Username, friend.ID)
	}

	// Use GetMessages instead of GetMessageHistory
	fmt.Printf("DEBUG GetMessages: Querying messages between user.ID=%d and friend.ID=%d\n", user.ID, friend.ID)
	messages, err := a.storage.GetMessages(a.ctx, user.ID, friend.ID, limit)
	if err != nil {
		fmt.Printf("DEBUG GetMessages: Query error: %v\n", err)
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	fmt.Printf("DEBUG GetMessages: Found %d messages\n", len(messages))

	result := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		fmt.Printf("DEBUG GetMessages: Message %d: FromUserID=%d, ToUserID=%d, Content=%s\n", i, msg.FromUserID, msg.ToUserID, msg.Content)
		result[i] = map[string]interface{}{
			"id":        msg.ID,
			"content":   msg.Content,
			"fromMe":    msg.FromUserID == user.ID,
			"delivered": msg.Delivered,
			"read":      msg.Read,
			"createdAt": msg.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	return result, nil
}

// GetUnreadCount returns the count of unread messages
func (a *App) GetUnreadCount() (int, error) {
	user, err := a.auth.CurrentUser()
	if err != nil {
		return 0, fmt.Errorf("not logged in: %w", err)
	}

	// Get all undelivered messages to the user
	messages, err := a.storage.GetUndeliveredMessages(a.ctx, user.ID)
	if err != nil {
		return 0, fmt.Errorf("failed to get unread messages: %w", err)
	}

	// Count messages that are unread
	count := 0
	for _, msg := range messages {
		if !msg.Read {
			count++
		}
	}

	return count, nil
}
