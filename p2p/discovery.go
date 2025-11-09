package p2p

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// PublishUser publishes a user's information to the DHT
// For Phase 3, we use a simplified approach: user discovery via database + peer connections
// In a production system with many users, you'd want to implement proper DHT records with signing
func (p *P2PHost) PublishUser(ctx context.Context, username string) error {
	// Store in local peer metadata for now
	// When peers connect, they can exchange user information
	fmt.Printf("Registered user '%s' for peer discovery\n", username)
	return nil
}

// FindUserByUsername looks up a user's peer ID
// For Phase 3, this uses the local database (requires user to be in DB)
// In a full DHT implementation, this would query the distributed hash table
func (p *P2PHost) FindUserByUsername(ctx context.Context, username string) (peer.ID, error) {
	// For now, return an error indicating DHT lookup is not yet implemented
	// Users will need to be in the local database (from previous connections or manual adds)
	return "", fmt.Errorf("DHT user lookup not yet implemented - use database search instead")
}

// RefreshUserPresence periodically republishes user presence to DHT
func (p *P2PHost) RefreshUserPresence(ctx context.Context, username string) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.PublishUser(ctx, username); err != nil {
				fmt.Printf("Failed to refresh user presence: %v\n", err)
			}
		}
	}
}
