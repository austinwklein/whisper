# Whisper: User Guide

Welcome to Whisper, a truly decentralized chat application. No servers, no central authority, no censorship. Just you and your peers, communicating freely.

---

## Table of Contents

1. [What is Whisper?](#what-is-whisper)
2. [Getting Started](#getting-started)
3. [Core Features](#core-features)
4. [How to Use](#how-to-use)
5. [Privacy & Security](#privacy--security)
6. [FAQ](#faq)
7. [Troubleshooting](#troubleshooting)

---

## What is Whisper?

Whisper is a peer-to-peer (P2P) chat application that puts you in complete control. Unlike traditional chat apps:

- **No Company Owns Your Data:** Your messages stay on your device. No servers collect them.
- **Censorship-Resistant:** No central authority can shut it down or block your communication.
- **No Tracking:** Your location, IP, and communication patterns are private.
- **Offline Works:** Messages can reach you even when you're offline.
- **True Ownership:** You own your account, your contacts, your messages.

**Use Cases:**
- Private conversations with friends
- Group discussions without corporate surveillance
- Communities in restricted regions
- Teams that need decentralized infrastructure

---

## Getting Started

### Installation

**Requirements:**
- Windows, macOS, or Linux
- ~50 MB disk space
- Internet connection (for peer discovery)

**Steps:**
1. Download Whisper from [GitHub releases](https://github.com/your-username/whisper/releases)
2. Extract and run the application
3. Complete the registration form (see below)

### First Run

**Welcome Screen → Registration**

Fill out:
- **Username:** Your unique handle (e.g., `alice`)
- **Password:** Strong password (you're the only one who stores it)
- **Full Name:** Your real or display name (e.g., `Alice Smith`)

That's it! Your account is created locally on your device.

**Important:**
- Your credentials are stored **only** on your computer
- If you lose your device, you lose access to your account (no password recovery)
- Write down your username and password somewhere safe

---

## Core Features

### 1. Friend System

**Adding Friends:**
1. Go to "Friends" tab
2. Click "Search for Friends"
3. Enter their name (or part of it, e.g., "Smith")
4. Select from results
5. Send friend request
6. Wait for them to authorize

**What Happens:**
- When you search, Whisper queries the P2P network for users matching that name
- Your friend gets a notification: "[Your Name] wants to be friends"
- Once they authorize, you become friends
- Now you can send each other messages

**Online Status:**
- Green dot = Friend is currently online
- Gray dot = Friend is offline
- "Appeared 2h ago" = Last seen 2 hours ago

### 2. Direct Messaging

**Send a Message:**
1. Click on a friend's name
2. Type your message
3. Press Enter or click Send

**Online Messages:**
- Friend receives immediately (if online)
- Message shows "Delivered"

**Offline Messages:**
- Friend receives when they log in next
- Message shows "Sent" (not yet delivered)
- Once delivered, changes to "Delivered"

**Important:**
- Direct messages are **end-to-end** between you and your friend
- No one else can see them
- Messages persist in your local history even if sender goes offline

### 3. Conference Chats (Groups)

**Create a Conference:**
1. Go to "Conferences" tab
2. Click "Create Conference"
3. Name it (e.g., "Dev Team", "Book Club")
4. Add participants
5. Start chatting

**Add People to a Conference:**
- Anyone in the conference can add new members
- New members can't join without invitation
- Members see all previous messages (stored locally)

**Leave a Conference:**
- Click "Leave" in conference info
- You'll no longer receive new messages
- To rejoin, someone must invite you again

**What Happens When You Log Out:**
- You automatically leave all conferences
- You won't see new messages sent after you left
- If re-invited, you'll start fresh (won't see old history)

---

## How to Use: Step-by-Step Workflows

### Workflow 1: Start Chatting with a Friend

**You have their contact info (phone, email, etc.):**

1. **Ask for their Peer Address**
    - They go: "Menu → My Peer Address"
    - They share the address with you (text, email, etc.)
    - Example: `/ip4/192.168.1.5/tcp/9999/p2p/QmAbcDef123`

2. **Connect to Them**
    - You go: "Menu → Connect to Peer"
    - Paste their address
    - Click "Connect"
    - Wait for connection (should be instant)

3. **Search for Them**
    - Go to "Friends" tab
    - Search by name
    - Send friend request
    - Wait for authorization

4. **Start Chatting**
    - Once authorized, click on their name
    - Type and send messages
    - Chat works online and offline

### Workflow 2: Create a Group Chat

1. **Click "Create Conference"**
2. **Name it** (e.g., "Project Alpha")
3. **Add people:**
    - Search by name
    - Select each person to add
    - Click "Create Conference"
4. **Start typing**
    - Everyone in the conference sees your messages
    - Messages appear for all participants in real-time

### Workflow 3: Invite Someone to Your Network

**You want to bring a new person into your friend circle:**

1. **Tell them to install Whisper** and register
2. **Exchange peer addresses with them:**
    - You: "Menu → My Peer Address" (copy)
    - Share with them
3. **They connect to you:**
    - They paste your address in "Connect to Peer"
4. **Search and friend request** (they search for you)
5. **Now they can reach your other friends** through peer exchange
    - They don't need everyone's address, just one person
    - The network will introduce them to others

---

## Privacy & Security

### What Whisper Protects

✅ **Messages Only You See**
- Direct messages between you and your friend are private
- Group messages visible only to conference members

✅ **No Tracking**
- Whisper doesn't track your location
- Doesn't know who your friends are (except those you authorize)
- Can't see your messages (they're local on your device)

✅ **No Metadata Collection**
- No server logs when you log in
- No server knows how many messages you send
- Connection times are private (except to peers you connect to)

✅ **Local Control**
- All your data stays on your computer
- You decide what to store
- You can delete everything anytime

### What Whisper Does NOT Protect (Important!)

❌ **Your IP Address (without VPN/Tor)**
- Your IP is published to the DHT (Distributed Hash Table) for peer discovery
- Peers can see your IP when you search for them or they search for you
- Your ISP can approximate your geographic location from your IP
- Attackers on the network could DDoS your IP if they identify you

❌ **Message Encryption**
- Messages use signing (verification that sender is real)
- Not encrypted end-to-end (encrypted upgrade planned for v0.2+)
- Messages stored locally in plain text
- Use VPN for additional privacy on public WiFi

❌ **Query Patterns**
- The DHT nodes can see that an IP queried for "user:alice"
- Pattern analysis could reveal your friend groups
- Timing of queries could leak behavioral information

❌ **Deleted Messages**
- Other peers may have copies
- Once sent, message can't be recalled
- History is local but not automatically deleted

---

### Privacy Threat Model & Mitigations

**Threat: Someone identifies you by IP and DDoS attacks you**

| Threat Level | Who | How | Mitigation |
|---|---|---|---|
| Friend | Medium | They can see your IP on DHT | Trust only peers you know |
| ISP | Low | ISP sees IP + patterns | Use VPN to hide activity |
| Govt | Context | Can subpoena ISP for user info | Depends on jurisdiction, use Tor |
| Network attacker | Low-Medium | Man-in-the-middle on your connection | Use VPN or Tor, Whisper uses TLS by default |

---

### Best Practices for Privacy

#### 1. Protect Your IP Address
**Option A: Use VPN** (Recommended for casual users)
- Download a reputable VPN (Mullvad, ProtonVPN, etc.)
- Connect to VPN first
- Run Whisper
- Your VPN exit node IP is published to DHT, not your real IP
- Even if someone traces DHT, they find the VPN provider, not you

**Option B: Use Tor Browser** (Recommended for high privacy)
- Download Tor Browser
- Use Whisper through Tor
- Your Tor exit node IP is published to DHT
- Much harder to trace, but slower

**Option C: No Protection** (If you don't care about geographic privacy)
- Run Whisper directly
- Your real IP is visible to peers
- Local network admin can see your activity

#### 2. Use Strong Passwords
- Mix uppercase, lowercase, numbers, symbols
- At least 12 characters
- Don't reuse passwords from other apps
- Use a password manager if possible

#### 3. Protect Your Device
- Use device lock (fingerprint, PIN)
- Enable full-disk encryption
- Keep OS and apps updated
- Use antivirus/malware protection

#### 4. Backup Important Data
- Export your chat history periodically
- Store in secure location (encrypted USB)
- If device lost, account is gone (no recovery)

#### 5. Be Selective with Peer Connections
- Only connect to peer addresses from **trusted people**
- Don't share your peer address publicly or on untrusted platforms
- Verify peer address is correct (typos could connect you to attacker)

#### 6. Use on Trusted Networks
- On public WiFi? Use VPN
- Corporate network? Use VPN (employer can monitor)
- Very sensitive data? Use Tor Browser
- Home network? Safe from outsiders, but ISP can see activity

#### 7. Assume Peers Are Not Trustworthy
- Don't send sensitive information even to friends
- Assume peers might be compromised or malicious
- Messages are signed but not encrypted (upgrade in v0.2+)
- Think twice before adding users you don't know

---

### Understanding DHT & Privacy

**What is the DHT?**
- Distributed Hash Table: a decentralized phonebook
- Maps "user:alice" to "her peer address"
- No central server (all peers maintain it together)

**How it affects privacy:**
- When you search for a friend, your search goes to DHT nodes
- DHT nodes could theoretically track who is searching for whom
- Mitigation planned for v0.2: onion routing (queries relayed through peers)

**Current limitation:**
- DHT nodes can correlate: "IP at 203.0.113.50 searched for alice"
- If you use VPN/Tor: "VPN exit node X searched for alice" (much better)
- Pattern analysis could still leak friend groups

**Example:**
```
Without VPN:
  Your real IP → DHT node → sees you query for "alice", "bob", "charlie"
  Attacker knows: you're in friend group with alice, bob, charlie

With VPN:
  Your VPN exit IP → DHT node → sees VPN IP query for "alice", etc.
  Attacker knows: a VPN user is in that friend group (much harder to trace back to you)

With Tor (v0.2+):
  Onion-routed query → DHT node doesn't see your query at all
  Query goes through random peers to DHT
  Attacker can't correlate queries to identity
```

---

### Future Privacy Improvements (v0.2+)

**Planned Upgrades:**
- ✅ End-to-end encryption (messages encrypted, signed)
- ✅ Onion routing for DHT queries (hide what you search for)
- ✅ Circuit relay option (hide IP from direct peers)
- ✅ Message encryption at rest (local database encrypted)

---

### Is Whisper Right for You?

| Your Need                             | Recommended? | Why                                              |
|---------------------------------------|--------------|--------------------------------------------------|
| Private chat with friends             | ✅ Yes        | Messages stay local, friends can't spy           |
| Hide from ISP/govt                    | ⚠️ Partial   | Use with VPN/Tor, IP protection not built-in yet |
| Hide from friend you're chatting with | ❌ No         | They see your peer info (unless you use relay)   |
| Complete anonymity                    | ⚠️ Partial   | Use Tor + VPN, query privacy coming in v0.2      |
| Secure from company                   | ⚠️ Depends   | On corporate network? Use VPN outside network    |
| Casual group chat                     | ✅ Yes        | Simple, no central server tracking you           |
| Whistleblowing/activism               | ⚠️ Careful   | Recommended with Tor, but stay vigilant          |



---

## FAQ

**Q: Is Whisper encrypted?**
A: Messages are digitally signed (proof of sender). They're not end-to-end encrypted (not yet), so don't send passwords or highly sensitive data. Upgrades planned for v0.2+. For privacy, use VPN/Tor (see below).

**Q: What is this DHT thing I keep hearing about?**
A: DHT stands for "Distributed Hash Table." Think of it as a decentralized phonebook. When you sign up, Whisper publishes: "alice maps to my peer address." When your friend searches for you, Whisper queries the DHT to find your current address. No central server maintains the phonebook—all peers work together to maintain it.

**Q: What happens if the person I'm chatting with goes offline?**
A: Messages you send are stored locally. They receive them when they log in next. No messages are lost.

**Q: How does Whisper handle my IP address changing?**
A: When your IP changes (switched networks, ISP reassignment, etc.), Whisper automatically republishes your new address to the DHT. Your friends' next search for you gets the new address. You don't need to manually re-share addresses.

**Q: Can I use Whisper on my phone?**
A: Not yet. Currently desktop only (Windows, macOS, Linux). Mobile apps planned for future.

**Q: How many people can I have in a conference?**
A: Technically unlimited, but performance degrades with large groups. Tested up to ~50 people. For very large groups, create multiple smaller conferences.

**Q: What if I lose my device?**
A: Your account is tied to that device. If you lose it, you lose access. No cloud backup. Best practice: write down credentials, consider creating a second installation on a secure backup device.

**Q: Can I message people who aren't my friends?**
A: No. Messages only work with authorized friends or conference members. This prevents spam and harassment.

**Q: How do I backup my messages?**
A: Whisper stores everything locally. Backup your application data folder. Location depends on OS:
- Windows: `%APPDATA%\Whisper`
- macOS: `~/Library/Application Support/Whisper`
- Linux: `~/.local/share/whisper`

**Q: Should I be worried about my IP being visible in DHT?**
A: Depends on your threat model. If you're concerned:
- Use a VPN: Your VPN exit node IP is published, not your real IP
- Use Tor: Even better, but slower
- At minimum: Don't share your peer address publicly
- Planned for v0.2: Onion routing to further protect query privacy

**Q: If I use VPN, am I completely anonymous?**
A: VPN protects your IP address from being traced to your location. However:
- VPN provider can see your traffic (choose a trustworthy one)
- Behavioral patterns could still reveal friend groups (DHT query timing analysis)
- For maximum privacy: Use Tor + VPN + Whisper together
- Planned upgrade (v0.2): Onion routing in DHT queries for additional privacy

**Q: Can DHT nodes see who I'm talking to?**
A: DHT nodes can see IP addresses querying for usernames. Current threat:
- "IP 203.0.113.50 searched for alice, bob, charlie" → reveals friend group
- Mitigation: Use VPN/Tor (DHT sees "VPN exit X searched" instead of your real IP)
- Future: Onion routing in v0.2 (queries routed through peers, DHT can't correlate)

**Q: Can Whisper be used for dangerous/illegal activities?**
A: Whisper is a tool. Like any tool, it can be misused. Whisper developers don't condone illegal activity. Users are responsible for complying with laws.

**Q: Who runs Whisper?**
A: No one owns or runs Whisper. It's open-source software. Each user runs it on their own device. The network is maintained by all users together.

**Q: Is there a Whisper server?**
A: No. The entire system is peer-to-peer. There's no central server. If it existed, it would defeat the purpose of Whisper.

**Q: Can Whisper be censored or shut down?**
A: Not by a central authority. To censor Whisper, you'd have to block every user individually. If everyone stopped using it, it would die, but there's no kill switch.

---

## Troubleshooting

### Can't Find Friends

**Problem:** Search returns no results

**Solutions:**
1. Make sure friend is online (they need to be running Whisper)
2. Check spelling of their name
3. They must have run Whisper at least once (to broadcast presence)
4. Try searching by full name instead of username
5. Ask them for their peer address, connect directly, then try searching again

### Messages Not Delivering

**Problem:** Message shows "Sent" but friend says they didn't receive

**Solutions (if friend offline):**
1. Wait for them to log in (message will deliver)
2. Messages are stored, they'll get them

**Solutions (if friend online):**
1. Ask them to restart Whisper
2. Check if you're really friends (look at friend list)
3. Try sending a different message
4. Restart both apps and try again

### Can't Connect to Peer

**Problem:** "Connection Failed" when entering peer address

**Solutions:**
1. Double-check the peer address (copy-paste to avoid typos)
2. Make sure they're running Whisper
3. Both peers must be on the network (connected to internet)
4. Try again after 10 seconds
5. Ask them to share peer address again (may have changed)

### Slow/Laggy Messages

**Problem:** Messages take a long time to send/receive

**Causes:**
1. Network connection is slow
2. Too many peers connected (rare)
3. Computer is under heavy load

**Solutions:**
1. Check your internet connection
2. Restart Whisper
3. Disconnect from unused peers (Advanced → Peer Management)
4. Restart your computer

### UI Issues / Crashes

**Problem:** App freezes, buttons don't work

**Solutions:**
1. Restart the application
2. Check if an update is available
3. Restart your computer
4. Clear app data (back it up first!) and reinstall
5. Report the bug on GitHub with details

### Lost Password

**Problem:** Forgot password

**Solutions:**
1. Whisper has no password reset (by design)
2. You'll need to reinstall and create a new account
3. You'll lose access to old account and messages
4. **Prevention:** Use password manager or write it down safely

### Running Multiple Instances

**Problem:** Want to test with multiple users on same computer

**Solutions:**
1. Create different user accounts on your computer
2. Log in as each user and run Whisper separately
3. Or use virtual machines
4. Or run Docker containers (advanced)

---

## Advanced Tips

### Export Chat History

**Backup conversations:**
1. Go to "Settings → Export Data"
2. Choose which conversations to export
3. Save as JSON/CSV
4. Keep in secure location

### Change Password

**Update your password:**
1. Go to "Settings → Change Password"
2. Enter current password
3. Enter new password
4. Confirm
5. All future logins use new password

### View My Peer Address

**Share with others to connect:**
1. Go to "Menu → My Peer Address"
2. Copy to clipboard
3. Share via any channel (email, messaging, etc.)

### Advanced Network Settings

**For power users:**
1. Go to "Settings → Network"
2. Adjust connection limits
3. Change port (if needed for port forwarding)
4. View connected peers
5. Manually manage peer connections

---

## Support

- **Bug Reports:** GitHub Issues  
- **Feature Requests:** GitHub Discussions  
- **General Help:** Read documentation or GitHub Discussions  
- **Security Issues:** Email awklein@alaska.edu  

---

## License & Attribution

Whisper is open-source and free to use.

Built with:
- Go + libp2p (networking)
- Svelte + TypeScript (frontend)
- Wails (desktop framework)

See LICENSE file for details.

---

**Happy Whispering! 🤫**

Remember: True privacy comes from true decentralization.
