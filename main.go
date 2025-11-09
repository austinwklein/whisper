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
	"github.com/austinwklein/whisper/p2p"
	"github.com/austinwklein/whisper/storage"
)

type App struct {
	config  *config.Config
	storage storage.Storage
	p2p     *p2p.P2PHost
	auth    *auth.AuthService
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

	// Create app
	app := &App{
		config:  cfg,
		storage: store,
		p2p:     p2pHost,
		auth:    authService,
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
	// Initialize DHT, GossipSub, etc.
	// This will be expanded in future phases
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
			// Join remaining parts as full name
			fullName := strings.Join(parts[3:], " ")
			// Remove quotes if present
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
			}

		case "logout":
			if !a.auth.IsAuthenticated() {
				fmt.Println("You are not logged in")
				break
			}
			user, _ := a.auth.CurrentUser()
			a.auth.Logout()
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
	fmt.Println("=== P2P Commands ===")
	fmt.Println("  connect <multiaddr>                         - Connect to a peer")
	fmt.Println("  peers                                       - List connected peers")
	fmt.Println()
	fmt.Println("=== General Commands ===")
	fmt.Println("  help                                        - Show this help")
	fmt.Println("  quit                                        - Exit the application")
	fmt.Println()
}
