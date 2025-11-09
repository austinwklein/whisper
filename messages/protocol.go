package messages

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const (
	// Protocol IDs
	ProtocolDirectMessage = protocol.ID("/whisper/message/direct/1.0.0")
	ProtocolMessageAck    = protocol.ID("/whisper/message/ack/1.0.0")
	ProtocolMessageRead   = protocol.ID("/whisper/message/read/1.0.0")
)

// DirectMessage represents a direct message between users
type DirectMessage struct {
	MessageID    int64  `json:"message_id,omitempty"` // Set by sender if stored locally
	FromUsername string `json:"from_username"`
	FromFullName string `json:"from_full_name"`
	FromPeerID   string `json:"from_peer_id"`
	ToUsername   string `json:"to_username"`
	Content      string `json:"content"`
	Timestamp    int64  `json:"timestamp"` // Unix timestamp
}

// MessageAck represents acknowledgment that a message was received
type MessageAck struct {
	MessageID int64  `json:"message_id"`
	FromPeer  string `json:"from_peer"`
	ToPeer    string `json:"to_peer"`
	Timestamp int64  `json:"timestamp"`
}

// MessageRead represents notification that a message was read
type MessageRead struct {
	MessageID int64  `json:"message_id"`
	FromPeer  string `json:"from_peer"`
	ToPeer    string `json:"to_peer"`
	Timestamp int64  `json:"timestamp"`
}

// Protocol handles direct messaging protocol
type Protocol struct {
	messageHandler func(message *DirectMessage, fromPeer peer.ID)
	ackHandler     func(ack *MessageAck, fromPeer peer.ID)
	readHandler    func(read *MessageRead, fromPeer peer.ID)
}

// NewProtocol creates a new message protocol handler
func NewProtocol() *Protocol {
	return &Protocol{}
}

// SetMessageHandler sets the handler for incoming direct messages
func (p *Protocol) SetMessageHandler(handler func(*DirectMessage, peer.ID)) {
	p.messageHandler = handler
}

// SetAckHandler sets the handler for message acknowledgments
func (p *Protocol) SetAckHandler(handler func(*MessageAck, peer.ID)) {
	p.ackHandler = handler
}

// SetReadHandler sets the handler for message read receipts
func (p *Protocol) SetReadHandler(handler func(*MessageRead, peer.ID)) {
	p.readHandler = handler
}

// HandleDirectMessage handles incoming direct messages
func (p *Protocol) HandleDirectMessage(s network.Stream) {
	defer s.Close()

	reader := bufio.NewReader(s)
	data, err := reader.ReadBytes('\n')
	if err != nil && err != io.EOF {
		fmt.Printf("Error reading direct message: %v\n", err)
		return
	}

	var message DirectMessage
	if err := json.Unmarshal(data, &message); err != nil {
		fmt.Printf("Error unmarshaling direct message: %v\n", err)
		return
	}

	if p.messageHandler != nil {
		p.messageHandler(&message, s.Conn().RemotePeer())
	}
}

// HandleMessageAck handles message acknowledgments
func (p *Protocol) HandleMessageAck(s network.Stream) {
	defer s.Close()

	reader := bufio.NewReader(s)
	data, err := reader.ReadBytes('\n')
	if err != nil && err != io.EOF {
		fmt.Printf("Error reading message ack: %v\n", err)
		return
	}

	var ack MessageAck
	if err := json.Unmarshal(data, &ack); err != nil {
		fmt.Printf("Error unmarshaling message ack: %v\n", err)
		return
	}

	if p.ackHandler != nil {
		p.ackHandler(&ack, s.Conn().RemotePeer())
	}
}

// HandleMessageRead handles message read receipts
func (p *Protocol) HandleMessageRead(s network.Stream) {
	defer s.Close()

	reader := bufio.NewReader(s)
	data, err := reader.ReadBytes('\n')
	if err != nil && err != io.EOF {
		fmt.Printf("Error reading message read: %v\n", err)
		return
	}

	var read MessageRead
	if err := json.Unmarshal(data, &read); err != nil {
		fmt.Printf("Error unmarshaling message read: %v\n", err)
		return
	}

	if p.readHandler != nil {
		p.readHandler(&read, s.Conn().RemotePeer())
	}
}

// SendDirectMessage sends a direct message to a peer
func SendDirectMessage(ctx context.Context, s network.Stream, message *DirectMessage) error {
	defer s.Close()

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	data = append(data, '\n')
	_, err = s.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// SendMessageAck sends a message acknowledgment to a peer
func SendMessageAck(ctx context.Context, s network.Stream, ack *MessageAck) error {
	defer s.Close()

	data, err := json.Marshal(ack)
	if err != nil {
		return fmt.Errorf("failed to marshal ack: %w", err)
	}

	data = append(data, '\n')
	_, err = s.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write ack: %w", err)
	}

	return nil
}

// SendMessageRead sends a message read receipt to a peer
func SendMessageRead(ctx context.Context, s network.Stream, read *MessageRead) error {
	defer s.Close()

	data, err := json.Marshal(read)
	if err != nil {
		return fmt.Errorf("failed to marshal read: %w", err)
	}

	data = append(data, '\n')
	_, err = s.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write read: %w", err)
	}

	return nil
}
