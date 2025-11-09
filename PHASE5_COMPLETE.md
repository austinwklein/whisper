# Phase 5: Conference Chat - Complete

**Date:** 2025-11-08  
**Status:** ✅ Complete

## Overview

Phase 5 adds group messaging capabilities to Whisper using libp2p's GossipSub protocol. Users can create conferences, invite friends, and participate in real-time group conversations with message persistence.

## Features Implemented

### 1. GossipSub Integration

**File:** `p2p/host.go`

Added GossipSub (libp2p's pub/sub protocol) to the P2P host:

```go
type P2PHost struct {
    host      host.Host
    dht       *dht.IpfsDHT
    pubsub    *pubsub.PubSub  // Added for conference chat
    ctx       context.Context
    discovery mdns.Service
    mu        sync.RWMutex
    peers     map[peer.ID]*PeerInfo
}
```

**Key Functions:**
- `NewP2PHost()` - Initializes GossipSub instance
- `PubSub()` - Accessor for conference manager

**Why GossipSub?**
- Efficient group message broadcasting
- Built-in redundancy and reliability
- Scalable to many participants
- Native libp2p protocol

### 2. Conference Protocol

**File:** `conference/protocol.go`

Defines the conference invitation protocol and message format:

```go
const ProtocolConferenceInvite = protocol.ID("/whisper/conference/invite/1.0.0")

type ConferenceInvite struct {
    ConferenceID   int64  `json:"conference_id"`
    ConferenceName string `json:"conference_name"`
    FromUsername   string `json:"from_username"`
    FromFullName   string `json:"from_full_name"`
}

type ConferenceGossipMessage struct {
    ConferenceID int64  `json:"conference_id"`
    FromUsername string `json:"from_username"`
    FromFullName string `json:"from_full_name"`
    FromPeerID   string `json:"from_peer_id"`
    Content      string `json:"content"`
    Timestamp    int64  `json:"timestamp"`
}
```

**Two Communication Channels:**
1. **Stream-based invites** - Direct 1-1 invitation messages
2. **GossipSub broadcasts** - Group message distribution

### 3. Conference Manager

**File:** `conference/manager.go`

Core business logic for conference management:

```go
type Manager struct {
    storage       storage.Storage
    host          host.Host
    pubsub        *pubsub.PubSub
    currentUserID int64
    
    subscriptions map[int64]*pubsub.Subscription  // Per-conference subscriptions
    topics        map[int64]*pubsub.Topic         // Per-conference topics
    cancelFuncs   map[int64]context.CancelFunc    // Cleanup functions
    mu            sync.RWMutex
}
```

**Key Methods:**

- `CreateConference(ctx, user, name)` - Creates new conference, auto-joins creator
- `InviteToConference(ctx, user, confID, username)` - Sends stream-based invite to friend
- `JoinConference(ctx, user, confID)` - Subscribes to GossipSub topic, starts listening
- `SendMessage(ctx, user, confID, content)` - Publishes message to topic
- `SubscribeToConference(ctx, user, confID)` - Background goroutine for receiving messages
- `LeaveConference(ctx, user, confID)` - Unsubscribes and marks inactive

**Real-Time Message Reception:**

Each conference spawns a background goroutine that listens to the GossipSub topic:

```go
func (m *Manager) SubscribeToConference(ctx context.Context, currentUser *storage.User, conferenceID int64) error {
    // Subscribe to topic: /whisper/conf/<id>
    // Start goroutine to listen for messages
    // Store messages in database
    // Display real-time notifications
}
```

### 4. Database Schema

**Files:** `storage/models.go`, `storage/storage.go`, `storage/sqlite.go`

#### Conferences Table
```sql
CREATE TABLE conferences (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    creator_id INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (creator_id) REFERENCES users(id)
);
```

#### Conference Participants Table
```sql
CREATE TABLE conference_participants (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    conference_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    active BOOLEAN DEFAULT TRUE,
    FOREIGN KEY (conference_id) REFERENCES conferences(id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE(conference_id, user_id)
);
```

#### Conference Messages Table
```sql
CREATE TABLE conference_messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    conference_id INTEGER NOT NULL,
    from_peer_id TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (conference_id) REFERENCES conferences(id)
);
```

**Storage Interface Methods:**
- `CreateConference(ctx, conference)` - Insert new conference
- `GetConference(ctx, id)` - Retrieve conference by ID
- `GetConferencesByUser(ctx, userID)` - List user's conferences
- `AddParticipant(ctx, confID, userID)` - Add participant
- `GetParticipants(ctx, confID)` - List active participants
- `RemoveParticipant(ctx, confID, userID)` - Mark participant as inactive
- `CreateConferenceMessage(ctx, msg)` - Store message
- `GetConferenceMessages(ctx, confID, limit)` - Retrieve history

### 5. CLI Commands

**File:** `main.go`

Eight new commands for conference management:

#### create-conf
```bash
create-conf <name>
```
Creates a new conference and automatically joins the creator.

**Example:**
```bash
> create-conf "Study Group"
✓ Conference 'Study Group' created! (ID: 1)
```

#### invite-conf
```bash
invite-conf <conference-id> <username>
```
Invites a friend to join a conference. Only friends can be invited.

**Example:**
```bash
> invite-conf 1 bob
✓ Invitation sent to bob
```

The invited user receives a real-time notification:
```
📨 Conference invitation from Alice Smith: Study Group (ID: 1)
Use 'join-conf 1' to join
```

#### join-conf
```bash
join-conf <conference-id>
```
Joins a conference and subscribes to receive messages.

**Example:**
```bash
> join-conf 1
✓ Joined conference 'Study Group'
```

#### conf-msg
```bash
conf-msg <conference-id> <message>
```
Sends a message to the conference via GossipSub.

**Example:**
```bash
> conf-msg 1 Hello everyone!
✓ Message sent to conference
```

Other participants see the message in real-time:
```
📨 [Study Group] Alice Smith: Hello everyone!
```

#### conf-list
```bash
conf-list
```
Lists all conferences the user is a member of.

**Example:**
```bash
> conf-list
Your conferences (2):
  1. Study Group (ID: 1)
  2. Project Team (ID: 2)
```

#### conf-history
```bash
conf-history <conference-id> [limit]
```
Views message history for a conference (default limit: 20).

**Example:**
```bash
> conf-history 1 10

=== Conference: Study Group (10 messages) ===
[14:23:15] Alice Smith: Hello everyone!
[14:23:42] Bob Jones: Hey Alice!
[14:24:01] Charlie Brown: Hi all!
```

#### conf-members
```bash
conf-members <conference-id>
```
Lists all participants in the conference.

**Example:**
```bash
> conf-members 1
Conference participants (3):
  1. Alice Smith (active) - Nov 8
  2. Bob Jones (active) - Nov 8
  3. Charlie Brown (left) - Nov 7
```

#### leave-conf
```bash
leave-conf <conference-id>
```
Leaves a conference and unsubscribes from the topic.

**Example:**
```bash
> leave-conf 1
✓ Left conference 'Study Group'
```

### 6. Help Text

Updated `showHelp()` to include conference commands section between Messaging and P2P commands.

## Architecture

### GossipSub Topics

Each conference has a unique GossipSub topic:
```
/whisper/conf/<conference-id>
```

**Example:** Conference ID 1 uses topic `/whisper/conf/1`

### Message Flow

#### Creating and Joining

1. **Alice creates conference:**
   - Database: Insert into `conferences` table
   - Database: Add Alice to `conference_participants`
   - Manager: Auto-join (subscribe to topic)

2. **Alice invites Bob:**
   - Check: Bob must be Alice's friend
   - P2P: Send invite via stream protocol
   - Bob receives real-time notification

3. **Bob joins:**
   - Database: Add Bob to `conference_participants`
   - GossipSub: Subscribe to `/whisper/conf/1`
   - Goroutine: Start listening for messages

#### Sending Messages

1. **Alice sends message:**
   - Encode: JSON ConferenceGossipMessage
   - GossipSub: Publish to `/whisper/conf/1`
   - Database: Store in `conference_messages`

2. **Propagation:**
   - GossipSub automatically propagates to all subscribers
   - Redundant paths ensure delivery
   - No central server needed

3. **Bob receives message:**
   - Goroutine: Receives from subscription
   - Database: Store message locally
   - Display: Real-time notification to user

### Conference Manager Lifecycle

```go
// On JoinConference:
1. Subscribe to GossipSub topic
2. Create background goroutine
3. Store subscription, topic, cancelFunc

// Background goroutine:
for {
    msg, err := sub.Next(ctx)
    // Decode message
    // Store in database
    // Display notification
}

// On LeaveConference:
1. Call cancelFunc (stops goroutine)
2. Unsubscribe from topic
3. Mark participant as inactive
4. Clean up maps
```

## Dependencies

### New Dependency Added

```bash
go get github.com/libp2p/go-libp2p-pubsub@v0.15.0
```

### libp2p Upgrade

Phase 5 implementation resulted in libp2p being upgraded:
- **Previous:** v0.36.5
- **Current:** v0.39.1

## Testing

### Test Setup

Three peer instances are required to properly test conference chat:

```bash
# Setup
mkdir -p ~/whisper-test/{alice,bob,charlie}
cp ./whisper ~/whisper-test/alice/
cp ./whisper ~/whisper-test/bob/
cp ./whisper ~/whisper-test/charlie/
```

### Test Scenario

See `DEMO_PHASE5.md` for complete testing walkthrough with three peers demonstrating:
- Conference creation
- Friend invitations
- Real-time group messaging
- Message history
- Participant management

## Design Decisions

### 1. GossipSub vs Direct Streams

**Why GossipSub for conferences?**
- Efficient broadcasting (no need to send to each peer individually)
- Built-in redundancy (messages propagate through multiple paths)
- Scalable (works with many participants)
- Standard libp2p protocol

**Why Streams for invites?**
- Direct, private communication
- Immediate feedback (success/failure)
- Fits existing friend protocol pattern

### 2. Friend-Only Invites

Users can only invite their friends to conferences:
```go
// Check if invitee is a friend
friends, err := m.storage.GetFriends(ctx, currentUser.ID)
isFriend := false
for _, f := range friends {
    if f.Username == inviteeUsername {
        isFriend = true
        break
    }
}
if !isFriend {
    return fmt.Errorf("you can only invite friends to conferences")
}
```

**Rationale:**
- Prevents spam
- Maintains trust network
- Aligns with P2P social model

### 3. Automatic Message Persistence

All conference messages are stored in the database:
```go
// Store message from GossipSub
message := &storage.ConferenceMessage{
    ConferenceID: gossipMsg.ConferenceID,
    FromPeerID:   gossipMsg.FromPeerID,
    Content:      gossipMsg.Content,
    CreatedAt:    time.Unix(gossipMsg.Timestamp, 0),
}
m.storage.CreateConferenceMessage(ctx, message)
```

**Benefits:**
- Users can view history
- Offline users catch up when they rejoin
- Audit trail for group conversations

### 4. Active Participant Tracking

Participants are marked as "active" or "left":
```sql
active BOOLEAN DEFAULT TRUE
```

**Purpose:**
- Track who's currently in the conference
- Allow rejoining (don't use unique constraint for status)
- Display accurate member lists

### 5. Conference ID in Topic Name

Topics use conference ID directly:
```go
topicName := fmt.Sprintf("/whisper/conf/%d", conferenceID)
```

**Alternative considered:** Using conference name (rejected)
- Names could conflict
- IDs are unique and immutable
- Simpler implementation

## Known Limitations

### Current Limitations

1. **Cannot remove participants** - Only self-removal via `leave-conf`
2. **No conference deletion** - Conferences persist indefinitely
3. **No admin/moderator roles** - All participants have equal privileges
4. **Cannot rename conferences** - Names are immutable after creation
5. **No conference discovery** - Must be invited by existing member
6. **No message encryption** - Conference messages are plaintext over GossipSub

### By Design

1. **Creator has no special powers** - Decentralized model (no "owner")
2. **No read receipts** - Would be complex with multiple readers
3. **No participant limits** - GossipSub handles scaling

## Future Enhancements

Possible improvements for Phase 5+:

1. **Conference Encryption** - End-to-end encryption for group messages
2. **Moderation** - Admin roles, kick/ban capabilities
3. **Conference Discovery** - Public conferences, search functionality
4. **Rich Media** - File sharing, images, reactions
5. **Participant Presence** - Online/offline status in conferences
6. **Message Threading** - Reply to specific messages
7. **Notifications Settings** - Mute, alerts preferences

## Files Modified/Created

### Created
- `conference/protocol.go` - Protocol definitions and message types
- `conference/manager.go` - Conference business logic

### Modified
- `p2p/host.go` - Added GossipSub integration
- `storage/models.go` - Added Conference, ConferenceParticipant, ConferenceMessage models
- `storage/storage.go` - Added conference-related interface methods
- `storage/sqlite.go` - Implemented conference storage methods, added migrations
- `main.go` - Added 8 conference CLI commands, updated help text
- `go.mod` / `go.sum` - Added libp2p-pubsub dependency

## Conclusion

Phase 5 successfully implements group messaging for Whisper using GossipSub. The implementation provides:

- ✅ Efficient group message broadcasting
- ✅ Real-time message delivery
- ✅ Message persistence and history
- ✅ Friend-based invitation system
- ✅ Participant management
- ✅ Clean command-line interface
- ✅ Fully distributed architecture (no central server)

The system is now ready for GUI implementation in Phase 6 using Wails.

---

**Next Phase:** Phase 6 - Wails Desktop GUI
