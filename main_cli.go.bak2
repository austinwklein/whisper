package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/austinwklein/whisper/auth"
	"github.com/austinwklein/whisper/conference"
	"github.com/austinwklein/whisper/config"
	"github.com/austinwklein/whisper/friends"
	"github.com/austinwklein/whisper/messages"
	"github.com/austinwklein/whisper/p2p"
	"github.com/austinwklein/whisper/storage"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

type App struct {
	config            *config.Config
	storage           storage.Storage
	p2p               *p2p.P2PHost
	auth              *auth.AuthService
	friendManager     *friends.Manager
	messageManager    *messages.Manager
	conferenceManager *conference.Manager
}

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize storage
	store, err := storage.NewSQLiteStorage(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Initialize P2P host (no private key = generate new one)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p2pHost, err := p2p.NewP2PHost(ctx, cfg.Port, nil)
	if err != nil {
		log.Fatalf("Failed to initialize P2P host: %v", err)
	}
	defer p2pHost.Close()

	// Initialize auth service
	authService := auth.NewAuthService(store)

	// Initialize friend manager
	friendManager := friends.NewManager(store, p2pHost.Host())

	// Initialize message manager
	messageManager := messages.NewManager(store, p2pHost.Host())

	// Initialize conference manager
	conferenceManager := conference.NewManager(store, p2pHost.Host(), p2pHost.PubSub())

	// Create app
	app := &App{
		config:            cfg,
		storage:           store,
		p2p:               p2pHost,
		auth:              authService,
		friendManager:     friendManager,
		messageManager:    messageManager,
		conferenceManager: conferenceManager,
	}

	// Start app services
	if err := app.Start(ctx); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}

	fmt.Println("\n=== Whisper P2P Chat ===")
	fmt.Printf("Peer ID: %s\n", p2pHost.PeerID())
	fmt.Println("\nYour multiaddresses:")
	for _, addr := range p2pHost.GetFullAddrs() {
		fmt.Printf("  %s\n", addr)
	}
	fmt.Println("\n=== Getting Started ===")
	fmt.Println("1. Register or login:")
	fmt.Println("   register <username> <password> <full-name>")
	fmt.Println("   login <username> <password>")
	fmt.Println()
	fmt.Println("2. Share your multiaddress (above) with a friend")
	fmt.Println()
	fmt.Println("3. Connect to your friend's multiaddress:")
	fmt.Println("   connect <their-multiaddr>")
	fmt.Println("   (This automatically sends a friend request!)")
	fmt.Println()
	fmt.Println("4. Accept their friend request:")
	fmt.Println("   accept <their-username>")
	fmt.Println()
	fmt.Println("Type 'help' for all available commands")
	fmt.Println()

	// Start command loop in a goroutine
	go app.commandLoop(ctx)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down...")
	cancel()
}

func (a *App) Start(ctx context.Context) error {
	// Future: Initialize additional services
	return nil
}

