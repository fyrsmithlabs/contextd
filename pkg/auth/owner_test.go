package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestDeriveOwnerID(t *testing.T) {
	tests := []struct {
		name     string
		username string
		want     string
		wantErr  bool
	}{
		{
			name:     "valid username",
			username: "alice",
			want:     computeExpectedHash("alice"),
			wantErr:  false,
		},
		{
			name:     "another valid username",
			username: "bob@example.com",
			want:     computeExpectedHash("bob@example.com"),
			wantErr:  false,
		},
		{
			name:     "empty username",
			username: "",
			want:     "",
			wantErr:  true,
		},
		{
			name:     "username with spaces",
			username: "user name",
			want:     computeExpectedHash("user name"),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DeriveOwnerID(tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeriveOwnerID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DeriveOwnerID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeriveOwnerID_Consistency(t *testing.T) {
	// Test that same input produces same output (idempotency)
	username := "testuser"

	id1, err := DeriveOwnerID(username)
	if err != nil {
		t.Fatalf("first DeriveOwnerID() failed: %v", err)
	}

	id2, err := DeriveOwnerID(username)
	if err != nil {
		t.Fatalf("second DeriveOwnerID() failed: %v", err)
	}

	if id1 != id2 {
		t.Errorf("DeriveOwnerID() not consistent: got %v and %v for same input", id1, id2)
	}
}

func TestDeriveOwnerID_Uniqueness(t *testing.T) {
	// Test that different inputs produce different outputs
	id1, err := DeriveOwnerID("alice")
	if err != nil {
		t.Fatalf("DeriveOwnerID(alice) failed: %v", err)
	}

	id2, err := DeriveOwnerID("bob")
	if err != nil {
		t.Fatalf("DeriveOwnerID(bob) failed: %v", err)
	}

	if id1 == id2 {
		t.Errorf("DeriveOwnerID() produced same output for different inputs")
	}
}

// Helper function to compute expected SHA256 hash
func computeExpectedHash(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}
