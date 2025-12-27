// Package hasher provides password/key hashing implementations.
package hasher

import (
	"github.com/artpar/apigate/ports"
	"golang.org/x/crypto/bcrypt"
)

// Bcrypt uses bcrypt for hashing.
type Bcrypt struct {
	cost int
}

// NewBcrypt creates a bcrypt hasher with the given cost.
func NewBcrypt(cost int) *Bcrypt {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}
	return &Bcrypt{cost: cost}
}

// Hash generates a bcrypt hash from plaintext.
func (h *Bcrypt) Hash(plaintext string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(plaintext), h.cost)
}

// Compare checks if plaintext matches hash.
func (h *Bcrypt) Compare(hash []byte, plaintext string) bool {
	return bcrypt.CompareHashAndPassword(hash, []byte(plaintext)) == nil
}

// Ensure interface compliance.
var _ ports.Hasher = (*Bcrypt)(nil)

// Fake provides a no-op hasher for testing (NOT FOR PRODUCTION).
type Fake struct{}

// Hash returns the plaintext as bytes (no actual hashing).
func (Fake) Hash(plaintext string) ([]byte, error) {
	return []byte(plaintext), nil
}

// Compare does simple equality check.
func (Fake) Compare(hash []byte, plaintext string) bool {
	return string(hash) == plaintext
}

// Ensure interface compliance.
var _ ports.Hasher = Fake{}
