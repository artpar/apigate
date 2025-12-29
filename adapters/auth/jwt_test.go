package auth_test

import (
	"strings"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/auth"
)

func TestNewTokenService_WithSecret(t *testing.T) {
	svc := auth.NewTokenService("my-secret", time.Hour)
	if svc == nil {
		t.Fatal("expected service")
	}
}

func TestNewTokenService_EmptySecret(t *testing.T) {
	svc := auth.NewTokenService("", time.Hour)
	if svc == nil {
		t.Fatal("expected service with generated secret")
	}

	// Should still work
	token, _, err := svc.GenerateToken("user1", "test@example.com", "admin")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if token == "" {
		t.Error("expected token")
	}
}

func TestNewTokenService_DefaultExpiration(t *testing.T) {
	svc := auth.NewTokenService("secret", 0)
	if svc == nil {
		t.Fatal("expected service")
	}

	_, expiresAt, err := svc.GenerateToken("user1", "test@example.com", "user")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	// Default should be 24 hours
	expectedExpiry := time.Now().Add(24 * time.Hour)
	if expiresAt.Before(expectedExpiry.Add(-time.Minute)) || expiresAt.After(expectedExpiry.Add(time.Minute)) {
		t.Errorf("expiration should be ~24h, got %v", expiresAt)
	}
}

func TestTokenService_GenerateToken(t *testing.T) {
	svc := auth.NewTokenService("test-secret", time.Hour)

	token, expiresAt, err := svc.GenerateToken("user123", "user@example.com", "admin")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	if token == "" {
		t.Error("expected non-empty token")
	}

	// Token should be JWT format (3 parts separated by dots)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Errorf("expected JWT format with 3 parts, got %d", len(parts))
	}

	// Expiration should be ~1 hour from now
	expectedExpiry := time.Now().Add(time.Hour)
	if expiresAt.Before(expectedExpiry.Add(-time.Minute)) || expiresAt.After(expectedExpiry.Add(time.Minute)) {
		t.Errorf("expiration should be ~1h, got %v", expiresAt)
	}
}

func TestTokenService_ValidateToken_Success(t *testing.T) {
	svc := auth.NewTokenService("test-secret", time.Hour)

	token, _, err := svc.GenerateToken("user123", "user@example.com", "admin")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if claims.UserID != "user123" {
		t.Errorf("UserID = %s, want user123", claims.UserID)
	}
	if claims.Email != "user@example.com" {
		t.Errorf("Email = %s, want user@example.com", claims.Email)
	}
	if claims.Role != "admin" {
		t.Errorf("Role = %s, want admin", claims.Role)
	}
}

func TestTokenService_ValidateToken_InvalidToken(t *testing.T) {
	svc := auth.NewTokenService("test-secret", time.Hour)

	_, err := svc.ValidateToken("invalid-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestTokenService_ValidateToken_WrongSecret(t *testing.T) {
	svc1 := auth.NewTokenService("secret1", time.Hour)
	svc2 := auth.NewTokenService("secret2", time.Hour)

	token, _, _ := svc1.GenerateToken("user1", "test@example.com", "user")

	_, err := svc2.ValidateToken(token)
	if err == nil {
		t.Error("expected error for token with wrong secret")
	}
}

func TestTokenService_ValidateToken_Expired(t *testing.T) {
	// Create service with very short expiration
	svc := auth.NewTokenService("test-secret", time.Millisecond)

	token, _, _ := svc.GenerateToken("user1", "test@example.com", "user")

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	_, err := svc.ValidateToken(token)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestTokenService_RefreshToken_Success(t *testing.T) {
	svc := auth.NewTokenService("test-secret", time.Hour)

	originalToken, originalExpiry, _ := svc.GenerateToken("user1", "test@example.com", "admin")

	// Small delay to ensure new expiry is different
	time.Sleep(50 * time.Millisecond)

	newToken, newExpiry, err := svc.RefreshToken(originalToken)
	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}

	// New expiry should be later than original (tokens may be same if generated in same second)
	if newExpiry.Before(originalExpiry) {
		t.Error("new expiry should be later than original")
	}

	// Verify new token works
	claims, err := svc.ValidateToken(newToken)
	if err != nil {
		t.Fatalf("new token validation failed: %v", err)
	}
	if claims.UserID != "user1" {
		t.Errorf("UserID = %s, want user1", claims.UserID)
	}
}

func TestTokenService_RefreshToken_InvalidToken(t *testing.T) {
	svc := auth.NewTokenService("test-secret", time.Hour)

	_, _, err := svc.RefreshToken("invalid-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestGenerateSecret(t *testing.T) {
	secret1 := auth.GenerateSecret()
	secret2 := auth.GenerateSecret()

	if secret1 == "" {
		t.Error("expected non-empty secret")
	}

	// Should be hex string (64 chars for 32 bytes)
	if len(secret1) != 64 {
		t.Errorf("expected 64 char hex string, got %d chars", len(secret1))
	}

	// Each call should produce different secret
	if secret1 == secret2 {
		t.Error("secrets should be different")
	}
}

func TestClaims_Fields(t *testing.T) {
	svc := auth.NewTokenService("test-secret", time.Hour)

	token, _, _ := svc.GenerateToken("uid123", "email@test.com", "moderator")
	claims, _ := svc.ValidateToken(token)

	if claims.UserID != "uid123" {
		t.Errorf("UserID = %s", claims.UserID)
	}
	if claims.Email != "email@test.com" {
		t.Errorf("Email = %s", claims.Email)
	}
	if claims.Role != "moderator" {
		t.Errorf("Role = %s", claims.Role)
	}
	if claims.Issuer != "apigate" {
		t.Errorf("Issuer = %s", claims.Issuer)
	}
	if claims.Subject != "uid123" {
		t.Errorf("Subject = %s", claims.Subject)
	}
}
