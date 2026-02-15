package key

import (
	"testing"
)

func TestKey_WithScopes(t *testing.T) {
	_, k := Generate("ak_")

	scopes := []string{"meter:write", "admin:read"}
	k2 := k.WithScopes(scopes)

	// Verify scopes are set on the copy
	if len(k2.Scopes) != 2 {
		t.Fatalf("Expected 2 scopes, got %d", len(k2.Scopes))
	}
	if k2.Scopes[0] != "meter:write" {
		t.Errorf("Expected scope[0]='meter:write', got %s", k2.Scopes[0])
	}
	if k2.Scopes[1] != "admin:read" {
		t.Errorf("Expected scope[1]='admin:read', got %s", k2.Scopes[1])
	}

	// Verify original is unchanged (immutability)
	if len(k.Scopes) != 0 {
		t.Errorf("Original key should have no scopes, got %d", len(k.Scopes))
	}
}

func TestKey_WithScopes_Nil(t *testing.T) {
	_, k := Generate("ak_")
	k2 := k.WithScopes(nil)

	if k2.Scopes != nil {
		t.Errorf("Expected nil scopes, got %v", k2.Scopes)
	}
}

func TestKey_WithQuotaBypass(t *testing.T) {
	_, k := Generate("ak_")

	k2 := k.WithQuotaBypass(true)

	if !k2.QuotaBypass {
		t.Error("Expected QuotaBypass=true")
	}

	// Verify original is unchanged
	if k.QuotaBypass {
		t.Error("Original key should have QuotaBypass=false")
	}
}

func TestKey_WithQuotaBypass_False(t *testing.T) {
	_, k := Generate("ak_")
	k = k.WithQuotaBypass(true)

	k2 := k.WithQuotaBypass(false)

	if k2.QuotaBypass {
		t.Error("Expected QuotaBypass=false")
	}
	if !k.QuotaBypass {
		t.Error("Original key should still have QuotaBypass=true")
	}
}

func TestKey_BuilderChaining(t *testing.T) {
	_, k := Generate("ak_")

	k2 := k.
		WithUserID("user_123").
		WithName("Test Key").
		WithScopes([]string{"meter:write"}).
		WithQuotaBypass(true)

	if k2.UserID != "user_123" {
		t.Errorf("Expected UserID='user_123', got %s", k2.UserID)
	}
	if k2.Name != "Test Key" {
		t.Errorf("Expected Name='Test Key', got %s", k2.Name)
	}
	if len(k2.Scopes) != 1 || k2.Scopes[0] != "meter:write" {
		t.Errorf("Expected Scopes=['meter:write'], got %v", k2.Scopes)
	}
	if !k2.QuotaBypass {
		t.Error("Expected QuotaBypass=true")
	}

	// Original unchanged
	if k.UserID != "" {
		t.Error("Original key should have empty UserID")
	}
}
