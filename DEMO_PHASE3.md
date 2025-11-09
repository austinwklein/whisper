# Phase 3 Demo: Friend System

This demo showcases the complete friend management system with real P2P friend requests.

## Quick Demo (2 Terminals)

### Terminal 1 - Alice

```bash
# Start Whisper
./whisper

# Register and login
> register alice pass123 "Alice Smith"
✓ Registration successful! You can now login with: login alice <password>

> login alice pass123
✓ Welcome back, Alice Smith!
Registered user 'alice' for peer discovery

# Note your multiaddress for Bob to connect
# Copy one of the addresses shown at startup
```

### Terminal 2 - Bob

```bash
# Start on different port
WHISPER_PORT=10000 ./whisper

# Register and login
> register bob pass456 "Bob Smith"
✓ Registration successful!

> login bob pass456
✓ Welcome back, Bob Smith!

# Connect to Alice using her multiaddress
> connect /ip4/127.0.0.1/tcp/9999/p2p/12D3KooW...
✓ Successfully connected!

# Send friend request
> add alice
Looking up alice...
Connecting to alice...
✓ Friend request sent to Alice Smith (alice)
```

### Back to Terminal 1 - Alice

```bash
# You'll see a notification:
📨 Friend request from Bob Smith (bob)
   Message: Bob Smith wants to be your friend
   Use 'accept bob' or 'reject bob'

# View pending requests
> requests
Pending friend requests (1):
  1. Bob Smith (bob)

Use 'accept <username>' or 'reject <username>'

# Accept the request
> accept bob
✓ Accepted friend request from Bob Smith

# View friends list
> friends
Your friends (1):
  1. ● Bob Smith (bob)
```

### Back to Terminal 2 - Bob

```bash
# You'll see acceptance notification:
✓ Alice Smith accepted your friend request!
   You are now friends with Alice Smith (alice)

# View your friends
> friends
Your friends (1):
  1. ● Alice Smith (alice)
```

## Feature Demos

### 1. Online/Offline Status

Keep both terminals running:

**Terminal 1 (Alice):**
```bash
> friends
Your friends (1):
  1. ● Bob Smith (bob)    # Online (green dot)
```

Close Terminal 2 (Bob's instance), then in Terminal 1:

```bash
> friends
Your friends (1):
  1. ○ Bob Smith (bob)    # Offline (gray dot)
```

### 2. Reject Friend Request

**Terminal 3 - Charlie:**
```bash
WHISPER_PORT=10001 ./whisper

> register charlie pass789 "Charlie Jones"
> login charlie pass789

> connect /ip4/127.0.0.1/tcp/9999/p2p/<alice-peer-id>
> add alice
```

**Terminal 1 - Alice:**
```bash
📨 Friend request from Charlie Jones (charlie)
   Message: Charlie Jones wants to be your friend
   Use 'accept charlie' or 'reject charlie'

> reject charlie
✓ Rejected friend request from Charlie Jones

> friends
Your friends (1):
  1. ● Bob Smith (bob)
  # Charlie is NOT in the list
```

### 3. Duplicate Request Protection

**Terminal 1 - Alice:**
```bash
> add bob
Failed to send friend request: already friends
```

### 4. Search and Add Workflow

**Terminal 1 - Alice:**
```bash
# Search for users by name
> search Smith
Found 2 user(s):
  1. Alice Smith (alice) - Peer ID: 12D3KooW...
  2. Bob Smith (bob) - Peer ID: 12D3KooW...

# Search for partial names
> search Jo
Found 1 user(s):
  1. Charlie Jones (charlie) - Peer ID: 12D3KooW...
```

## Multi-User Demo (3+ Users)

### Setup

Start 3 instances:

**Terminal 1 - Alice (Port 9999)**
```bash
./whisper
> register alice pass123 "Alice Smith"
> login alice pass123
```

**Terminal 2 - Bob (Port 10000)**
```bash
WHISPER_PORT=10000 ./whisper
> register bob pass456 "Bob Jones"
> login bob pass456
> connect <alice-multiaddr>
```

**Terminal 3 - Charlie (Port 10001)**
```bash
WHISPER_PORT=10001 ./whisper
> register charlie pass789 "Charlie Brown"
> login charlie pass789
> connect <alice-multiaddr>
> connect <bob-multiaddr>
```

### Friend Network Building

**Bob adds Alice:**
```bash
# Terminal 2
> add alice
✓ Friend request sent to Alice Smith (alice)
```

**Alice accepts:**
```bash
# Terminal 1
> accept bob
✓ Accepted friend request from Bob Smith
```

**Charlie adds both:**
```bash
# Terminal 3
> add alice
✓ Friend request sent to Alice Smith (alice)

> add bob
✓ Friend request sent to Bob Jones (bob)
```

**Alice and Bob both accept:**
```bash
# Terminal 1
> accept charlie

# Terminal 2
> accept charlie
```

**View everyone's friend network:**
```bash
# Terminal 1 - Alice
> friends
Your friends (2):
  1. ● Bob Jones (bob)
  2. ● Charlie Brown (charlie)

# Terminal 2 - Bob  
> friends
Your friends (2):
  1. ● Alice Smith (alice)
  2. ● Charlie Brown (charlie)

# Terminal 3 - Charlie
> friends
Your friends (2):
  1. ● Alice Smith (alice)
  2. ● Bob Jones (bob)
```

## Error Handling Demos

### 1. Not Logged In
```bash
> add bob
You must be logged in to add friends
```

### 2. User Not Found
```bash
> add nonexistent
Failed to send friend request: target user not found
```

### 3. Cannot Add Self
```bash
> add alice
Failed to send friend request: cannot add yourself as friend
```

### 4. Already Pending
```bash
> add bob
✓ Friend request sent

> add bob
Failed to send friend request: friend request already pending
```

## Real-Time Notifications

All friend system events trigger real-time CLI notifications:

### Friend Request Received:
```
📨 Friend request from Alice Smith (alice)
   Message: Alice Smith wants to be your friend
   Use 'accept alice' or 'reject alice'
> 
```

### Request Accepted:
```
✓ Bob Smith accepted your friend request!
   You are now friends with Bob Smith (bob)
> 
```

### Request Rejected:
```
✗ Charlie Jones declined your friend request
> 
```

## Complete Command Flow

```bash
# Start instance
./whisper

# Setup
> register <username> <password> "<full name>"
> login <username> <password>

# Connect to network
> connect <peer-multiaddr>
> peers  # Verify connection

# Friend management
> search <name>           # Find users
> add <username>          # Send request
> requests                # View incoming
> accept <username>       # Accept
> reject <username>       # Reject
> friends                 # List all friends

# Utility
> whoami                  # Current user info
> help                    # All commands
> quit                    # Exit
```

## Tips for Demo

1. **Keep terminals side-by-side** to see real-time notifications
2. **Note multiaddresses early** for easy peer connections
3. **Use clear, different usernames** (alice, bob, charlie)
4. **Show both accept and reject** flows
5. **Demonstrate online/offline status** by starting/stopping instances
6. **Search before adding** to show user discovery
7. **Try duplicate requests** to show protection

## What's Next?

Phase 4 will add:
- Direct messaging between friends
- Message history
- Offline message delivery
- Read receipts
- Typing indicators

Stay tuned! 🚀
