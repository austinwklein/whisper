package friends

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/austinwklein/whisper/storage"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

var (
	ErrNotAuthenticated = errors.New("not authenticated")
	ErrAlreadyFriends   = errors.New("already friends")
	ErrPendingRequest   = errors.New("friend request already pending")
	ErrRequestNotFound  = errors.New("friend request not found")
	ErrCannotAddSelf    = errors.New("cannot add yourself as friend")
)

// Manager handles friend operations
type Manager struct {
	storage       storage.Storage
	host          host.Host
	protocol      *Protocol
	currentUserID int64
}

// NewManager creates a new friend manager
func NewManager(store storage.Storage, h host.Host) *Manager {
	protocol := NewProtocol()

	mgr := &Manager{
		storage:  store,
		host:     h,
		protocol: protocol,
	}

	// Set up protocol handlers
	protocol.SetRequestHandler(mgr.handleIncomingRequest)
	protocol.SetAcceptHandler(mgr.handleIncomingAccept)
	protocol.SetRejectHandler(mgr.handleIncomingReject)

	// Register stream handlers
	h.SetStreamHandler(ProtocolFriendRequest, protocol.HandleFriendRequest)
	h.SetStreamHandler(ProtocolFriendAccept, protocol.HandleFriendAccept)
	h.SetStreamHandler(ProtocolFriendReject, protocol.HandleFriendReject)

	return mgr
}

// SetCurrentUser sets the currently logged-in user
func (m *Manager) SetCurrentUser(userID int64) {
	m.currentUserID = userID
}

// SendFriendRequest sends a friend request to another user
func (m *Manager) SendFriendRequest(ctx context.Context, currentUser *storage.User, targetPeerID peer.ID) error {
	if m.currentUserID == 0 {
		return ErrNotAuthenticated
	}

	// Don't allow adding yourself
	if targetPeerID.String() == currentUser.PeerID {
		return ErrCannotAddSelf
	}

	// Check if already friends or request pending
	// First, we need to get the target user's ID from their peer ID
	targetUser, err := m.storage.GetUserByPeerID(ctx, targetPeerID.String())
	if err != nil {
		return fmt.Errorf("failed to get target user: %w", err)
	}
	if targetUser == nil {
		return errors.New("target user not found")
	}

	// Check if friend relationship already exists
	existingFriend, err := m.storage.GetFriendRequest(ctx, currentUser.ID, targetUser.ID)
	if err != nil {
		return fmt.Errorf("failed to check existing friendship: %w", err)
	}
	if existingFriend != nil {
		if existingFriend.Status == "accepted" {
			return ErrAlreadyFriends
		}
		return ErrPendingRequest
	}

	// Create friend request in database
	friend := &storage.Friend{
		UserID:   currentUser.ID,
		FriendID: targetUser.ID,
		PeerID:   targetUser.PeerID,
		Username: targetUser.Username,
		FullName: targetUser.FullName,
		Status:   "pending",
	}

	if err := m.storage.CreateFriendRequest(ctx, friend); err != nil {
		return fmt.Errorf("failed to create friend request: %w", err)
	}

	// Send friend request over P2P
	stream, err := m.host.NewStream(ctx, targetPeerID, ProtocolFriendRequest)
	if err != nil {
		return fmt.Errorf("failed to open stream: %w", err)
	}

	request := &FriendRequestMessage{
		FromUsername: currentUser.Username,
		FromFullName: currentUser.FullName,
		FromPeerID:   currentUser.PeerID,
		Message:      fmt.Sprintf("%s wants to be your friend", currentUser.FullName),
	}

	if err := SendFriendRequest(ctx, stream, request); err != nil {
		return fmt.Errorf("failed to send friend request: %w", err)
	}

	fmt.Printf("✓ Friend request sent to %s (%s)\n", targetUser.FullName, targetUser.Username)
	return nil
}

