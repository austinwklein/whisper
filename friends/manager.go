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
	fmt.Printf("DEBUG SetCurrentUser called: old=%d, new=%d\n", m.currentUserID, userID)
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

	// Check if target user exists in our local database
	targetUser, err := m.storage.GetUserByPeerID(ctx, targetPeerID.String())

	// If target user doesn't exist, create a placeholder record
	if err != nil || targetUser == nil {
		fmt.Printf("DEBUG SendFriendRequest: Target user not in DB, creating placeholder\n")
		// Create a placeholder user record - will be updated when they respond
		// Use full peer ID as temporary username to avoid conflicts (peer IDs are unique)
		placeholderUsername := fmt.Sprintf("unknown_%s", targetPeerID.String())
		targetUser = &storage.User{
			Username:     placeholderUsername, // Temporary unique username
			PasswordHash: "P2P_REMOTE_USER",
			FullName:     "Unknown User",
			PeerID:       targetPeerID.String(),
		}
		if err := m.storage.CreateUser(ctx, targetUser); err != nil {
			// If creation fails, it might be a race condition or duplicate
			// Try to fetch the user again by peer ID
			fmt.Printf("DEBUG SendFriendRequest: CreateUser failed (%v), retrying lookup\n", err)
			targetUser, err = m.storage.GetUserByPeerID(ctx, targetPeerID.String())
			if err != nil || targetUser == nil {
				return fmt.Errorf("failed to create or retrieve placeholder user: %w", err)
			}
			fmt.Printf("DEBUG SendFriendRequest: Found existing user after retry (ID: %d, Username: %s)\n", targetUser.ID, targetUser.Username)
		} else {
			fmt.Printf("DEBUG SendFriendRequest: Created placeholder user (ID: %d, Username: %s)\n", targetUser.ID, targetUser.Username)
		}
	} else {
		fmt.Printf("DEBUG SendFriendRequest: Found existing user (ID: %d, Username: %s)\n", targetUser.ID, targetUser.Username)
	}

	// Check if already friends or request pending
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

	// Create friend request in database (on sender's side)
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
	fmt.Printf("DEBUG SendFriendRequest: Created pending request on sender side: %d -> %d\n",
		currentUser.ID, targetUser.ID)

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

	if targetUser != nil {
		fmt.Printf("✓ Friend request sent to %s (%s)\n", targetUser.FullName, targetUser.Username)
	} else {
		fmt.Printf("✓ Friend request sent to peer %s\n", targetPeerID.String()[:16]+"...")
	}
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

	fmt.Printf("DEBUG: Creating reciprocal friendship: UserID=%d (%s) -> FriendID=%d (%s)\n",
		reciprocalFriend.UserID, currentUser.Username, reciprocalFriend.FriendID, fromUser.Username)

	if err := m.storage.CreateFriendRequest(ctx, reciprocalFriend); err != nil {
		return fmt.Errorf("failed to create reciprocal friendship: %w", err)
	}

	fmt.Printf("DEBUG: Reciprocal friendship created successfully\n")

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
		// IMPORTANT: Store the requester's info (fromUser) in username/fullname fields
		// This is what will be displayed to the recipient
		friendReq := &storage.Friend{
			UserID:   fromUser.ID,       // ID of requester (e.g., Bob's ID)
			FriendID: currentUser.ID,    // ID of recipient (e.g., Alice's ID)
			PeerID:   fromUser.PeerID,   // PeerID of requester
			Username: fromUser.Username, // Username of requester (e.g., "bob")
			FullName: fromUser.FullName, // Full name of requester (e.g., "Bob Jones")
			Status:   "pending",
		}

		fmt.Printf("DEBUG: Creating friend request: UserID=%d (%s), FriendID=%d (%s), Username=%s, FullName=%s\n",
			friendReq.UserID, fromUser.Username, friendReq.FriendID, currentUser.Username,
			friendReq.Username, friendReq.FullName)

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
	ctx := context.Background()

	fmt.Printf("DEBUG handleIncomingAccept: m.currentUserID = %d\n", m.currentUserID)

	// Ensure the accepting user exists in our database
	// First try by username, then by peer ID (in case placeholder was created)
	acceptingUser, err := m.storage.GetUserByUsername(ctx, response.Username)
	if err != nil || acceptingUser == nil {
		// Try by peer ID (might be a placeholder user)
		acceptingUser, err = m.storage.GetUserByPeerID(ctx, response.PeerID)
		if err != nil || acceptingUser == nil {
			// Still doesn't exist, create new user
			acceptingUser = &storage.User{
				Username:     response.Username,
				PasswordHash: "P2P_REMOTE_USER",
				FullName:     response.FullName,
				PeerID:       response.PeerID,
			}
			if err := m.storage.CreateUser(ctx, acceptingUser); err != nil {
				fmt.Printf("Error creating user record for %s: %v\n", response.Username, err)
				return
			}
			fmt.Printf("DEBUG handleIncomingAccept: Created user record for %s (ID: %d)\n", acceptingUser.Username, acceptingUser.ID)
		} else {
			// Found by peer ID - update the placeholder with real info
			fmt.Printf("DEBUG handleIncomingAccept: Found placeholder user by peer ID (ID: %d), updating with real info\n", acceptingUser.ID)
			acceptingUser.Username = response.Username
			acceptingUser.FullName = response.FullName
			if err := m.storage.UpdateUser(ctx, acceptingUser); err != nil {
				fmt.Printf("Warning: Failed to update placeholder user: %v\n", err)
			} else {
				fmt.Printf("DEBUG handleIncomingAccept: Updated placeholder user to %s (%s)\n", acceptingUser.Username, acceptingUser.FullName)
			}
		}
	} else {
		fmt.Printf("DEBUG handleIncomingAccept: Found existing user %s (ID: %d)\n", acceptingUser.Username, acceptingUser.ID)
	}

	// Get current user
	if m.currentUserID == 0 {
		fmt.Printf("DEBUG handleIncomingAccept: currentUserID is 0, skipping friendship creation\n")
		fmt.Printf("\n✓ %s accepted your friend request!\n", response.FullName)
		fmt.Printf("   You are now friends with %s (%s)\n", response.FullName, response.Username)
		fmt.Print("> ")
		return
	}

	currentUser, err := m.storage.GetUserByID(ctx, m.currentUserID)
	if err != nil || currentUser == nil {
		fmt.Printf("ERROR: Could not get current user (ID: %d): %v\n", m.currentUserID, err)
		fmt.Printf("\n✓ %s accepted your friend request!\n", response.FullName)
		fmt.Printf("   You are now friends with %s (%s)\n", response.FullName, response.Username)
		fmt.Print("> ")
		return
	}

	fmt.Printf("DEBUG handleIncomingAccept: Current user = %s (ID: %d)\n", currentUser.Username, currentUser.ID)

	// Create bidirectional friendship records if they don't exist
	// 1. Current user -> Accepting user (this should already exist as "pending")
	existingRequest, _ := m.storage.GetFriendRequest(ctx, currentUser.ID, acceptingUser.ID)
	fmt.Printf("DEBUG handleIncomingAccept: Looking for existing request: currentUser.ID=%d -> acceptingUser.ID=%d\n",
		currentUser.ID, acceptingUser.ID)
	if existingRequest != nil {
		fmt.Printf("DEBUG handleIncomingAccept: Found existing request, Status=%s\n", existingRequest.Status)
		if existingRequest.Status == "pending" {
			existingRequest.Status = "accepted"
			now := time.Now()
			existingRequest.AcceptedAt = now
			// Update with real username/fullname from response
			existingRequest.Username = acceptingUser.Username
			existingRequest.FullName = acceptingUser.FullName
			if err := m.storage.UpdateFriendRequest(ctx, existingRequest); err != nil {
				fmt.Printf("Warning: Failed to update friend request: %v\n", err)
			} else {
				fmt.Printf("DEBUG handleIncomingAccept: Updated existing request to accepted with real name %s\n", acceptingUser.FullName)
			}
		}
	} else {
		fmt.Printf("DEBUG handleIncomingAccept: No existing request found\n")
	}

	// 2. Accepting user -> Current user (reciprocal friendship)
	reciprocalFriend, _ := m.storage.GetFriendRequest(ctx, acceptingUser.ID, currentUser.ID)
	fmt.Printf("DEBUG handleIncomingAccept: Looking for reciprocal friendship: acceptingUser.ID=%d -> currentUser.ID=%d\n",
		acceptingUser.ID, currentUser.ID)
	if reciprocalFriend == nil {
		fmt.Printf("DEBUG handleIncomingAccept: Creating reciprocal friendship\n")
		reciprocalFriend = &storage.Friend{
			UserID:     acceptingUser.ID,
			FriendID:   currentUser.ID,
			PeerID:     acceptingUser.PeerID,
			Username:   acceptingUser.Username,
			FullName:   acceptingUser.FullName,
			Status:     "accepted",
			AcceptedAt: time.Now(),
		}
		if err := m.storage.CreateFriendRequest(ctx, reciprocalFriend); err != nil {
			fmt.Printf("Warning: Failed to create reciprocal friendship: %v\n", err)
		} else {
			fmt.Printf("DEBUG handleIncomingAccept: Reciprocal friendship created\n")
		}
	} else {
		fmt.Printf("DEBUG handleIncomingAccept: Reciprocal friendship already exists, Status=%s\n", reciprocalFriend.Status)
	}

	fmt.Printf("\n✓ %s accepted your friend request!\n", response.FullName)
	fmt.Printf("   You are now friends with %s (%s)\n", response.FullName, response.Username)
	fmt.Print("> ")
}

func (m *Manager) handleIncomingReject(response *FriendResponseMessage, fromPeer peer.ID) {
	fmt.Printf("\n✗ %s declined your friend request\n", response.FullName)
	fmt.Print("> ")
}
