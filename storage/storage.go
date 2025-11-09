package storage

import "context"

// Storage defines the interface for data persistence
type Storage interface {
	// User operations
	CreateUser(ctx context.Context, user *User) error
	GetUserByID(ctx context.Context, id int64) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUserByPeerID(ctx context.Context, peerID string) (*User, error)
	UpdateUser(ctx context.Context, user *User) error
	SearchUsersByName(ctx context.Context, name string) ([]*User, error)

	// Friend operations
	CreateFriendRequest(ctx context.Context, friend *Friend) error
	GetFriendRequest(ctx context.Context, userID, friendID int64) (*Friend, error)
	UpdateFriendRequest(ctx context.Context, friend *Friend) error
	GetFriends(ctx context.Context, userID int64) ([]*Friend, error)
	GetPendingFriendRequests(ctx context.Context, userID int64) ([]*Friend, error)

	// Message operations
	SaveMessage(ctx context.Context, message *Message) error
	GetMessages(ctx context.Context, userID, otherUserID int64, limit int) ([]*Message, error)
	GetUndeliveredMessages(ctx context.Context, userID int64) ([]*Message, error)
	MarkMessageDelivered(ctx context.Context, messageID int64) error
	MarkMessageRead(ctx context.Context, messageID int64) error

	// Conference operations
	CreateConference(ctx context.Context, conference *Conference) error
	GetConference(ctx context.Context, id int64) (*Conference, error)
	GetUserConferences(ctx context.Context, userID int64) ([]*Conference, error)
	AddConferenceParticipant(ctx context.Context, participant *ConferenceParticipant) error
	RemoveConferenceParticipant(ctx context.Context, conferenceID, userID int64) error
	GetConferenceParticipants(ctx context.Context, conferenceID int64) ([]*ConferenceParticipant, error)
	SaveConferenceMessage(ctx context.Context, message *ConferenceMessage) error
	GetConferenceMessages(ctx context.Context, conferenceID int64, limit int) ([]*ConferenceMessage, error)

	// Known peers operations
	SaveKnownPeer(ctx context.Context, peer *KnownPeer) error
	GetKnownPeers(ctx context.Context) ([]*KnownPeer, error)
	UpdateKnownPeer(ctx context.Context, peer *KnownPeer) error

	// Lifecycle
	Close() error
}
