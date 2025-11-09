# Phase 2 Demo: User Authentication

This demo shows the complete authentication flow implemented in Phase 2.

## Quick Start Demo

### 1. Start Whisper
```bash
./whisper
```

You'll see:
```
=== Whisper P2P Chat ===
Peer ID: 12D3KooWNYicZLS61Hi74tajw1PdUMhWDLB18XC8LDuLr6KFtWR2

Your multiaddresses:
  /ip4/10.22.23.16/tcp/9999/p2p/12D3KooWNYicZLS61Hi74tajw1PdUMhWDLB18XC8LDuLr6KFtWR2
  /ip4/127.0.0.1/tcp/9999/p2p/12D3KooWNYicZLS61Hi74tajw1PdUMhWDLB18XC8LDuLr6KFtWR2

=== Getting Started ===
To use Whisper, you need to register or login:
  register <username> <password> <full-name>
  login <username> <password>

Type 'help' for all available commands

> 
```

### 2. Register a New User
```bash
> register alice mypassword123 "Alice Smith"
```

Output:
```
✓ Registration successful! You can now login with: login alice <password>
```

### 3. Login
```bash
> login alice mypassword123
```

Output:
```
✓ Welcome back, Alice Smith!
```

### 4. Check Who You Are
```bash
> whoami
```

Output:
```
Username: alice
Full Name: Alice Smith
Peer ID: 12D3KooWNYicZLS61Hi74tajw1PdUMhWDLB18XC8LDuLr6KFtWR2
Account Created: 2024-11-08 14:32:10
```

### 5. Register More Users (in separate terminals)
Terminal 2:
```bash
WHISPER_PORT=10000 ./whisper
> register bob secretpass456 "Bob Smith"
> login bob secretpass456
```

Terminal 3:
```bash
WHISPER_PORT=10001 ./whisper
> register charlie pass789xyz "Charlie Jones"
> login charlie pass789xyz
```

### 6. Search for Users (back in Terminal 1)
```bash
> search Smith
```

Output:
```
Found 2 user(s):
  1. Alice Smith (alice) - Peer ID: 12D3KooW...
  2. Bob Smith (bob) - Peer ID: 12D3KooW...
```

### 7. Change Password
```bash
> passwd mypassword123 newsecurepass999
```

Output:
```
✓ Password changed successfully
```

### 8. Logout and Login Again
```bash
> logout
```

Output:
```
✓ Logged out alice
```

Try old password (should fail):
```bash
> login alice mypassword123
```

Output:
```
Login failed: invalid password
```

Try new password (should work):
```bash
> login alice newsecurepass999
```

Output:
```
✓ Welcome back, Alice Smith!
```

## Error Handling Examples

### Try to register duplicate username:
```bash
> register alice password123 "Alice Another"
```

Output:
```
Registration failed: username already exists
```

### Try weak password:
```bash
> register dave 123 "Dave Test"
```

Output:
```
Registration failed: password must be at least 8 characters
```

### Try to use commands without login:
```bash
> logout
> search John
```

Output:
```
You must be logged in to search for users
```

## Complete Command Reference

Type `help` in the application to see:

```
=== Authentication Commands ===
  register <username> <password> <full-name> - Create new account
  login <username> <password>                - Login to your account
  logout                                      - Logout from current account
  whoami                                      - Show current user info
  passwd <old-pass> <new-pass>               - Change your password
  search <name>                               - Search for users by name

=== P2P Commands ===
  connect <multiaddr>                         - Connect to a peer
  peers                                       - List connected peers

=== General Commands ===
  help                                        - Show this help
  quit                                        - Exit the application
```

## What's Working in Phase 2

✓ **User Registration** - Create accounts with secure password hashing
✓ **Login/Logout** - Session management
✓ **Password Changes** - Update passwords with verification
✓ **User Search** - Find other users by name
✓ **Peer ID Mapping** - Users tied to P2P identities
✓ **Data Persistence** - All data stored in SQLite
✓ **Security** - bcrypt password hashing, no plaintext storage
✓ **Error Handling** - Clear error messages for all failure cases

## Next Phase Preview

Phase 3 will add:
- Friend requests between users
- Accept/reject friend requests
- Online/offline friend status
- DHT-based user discovery
- Direct messaging between friends

Stay tuned! 🚀
