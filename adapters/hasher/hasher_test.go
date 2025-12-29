package hasher_test

import (
	"testing"

	"github.com/artpar/apigate/adapters/hasher"
	"golang.org/x/crypto/bcrypt"
)

func TestBcrypt_NewBcrypt_ValidCost(t *testing.T) {
	h := hasher.NewBcrypt(10)
	if h == nil {
		t.Fatal("expected hasher")
	}
}

func TestBcrypt_NewBcrypt_InvalidCost(t *testing.T) {
	// Too low cost should default
	h := hasher.NewBcrypt(1)
	if h == nil {
		t.Fatal("expected hasher with default cost")
	}

	// Too high cost should default
	h = hasher.NewBcrypt(100)
	if h == nil {
		t.Fatal("expected hasher with default cost")
	}
}

func TestBcrypt_Hash(t *testing.T) {
	h := hasher.NewBcrypt(bcrypt.MinCost) // Use min cost for speed in tests

	hash, err := h.Hash("password123")
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}

	if len(hash) == 0 {
		t.Error("expected non-empty hash")
	}

	// Hash should be bcrypt format
	if hash[0] != '$' {
		t.Error("expected bcrypt format starting with $")
	}
}

func TestBcrypt_Hash_DifferentInputs(t *testing.T) {
	h := hasher.NewBcrypt(bcrypt.MinCost)

	hash1, _ := h.Hash("password1")
	hash2, _ := h.Hash("password2")

	if string(hash1) == string(hash2) {
		t.Error("different passwords should produce different hashes")
	}
}

func TestBcrypt_Hash_SameInputDifferentOutput(t *testing.T) {
	h := hasher.NewBcrypt(bcrypt.MinCost)

	hash1, _ := h.Hash("password")
	hash2, _ := h.Hash("password")

	// Bcrypt uses random salt, so same input gives different hash
	if string(hash1) == string(hash2) {
		t.Error("same password should produce different hashes due to salt")
	}
}

func TestBcrypt_Compare_Match(t *testing.T) {
	h := hasher.NewBcrypt(bcrypt.MinCost)

	password := "mySecretPassword"
	hash, _ := h.Hash(password)

	if !h.Compare(hash, password) {
		t.Error("Compare should return true for matching password")
	}
}

func TestBcrypt_Compare_NoMatch(t *testing.T) {
	h := hasher.NewBcrypt(bcrypt.MinCost)

	hash, _ := h.Hash("correctPassword")

	if h.Compare(hash, "wrongPassword") {
		t.Error("Compare should return false for wrong password")
	}
}

func TestBcrypt_Compare_InvalidHash(t *testing.T) {
	h := hasher.NewBcrypt(bcrypt.MinCost)

	if h.Compare([]byte("not-a-hash"), "password") {
		t.Error("Compare should return false for invalid hash")
	}
}

func TestBcrypt_Compare_EmptyHash(t *testing.T) {
	h := hasher.NewBcrypt(bcrypt.MinCost)

	if h.Compare([]byte{}, "password") {
		t.Error("Compare should return false for empty hash")
	}
}

func TestBcrypt_Compare_EmptyPassword(t *testing.T) {
	h := hasher.NewBcrypt(bcrypt.MinCost)

	hash, _ := h.Hash("password")

	if h.Compare(hash, "") {
		t.Error("Compare should return false for empty password")
	}
}

func TestFake_Hash(t *testing.T) {
	h := hasher.Fake{}

	hash, err := h.Hash("plaintext")
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}

	if string(hash) != "plaintext" {
		t.Errorf("Fake hash should return plaintext, got %s", hash)
	}
}

func TestFake_Compare_Match(t *testing.T) {
	h := hasher.Fake{}

	if !h.Compare([]byte("password"), "password") {
		t.Error("Fake Compare should return true for matching values")
	}
}

func TestFake_Compare_NoMatch(t *testing.T) {
	h := hasher.Fake{}

	if h.Compare([]byte("password1"), "password2") {
		t.Error("Fake Compare should return false for different values")
	}
}

func TestFake_RoundTrip(t *testing.T) {
	h := hasher.Fake{}

	password := "testPassword123"
	hash, _ := h.Hash(password)

	if !h.Compare(hash, password) {
		t.Error("Fake should compare hashed value with original")
	}
}