// AcceptFriendRequest accepts a pending friend request
func (m *Manager) AcceptFriendRequest(ctx context.Context, currentUser *storage.User, fromUsername string) error {
	if m.currentUserID == 0 {
		return ErrNotAuthenticated
	}

	// Get the requesting user
	fromUser, err := m.storage.GetUserByUsername(ctx, fromUsername)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if fromUser == nil {
		return errors.New("requesting user not found")
	}

	// Get the friend request (where fromUser sent request to currentUser)
	friendRequest, err := m.storage.GetFriendRequest(ctx, fromUser.ID, currentUser.ID)
	if err != nil {
		return fmt.Errorf("failed to get friend request: %w", err)
	}
	if friendRequest == nil {
		return ErrRequestNotFound
	}

	if friendRequest.Status != "pending" {
		return errors.New("request is not pending")
	}

	// Update request status
	friendRequest.Status = "accepted"
	now := time.Now()
	friendRequest.AcceptedAt = now

	if err := m.storage.UpdateFriendRequest(ctx, friendRequest); err != nil {
		return fmt.Errorf("failed to update friend request: %w", err)
	}

	// Create reciprocal friendship (currentUser -> fromUser)
	reciprocalFriend := &storage.Friend{
		UserID:     currentUser.ID,
		FriendID:   fromUser.ID,
		PeerID:     fromUser.PeerID,
		Username:   fromUser.Username,
		FullName:   fromUser.FullName,
		Status:     "accepted",
		AcceptedAt: now,
	}

	if err := m.storage.CreateFriendRequest(ctx, reciprocalFriend); err != nil {
		return fmt.Errorf("failed to create reciprocal friendship: %w", err)
	}

	// Send acceptance notification
	peerID, err := peer.Decode(fromUser.PeerID)
	if err != nil {
		return fmt.Errorf("invalid peer ID: %w", err)
	}

	stream, err := m.host.NewStream(ctx, peerID, ProtocolFriendAccept)
	if err != nil {
		// Not fatal if we can't notify - friendship is still established
		fmt.Printf("Warning: Could not notify peer of acceptance: %v\n", err)
	} else {
		response := &FriendResponseMessage{
			Accepted: true,
			Username: currentUser.Username,
			FullName: currentUser.FullName,
			PeerID:   currentUser.PeerID,
			Message:  fmt.Sprintf("%s accepted your friend request", currentUser.FullName),
		}
		SendFriendResponse(ctx, stream, response)
	}

	fmt.Printf("✓ Accepted friend request from %s\n", fromUser.FullName)
	return nil
}

// RejectFriendRequest rejects a pending friend request
func (m *Manager) RejectFriendRequest(ctx context.Context, currentUser *storage.User, fromUsername string) error {
	if m.currentUserID == 0 {
		return ErrNotAuthenticated
	}

	// Get the requesting user
	fromUser, err := m.storage.GetUserByUsername(ctx, fromUsername)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if fromUser == nil {
		return errors.New("requesting user not found")
	}

	// Get the friend request
	friendRequest, err := m.storage.GetFriendRequest(ctx, fromUser.ID, currentUser.ID)
	if err != nil {
		return fmt.Errorf("failed to get friend request: %w", err)
	}
	if friendRequest == nil {
		return ErrRequestNotFound
	}

	if friendRequest.Status != "pending" {
		return errors.New("request is not pending")
	}

	// Update request status
	friendRequest.Status = "rejected"
	if err := m.storage.UpdateFriendRequest(ctx, friendRequest); err != nil {
		return fmt.Errorf("failed to update friend request: %w", err)
	}

	// Send rejection notification
	peerID, err := peer.Decode(fromUser.PeerID)
	if err != nil {
		return fmt.Errorf("invalid peer ID: %w", err)
	}

	stream, err := m.host.NewStream(ctx, peerID, ProtocolFriendReject)
	if err != nil {
		fmt.Printf("Warning: Could not notify peer of rejection: %v\n", err)
	} else {
		response := &FriendResponseMessage{
			Accepted: false,
			Username: currentUser.Username,
			FullName: currentUser.FullName,
			PeerID:   currentUser.PeerID,
			Message:  "Friend request was declined",
		}
		SendFriendResponse(ctx, stream, response)
	}

	fmt.Printf("✓ Rejected friend request from %s\n", fromUser.FullName)
	return nil
}

