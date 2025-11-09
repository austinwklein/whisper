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

	"github.com/austinwklein/whisper/config"
	"github.com/austinwklein/whisper/p2p"
	"github.com/austinwklein/whisper/storage"
)

type App struct {
	config  *config.Config
	storage storage.Storage
	p2p     *p2p.P2PHost
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

	// Create app
	app := &App{
		config:  cfg,
		storage: store,
		p2p:     p2pHost,
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
	fmt.Println("\n=== Commands ===")
	fmt.Println("  connect <multiaddr> - Connect to a peer")
	fmt.Println("  peers               - List connected peers")
	fmt.Println("  help                - Show this help")
	fmt.Println("  quit                - Exit the application")
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
	// This will be expanded in Phase 2
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
		case "connect":
			if len(parts) < 2 {
				fmt.Println("Usage: connect <multiaddr>")
				break
			}
			addr := parts[1]
			if err := a.p2p.ConnectToPeer(ctx, addr); err != nil {
				fmt.Printf("Failed to connect: %v\n", err)
			} else {
				fmt.Println("Successfully connected!")
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
			fmt.Println("\n=== Commands ===")
			fmt.Println("  connect <multiaddr> - Connect to a peer")
			fmt.Println("  peers               - List connected peers")
			fmt.Println("  help                - Show this help")
			fmt.Println("  quit                - Exit the application")

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
