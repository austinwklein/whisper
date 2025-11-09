package storage

import "time"

// User represents a user in the system
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // Don't serialize password
	FullName     string    `json:"full_name"`
	PeerID       string    `json:"peer_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Friend represents a friendship between two users
type Friend struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	FriendID   int64     `json:"friend_id"`
	PeerID     string    `json:"peer_id"`   // Friend's peer ID
	Username   string    `json:"username"`  // Friend's username
	FullName   string    `json:"full_name"` // Friend's full name
	Status     string    `json:"status"`    // pending, accepted, blocked
	CreatedAt  time.Time `json:"created_at"`
	AcceptedAt time.Time `json:"accepted_at,omitempty"`
}

// Message represents a direct message
type Message struct {
	ID          int64     `json:"id"`
	FromUserID  int64     `json:"from_user_id"`
	ToUserID    int64     `json:"to_user_id"`
	FromPeerID  string    `json:"from_peer_id"`
	ToPeerID    string    `json:"to_peer_id"`
	Content     string    `json:"content"`
	Delivered   bool      `json:"delivered"`
	Read        bool      `json:"read"`
	CreatedAt   time.Time `json:"created_at"`
	DeliveredAt time.Time `json:"delivered_at,omitempty"`
	ReadAt      time.Time `json:"read_at,omitempty"`
}

// Conference represents a group chat
type Conference struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatorID int64     `json:"creator_id"`
	CreatedAt time.Time `json:"created_at"`
}

// ConferenceParticipant represents a participant in a conference
type ConferenceParticipant struct {
	ID           int64     `json:"id"`
	ConferenceID int64     `json:"conference_id"`
	UserID       int64     `json:"user_id"`
	PeerID       string    `json:"peer_id"`
	Username     string    `json:"username"`
	JoinedAt     time.Time `json:"joined_at"`
	LeftAt       time.Time `json:"left_at,omitempty"`
	Active       bool      `json:"active"`
}

// ConferenceMessage represents a message in a conference
type ConferenceMessage struct {
	ID           int64     `json:"id"`
	ConferenceID int64     `json:"conference_id"`
	FromUserID   int64     `json:"from_user_id"`
	FromPeerID   string    `json:"from_peer_id"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
}

// KnownPeer represents a peer we've connected to before
type KnownPeer struct {
	ID        int64     `json:"id"`
	PeerID    string    `json:"peer_id"`
	Username  string    `json:"username"`
	Addrs     string    `json:"addrs"` // JSON array of multiaddresses
	LastSeen  time.Time `json:"last_seen"`
	CreatedAt time.Time `json:"created_at"`
}
