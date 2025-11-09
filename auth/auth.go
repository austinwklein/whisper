package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/austinwklein/whisper/storage"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserExists       = errors.New("username already exists")
	ErrInvalidPassword  = errors.New("invalid password")
	ErrUserNotFound     = errors.New("user not found")
	ErrNotAuthenticated = errors.New("not authenticated")
	ErrWeakPassword     = errors.New("password must be at least 8 characters")
)

// AuthService handles user authentication
type AuthService struct {
	storage       storage.Storage
	currentUser   *storage.User
	authenticated bool
}

// NewAuthService creates a new authentication service
func NewAuthService(store storage.Storage) *AuthService {
	return &AuthService{
		storage:       store,
		authenticated: false,
	}
}

// Register creates a new user account
func (a *AuthService) Register(ctx context.Context, username, password, fullName, peerID string) error {
	// Validate input
	if username == "" {
		return errors.New("username is required")
	}
	if password == "" {
		return errors.New("password is required")
	}
	if len(password) < 8 {
		return ErrWeakPassword
	}
	if fullName == "" {
		return errors.New("full name is required")
	}
	if peerID == "" {
		return errors.New("peer ID is required")
	}

	// Check if user already exists
	existingUser, err := a.storage.GetUserByUsername(ctx, username)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}
	if existingUser != nil {
		return ErrUserExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &storage.User{
		Username:     username,
		PasswordHash: string(hashedPassword),
		FullName:     fullName,
		PeerID:       peerID,
	}

	if err := a.storage.CreateUser(ctx, user); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// Login authenticates a user
func (a *AuthService) Login(ctx context.Context, username, password string) (*storage.User, error) {
	// Get user from storage
	user, err := a.storage.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, ErrInvalidPassword
	}

	// Set current user
	a.currentUser = user
	a.authenticated = true

	return user, nil
}

// Logout logs out the current user
func (a *AuthService) Logout() {
	a.currentUser = nil
	a.authenticated = false
}

// CurrentUser returns the currently authenticated user
func (a *AuthService) CurrentUser() (*storage.User, error) {
	if !a.authenticated || a.currentUser == nil {
		return nil, ErrNotAuthenticated
	}
	return a.currentUser, nil
}

// IsAuthenticated returns true if a user is logged in
func (a *AuthService) IsAuthenticated() bool {
	return a.authenticated
}

// ChangePassword changes the password for the current user
func (a *AuthService) ChangePassword(ctx context.Context, oldPassword, newPassword string) error {
	if !a.authenticated || a.currentUser == nil {
		return ErrNotAuthenticated
	}

	// Validate new password
	if len(newPassword) < 8 {
		return ErrWeakPassword
	}

	// Verify old password
	err := bcrypt.CompareHashAndPassword([]byte(a.currentUser.PasswordHash), []byte(oldPassword))
	if err != nil {
		return ErrInvalidPassword
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update user
	a.currentUser.PasswordHash = string(hashedPassword)
	if err := a.storage.UpdateUser(ctx, a.currentUser); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// GetUserByPeerID retrieves a user by their peer ID
func (a *AuthService) GetUserByPeerID(ctx context.Context, peerID string) (*storage.User, error) {
	user, err := a.storage.GetUserByPeerID(ctx, peerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by peer ID: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// SearchUsers searches for users by name
func (a *AuthService) SearchUsers(ctx context.Context, name string) ([]*storage.User, error) {
	if !a.authenticated {
		return nil, ErrNotAuthenticated
	}

	users, err := a.storage.SearchUsersByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	return users, nil
}
