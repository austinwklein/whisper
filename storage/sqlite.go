package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStorage implements the Storage interface using SQLite
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite storage instance
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	// Expand ~ to home directory
	if strings.HasPrefix(dbPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dbPath = filepath.Join(home, dbPath[2:])
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	storage := &SQLiteStorage{db: db}

	// Initialize schema
	if err := storage.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return storage, nil
}

// initSchema creates the database schema
func (s *SQLiteStorage) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		full_name TEXT NOT NULL,
		peer_id TEXT UNIQUE NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_users_peer_id ON users(peer_id);

	CREATE TABLE IF NOT EXISTS friends (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		friend_id INTEGER NOT NULL,
		peer_id TEXT NOT NULL,
		username TEXT NOT NULL,
		full_name TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		accepted_at DATETIME,
		FOREIGN KEY(user_id) REFERENCES users(id),
		UNIQUE(user_id, friend_id)
	);

	CREATE INDEX IF NOT EXISTS idx_friends_user_id ON friends(user_id);
	CREATE INDEX IF NOT EXISTS idx_friends_status ON friends(status);

	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		from_user_id INTEGER NOT NULL,
		to_user_id INTEGER NOT NULL,
		from_peer_id TEXT NOT NULL,
		to_peer_id TEXT NOT NULL,
		content TEXT NOT NULL,
		delivered BOOLEAN DEFAULT 0,
		read BOOLEAN DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		delivered_at DATETIME,
		read_at DATETIME,
		FOREIGN KEY(from_user_id) REFERENCES users(id),
		FOREIGN KEY(to_user_id) REFERENCES users(id)
	);

	CREATE INDEX IF NOT EXISTS idx_messages_to_user ON messages(to_user_id);
	CREATE INDEX IF NOT EXISTS idx_messages_delivered ON messages(delivered);

	CREATE TABLE IF NOT EXISTS conferences (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		creator_id INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(creator_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS conference_participants (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		conference_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		peer_id TEXT NOT NULL,
		username TEXT NOT NULL,
		joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		left_at DATETIME,
		active BOOLEAN DEFAULT 1,
		FOREIGN KEY(conference_id) REFERENCES conferences(id),
		FOREIGN KEY(user_id) REFERENCES users(id)
	);

	CREATE INDEX IF NOT EXISTS idx_conference_participants_conf ON conference_participants(conference_id);
	CREATE INDEX IF NOT EXISTS idx_conference_participants_user ON conference_participants(user_id);

	CREATE TABLE IF NOT EXISTS conference_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		conference_id INTEGER NOT NULL,
		from_user_id INTEGER NOT NULL,
		from_peer_id TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(conference_id) REFERENCES conferences(id),
		FOREIGN KEY(from_user_id) REFERENCES users(id)
	);

	CREATE INDEX IF NOT EXISTS idx_conference_messages_conf ON conference_messages(conference_id);

	CREATE TABLE IF NOT EXISTS known_peers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		peer_id TEXT UNIQUE NOT NULL,
		username TEXT,
		addrs TEXT,
		last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_known_peers_peer_id ON known_peers(peer_id);
	`

	_, err := s.db.Exec(schema)
	return err
}

// User operations
func (s *SQLiteStorage) CreateUser(ctx context.Context, user *User) error {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO users (username, password_hash, full_name, peer_id)
		VALUES (?, ?, ?, ?)
	`, user.Username, user.PasswordHash, user.FullName, user.PeerID)
	if err != nil {
		return err
	}
	user.ID, _ = result.LastInsertId()
	return nil
}

func (s *SQLiteStorage) GetUserByID(ctx context.Context, id int64) (*User, error) {
	user := &User{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, full_name, peer_id, created_at, updated_at
		FROM users WHERE id = ?
	`, id).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.FullName, &user.PeerID, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return user, err
}

func (s *SQLiteStorage) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	user := &User{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, full_name, peer_id, created_at, updated_at
		FROM users WHERE username = ?
	`, username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.FullName, &user.PeerID, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return user, err
}

func (s *SQLiteStorage) GetUserByPeerID(ctx context.Context, peerID string) (*User, error) {
	user := &User{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, full_name, peer_id, created_at, updated_at
		FROM users WHERE peer_id = ?
	`, peerID).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.FullName, &user.PeerID, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return user, err
}

func (s *SQLiteStorage) UpdateUser(ctx context.Context, user *User) error {
	user.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET password_hash = ?, full_name = ?, peer_id = ?, updated_at = ?
		WHERE id = ?
	`, user.PasswordHash, user.FullName, user.PeerID, user.UpdatedAt, user.ID)
	return err
}

func (s *SQLiteStorage) SearchUsersByName(ctx context.Context, name string) ([]*User, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, username, password_hash, full_name, peer_id, created_at, updated_at
		FROM users WHERE full_name LIKE ?
	`, "%"+name+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []*User{}
	for rows.Next() {
		user := &User{}
		if err := rows.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.FullName, &user.PeerID, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

// Friend operations
func (s *SQLiteStorage) CreateFriendRequest(ctx context.Context, friend *Friend) error {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO friends (user_id, friend_id, peer_id, username, full_name, status)
		VALUES (?, ?, ?, ?, ?, ?)
	`, friend.UserID, friend.FriendID, friend.PeerID, friend.Username, friend.FullName, friend.Status)
	if err != nil {
		return err
	}
	friend.ID, _ = result.LastInsertId()
	return nil
}

