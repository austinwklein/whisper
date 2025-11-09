# Phase 4: Direct Messaging - Demo Walkthrough

This document demonstrates the direct messaging functionality added in Phase 4.

## Features Demonstrated

1. Sending direct messages between friends
2. Real-time message delivery
3. Message persistence and offline queue
4. Message history with delivery/read receipts
5. Unread message notifications
6. Automatic retry of undelivered messages on login

## Prerequisites

- Two terminal windows
- Clean database (recommended: `make reset`)
- Both users must be friends (from Phase 3)

## Demo Steps

### Step 1: Setup Environment

```bash
# Terminal 1
pkill -f whisper
make reset
./whisper
```

```bash
# Terminal 2
WHISPER_PORT=10000 ./whisper
```

### Step 2: Register and Login Users

**Terminal 1 (Alice):**
```
> register alice pass123 "Alice Smith"
✓ Registration successful! You can now login with: login alice <password>

> login alice pass123
✓ Welcome back, Alice Smith!
```

**Terminal 2 (Bob):**
```
> register bob pass456 "Bob Jones"
✓ Registration successful! You can now login with: login bob <password>

> login bob pass456
✓ Welcome back, Bob Jones!
```

### Step 3: Connect Peers and Establish Friendship

**Terminal 2 (Bob):**
Copy Alice's multiaddress from Terminal 1, then:
```
> connect /ip4/127.0.0.1/tcp/9999/p2p/<alice-peer-id>
✓ Successfully connected!

> add alice
📤 Friend request sent to alice
```

**Terminal 1 (Alice):**
```
📨 Friend request from Bob Jones (bob)

> accept bob
✓ Friend request accepted!
📨 Bob Jones accepted your friend request!
```

### Step 4: Send Direct Messages

**Terminal 1 (Alice):**
```
> msg bob Hey Bob! How are you doing?
✓ Message sent to bob
```

**Terminal 2 (Bob):**
You should see:
```
📨 New message from Alice Smith (alice): Hey Bob! How are you doing?
```

**Terminal 2 (Bob):**
```
> msg alice I'm doing great, thanks for asking! How about you?
✓ Message sent to alice
```

**Terminal 1 (Alice):**
```
📨 New message from Bob Jones (bob): I'm doing great, thanks for asking! How about you?
```

### Step 5: View Message History

**Terminal 1 (Alice):**
```
> history bob

=== Conversation with Bob Jones (3 messages) ===
[14:23:45] You: Hey Bob! How are you doing? ✓✓
[14:24:12] Bob Jones: I'm doing great, thanks for asking! How about you?
[14:24:35] You: Pretty good! Working on this P2P chat app. ✓
```

**Legend:**
- `✓` = Message delivered
- `✓✓` = Message read
- No checkmark = Not yet delivered

**Terminal 2 (Bob):**
```
> history alice 10

=== Conversation with Alice Smith (3 messages) ===
[14:23:45] Alice Smith: Hey Bob! How are you doing?
[14:24:12] You: I'm doing great, thanks for asking! How about you? ✓✓
[14:24:35] Alice Smith: Pretty good! Working on this P2P chat app.
```

### Step 6: Test Unread Message Notifications

**Terminal 1 (Alice):**
```
> msg bob Here's another message!
✓ Message sent to bob

> msg bob And one more!
✓ Message sent to bob
```

**Terminal 2 (Bob):**
```
> unread

=== Unread Messages ===
Alice Smith (alice): 2 unread message(s)

Use 'history <username>' to read messages

> history alice
=== Conversation with Alice Smith (5 messages) ===
[14:23:45] Alice Smith: Hey Bob! How are you doing?
[14:24:12] You: I'm doing great, thanks for asking! How about you? ✓✓
[14:24:35] Alice Smith: Pretty good! Working on this P2P chat app.
[14:25:10] Alice Smith: Here's another message!
[14:25:15] Alice Smith: And one more!

> unread
No unread messages
```

### Step 7: Test Offline Message Queue

**Terminal 2 (Bob):**
```
> logout
✓ Logged out bob

> quit
```

**Terminal 1 (Alice):**
```
> msg bob This message will be queued since you're offline
✓ Message saved (user offline, will deliver when online)

> msg bob Another queued message
✓ Message saved (user offline, will deliver when online)
```

**Terminal 2 (Bob) - Restart and login:**
```
WHISPER_PORT=10000 ./whisper

> login bob pass456
✓ Welcome back, Bob Jones!
Found 2 undelivered message(s), attempting delivery...
✓ Delivered message to bob
✓ Delivered message to bob

📨 New message from Alice Smith (alice): This message will be queued since you're offline
📨 New message from Alice Smith (alice): Another queued message
```

### Step 8: View All Friends and Their Status

**Terminal 1 (Alice):**
```
> friends
Your friends (1):
  1. ● Alice Smith (alice)
```

