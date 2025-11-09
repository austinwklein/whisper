package conference

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
	// Protocol IDs for conference management
	ProtocolConferenceInvite = protocol.ID("/whisper/conference/invite/1.0.0")
)

// ConferenceInvite represents an invitation to join a conference
type ConferenceInvite struct {
	ConferenceID   int64  `json:"conference_id"`
	ConferenceName string `json:"conference_name"`
	FromUsername   string `json:"from_username"`
	FromFullName   string `json:"from_full_name"`
	FromPeerID     string `json:"from_peer_id"`
	Message        string `json:"message,omitempty"`
}

// ConferenceGossipMessage represents a message broadcast in a conference via GossipSub
type ConferenceGossipMessage struct {
	ConferenceID int64  `json:"conference_id"`
	FromUsername string `json:"from_username"`
	FromFullName string `json:"from_full_name"`
	FromPeerID   string `json:"from_peer_id"`
	Content      string `json:"content"`
	Timestamp    int64  `json:"timestamp"` // Unix timestamp
}

// Protocol handles conference invitation protocol
type Protocol struct {
	inviteHandler func(invite *ConferenceInvite, fromPeer peer.ID)
}

// NewProtocol creates a new conference protocol handler
func NewProtocol() *Protocol {
	return &Protocol{}
}

// SetInviteHandler sets the handler for incoming conference invites
func (p *Protocol) SetInviteHandler(handler func(*ConferenceInvite, peer.ID)) {
	p.inviteHandler = handler
}

// HandleConferenceInvite handles incoming conference invitations
func (p *Protocol) HandleConferenceInvite(s network.Stream) {
	defer s.Close()

	reader := bufio.NewReader(s)
	data, err := reader.ReadBytes('\n')
	if err != nil && err != io.EOF {
		fmt.Printf("Error reading conference invite: %v\n", err)
		return
	}

	var invite ConferenceInvite
	if err := json.Unmarshal(data, &invite); err != nil {
		fmt.Printf("Error unmarshaling conference invite: %v\n", err)
		return
	}

	if p.inviteHandler != nil {
		p.inviteHandler(&invite, s.Conn().RemotePeer())
	}
}

// SendConferenceInvite sends a conference invitation to a peer
func SendConferenceInvite(ctx context.Context, s network.Stream, invite *ConferenceInvite) error {
	defer s.Close()

	data, err := json.Marshal(invite)
	if err != nil {
		return fmt.Errorf("failed to marshal invite: %w", err)
	}

	data = append(data, '\n')
	_, err = s.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write invite: %w", err)
	}

	return nil
}
