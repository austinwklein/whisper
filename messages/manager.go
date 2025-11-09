package messages

import (
	"context"
	"fmt"
	"time"

	"github.com/austinwklein/whisper/storage"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Manager handles message operations
type Manager struct {
	storage       storage.Storage
	host          host.Host
	protocol      *Protocol
	currentUserID int64
}

// NewManager creates a new message manager
func NewManager(store storage.Storage, h host.Host) *Manager {
	m := &Manager{
		storage:  store,
		host:     h,
		protocol: NewProtocol(),
	}

	// Set protocol handlers
	m.protocol.SetMessageHandler(m.handleIncomingMessage)
	m.protocol.SetAckHandler(m.handleMessageAck)
	m.protocol.SetReadHandler(m.handleMessageRead)

	// Register stream handlers
	h.SetStreamHandler(ProtocolDirectMessage, m.protocol.HandleDirectMessage)
	h.SetStreamHandler(ProtocolMessageAck, m.protocol.HandleMessageAck)
	h.SetStreamHandler(ProtocolMessageRead, m.protocol.HandleMessageRead)

	return m
}

// SetCurrentUser sets the currently logged in user
func (m *Manager) SetCurrentUser(userID int64) {
	m.currentUserID = userID
}

// SendMessage sends a direct message to a friend
func (m *Manager) SendMessage(ctx context.Context, currentUser *storage.User, toUsername string, content string) error {
	// Look up recipient user
	toUser, err := m.storage.GetUserByUsername(ctx, toUsername)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Check if they are friends
	friendship, err := m.storage.GetFriendRequest(ctx, currentUser.ID, toUser.ID)
	if err != nil || friendship == nil || friendship.Status != "accepted" {
		// Check reverse direction
		friendship, err = m.storage.GetFriendRequest(ctx, toUser.ID, currentUser.ID)
		if err != nil || friendship == nil || friendship.Status != "accepted" {
			return fmt.Errorf("you must be friends with %s to send messages", toUsername)
		}
	}

	// Create message
	msg := &storage.Message{
		FromUserID: currentUser.ID,
		ToUserID:   toUser.ID,
		FromPeerID: currentUser.PeerID,
		ToPeerID:   toUser.PeerID,
		Content:    content,
		Delivered:  false,
		Read:       false,
		CreatedAt:  time.Now(),
	}

	// Save message to database
	if err := m.storage.SaveMessage(ctx, msg); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	// Try to deliver message if peer is online
	toPeerID, err := peer.Decode(toUser.PeerID)
	if err != nil {
		return fmt.Errorf("invalid peer ID: %w", err)
	}

	// Check if peer is connected
	if m.host.Network().Connectedness(toPeerID) != 1 { // 1 = Connected
		fmt.Printf("✓ Message saved (user offline, will deliver when online)\n")
		return nil
	}

	// Open stream and send message
	stream, err := m.host.NewStream(ctx, toPeerID, ProtocolDirectMessage)
	if err != nil {
		fmt.Printf("✓ Message saved (delivery failed, will retry: %v)\n", err)
		return nil
	}

	directMsg := &DirectMessage{
		MessageID:    msg.ID,
		FromUsername: currentUser.Username,
		FromFullName: currentUser.FullName,
		FromPeerID:   currentUser.PeerID,
		ToUsername:   toUser.Username,
		Content:      content,
		Timestamp:    msg.CreatedAt.Unix(),
	}

	if err := SendDirectMessage(ctx, stream, directMsg); err != nil {
		fmt.Printf("✓ Message saved (delivery failed, will retry: %v)\n", err)
		return nil
	}

	// Mark as delivered
	if err := m.storage.MarkMessageDelivered(ctx, msg.ID); err != nil {
		fmt.Printf("Warning: Failed to mark message as delivered: %v\n", err)
	}

	fmt.Printf("✓ Message sent to %s\n", toUsername)
	return nil
}

// handleIncomingMessage handles incoming direct messages
func (m *Manager) handleIncomingMessage(message *DirectMessage, fromPeer peer.ID) {
	ctx := context.Background()

	// Look up sender
	fromUser, err := m.storage.GetUserByUsername(ctx, message.FromUsername)
	if err != nil {
		fmt.Printf("Error: Message from unknown user %s\n", message.FromUsername)
		return
	}

	// Look up recipient (should be current user)
	toUser, err := m.storage.GetUserByUsername(ctx, message.ToUsername)
	if err != nil {
		fmt.Printf("Error: Message to unknown user %s\n", message.ToUsername)
		return
	}

	// Save message
	msg := &storage.Message{
		FromUserID: fromUser.ID,
		ToUserID:   toUser.ID,
		FromPeerID: fromUser.PeerID,
		ToPeerID:   toUser.PeerID,
		Content:    message.Content,
		Delivered:  true,
		Read:       false,
		CreatedAt:  time.Unix(message.Timestamp, 0),
	}

	if err := m.storage.SaveMessage(ctx, msg); err != nil {
		fmt.Printf("Error saving message: %v\n", err)
		return
	}

	// Mark as delivered immediately
	if err := m.storage.MarkMessageDelivered(ctx, msg.ID); err != nil {
		fmt.Printf("Warning: Failed to mark message as delivered: %v\n", err)
	}

	// Send acknowledgment
	stream, err := m.host.NewStream(ctx, fromPeer, ProtocolMessageAck)
	if err != nil {
		fmt.Printf("Warning: Failed to send message ack: %v\n", err)
	} else {
		ack := &MessageAck{
			MessageID: message.MessageID,
			FromPeer:  toUser.PeerID,
			ToPeer:    fromUser.PeerID,
			Timestamp: time.Now().Unix(),
		}
		if err := SendMessageAck(ctx, stream, ack); err != nil {
			fmt.Printf("Warning: Failed to send ack: %v\n", err)
		}
	}

	// Display notification
	fmt.Printf("\n📨 New message from %s (%s): %s\n> ", message.FromFullName, message.FromUsername, message.Content)
}

// handleMessageAck handles message delivery acknowledgments
func (m *Manager) handleMessageAck(ack *MessageAck, fromPeer peer.ID) {
	ctx := context.Background()

	if ack.MessageID > 0 {
		if err := m.storage.MarkMessageDelivered(ctx, ack.MessageID); err != nil {
			fmt.Printf("Warning: Failed to mark message as delivered: %v\n", err)
		}
	}
}

// handleMessageRead handles message read receipts
func (m *Manager) handleMessageRead(read *MessageRead, fromPeer peer.ID) {
	ctx := context.Background()

	if read.MessageID > 0 {
		if err := m.storage.MarkMessageRead(ctx, read.MessageID); err != nil {
			fmt.Printf("Warning: Failed to mark message as read: %v\n", err)
		}
	}
}

// GetConversation retrieves message history with another user
func (m *Manager) GetConversation(ctx context.Context, currentUserID, otherUserID int64, limit int) ([]*storage.Message, error) {
	return m.storage.GetMessages(ctx, currentUserID, otherUserID, limit)
}

// GetUndeliveredMessages retrieves messages that haven't been delivered yet
func (m *Manager) GetUndeliveredMessages(ctx context.Context, userID int64) ([]*storage.Message, error) {
	return m.storage.GetUndeliveredMessages(ctx, userID)
}

// MarkAsRead marks messages from a specific user as read
func (m *Manager) MarkAsRead(ctx context.Context, currentUser *storage.User, fromUsername string) error {
	// Look up the other user
	fromUser, err := m.storage.GetUserByUsername(ctx, fromUsername)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Get all unread messages from that user
	messages, err := m.storage.GetMessages(ctx, currentUser.ID, fromUser.ID, 100)
	if err != nil {
		return fmt.Errorf("failed to get messages: %w", err)
	}

	// Mark each unread message as read
	for _, msg := range messages {
		if msg.FromUserID == fromUser.ID && !msg.Read {
			if err := m.storage.MarkMessageRead(ctx, msg.ID); err != nil {
				fmt.Printf("Warning: Failed to mark message %d as read: %v\n", msg.ID, err)
			}

			// Send read receipt if peer is online
			toPeerID, err := peer.Decode(fromUser.PeerID)
			if err != nil {
				continue
			}

			if m.host.Network().Connectedness(toPeerID) == 1 { // Connected
				stream, err := m.host.NewStream(ctx, toPeerID, ProtocolMessageRead)
				if err != nil {
					continue
				}

				readReceipt := &MessageRead{
					MessageID: msg.ID,
					FromPeer:  currentUser.PeerID,
					ToPeer:    fromUser.PeerID,
					Timestamp: time.Now().Unix(),
				}
				SendMessageRead(ctx, stream, readReceipt)
			}
		}
	}

	return nil
}

// RetryUndeliveredMessages attempts to deliver queued messages to online peers
func (m *Manager) RetryUndeliveredMessages(ctx context.Context, currentUserID int64) error {
	messages, err := m.storage.GetUndeliveredMessages(ctx, currentUserID)
	if err != nil {
		return fmt.Errorf("failed to get undelivered messages: %w", err)
	}

	if len(messages) == 0 {
		return nil
	}

	fmt.Printf("Found %d undelivered message(s), attempting delivery...\n", len(messages))

	for _, msg := range messages {
		// Look up sender and recipient
		fromUser, err := m.storage.GetUserByID(ctx, msg.FromUserID)
		if err != nil {
			continue
		}

		toUser, err := m.storage.GetUserByID(ctx, msg.ToUserID)
		if err != nil {
			continue
		}

		// Try to deliver
		toPeerID, err := peer.Decode(toUser.PeerID)
		if err != nil {
			continue
		}

		if m.host.Network().Connectedness(toPeerID) != 1 {
			continue // Still offline
		}

		stream, err := m.host.NewStream(ctx, toPeerID, ProtocolDirectMessage)
		if err != nil {
			continue
		}

		directMsg := &DirectMessage{
			MessageID:    msg.ID,
			FromUsername: fromUser.Username,
			FromFullName: fromUser.FullName,
			FromPeerID:   fromUser.PeerID,
			ToUsername:   toUser.Username,
			Content:      msg.Content,
			Timestamp:    msg.CreatedAt.Unix(),
		}

		if err := SendDirectMessage(ctx, stream, directMsg); err != nil {
			continue
		}

		// Mark as delivered
		if err := m.storage.MarkMessageDelivered(ctx, msg.ID); err != nil {
			fmt.Printf("Warning: Failed to mark message as delivered: %v\n", err)
		} else {
			fmt.Printf("✓ Delivered message to %s\n", toUser.Username)
		}
	}

	return nil
}
