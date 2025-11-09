package p2p

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/multiformats/go-multiaddr"
)

const (
	// Protocol IDs for different message types
	ProtocolFriendRequest = "/whisper/friend/request/1.0.0"
	ProtocolFriendAccept  = "/whisper/friend/accept/1.0.0"
	ProtocolDirectMessage = "/whisper/message/direct/1.0.0"
	ProtocolUserSearch    = "/whisper/user/search/1.0.0"
)

// P2PHost wraps libp2p host and provides Whisper-specific functionality
type P2PHost struct {
	host      host.Host
	dht       *dht.IpfsDHT
	ctx       context.Context
	discovery mdns.Service
	mu        sync.RWMutex
	peers     map[peer.ID]*PeerInfo
}

// PeerInfo stores information about a connected peer
type PeerInfo struct {
	ID        peer.ID
	Addrs     []multiaddr.Multiaddr
	Connected bool
	Username  string // Will be populated after user identification
}

// isPortAvailable checks if a TCP port is available
func isPortAvailable(port int) bool {
	if port == 0 {
		return true // Port 0 means auto-select
	}
	// Try to bind to 0.0.0.0 (all interfaces) to match how libp2p binds
	addr := fmt.Sprintf("0.0.0.0:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// NewP2PHost creates a new P2P host instance
func NewP2PHost(ctx context.Context, port int, privKey crypto.PrivKey) (*P2PHost, error) {
	// Generate a new identity if not provided
	if privKey == nil {
		var err error
		privKey, _, err = crypto.GenerateKeyPair(crypto.Ed25519, -1)
		if err != nil {
			return nil, fmt.Errorf("failed to generate key pair: %w", err)
		}
	}

	// Check if requested port is available
	if !isPortAvailable(port) {
		fmt.Printf("Port %d is already in use, selecting an available port automatically...\n", port)
		port = 0 // Let OS select an available port
	}

	// Create listen address
	// If port is 0, libp2p will automatically select an available port
	listenAddr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port)

	// Create libp2p host
	h, err := libp2p.New(
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(listenAddr),
		libp2p.DefaultTransports,
		libp2p.DefaultMuxers,
		libp2p.DefaultSecurity,
		libp2p.NATPortMap(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	// Create DHT for peer discovery
	kdht, err := dht.New(ctx, h, dht.Mode(dht.ModeServer))
	if err != nil {
		h.Close()
		return nil, fmt.Errorf("failed to create DHT: %w", err)
	}

	// Bootstrap the DHT
	if err = kdht.Bootstrap(ctx); err != nil {
		h.Close()
		return nil, fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	p2pHost := &P2PHost{
		host:  h,
		dht:   kdht,
		ctx:   ctx,
		peers: make(map[peer.ID]*PeerInfo),
	}

	// Set up connection notifications
	h.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, conn network.Conn) {
			p2pHost.handleNewConnection(conn.RemotePeer())
		},
		DisconnectedF: func(n network.Network, conn network.Conn) {
			p2pHost.handleDisconnection(conn.RemotePeer())
		},
	})

	// Setup mDNS discovery for local network peers
	disc := &discoveryNotifee{h: p2pHost}
	ser := mdns.NewMdnsService(h, "whisper-mdns", disc)
	p2pHost.discovery = ser

	return p2pHost, nil
}

// PeerID returns the local peer ID
func (p *P2PHost) PeerID() peer.ID {
	return p.host.ID()
}

// Host returns the underlying libp2p host
func (p *P2PHost) Host() host.Host {
	return p.host
}

// Addrs returns the local multiaddresses
func (p *P2PHost) Addrs() []multiaddr.Multiaddr {
	return p.host.Addrs()
}

// GetFullAddrs returns the full multiaddresses including peer ID
func (p *P2PHost) GetFullAddrs() []string {
	addrs := make([]string, 0)
	for _, addr := range p.host.Addrs() {
		// Combine address with peer ID
		fullAddr := fmt.Sprintf("%s/p2p/%s", addr.String(), p.host.ID().String())
		addrs = append(addrs, fullAddr)
	}
	return addrs
}

// ConnectToPeer connects to a peer using its multiaddress
func (p *P2PHost) ConnectToPeer(ctx context.Context, addrStr string) error {
	// Parse the multiaddress
	maddr, err := multiaddr.NewMultiaddr(addrStr)
	if err != nil {
		return fmt.Errorf("invalid multiaddress: %w", err)
	}

	// Extract peer ID and address
	addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("failed to parse peer info: %w", err)
	}

	// Connect to the peer
	if err := p.host.Connect(ctx, *addrInfo); err != nil {
		return fmt.Errorf("failed to connect to peer: %w", err)
	}

	return nil
}

// GetConnectedPeers returns a list of currently connected peers
func (p *P2PHost) GetConnectedPeers() []*PeerInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	peers := make([]*PeerInfo, 0, len(p.peers))
	for _, peerInfo := range p.peers {
		if peerInfo.Connected {
			peers = append(peers, peerInfo)
		}
	}
	return peers
}

// SetStreamHandler sets a handler for a specific protocol
func (p *P2PHost) SetStreamHandler(protocolID protocol.ID, handler network.StreamHandler) {
	p.host.SetStreamHandler(protocolID, handler)
}

// NewStream opens a new stream to a peer for a specific protocol
func (p *P2PHost) NewStream(ctx context.Context, peerID peer.ID, protocolID protocol.ID) (network.Stream, error) {
	return p.host.NewStream(ctx, peerID, protocolID)
}

// handleNewConnection handles new peer connections
func (p *P2PHost) handleNewConnection(peerID peer.ID) {
	p.mu.Lock()
	defer p.mu.Unlock()

	peerInfo, exists := p.peers[peerID]
	if !exists {
		peerInfo = &PeerInfo{
			ID:        peerID,
			Connected: true,
		}
		p.peers[peerID] = peerInfo
	} else {
		peerInfo.Connected = true
	}

	// Get peer addresses
	peerInfo.Addrs = p.host.Peerstore().Addrs(peerID)

	fmt.Printf("Peer connected: %s\n", peerID.String())
}

// handleDisconnection handles peer disconnections
func (p *P2PHost) handleDisconnection(peerID peer.ID) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if peerInfo, exists := p.peers[peerID]; exists {
		peerInfo.Connected = false
		fmt.Printf("Peer disconnected: %s\n", peerID.String())
	}
}

// Close shuts down the P2P host
func (p *P2PHost) Close() error {
	if p.discovery != nil {
		p.discovery.Close()
	}
	if p.dht != nil {
		p.dht.Close()
	}
	return p.host.Close()
}

// discoveryNotifee implements mdns.Notifee for local peer discovery
type discoveryNotifee struct {
	h *P2PHost
}

// HandlePeerFound is called when a peer is discovered via mDNS
func (n *discoveryNotifee) HandlePeerFound(peerInfo peer.AddrInfo) {
	// Try to connect to the discovered peer
	if err := n.h.host.Connect(n.h.ctx, peerInfo); err != nil {
		fmt.Printf("Failed to connect to discovered peer %s: %v\n", peerInfo.ID, err)
	} else {
		fmt.Printf("Connected to peer via mDNS: %s\n", peerInfo.ID)
	}
}