// GetFriends returns all accepted friends
func (m *Manager) GetFriends(ctx context.Context, userID int64) ([]*storage.Friend, error) {
	return m.storage.GetFriends(ctx, userID)
}

// GetPendingRequests returns all pending friend requests for a user
func (m *Manager) GetPendingRequests(ctx context.Context, userID int64) ([]*storage.Friend, error) {
	return m.storage.GetPendingFriendRequests(ctx, userID)
}

// Protocol message handlers
func (m *Manager) handleIncomingRequest(request *FriendRequestMessage, fromPeer peer.ID) {
	ctx := context.Background()

	// First, check if this user exists in our database, if not create them
	fromUser, err := m.storage.GetUserByUsername(ctx, request.FromUsername)
	if err != nil || fromUser == nil {
		// User doesn't exist - this is normal in P2P when someone contacts us
		// Create a basic user record so we can store the friend request
		fromUser = &storage.User{
			Username:     request.FromUsername,
			PasswordHash: "P2P_REMOTE_USER", // Placeholder - they registered on another peer
			FullName:     request.FromFullName,
			PeerID:       request.FromPeerID,
		}
		if err := m.storage.CreateUser(ctx, fromUser); err != nil {
			fmt.Printf("Error creating user record for %s: %v\n", request.FromUsername, err)
			return
		}
	}

	// Get current user
	if m.currentUserID == 0 {
		fmt.Printf("\n📨 Friend request from %s (%s) - login to accept/reject\n", request.FromFullName, request.FromUsername)
		return
	}

	currentUser, err := m.storage.GetUserByID(ctx, m.currentUserID)
	if err != nil || currentUser == nil {
		fmt.Printf("Error: Could not get current user\n")
		return
	}

	// If fromUser exists in DB, create the friend request record
	if fromUser != nil && fromUser.ID > 0 {
		// Check if request already exists
		existing, _ := m.storage.GetFriendRequest(ctx, fromUser.ID, currentUser.ID)
		if existing != nil {
			fmt.Printf("\n📨 Friend request from %s (%s) already exists\n", request.FromFullName, request.FromUsername)
			return
		}

		// Create friend request
		friendReq := &storage.Friend{
			UserID:   fromUser.ID,
			FriendID: currentUser.ID,
			PeerID:   fromUser.PeerID,
			Username: fromUser.Username,
			FullName: fromUser.FullName,
			Status:   "pending",
		}

		if err := m.storage.CreateFriendRequest(ctx, friendReq); err != nil {
			fmt.Printf("Error saving friend request: %v\n", err)
		}
	}

	fmt.Printf("\n📨 Friend request from %s (%s)\n", request.FromFullName, request.FromUsername)
	fmt.Printf("   Message: %s\n", request.Message)
	fmt.Printf("   Use 'accept %s' or 'reject %s'\n", request.FromUsername, request.FromUsername)
	fmt.Print("> ")
}

func (m *Manager) handleIncomingAccept(response *FriendResponseMessage, fromPeer peer.ID) {
	fmt.Printf("\n✓ %s accepted your friend request!\n", response.FullName)
	fmt.Printf("   You are now friends with %s (%s)\n", response.FullName, response.Username)
	fmt.Print("> ")
}

func (m *Manager) handleIncomingReject(response *FriendResponseMessage, fromPeer peer.ID) {
	fmt.Printf("\n✗ %s declined your friend request\n", response.FullName)
	fmt.Print("> ")
}
