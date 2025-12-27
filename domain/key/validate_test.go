package key_test

import (
	"testing"
	"time"

	"github.com/artpar/apigate/domain/key"
)

// Test fixtures
var (
	baseTime   = time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	pastTime   = baseTime.Add(-24 * time.Hour)
	futureTime = baseTime.Add(24 * time.Hour)
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name       string
		key        key.Key
		now        time.Time
		wantValid  bool
		wantReason string
	}{
		{
			name: "valid key",
			key: key.Key{
				ID:        "key-1",
				UserID:    "user-1",
				CreatedAt: pastTime,
			},
			now:       baseTime,
			wantValid: true,
		},
		{
			name: "valid key with future expiry",
			key: key.Key{
				ID:        "key-2",
				UserID:    "user-1",
				ExpiresAt: &futureTime,
				CreatedAt: pastTime,
			},
			now:       baseTime,
			wantValid: true,
		},
		{
			name: "expired key",
			key: key.Key{
				ID:        "key-3",
				UserID:    "user-1",
				ExpiresAt: &pastTime,
				CreatedAt: pastTime.Add(-48 * time.Hour),
			},
			now:        baseTime,
			wantValid:  false,
			wantReason: key.ReasonExpired,
		},
		{
			name: "revoked key",
			key: key.Key{
				ID:        "key-4",
				UserID:    "user-1",
				RevokedAt: &pastTime,
				CreatedAt: pastTime.Add(-48 * time.Hour),
			},
			now:        baseTime,
			wantValid:  false,
			wantReason: key.ReasonRevoked,
		},
		{
			name: "revoked takes precedence over expired",
			key: key.Key{
				ID:        "key-5",
				UserID:    "user-1",
				ExpiresAt: &pastTime,
				RevokedAt: &pastTime,
				CreatedAt: pastTime.Add(-48 * time.Hour),
			},
			now:        baseTime,
			wantValid:  false,
			wantReason: key.ReasonRevoked,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := key.Validate(tt.key, tt.now)

			if result.Valid != tt.wantValid {
				t.Errorf("Validate() valid = %v, want %v", result.Valid, tt.wantValid)
			}

			if result.Reason != tt.wantReason {
				t.Errorf("Validate() reason = %q, want %q", result.Reason, tt.wantReason)
			}

			if tt.wantValid && result.Key.ID != tt.key.ID {
				t.Errorf("Validate() key.ID = %q, want %q", result.Key.ID, tt.key.ID)
			}
		})
	}
}

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		name       string
		rawKey     string
		prefix     string
		wantPrefix string
		wantValid  bool
	}{
		{
			name:       "valid key format",
			rawKey:     "ak_abcd1234efgh5678901234567890123456789012345678901234567890123456",
			prefix:     "ak_",
			wantPrefix: "ak_abcd1234e", // First 12 chars
			wantValid:  true,
		},
		{
			name:      "wrong prefix",
			rawKey:    "sk_abcd1234efgh5678901234567890123456789012345678901234567890123456",
			prefix:    "ak_",
			wantValid: false,
		},
		{
			name:      "too short",
			rawKey:    "ak_short",
			prefix:    "ak_",
			wantValid: false,
		},
		{
			name:      "empty key",
			rawKey:    "",
			prefix:    "ak_",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, valid := key.ValidateFormat(tt.rawKey, tt.prefix)

			if valid != tt.wantValid {
				t.Errorf("ValidateFormat() valid = %v, want %v", valid, tt.wantValid)
			}

			if tt.wantValid && prefix != tt.wantPrefix {
				t.Errorf("ValidateFormat() prefix = %q, want %q", prefix, tt.wantPrefix)
			}
		})
	}
}

func TestHasScope(t *testing.T) {
	tests := []struct {
		name     string
		key      key.Key
		scope    string
		wantHas  bool
	}{
		{
			name:    "empty scopes grants all access",
			key:     key.Key{Scopes: nil},
			scope:   "/api/anything",
			wantHas: true,
		},
		{
			name:    "exact scope match",
			key:     key.Key{Scopes: []string{"/api/users", "/api/items"}},
			scope:   "/api/users",
			wantHas: true,
		},
		{
			name:    "scope not in list",
			key:     key.Key{Scopes: []string{"/api/users"}},
			scope:   "/api/admin",
			wantHas: false,
		},
		{
			name:    "wildcard scope grants all",
			key:     key.Key{Scopes: []string{"*"}},
			scope:   "/api/anything",
			wantHas: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := key.HasScope(tt.key, tt.scope)
			if got != tt.wantHas {
				t.Errorf("HasScope() = %v, want %v", got, tt.wantHas)
			}
		})
	}
}

func TestMatchPath(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{"exact match", "/api/users", "/api/users", true},
		{"no match", "/api/users", "/api/items", false},
		{"wildcard matches subpath", "/api/*", "/api/users", true},
		{"wildcard matches deep path", "/api/*", "/api/users/123/profile", true},
		{"wildcard matches base", "/api/*", "/api", true},
		{"wildcard no match different prefix", "/api/*", "/admin/users", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := key.MatchPath(tt.pattern, tt.path)
			if got != tt.want {
				t.Errorf("MatchPath(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}

// Benchmark to ensure validation is fast
func BenchmarkValidate(b *testing.B) {
	k := key.Key{
		ID:        "key-1",
		UserID:    "user-1",
		ExpiresAt: &futureTime,
		CreatedAt: pastTime,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key.Validate(k, baseTime)
	}
}
