package friends

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
	ProtocolFriendRequest = protocol.ID("/whisper/friend/request/1.0.0")
	ProtocolFriendAccept  = protocol.ID("/whisper/friend/accept/1.0.0")
	ProtocolFriendReject  = protocol.ID("/whisper/friend/reject/1.0.0")
)

// FriendRequestMessage represents a friend request
type FriendRequestMessage struct {
	FromUsername string `json:"from_username"`
	FromFullName string `json:"from_full_name"`
	FromPeerID   string `json:"from_peer_id"`
	Message      string `json:"message,omitempty"`
}

// FriendResponseMessage represents a response to a friend request
type FriendResponseMessage struct {
	Accepted bool   `json:"accepted"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
	PeerID   string `json:"peer_id"`
	Message  string `json:"message,omitempty"`
}

// Protocol handles friend request protocol
type Protocol struct {
	requestHandler func(request *FriendRequestMessage, fromPeer peer.ID)
	acceptHandler  func(response *FriendResponseMessage, fromPeer peer.ID)
	rejectHandler  func(response *FriendResponseMessage, fromPeer peer.ID)
}

// NewProtocol creates a new friend protocol handler
func NewProtocol() *Protocol {
	return &Protocol{}
}

// SetRequestHandler sets the handler for incoming friend requests
func (p *Protocol) SetRequestHandler(handler func(*FriendRequestMessage, peer.ID)) {
	p.requestHandler = handler
}

// SetAcceptHandler sets the handler for friend request acceptances
func (p *Protocol) SetAcceptHandler(handler func(*FriendResponseMessage, peer.ID)) {
	p.acceptHandler = handler
}

// SetRejectHandler sets the handler for friend request rejections
func (p *Protocol) SetRejectHandler(handler func(*FriendResponseMessage, peer.ID)) {
	p.rejectHandler = handler
}

// HandleFriendRequest handles incoming friend requests
func (p *Protocol) HandleFriendRequest(s network.Stream) {
	defer s.Close()

	reader := bufio.NewReader(s)
	data, err := reader.ReadBytes('\n')
	if err != nil && err != io.EOF {
		fmt.Printf("Error reading friend request: %v\n", err)
		return
	}

	var request FriendRequestMessage
	if err := json.Unmarshal(data, &request); err != nil {
		fmt.Printf("Error unmarshaling friend request: %v\n", err)
		return
	}

	if p.requestHandler != nil {
		p.requestHandler(&request, s.Conn().RemotePeer())
	}
}

// HandleFriendAccept handles friend request acceptances
func (p *Protocol) HandleFriendAccept(s network.Stream) {
	defer s.Close()

	reader := bufio.NewReader(s)
	data, err := reader.ReadBytes('\n')
	if err != nil && err != io.EOF {
		fmt.Printf("Error reading friend accept: %v\n", err)
		return
	}

	var response FriendResponseMessage
	if err := json.Unmarshal(data, &response); err != nil {
		fmt.Printf("Error unmarshaling friend accept: %v\n", err)
		return
	}

	if p.acceptHandler != nil {
		p.acceptHandler(&response, s.Conn().RemotePeer())
	}
}

// HandleFriendReject handles friend request rejections
func (p *Protocol) HandleFriendReject(s network.Stream) {
	defer s.Close()

	reader := bufio.NewReader(s)
	data, err := reader.ReadBytes('\n')
	if err != nil && err != io.EOF {
		fmt.Printf("Error reading friend reject: %v\n", err)
		return
	}

	var response FriendResponseMessage
	if err := json.Unmarshal(data, &response); err != nil {
		fmt.Printf("Error unmarshaling friend reject: %v\n", err)
		return
	}

	if p.rejectHandler != nil {
		p.rejectHandler(&response, s.Conn().RemotePeer())
	}
}

// SendFriendRequest sends a friend request to a peer
func SendFriendRequest(ctx context.Context, s network.Stream, request *FriendRequestMessage) error {
	defer s.Close()

	data, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	data = append(data, '\n')
	_, err = s.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write request: %w", err)
	}

	return nil
}

// SendFriendResponse sends a response to a friend request
func SendFriendResponse(ctx context.Context, s network.Stream, response *FriendResponseMessage) error {
	defer s.Close()

	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	data = append(data, '\n')
	_, err = s.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	return nil
}
