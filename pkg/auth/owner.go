// Package auth provides authentication and authorization utilities for contextd.
//
// This package implements owner-scoped authentication using cryptographic hashing
// to derive stable, unique owner identifiers from usernames.
package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

var (
	// ErrEmptyUsername is returned when an empty username is provided
	ErrEmptyUsername = errors.New("username cannot be empty")
)

// DeriveOwnerID derives a stable owner ID from a username using SHA256 hashing.
//
// The owner ID is computed as SHA256(username) and returned as a hex-encoded string.
// This provides a one-way, deterministic mapping from username to owner ID that:
//   - Is consistent (same username always produces same owner ID)
//   - Is unique (different usernames produce different owner IDs with high probability)
//   - Is irreversible (cannot recover username from owner ID)
//
// This owner ID is used for multi-tenant isolation in 0.9.0-rc-1, ensuring that users
// can only access their own data.
//
// Example:
//
//	ownerID, err := auth.DeriveOwnerID("alice")
//	if err != nil {
//	    return fmt.Errorf("derive owner ID: %w", err)
//	}
//	// ownerID = "2bd806c97f0e00af1a1fc3328fa763a9269723c8db8fac4f93af71db186d6e90"
//
// Returns ErrEmptyUsername if username is empty.
func DeriveOwnerID(username string) (string, error) {
	// Validate input
	if username == "" {
		return "", ErrEmptyUsername
	}

	// Compute SHA256 hash of username
	hash := sha256.Sum256([]byte(username))

	// Return hex-encoded hash
	return hex.EncodeToString(hash[:]), nil
}