func (s *SQLiteStorage) GetFriendRequest(ctx context.Context, userID, friendID int64) (*Friend, error) {
	friend := &Friend{}
	var acceptedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, friend_id, peer_id, username, full_name, status, created_at, accepted_at
		FROM friends WHERE user_id = ? AND friend_id = ?
	`, userID, friendID).Scan(&friend.ID, &friend.UserID, &friend.FriendID, &friend.PeerID, &friend.Username, &friend.FullName, &friend.Status, &friend.CreatedAt, &acceptedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if acceptedAt.Valid {
		friend.AcceptedAt = acceptedAt.Time
	}
	return friend, err
}

func (s *SQLiteStorage) UpdateFriendRequest(ctx context.Context, friend *Friend) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE friends SET status = ?, accepted_at = ?
		WHERE id = ?
	`, friend.Status, friend.AcceptedAt, friend.ID)
	return err
}

func (s *SQLiteStorage) GetFriends(ctx context.Context, userID int64) ([]*Friend, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, friend_id, peer_id, username, full_name, status, created_at, accepted_at
		FROM friends WHERE user_id = ? AND status = 'accepted'
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	friends := []*Friend{}
	for rows.Next() {
		friend := &Friend{}
		var acceptedAt sql.NullTime
		if err := rows.Scan(&friend.ID, &friend.UserID, &friend.FriendID, &friend.PeerID, &friend.Username, &friend.FullName, &friend.Status, &friend.CreatedAt, &acceptedAt); err != nil {
			return nil, err
		}
		if acceptedAt.Valid {
			friend.AcceptedAt = acceptedAt.Time
		}
		friends = append(friends, friend)
	}
	return friends, rows.Err()
}

func (s *SQLiteStorage) GetPendingFriendRequests(ctx context.Context, userID int64) ([]*Friend, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, friend_id, peer_id, username, full_name, status, created_at, accepted_at
		FROM friends WHERE friend_id = ? AND status = 'pending'
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	requests := []*Friend{}
	for rows.Next() {
		friend := &Friend{}
		var acceptedAt sql.NullTime
		if err := rows.Scan(&friend.ID, &friend.UserID, &friend.FriendID, &friend.PeerID, &friend.Username, &friend.FullName, &friend.Status, &friend.CreatedAt, &acceptedAt); err != nil {
			return nil, err
		}
		if acceptedAt.Valid {
			friend.AcceptedAt = acceptedAt.Time
		}
		requests = append(requests, friend)
	}
	return requests, rows.Err()
}

// Message operations
func (s *SQLiteStorage) SaveMessage(ctx context.Context, message *Message) error {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO messages (from_user_id, to_user_id, from_peer_id, to_peer_id, content, delivered, read)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, message.FromUserID, message.ToUserID, message.FromPeerID, message.ToPeerID, message.Content, message.Delivered, message.Read)
	if err != nil {
		return err
	}
	message.ID, _ = result.LastInsertId()
	return nil
}

func (s *SQLiteStorage) GetMessages(ctx context.Context, userID, otherUserID int64, limit int) ([]*Message, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, from_user_id, to_user_id, from_peer_id, to_peer_id, content, delivered, read, created_at, delivered_at, read_at
		FROM messages
		WHERE (from_user_id = ? AND to_user_id = ?) OR (from_user_id = ? AND to_user_id = ?)
		ORDER BY created_at DESC
		LIMIT ?
	`, userID, otherUserID, otherUserID, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []*Message{}
	for rows.Next() {
		msg := &Message{}
		var deliveredAt, readAt sql.NullTime
		if err := rows.Scan(&msg.ID, &msg.FromUserID, &msg.ToUserID, &msg.FromPeerID, &msg.ToPeerID, &msg.Content, &msg.Delivered, &msg.Read, &msg.CreatedAt, &deliveredAt, &readAt); err != nil {
			return nil, err
		}
		if deliveredAt.Valid {
			msg.DeliveredAt = deliveredAt.Time
		}
		if readAt.Valid {
			msg.ReadAt = readAt.Time
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func (s *SQLiteStorage) GetUndeliveredMessages(ctx context.Context, userID int64) ([]*Message, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, from_user_id, to_user_id, from_peer_id, to_peer_id, content, delivered, read, created_at, delivered_at, read_at
		FROM messages
		WHERE to_user_id = ? AND delivered = 0
		ORDER BY created_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []*Message{}
	for rows.Next() {
		msg := &Message{}
		var deliveredAt, readAt sql.NullTime
		if err := rows.Scan(&msg.ID, &msg.FromUserID, &msg.ToUserID, &msg.FromPeerID, &msg.ToPeerID, &msg.Content, &msg.Delivered, &msg.Read, &msg.CreatedAt, &deliveredAt, &readAt); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func (s *SQLiteStorage) MarkMessageDelivered(ctx context.Context, messageID int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE messages SET delivered = 1, delivered_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, messageID)
	return err
}

func (s *SQLiteStorage) MarkMessageRead(ctx context.Context, messageID int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE messages SET read = 1, read_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, messageID)
	return err
}

// Conference operations
func (s *SQLiteStorage) CreateConference(ctx context.Context, conference *Conference) error {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO conferences (name, creator_id)
		VALUES (?, ?)
	`, conference.Name, conference.CreatorID)
	if err != nil {
		return err
	}
	conference.ID, _ = result.LastInsertId()
	return nil
}

