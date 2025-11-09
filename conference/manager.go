package conference

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/austinwklein/whisper/storage"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Manager handles conference operations
type Manager struct {
	storage       storage.Storage
	host          host.Host
	pubsub        *pubsub.PubSub
	protocol      *Protocol
	currentUserID int64
	subscriptions map[int64]*pubsub.Subscription // conference_id -> subscription
	topics        map[int64]*pubsub.Topic        // conference_id -> topic
}

// NewManager creates a new conference manager
func NewManager(store storage.Storage, h host.Host, ps *pubsub.PubSub) *Manager {
	m := &Manager{
		storage:       store,
		host:          h,
		pubsub:        ps,
		protocol:      NewProtocol(),
		subscriptions: make(map[int64]*pubsub.Subscription),
		topics:        make(map[int64]*pubsub.Topic),
	}

	// Set protocol handlers
	m.protocol.SetInviteHandler(m.handleIncomingInvite)

	// Register stream handlers
	h.SetStreamHandler(ProtocolConferenceInvite, m.protocol.HandleConferenceInvite)

	return m
}

// SetCurrentUser sets the currently logged in user
func (m *Manager) SetCurrentUser(userID int64) {
	m.currentUserID = userID
}

// CreateConference creates a new conference
func (m *Manager) CreateConference(ctx context.Context, currentUser *storage.User, name string) (*storage.Conference, error) {
	if m.currentUserID == 0 {
		return nil, fmt.Errorf("not authenticated")
	}

	// Create conference
	conf := &storage.Conference{
		Name:      name,
		CreatorID: currentUser.ID,
		CreatedAt: time.Now(),
	}

	if err := m.storage.CreateConference(ctx, conf); err != nil {
		return nil, fmt.Errorf("failed to create conference: %w", err)
	}

	// Add creator as first participant
	participant := &storage.ConferenceParticipant{
		ConferenceID: conf.ID,
		UserID:       currentUser.ID,
		PeerID:       currentUser.PeerID,
		Username:     currentUser.Username,
		JoinedAt:     time.Now(),
		Active:       true,
	}

	if err := m.storage.AddConferenceParticipant(ctx, participant); err != nil {
		return nil, fmt.Errorf("failed to add creator as participant: %w", err)
	}

	// Subscribe to conference topic
	if err := m.SubscribeToConference(ctx, currentUser, conf.ID); err != nil {
		return nil, fmt.Errorf("failed to subscribe to conference: %w", err)
	}

	fmt.Printf("✓ Conference '%s' created (ID: %d)\n", name, conf.ID)
	return conf, nil
}

// InviteToConference invites a friend to a conference
func (m *Manager) InviteToConference(ctx context.Context, currentUser *storage.User, conferenceID int64, friendUsername string) error {
	// Get the conference
	conf, err := m.storage.GetConference(ctx, conferenceID)
	if err != nil || conf == nil {
		return fmt.Errorf("conference not found")
	}

	// Verify current user is a participant
	participants, err := m.storage.GetConferenceParticipants(ctx, conferenceID)
	if err != nil {
		return fmt.Errorf("failed to get participants: %w", err)
	}

	isParticipant := false
	for _, p := range participants {
		if p.UserID == currentUser.ID && p.Active {
			isParticipant = true
			break
		}
	}

	if !isParticipant {
		return fmt.Errorf("you are not a participant in this conference")
	}

	// Get friend
	friend, err := m.storage.GetUserByUsername(ctx, friendUsername)
	if err != nil || friend == nil {
		return fmt.Errorf("user not found: %s", friendUsername)
	}

	// Check if they're friends
	friendship, err := m.storage.GetFriendRequest(ctx, currentUser.ID, friend.ID)
	if err != nil || friendship == nil || friendship.Status != "accepted" {
		friendship, err = m.storage.GetFriendRequest(ctx, friend.ID, currentUser.ID)
		if err != nil || friendship == nil || friendship.Status != "accepted" {
			return fmt.Errorf("you must be friends with %s to invite them", friendUsername)
		}
	}

	// Check if already a participant
	for _, p := range participants {
		if p.UserID == friend.ID && p.Active {
			return fmt.Errorf("%s is already in this conference", friendUsername)
		}
	}

	// Send invite
	friendPeerID, err := peer.Decode(friend.PeerID)
	if err != nil {
		return fmt.Errorf("invalid peer ID: %w", err)
	}

	// Check if friend is online
	if m.host.Network().Connectedness(friendPeerID) != 1 {
		return fmt.Errorf("%s is not online - invites require recipient to be connected", friendUsername)
	}

	stream, err := m.host.NewStream(ctx, friendPeerID, ProtocolConferenceInvite)
	if err != nil {
		return fmt.Errorf("failed to open stream: %w", err)
	}

	invite := &ConferenceInvite{
		ConferenceID:   conf.ID,
		ConferenceName: conf.Name,
		FromUsername:   currentUser.Username,
		FromFullName:   currentUser.FullName,
		FromPeerID:     currentUser.PeerID,
		Message:        fmt.Sprintf("%s invited you to conference '%s'", currentUser.FullName, conf.Name),
	}

	if err := SendConferenceInvite(ctx, stream, invite); err != nil {
		return fmt.Errorf("failed to send invite: %w", err)
	}

	fmt.Printf("✓ Invited %s to conference '%s'\n", friendUsername, conf.Name)
	return nil
}