func (a *App) commandLoop(ctx context.Context) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			fmt.Print("> ")
			continue
		}

		parts := strings.Fields(line)
		cmd := parts[0]

		switch cmd {
		case "register":
			if len(parts) < 4 {
				fmt.Println("Usage: register <username> <password> <full-name>")
				fmt.Println("Example: register alice mypassword123 \"Alice Smith\"")
				break
			}
			username := parts[1]
			password := parts[2]
			fullName := strings.Join(parts[3:], " ")
			fullName = strings.Trim(fullName, "\"")

			peerID := a.p2p.PeerID().String()
			err := a.auth.Register(ctx, username, password, fullName, peerID)
			if err != nil {
				fmt.Printf("Registration failed: %v\n", err)
			} else {
				fmt.Printf("✓ Registration successful! You can now login with: login %s <password>\n", username)
			}

		case "login":
			if len(parts) < 3 {
				fmt.Println("Usage: login <username> <password>")
				break
			}
			username := parts[1]
			password := parts[2]

			user, err := a.auth.Login(ctx, username, password)
			if err != nil {
				fmt.Printf("Login failed: %v\n", err)
			} else {
				// Update user's peer ID to current one (in case it changed after restart)
				currentPeerID := a.p2p.PeerID().String()
				if user.PeerID != currentPeerID {
					user.PeerID = currentPeerID
					if err := a.storage.UpdateUser(ctx, user); err != nil {
						fmt.Printf("Warning: Failed to update peer ID: %v\n", err)
					}
				}

				fmt.Printf("✓ Welcome back, %s!\n", user.FullName)
				// Set current user for friend manager, message manager, and conference manager
				a.friendManager.SetCurrentUser(user.ID)
				a.messageManager.SetCurrentUser(user.ID)
				a.conferenceManager.SetCurrentUser(user.ID)
				// Publish user to DHT
				go func() {
					if err := a.p2p.PublishUser(ctx, username); err != nil {
						fmt.Printf("Warning: Failed to publish to DHT: %v\n", err)
					}
					// Keep refreshing presence
					a.p2p.RefreshUserPresence(ctx, username)
				}()
				// Try to deliver any undelivered messages
				go func() {
					if err := a.messageManager.RetryUndeliveredMessages(ctx, user.ID); err != nil {
						fmt.Printf("Warning: Failed to retry undelivered messages: %v\n", err)
					}
				}()
			}

		case "logout":
			if !a.auth.IsAuthenticated() {
				fmt.Println("You are not logged in")
				break
			}
			user, _ := a.auth.CurrentUser()
			a.auth.Logout()
			a.friendManager.SetCurrentUser(0)
			a.messageManager.SetCurrentUser(0)
			a.conferenceManager.SetCurrentUser(0)
			fmt.Printf("✓ Logged out %s\n", user.Username)

		case "whoami":
			if !a.auth.IsAuthenticated() {
				fmt.Println("Not authenticated. Please login first.")
				break
			}
			user, _ := a.auth.CurrentUser()
			fmt.Printf("Username: %s\n", user.Username)
			fmt.Printf("Full Name: %s\n", user.FullName)
			fmt.Printf("Peer ID: %s\n", user.PeerID)
			fmt.Printf("Account Created: %s\n", user.CreatedAt.Format("2006-01-02 15:04:05"))

		case "passwd":
			if !a.auth.IsAuthenticated() {
				fmt.Println("You must be logged in to change password")
				break
			}
			if len(parts) < 3 {
				fmt.Println("Usage: passwd <old-password> <new-password>")
				break
			}
			oldPassword := parts[1]
			newPassword := parts[2]

			err := a.auth.ChangePassword(ctx, oldPassword, newPassword)
			if err != nil {
				fmt.Printf("Failed to change password: %v\n", err)
			} else {
				fmt.Println("✓ Password changed successfully")
			}

		case "search":
			if !a.auth.IsAuthenticated() {
				fmt.Println("You must be logged in to search for users")
				break
			}
			if len(parts) < 2 {
				fmt.Println("Usage: search <name>")
				break
			}
			searchName := strings.Join(parts[1:], " ")
			searchName = strings.Trim(searchName, "\"")

			users, err := a.auth.SearchUsers(ctx, searchName)
			if err != nil {
				fmt.Printf("Search failed: %v\n", err)
				break
			}

			if len(users) == 0 {
				fmt.Println("No users found")
			} else {
				fmt.Printf("Found %d user(s):\n", len(users))
				for i, user := range users {
					fmt.Printf("  %d. %s (%s) - Peer ID: %s\n", i+1, user.FullName, user.Username, user.PeerID)
				}
			}

		case "add":
			if !a.auth.IsAuthenticated() {
				fmt.Println("You must be logged in to add friends")
				break
			}
			if len(parts) < 2 {
				fmt.Println("Usage: add <username>")
				fmt.Println("Alternative: add-peer <peer-id> to add a connected peer")
				break
			}
			targetUsername := parts[1]

			currentUser, _ := a.auth.CurrentUser()

			// First, look up the user in DHT
			fmt.Printf("Looking up %s in DHT...\n", targetUsername)
			targetPeerID, err := a.p2p.FindUserByUsername(ctx, targetUsername)
			if err != nil {
				// Try local database as fallback
				targetUser, dbErr := a.storage.GetUserByUsername(ctx, targetUsername)
				if dbErr != nil || targetUser == nil {
					fmt.Printf("User not found: %v\n", err)
					fmt.Println("Tip: User must be online and registered, or use 'add-peer <peer-id>' for connected peers")
					break
				}
				targetPeerID, _ = peer.Decode(targetUser.PeerID)
			}

			// Connect to the peer if not already connected
			fmt.Printf("Connecting to %s...\n", targetUsername)
			err = a.p2p.ConnectToPeer(ctx, fmt.Sprintf("/p2p/%s", targetPeerID.String()))
			if err != nil {
				fmt.Printf("Warning: Could not connect directly: %v\n", err)
				fmt.Println("Attempting to send request anyway...")
			}

			// Send friend request
			err = a.friendManager.SendFriendRequest(ctx, currentUser, targetPeerID)
			if err != nil {
				fmt.Printf("Failed to send friend request: %v\n", err)
			}

		case "add-peer":
			if !a.auth.IsAuthenticated() {
				fmt.Println("You must be logged in to add friends")
				break
			}
			if len(parts) < 2 {
				fmt.Println("Usage: add-peer <peer-id>")
				fmt.Println("Example: add-peer 12D3KooW...")
				fmt.Println("Use 'peers' to see connected peer IDs")
				break
			}
			peerIDStr := parts[1]

			currentUser, _ := a.auth.CurrentUser()

			// Decode peer ID
			targetPeerID, err := peer.Decode(peerIDStr)
			if err != nil {
				fmt.Printf("Invalid peer ID: %v\n", err)
				break
			}

			// Send friend request
			err = a.friendManager.SendFriendRequest(ctx, currentUser, targetPeerID)
			if err != nil {
				fmt.Printf("Failed to send friend request: %v\n", err)
			}

		case "accept":
			if !a.auth.IsAuthenticated() {
				fmt.Println("You must be logged in to accept friend requests")
				break
			}
			if len(parts) < 2 {
				fmt.Println("Usage: accept <username>")
				break
			}
			fromUsername := parts[1]
			currentUser, _ := a.auth.CurrentUser()

			err := a.friendManager.AcceptFriendRequest(ctx, currentUser, fromUsername)
			if err != nil {
				fmt.Printf("Failed to accept friend request: %v\n", err)
			}

		case "reject":
			if !a.auth.IsAuthenticated() {
				fmt.Println("You must be logged in to reject friend requests")
				break
			}
			if len(parts) < 2 {
				fmt.Println("Usage: reject <username>")
				break
			}
			fromUsername := parts[1]
			currentUser, _ := a.auth.CurrentUser()

			err := a.friendManager.RejectFriendRequest(ctx, currentUser, fromUsername)
			if err != nil {
				fmt.Printf("Failed to reject friend request: %v\n", err)
			}

		case "friends":
			if !a.auth.IsAuthenticated() {
				fmt.Println("You must be logged in to view friends")
				break
			}
			currentUser, _ := a.auth.CurrentUser()

			friends, err := a.friendManager.GetFriends(ctx, currentUser.ID)
			if err != nil {
				fmt.Printf("Failed to get friends: %v\n", err)
				break
			}

			if len(friends) == 0 {
				fmt.Println("You don't have any friends yet")
				fmt.Println("Use 'add <username>' to send friend requests")
			} else {
				fmt.Printf("Your friends (%d):\n", len(friends))
				for i, friend := range friends {
					// Check if friend is online
					status := "offline"
					connectedPeers := a.p2p.GetConnectedPeers()
					for _, peer := range connectedPeers {
						if peer.ID.String() == friend.PeerID {
							status = "online"
							break
						}
					}
					statusIcon := "○"
					if status == "online" {
						statusIcon = "●"
					}
					fmt.Printf("  %d. %s %s (%s)\n", i+1, statusIcon, friend.FullName, friend.Username)
				}
			}

		case "requests":
			if !a.auth.IsAuthenticated() {
				fmt.Println("You must be logged in to view friend requests")
				break
			}
			currentUser, _ := a.auth.CurrentUser()

			requests, err := a.friendManager.GetPendingRequests(ctx, currentUser.ID)
			if err != nil {
				fmt.Printf("Failed to get friend requests: %v\n", err)
				break
			}

			if len(requests) == 0 {
				fmt.Println("No pending friend requests")
			} else {
				fmt.Printf("Pending friend requests (%d):\n", len(requests))
				for i, req := range requests {
					fmt.Printf("  %d. %s (%s)\n", i+1, req.FullName, req.Username)
				}
				fmt.Println("\nUse 'accept <username>' or 'reject <username>'")
			}

		case "connect":
			if !a.auth.IsAuthenticated() {
				fmt.Println("You must be logged in to connect to peers")
				break
			}
			if len(parts) < 2 {
				fmt.Println("Usage: connect <multiaddr>")
				fmt.Println("Example: connect /ip4/127.0.0.1/tcp/9999/p2p/12D3KooW...")
				fmt.Println()
				fmt.Println("This will connect to the peer AND automatically send a friend request")
				break
			}
			addr := parts[1]

			// Connect to the peer
			if err := a.p2p.ConnectToPeer(ctx, addr); err != nil {
				fmt.Printf("Failed to connect: %v\n", err)
				break
			}
			fmt.Println("✓ Successfully connected!")

			// Extract peer ID from multiaddr and auto-send friend request
			maddr, err := multiaddr.NewMultiaddr(addr)
			if err != nil {
				fmt.Printf("Note: Could not parse multiaddr to send friend request: %v\n", err)
				break
			}

			// Extract peer ID from the multiaddr
			var targetPeerID peer.ID
			multiaddr.ForEach(maddr, func(c multiaddr.Component) bool {
				if c.Protocol().Code == multiaddr.P_P2P {
					targetPeerID, _ = peer.Decode(c.Value())
					return false
				}
				return true
			})

			if targetPeerID == "" {
				fmt.Println("Note: No peer ID found in multiaddr, skipping auto friend request")
				break
			}

			// Auto-send friend request
			currentUser, err := a.auth.CurrentUser()
			if err != nil {
				fmt.Println("Note: Please login first to send friend request")
				break
			}
			fmt.Printf("Automatically sending friend request to %s...\n", targetPeerID.String()[:16]+"...")
			err = a.friendManager.SendFriendRequest(ctx, currentUser, targetPeerID)
			if err != nil {
				fmt.Printf("Note: Friend request not sent: %v\n", err)
				fmt.Println("(You may already be friends or have a pending request)")
			}

		case "peers":
			peers := a.p2p.GetConnectedPeers()
			if len(peers) == 0 {
				fmt.Println("No connected peers")
			} else {
				fmt.Printf("Connected peers (%d):\n", len(peers))
				for i, peer := range peers {
					fmt.Printf("  %d. %s\n", i+1, peer.ID.String())
					if peer.Username != "" {
						fmt.Printf("     Username: %s\n", peer.Username)
					}
				}
			}

		case "msg":
			if !a.auth.IsAuthenticated() {
				fmt.Println("You must be logged in to send messages")
				break
			}
			if len(parts) < 3 {
				fmt.Println("Usage: msg <username> <message>")
				fmt.Println("Example: msg alice Hello, how are you?")
				break
			}
			toUsername := parts[1]
			message := strings.Join(parts[2:], " ")

			currentUser, _ := a.auth.CurrentUser()
			err := a.messageManager.SendMessage(ctx, currentUser, toUsername, message)
			if err != nil {
				fmt.Printf("Failed to send message: %v\n", err)
			}

		case "history":
			if !a.auth.IsAuthenticated() {
				fmt.Println("You must be logged in to view message history")
				break
			}
			if len(parts) < 2 {
				fmt.Println("Usage: history <username> [limit]")
				fmt.Println("Example: history alice 20")
				break
			}
			otherUsername := parts[1]
			limit := 20
			if len(parts) >= 3 {
				fmt.Sscanf(parts[2], "%d", &limit)
			}

			currentUser, _ := a.auth.CurrentUser()
			otherUser, err := a.storage.GetUserByUsername(ctx, otherUsername)
			if err != nil || otherUser == nil {
				fmt.Printf("User not found: %s\n", otherUsername)
				break
			}

			messages, err := a.messageManager.GetConversation(ctx, currentUser.ID, otherUser.ID, limit)
			if err != nil {
				fmt.Printf("Failed to get messages: %v\n", err)
				break
			}

			if len(messages) == 0 {
				fmt.Printf("No message history with %s\n", otherUsername)
			} else {
				fmt.Printf("\n=== Conversation with %s (%d messages) ===\n", otherUser.FullName, len(messages))
				// Messages are in DESC order, so reverse them for display
				for i := len(messages) - 1; i >= 0; i-- {
					msg := messages[i]
					timestamp := msg.CreatedAt.Format("15:04:05")

					var sender string
					if msg.FromUserID == currentUser.ID {
						sender = "You"
					} else {
						sender = otherUser.FullName
					}

					status := ""
					if msg.FromUserID == currentUser.ID {
						if msg.Read {
							status = " ✓✓"
						} else if msg.Delivered {
							status = " ✓"
						}
					}

					fmt.Printf("[%s] %s: %s%s\n", timestamp, sender, msg.Content, status)
				}
				fmt.Println()
			}

			// Mark messages as read
			if err := a.messageManager.MarkAsRead(ctx, currentUser, otherUsername); err != nil {
				fmt.Printf("Warning: Failed to mark messages as read: %v\n", err)
			}

		case "unread":
			if !a.auth.IsAuthenticated() {
				fmt.Println("You must be logged in to view unread messages")
				break
			}

			currentUser, _ := a.auth.CurrentUser()

			// Get all friends
			friends, err := a.friendManager.GetFriends(ctx, currentUser.ID)
			if err != nil {
				fmt.Printf("Failed to get friends: %v\n", err)
				break
			}

			hasUnread := false
			for _, friend := range friends {
				// Get messages with this friend
				messages, err := a.messageManager.GetConversation(ctx, currentUser.ID, friend.FriendID, 50)
				if err != nil {
					continue
				}

				unreadCount := 0
				for _, msg := range messages {
					if msg.ToUserID == currentUser.ID && !msg.Read {
						unreadCount++
					}
				}

				if unreadCount > 0 {
					if !hasUnread {
						fmt.Println("\n=== Unread Messages ===")
						hasUnread = true
					}
					fmt.Printf("%s (%s): %d unread message(s)\n", friend.FullName, friend.Username, unreadCount)
				}
			}

			if !hasUnread {
				fmt.Println("No unread messages")
			} else {
				fmt.Println("\nUse 'history <username>' to read messages")
			}

		case "create-conf":
			if !a.auth.IsAuthenticated() {
				fmt.Println("You must be logged in to create conferences")
				break
			}
			if len(parts) < 2 {
				fmt.Println("Usage: create-conf <name>")
				fmt.Println("Example: create-conf \"Study Group\"")
				break
			}
			confName := strings.Join(parts[1:], " ")
			confName = strings.Trim(confName, "\"")

			currentUser, _ := a.auth.CurrentUser()
			_, err := a.conferenceManager.CreateConference(ctx, currentUser, confName)
			if err != nil {
				fmt.Printf("Failed to create conference: %v\n", err)
			}

		case "invite-conf":
			if !a.auth.IsAuthenticated() {
				fmt.Println("You must be logged in to invite to conferences")
				break
			}
			if len(parts) < 3 {
				fmt.Println("Usage: invite-conf <conference-id> <username>")
				fmt.Println("Example: invite-conf 1 alice")
				break
			}
			var confID int64
			fmt.Sscanf(parts[1], "%d", &confID)
			username := parts[2]

			currentUser, _ := a.auth.CurrentUser()
			err := a.conferenceManager.InviteToConference(ctx, currentUser, confID, username)
			if err != nil {
				fmt.Printf("Failed to invite: %v\n", err)
			}

		case "join-conf":
			if !a.auth.IsAuthenticated() {
				fmt.Println("You must be logged in to join conferences")
				break
			}
			if len(parts) < 2 {
				fmt.Println("Usage: join-conf <conference-id>")
				fmt.Println("Example: join-conf 1")
				break
			}
			var confID int64
			fmt.Sscanf(parts[1], "%d", &confID)

			currentUser, _ := a.auth.CurrentUser()
			err := a.conferenceManager.JoinConference(ctx, currentUser, confID)
			if err != nil {
				fmt.Printf("Failed to join conference: %v\n", err)
			}

		case "conf-msg":
			if !a.auth.IsAuthenticated() {
				fmt.Println("You must be logged in to send conference messages")
				break
			}
			if len(parts) < 3 {
				fmt.Println("Usage: conf-msg <conference-id> <message>")
				fmt.Println("Example: conf-msg 1 Hello everyone!")
				break
			}
			var confID int64
			fmt.Sscanf(parts[1], "%d", &confID)
			message := strings.Join(parts[2:], " ")

			currentUser, _ := a.auth.CurrentUser()
			err := a.conferenceManager.SendMessage(ctx, currentUser, confID, message)
			if err != nil {
				fmt.Printf("Failed to send message: %v\n", err)
			} else {
				fmt.Printf("✓ Message sent to conference\n")
			}

		case "conf-list":
			if !a.auth.IsAuthenticated() {
				fmt.Println("You must be logged in to view conferences")
				break
			}
			currentUser, _ := a.auth.CurrentUser()
			conferences, err := a.conferenceManager.GetConferences(ctx, currentUser.ID)
			if err != nil {
				fmt.Printf("Failed to get conferences: %v\n", err)
				break
			}

			if len(conferences) == 0 {
				fmt.Println("You are not in any conferences")
				fmt.Println("Use 'create-conf <name>' to create one")
			} else {
				fmt.Printf("Your conferences (%d):\n", len(conferences))
				for i, conf := range conferences {
					fmt.Printf("  %d. %s (ID: %d)\n", i+1, conf.Name, conf.ID)
				}
			}

		case "conf-history":
			if !a.auth.IsAuthenticated() {
				fmt.Println("You must be logged in to view conference history")
				break
			}
			if len(parts) < 2 {
				fmt.Println("Usage: conf-history <conference-id> [limit]")
				fmt.Println("Example: conf-history 1 20")
				break
			}
			var confID int64
			fmt.Sscanf(parts[1], "%d", &confID)
			limit := 20
			if len(parts) >= 3 {
				fmt.Sscanf(parts[2], "%d", &limit)
			}

			// Get conference
			conf, err := a.storage.GetConference(ctx, confID)
			if err != nil || conf == nil {
				fmt.Printf("Conference not found\n")
				break
			}

			messages, err := a.conferenceManager.GetConferenceMessages(ctx, confID, limit)
			if err != nil {
				fmt.Printf("Failed to get messages: %v\n", err)
				break
			}

			if len(messages) == 0 {
				fmt.Printf("No messages in conference '%s'\n", conf.Name)
			} else {
				fmt.Printf("\n=== Conference: %s (%d messages) ===\n", conf.Name, len(messages))
				// Messages are in DESC order, so reverse them
				for i := len(messages) - 1; i >= 0; i-- {
					msg := messages[i]
					timestamp := msg.CreatedAt.Format("15:04:05")

					// Try to get username from peer ID
					fromUsername := msg.FromPeerID[:8] + "..." // Fallback
					fromUser, err := a.storage.GetUserByPeerID(ctx, msg.FromPeerID)
					if err == nil && fromUser != nil {
						fromUsername = fromUser.FullName
					}

					fmt.Printf("[%s] %s: %s\n", timestamp, fromUsername, msg.Content)
				}
				fmt.Println()
			}

		case "conf-members":
			if !a.auth.IsAuthenticated() {
				fmt.Println("You must be logged in to view conference members")
				break
			}
			if len(parts) < 2 {
				fmt.Println("Usage: conf-members <conference-id>")
				fmt.Println("Example: conf-members 1")
				break
			}
			var confID int64
			fmt.Sscanf(parts[1], "%d", &confID)

			participants, err := a.conferenceManager.GetConferenceParticipants(ctx, confID)
			if err != nil {
				fmt.Printf("Failed to get participants: %v\n", err)
				break
			}

			if len(participants) == 0 {
				fmt.Println("No participants in conference")
			} else {
				fmt.Printf("Conference participants (%d):\n", len(participants))
				for i, p := range participants {
					status := "active"
					if !p.Active {
						status = "left"
					}
					fmt.Printf("  %d. %s (%s) - %s\n", i+1, p.Username, status, p.JoinedAt.Format("Jan 2"))
				}
			}

		case "leave-conf":
			if !a.auth.IsAuthenticated() {
				fmt.Println("You must be logged in to leave conferences")
				break
			}
			if len(parts) < 2 {
				fmt.Println("Usage: leave-conf <conference-id>")
				fmt.Println("Example: leave-conf 1")
				break
			}
			var confID int64
			fmt.Sscanf(parts[1], "%d", &confID)

			currentUser, _ := a.auth.CurrentUser()
			err := a.conferenceManager.LeaveConference(ctx, currentUser, confID)
			if err != nil {
				fmt.Printf("Failed to leave conference: %v\n", err)
			}

		case "help":
			a.showHelp()

		case "quit", "exit":
			fmt.Println("Exiting...")
			syscall.Kill(syscall.Getpid(), syscall.SIGINT)
			return

		default:
			fmt.Printf("Unknown command: %s (type 'help' for available commands)\n", cmd)
		}

		fmt.Print("> ")
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading input: %v\n", err)
	}
}