func (s *SQLiteStorage) GetConference(ctx context.Context, id int64) (*Conference, error) {
	conf := &Conference{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, creator_id, created_at
		FROM conferences WHERE id = ?
	`, id).Scan(&conf.ID, &conf.Name, &conf.CreatorID, &conf.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return conf, err
}

func (s *SQLiteStorage) GetUserConferences(ctx context.Context, userID int64) ([]*Conference, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.name, c.creator_id, c.created_at
		FROM conferences c
		INNER JOIN conference_participants cp ON c.id = cp.conference_id
		WHERE cp.user_id = ? AND cp.active = 1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	conferences := []*Conference{}
	for rows.Next() {
		conf := &Conference{}
		if err := rows.Scan(&conf.ID, &conf.Name, &conf.CreatorID, &conf.CreatedAt); err != nil {
			return nil, err
		}
		conferences = append(conferences, conf)
	}
	return conferences, rows.Err()
}

func (s *SQLiteStorage) AddConferenceParticipant(ctx context.Context, participant *ConferenceParticipant) error {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO conference_participants (conference_id, user_id, peer_id, username, active)
		VALUES (?, ?, ?, ?, ?)
	`, participant.ConferenceID, participant.UserID, participant.PeerID, participant.Username, participant.Active)
	if err != nil {
		return err
	}
	participant.ID, _ = result.LastInsertId()
	return nil
}

func (s *SQLiteStorage) RemoveConferenceParticipant(ctx context.Context, conferenceID, userID int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE conference_participants
		SET active = 0, left_at = CURRENT_TIMESTAMP
		WHERE conference_id = ? AND user_id = ?
	`, conferenceID, userID)
	return err
}

func (s *SQLiteStorage) GetConferenceParticipants(ctx context.Context, conferenceID int64) ([]*ConferenceParticipant, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, conference_id, user_id, peer_id, username, joined_at, left_at, active
		FROM conference_participants
		WHERE conference_id = ? AND active = 1
	`, conferenceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	participants := []*ConferenceParticipant{}
	for rows.Next() {
		p := &ConferenceParticipant{}
		if err := rows.Scan(&p.ID, &p.ConferenceID, &p.UserID, &p.PeerID, &p.Username, &p.JoinedAt, &p.LeftAt, &p.Active); err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}
	return participants, rows.Err()
}

func (s *SQLiteStorage) SaveConferenceMessage(ctx context.Context, message *ConferenceMessage) error {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO conference_messages (conference_id, from_user_id, from_peer_id, content)
		VALUES (?, ?, ?, ?)
	`, message.ConferenceID, message.FromUserID, message.FromPeerID, message.Content)
	if err != nil {
		return err
	}
	message.ID, _ = result.LastInsertId()
	return nil
}

func (s *SQLiteStorage) GetConferenceMessages(ctx context.Context, conferenceID int64, limit int) ([]*ConferenceMessage, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, conference_id, from_user_id, from_peer_id, content, created_at
		FROM conference_messages
		WHERE conference_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, conferenceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []*ConferenceMessage{}
	for rows.Next() {
		msg := &ConferenceMessage{}
		if err := rows.Scan(&msg.ID, &msg.ConferenceID, &msg.FromUserID, &msg.FromPeerID, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

// Known peers operations
func (s *SQLiteStorage) SaveKnownPeer(ctx context.Context, peer *KnownPeer) error {
	result, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO known_peers (peer_id, username, addrs, last_seen)
		VALUES (?, ?, ?, ?)
	`, peer.PeerID, peer.Username, peer.Addrs, peer.LastSeen)
	if err != nil {
		return err
	}
	peer.ID, _ = result.LastInsertId()
	return nil
}

func (s *SQLiteStorage) GetKnownPeers(ctx context.Context) ([]*KnownPeer, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, peer_id, username, addrs, last_seen, created_at
		FROM known_peers
		ORDER BY last_seen DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	peers := []*KnownPeer{}
	for rows.Next() {
		peer := &KnownPeer{}
		if err := rows.Scan(&peer.ID, &peer.PeerID, &peer.Username, &peer.Addrs, &peer.LastSeen, &peer.CreatedAt); err != nil {
			return nil, err
		}
		peers = append(peers, peer)
	}
	return peers, rows.Err()
}

func (s *SQLiteStorage) UpdateKnownPeer(ctx context.Context, peer *KnownPeer) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE known_peers
		SET username = ?, addrs = ?, last_seen = ?
		WHERE peer_id = ?
	`, peer.Username, peer.Addrs, peer.LastSeen, peer.PeerID)
	return err
}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