// JoinConference joins a conference by ID
func (m *Manager) JoinConference(ctx context.Context, currentUser *storage.User, conferenceID int64) error {
	// Get the conference
	conf, err := m.storage.GetConference(ctx, conferenceID)
	if err != nil || conf == nil {
		return fmt.Errorf("conference not found")
	}

	// Check if already a participant
	participants, err := m.storage.GetConferenceParticipants(ctx, conferenceID)
	if err != nil {
		return fmt.Errorf("failed to get participants: %w", err)
	}

	for _, p := range participants {
		if p.UserID == currentUser.ID {
			if p.Active {
				return fmt.Errorf("you are already in this conference")
			}
			// Reactivate if previously left
			p.Active = true
			// Note: We'd need an UpdateConferenceParticipant method for this
		}
	}

	// Add as participant
	participant := &storage.ConferenceParticipant{
		ConferenceID: conf.ID,
		UserID:       currentUser.ID,
		PeerID:       currentUser.PeerID,
		Username:     currentUser.Username,
		JoinedAt:     time.Now(),
		Active:       true,
	}

	if err := m.storage.AddConferenceParticipant(ctx, participant); err != nil {
		return fmt.Errorf("failed to add participant: %w", err)
	}

	// Subscribe to conference topic
	if err := m.SubscribeToConference(ctx, currentUser, conf.ID); err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	fmt.Printf("✓ Joined conference '%s'\n", conf.Name)
	return nil
}

// SendMessage sends a message to a conference via GossipSub
func (m *Manager) SendMessage(ctx context.Context, currentUser *storage.User, conferenceID int64, content string) error {
	// Verify user is a participant
	participants, err := m.storage.GetConferenceParticipants(ctx, conferenceID)
	if err != nil {
		return fmt.Errorf("failed to get participants: %w", err)
	}

	isParticipant := false
	for _, p := range participants {
		if p.UserID == currentUser.ID && p.Active {
			isParticipant = true
			break
		}
	}

	if !isParticipant {
		return fmt.Errorf("you are not a participant in this conference")
	}

	// Get topic
	topic, ok := m.topics[conferenceID]
	if !ok {
		return fmt.Errorf("not subscribed to conference - use 'join-conf %d' first", conferenceID)
	}

	// Create message
	msg := &ConferenceGossipMessage{
		ConferenceID: conferenceID,
		FromUsername: currentUser.Username,
		FromFullName: currentUser.FullName,
		FromPeerID:   currentUser.PeerID,
		Content:      content,
		Timestamp:    time.Now().Unix(),
	}

	// Marshal to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Publish to topic
	if err := topic.Publish(ctx, data); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	// Save to local database
	confMsg := &storage.ConferenceMessage{
		ConferenceID: conferenceID,
		FromUserID:   currentUser.ID,
		FromPeerID:   currentUser.PeerID,
		Content:      content,
		CreatedAt:    time.Now(),
	}

	if err := m.storage.SaveConferenceMessage(ctx, confMsg); err != nil {
		fmt.Printf("Warning: Failed to save message locally: %v\n", err)
	}

	return nil
}