`●` indicates the friend is online and messages will be delivered immediately.
`○` indicates the friend is offline and messages will be queued.

### Step 9: Test Message Persistence

Messages are stored in the SQLite database and persist across restarts.

**Terminal 1 (Alice):**
```
> quit
```

**Restart Alice:**
```
./whisper

> login alice pass123
✓ Welcome back, Alice Smith!

> history bob
=== Conversation with Bob Jones (7 messages) ===
[... all previous messages are still there ...]
```

## CLI Commands Reference

### Messaging Commands

| Command | Description | Example |
|---------|-------------|---------|
| `msg <username> <message>` | Send a direct message to a friend | `msg alice Hello there!` |
| `history <username> [limit]` | View message history (default 20) | `history bob 50` |
| `unread` | Show all unread messages | `unread` |

## Features Explained

### 1. Real-Time Message Delivery
When both users are online and connected, messages are delivered instantly via libp2p streams.

### 2. Offline Message Queue
If the recipient is offline, messages are:
- Saved to the local database
- Marked as "undelivered"
- Automatically delivered when the recipient comes online
- Retried on each login

### 3. Delivery Receipts
- Single checkmark (✓): Message delivered to recipient
- Double checkmark (✓✓): Message read by recipient
- No checkmark: Message not yet delivered (queued)

### 4. Read Receipts
When you view message history with a friend:
- Messages are automatically marked as read
- A read receipt is sent to the sender (if online)
- The sender sees ✓✓ in their history

### 5. Friend-Only Messaging
You can only send messages to accepted friends. This prevents spam and unwanted messages.

## Protocol Details

### libp2p Protocols Used

1. **`/whisper/message/direct/1.0.0`** - Direct message delivery
2. **`/whisper/message/ack/1.0.0`** - Delivery acknowledgment
3. **`/whisper/message/read/1.0.0`** - Read receipt notification

### Message Flow

```
[Sender]                    [libp2p]                    [Recipient]
   |                           |                            |
   |-- Save to local DB ------>|                            |
   |                           |                            |
   |-- Open stream ----------->|                            |
   |                           |-- DirectMessage ---------->|
   |                           |                            |-- Save to DB
   |                           |                            |-- Display notification
   |                           |                            |
   |                           |<-- Ack (delivered) --------|
   |<-- Mark delivered --------|                            |
```

### Offline Message Handling

```
[Sender]                    [Database]                  [Recipient]
   |                           |                            |
   |-- Save message ---------->|                            |
   |    (delivered=false)      |                            |
   |                           |                            |
   |                           |                            |-- Login
   |                           |                            |
   |                           |<-- Get undelivered --------|
   |<-- Retry delivery --------|                            |
   |                           |                            |
   |-- Mark delivered -------->|                            |
```

## Database Schema

Messages table:
```sql
CREATE TABLE messages (
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
```

## Troubleshooting

### Messages Not Sending
**Problem:** "You must be friends with X to send messages"
**Solution:** 
1. Use `add <username>` to send friend request
2. Have recipient accept with `accept <username>`
3. Verify with `friends` command

### Messages Not Delivered
**Problem:** Message says "user offline, will deliver when online"
**Solution:**
- This is normal behavior when recipient is not connected
- Message will be delivered automatically when recipient logs in
- Check connection with `peers` command

### Can't See Message History
**Problem:** `history` shows "No message history"
**Solution:**
- Ensure you're friends with the user
- Check that you've sent or received messages
- Verify username spelling

### Unread Count Not Updating
**Problem:** `unread` still shows messages after reading
**Solution:**
- Use `history <username>` to view and mark messages as read
- Simply receiving messages doesn't mark them as read
- The `unread` command automatically refreshes

## Performance Notes

- Messages are stored locally in SQLite
- Message delivery uses libp2p streams (TCP)
- No message size limit currently (consider adding one)
- History query is limited (default 20, max configurable)
- Unread check scans last 50 messages per friend

## Security Considerations

- Messages are **not encrypted** in this phase (plaintext over libp2p)
- Messages are stored in plaintext in the database
- No message deletion functionality
- Friend-only messaging provides basic access control
- Consider adding encryption in future phases

## Next Steps (Phase 5)

Phase 5 will add:
- Conference (group) chat using GossipSub
- Multi-party messaging
- Conference creation and management
- Participant management

## Testing Checklist

- [x] Send message between online friends
- [x] Receive real-time notifications
- [x] View message history
- [x] Check delivery receipts (✓)
- [x] Check read receipts (✓✓)
- [x] Send message to offline friend
- [x] Verify message queued
- [x] Recipient logs in and receives queued messages
- [x] View unread messages
- [x] Mark messages as read
- [x] Message persistence across restarts
- [x] Friend-only messaging enforcement

---

**Phase 4 Status:** ✅ Complete
**Date:** 2025-11-08
**Features:** Direct messaging, offline queue, delivery/read receipts
