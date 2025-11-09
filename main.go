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
	"github.com/austinwklein/whisper/config"
	"github.com/austinwklein/whisper/friends"
	"github.com/austinwklein/whisper/p2p"
	"github.com/austinwklein/whisper/storage"
	"github.com/libp2p/go-libp2p/core/peer"
)

type App struct {
	config        *config.Config
	storage       storage.Storage
	p2p           *p2p.P2PHost
	auth          *auth.AuthService
	friendManager *friends.Manager
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

	// Create app
	app := &App{
		config:        cfg,
		storage:       store,
		p2p:           p2pHost,
		auth:          authService,
		friendManager: friendManager,
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
	fmt.Println("To use Whisper, you need to register or login:")
	fmt.Println("  register <username> <password> <full-name>")
	fmt.Println("  login <username> <password>")
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
				fmt.Printf("✓ Welcome back, %s!\n", user.FullName)
				// Set current user for friend manager
				a.friendManager.SetCurrentUser(user.ID)
				// Publish user to DHT
				go func() {
					if err := a.p2p.PublishUser(ctx, username); err != nil {
						fmt.Printf("Warning: Failed to publish to DHT: %v\n", err)
					}
					// Keep refreshing presence
					a.p2p.RefreshUserPresence(ctx, username)
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
				fmt.Println("Find users with: search <name>")
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
					fmt.Println("Tip: User must be online and registered")
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
			if len(parts) < 2 {
				fmt.Println("Usage: connect <multiaddr>")
				break
			}
			addr := parts[1]
			if err := a.p2p.ConnectToPeer(ctx, addr); err != nil {
				fmt.Printf("Failed to connect: %v\n", err)
			} else {
				fmt.Println("✓ Successfully connected!")
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
	fmt.Println("=== Friend Commands ===")
	fmt.Println("  add <username>                              - Send friend request")
	fmt.Println("  accept <username>                           - Accept friend request")
	fmt.Println("  reject <username>                           - Reject friend request")
	fmt.Println("  friends                                     - List your friends")
	fmt.Println("  requests                                    - View pending friend requests")
	fmt.Println()
	fmt.Println("=== P2P Commands ===")
	fmt.Println("  connect <multiaddr>                         - Connect to a peer")
	fmt.Println("  peers                                       - List connected peers")
	fmt.Println()
	fmt.Println("=== General Commands ===")
	fmt.Println("  help                                        - Show this help")
	fmt.Println("  quit                                        - Exit the application")
	fmt.Println()
}