// SubscribeToConference subscribes to a conference's GossipSub topic
func (m *Manager) SubscribeToConference(ctx context.Context, currentUser *storage.User, conferenceID int64) error {
	// Check if already subscribed
	if _, ok := m.subscriptions[conferenceID]; ok {
		return nil // Already subscribed
	}

	// Create topic name
	topicName := fmt.Sprintf("/whisper/conf/%d", conferenceID)

	// Join topic
	topic, err := m.pubsub.Join(topicName)
	if err != nil {
		return fmt.Errorf("failed to join topic: %w", err)
	}

	// Subscribe to topic
	sub, err := topic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	// Store subscription and topic
	m.subscriptions[conferenceID] = sub
	m.topics[conferenceID] = topic

	// Start listening for messages in background
	go m.listenToConference(ctx, currentUser, conferenceID, sub)

	return nil
}

// listenToConference listens for messages on a conference subscription
func (m *Manager) listenToConference(ctx context.Context, currentUser *storage.User, conferenceID int64, sub *pubsub.Subscription) {
	for {
		msg, err := sub.Next(ctx)
		if err != nil {
			// Subscription closed or context canceled
			return
		}

		// Skip messages from self
		if msg.ReceivedFrom == m.host.ID() {
			continue
		}

		// Parse message
		var gossipMsg ConferenceGossipMessage
		if err := json.Unmarshal(msg.Data, &gossipMsg); err != nil {
			fmt.Printf("Error parsing conference message: %v\n", err)
			continue
		}

		// Save to database
		confMsg := &storage.ConferenceMessage{
			ConferenceID: gossipMsg.ConferenceID,
			FromUserID:   0, // We might not know their user ID
			FromPeerID:   gossipMsg.FromPeerID,
			Content:      gossipMsg.Content,
			CreatedAt:    time.Unix(gossipMsg.Timestamp, 0),
		}

		// Try to find user by peer ID
		fromUser, err := m.storage.GetUserByPeerID(ctx, gossipMsg.FromPeerID)
		if err == nil && fromUser != nil {
			confMsg.FromUserID = fromUser.ID
		}

		if err := m.storage.SaveConferenceMessage(ctx, confMsg); err != nil {
			fmt.Printf("Warning: Failed to save conference message: %v\n", err)
		}

		// Display notification
		fmt.Printf("\n📢 [Conference] %s: %s\n> ", gossipMsg.FromFullName, gossipMsg.Content)
	}
}

// LeaveConference leaves a conference
func (m *Manager) LeaveConference(ctx context.Context, currentUser *storage.User, conferenceID int64) error {
	// Remove from participants
	if err := m.storage.RemoveConferenceParticipant(ctx, conferenceID, currentUser.ID); err != nil {
		return fmt.Errorf("failed to leave conference: %w", err)
	}

	// Unsubscribe from topic
	if sub, ok := m.subscriptions[conferenceID]; ok {
		sub.Cancel()
		delete(m.subscriptions, conferenceID)
	}

	if topic, ok := m.topics[conferenceID]; ok {
		topic.Close()
		delete(m.topics, conferenceID)
	}

	fmt.Printf("✓ Left conference\n")
	return nil
}

// GetConferences returns all conferences the user is in
func (m *Manager) GetConferences(ctx context.Context, userID int64) ([]*storage.Conference, error) {
	return m.storage.GetUserConferences(ctx, userID)
}

// GetConferenceMessages returns messages from a conference
func (m *Manager) GetConferenceMessages(ctx context.Context, conferenceID int64, limit int) ([]*storage.ConferenceMessage, error) {
	return m.storage.GetConferenceMessages(ctx, conferenceID, limit)
}

// GetConferenceParticipants returns participants in a conference
func (m *Manager) GetConferenceParticipants(ctx context.Context, conferenceID int64) ([]*storage.ConferenceParticipant, error) {
	return m.storage.GetConferenceParticipants(ctx, conferenceID)
}

// handleIncomingInvite handles incoming conference invitations
func (m *Manager) handleIncomingInvite(invite *ConferenceInvite, fromPeer peer.ID) {
	fmt.Printf("\n📨 Conference invite from %s (%s)\n", invite.FromFullName, invite.FromUsername)
	fmt.Printf("   Conference: %s (ID: %d)\n", invite.ConferenceName, invite.ConferenceID)
	fmt.Printf("   Message: %s\n", invite.Message)
	fmt.Printf("   Use 'join-conf %d' to join\n", invite.ConferenceID)
	fmt.Print("> ")
}