func (a *App) showHelp() {
	fmt.Println("\n=== Authentication Commands ===")
	fmt.Println("  register <username> <password> <full-name> - Create new account")
	fmt.Println("  login <username> <password>                - Login to your account")
	fmt.Println("  logout                                      - Logout from current account")
	fmt.Println("  whoami                                      - Show current user info")
	fmt.Println("  passwd <old-pass> <new-pass>               - Change your password")
	fmt.Println("  search <name>                               - Search for users by name")
	fmt.Println()
	fmt.Println("=== Getting Started ===")
	fmt.Println("  connect <multiaddr>                         - Connect to peer & send friend request")
	fmt.Println("  accept <username>                           - Accept friend request")
	fmt.Println()
	fmt.Println("=== Friend Commands ===")
	fmt.Println("  add <username>                              - Send friend request by username")
	fmt.Println("  add-peer <peer-id>                          - Send friend request by peer ID")
	fmt.Println("  reject <username>                           - Reject friend request")
	fmt.Println("  friends                                     - List your friends")
	fmt.Println("  requests                                    - View pending friend requests")
	fmt.Println()
	fmt.Println("=== Messaging Commands ===")
	fmt.Println("  msg <username> <message>                    - Send a direct message")
	fmt.Println("  history <username> [limit]                  - View message history")
	fmt.Println("  unread                                      - Show unread messages")
	fmt.Println()
	fmt.Println("=== Conference Commands ===")
	fmt.Println("  create-conf <name>                          - Create a new conference")
	fmt.Println("  invite-conf <conf-id> <username>            - Invite friend to conference")
	fmt.Println("  join-conf <conference-id>                   - Join a conference")
	fmt.Println("  conf-msg <conf-id> <message>                - Send conference message")
	fmt.Println("  conf-list                                   - List your conferences")
	fmt.Println("  conf-history <conf-id> [limit]              - View conference history")
	fmt.Println("  conf-members <conf-id>                      - List conference members")
	fmt.Println("  leave-conf <conf-id>                        - Leave a conference")
	fmt.Println()
	fmt.Println("=== Advanced Commands ===")
	fmt.Println("  peers                                       - List connected peers")
	fmt.Println()
	fmt.Println("=== General Commands ===")
	fmt.Println("  help                                        - Show this help")
	fmt.Println("  quit                                        - Exit the application")
	fmt.Println()
}
