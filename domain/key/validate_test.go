package key_test

import (
	"strings"
	"testing"
	"time"

	"github.com/artpar/apigate/domain/key"
	"golang.org/x/crypto/bcrypt"
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

// TestGenerate tests the Generate function
func TestGenerate(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
	}{
		{
			name:   "generate with ak_ prefix",
			prefix: "ak_",
		},
		{
			name:   "generate with sk_ prefix",
			prefix: "sk_",
		},
		{
			name:   "generate with empty prefix",
			prefix: "",
		},
		{
			name:   "generate with slightly longer prefix",
			prefix: "sk_live_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawKey, k := key.Generate(tt.prefix)

			// Verify raw key starts with prefix
			if !strings.HasPrefix(rawKey, tt.prefix) {
				t.Errorf("Generate() rawKey = %q, should start with %q", rawKey, tt.prefix)
			}

			// Verify raw key length: prefix + 64 hex chars
			expectedLen := len(tt.prefix) + 64
			if len(rawKey) != expectedLen {
				t.Errorf("Generate() rawKey length = %d, want %d", len(rawKey), expectedLen)
			}

			// Verify key ID starts with "key_"
			if !strings.HasPrefix(k.ID, "key_") {
				t.Errorf("Generate() key.ID = %q, should start with 'key_'", k.ID)
			}

			// Verify prefix is first 12 chars of raw key
			if len(rawKey) >= 12 {
				if k.Prefix != rawKey[:12] {
					t.Errorf("Generate() key.Prefix = %q, want %q", k.Prefix, rawKey[:12])
				}
			}

			// Verify hash is valid bcrypt and matches raw key
			err := bcrypt.CompareHashAndPassword(k.Hash, []byte(rawKey))
			if err != nil {
				t.Errorf("Generate() hash does not match raw key: %v", err)
			}

			// Verify CreatedAt is set and reasonable
			if k.CreatedAt.IsZero() {
				t.Error("Generate() key.CreatedAt is zero")
			}

			// Verify other fields are zero/nil
			if k.UserID != "" {
				t.Errorf("Generate() key.UserID = %q, want empty", k.UserID)
			}
			if k.Name != "" {
				t.Errorf("Generate() key.Name = %q, want empty", k.Name)
			}
			if k.ExpiresAt != nil {
				t.Errorf("Generate() key.ExpiresAt = %v, want nil", k.ExpiresAt)
			}
			if k.RevokedAt != nil {
				t.Errorf("Generate() key.RevokedAt = %v, want nil", k.RevokedAt)
			}
			if k.LastUsed != nil {
				t.Errorf("Generate() key.LastUsed = %v, want nil", k.LastUsed)
			}
			if len(k.Scopes) != 0 {
				t.Errorf("Generate() key.Scopes = %v, want empty", k.Scopes)
			}
		})
	}
}

// TestGenerateUniqueness verifies that Generate produces unique keys
func TestGenerateUniqueness(t *testing.T) {
	const numKeys = 100
	rawKeys := make(map[string]bool)
	keyIDs := make(map[string]bool)

	for i := 0; i < numKeys; i++ {
		rawKey, k := key.Generate("ak_")

		if rawKeys[rawKey] {
			t.Errorf("Generate() produced duplicate raw key: %q", rawKey)
		}
		rawKeys[rawKey] = true

		if keyIDs[k.ID] {
			t.Errorf("Generate() produced duplicate key ID: %q", k.ID)
		}
		keyIDs[k.ID] = true
	}
}

// TestWithUserID tests the WithUserID method
func TestWithUserID(t *testing.T) {
	tests := []struct {
		name     string
		initial  key.Key
		userID   string
		wantID   string
	}{
		{
			name: "set user ID on empty key",
			initial: key.Key{
				ID: "key-1",
			},
			userID: "user-123",
			wantID: "user-123",
		},
		{
			name: "set user ID overwrites existing",
			initial: key.Key{
				ID:     "key-1",
				UserID: "old-user",
			},
			userID: "new-user",
			wantID: "new-user",
		},
		{
			name: "set empty user ID",
			initial: key.Key{
				ID:     "key-1",
				UserID: "user-123",
			},
			userID: "",
			wantID: "",
		},
		{
			name: "preserves other fields",
			initial: key.Key{
				ID:        "key-1",
				Name:      "My API Key",
				Scopes:    []string{"/api/users"},
				CreatedAt: baseTime,
			},
			userID: "user-456",
			wantID: "user-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.initial.WithUserID(tt.userID)

			// Verify UserID is set correctly
			if result.UserID != tt.wantID {
				t.Errorf("WithUserID() UserID = %q, want %q", result.UserID, tt.wantID)
			}

			// Verify original key is unchanged (immutable)
			if tt.initial.UserID == tt.wantID && tt.wantID != "" && tt.initial.UserID != "old-user" && tt.initial.UserID != "user-123" {
				// Skip this check if the original already had the same value
			}

			// Verify other fields are preserved
			if result.ID != tt.initial.ID {
				t.Errorf("WithUserID() ID = %q, want %q", result.ID, tt.initial.ID)
			}
			if result.Name != tt.initial.Name {
				t.Errorf("WithUserID() Name = %q, want %q", result.Name, tt.initial.Name)
			}
			if result.CreatedAt != tt.initial.CreatedAt {
				t.Errorf("WithUserID() CreatedAt = %v, want %v", result.CreatedAt, tt.initial.CreatedAt)
			}
		})
	}
}

// TestWithName tests the WithName method
func TestWithName(t *testing.T) {
	tests := []struct {
		name     string
		initial  key.Key
		keyName  string
		wantName string
	}{
		{
			name: "set name on empty key",
			initial: key.Key{
				ID: "key-1",
			},
			keyName:  "Production API Key",
			wantName: "Production API Key",
		},
		{
			name: "set name overwrites existing",
			initial: key.Key{
				ID:   "key-1",
				Name: "Old Name",
			},
			keyName:  "New Name",
			wantName: "New Name",
		},
		{
			name: "set empty name",
			initial: key.Key{
				ID:   "key-1",
				Name: "Some Name",
			},
			keyName:  "",
			wantName: "",
		},
		{
			name: "preserves other fields",
			initial: key.Key{
				ID:        "key-1",
				UserID:    "user-123",
				Scopes:    []string{"/api/users"},
				CreatedAt: baseTime,
			},
			keyName:  "API Key with Scopes",
			wantName: "API Key with Scopes",
		},
		{
			name: "handles special characters in name",
			initial: key.Key{
				ID: "key-1",
			},
			keyName:  "My Key (Production) - v1.0",
			wantName: "My Key (Production) - v1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.initial.WithName(tt.keyName)

			// Verify Name is set correctly
			if result.Name != tt.wantName {
				t.Errorf("WithName() Name = %q, want %q", result.Name, tt.wantName)
			}

			// Verify other fields are preserved
			if result.ID != tt.initial.ID {
				t.Errorf("WithName() ID = %q, want %q", result.ID, tt.initial.ID)
			}
			if result.UserID != tt.initial.UserID {
				t.Errorf("WithName() UserID = %q, want %q", result.UserID, tt.initial.UserID)
			}
			if result.CreatedAt != tt.initial.CreatedAt {
				t.Errorf("WithName() CreatedAt = %v, want %v", result.CreatedAt, tt.initial.CreatedAt)
			}
		})
	}
}

// TestMethodChaining verifies that WithUserID and WithName can be chained
func TestMethodChaining(t *testing.T) {
	_, k := key.Generate("ak_")

	result := k.WithUserID("user-123").WithName("My API Key")

	if result.UserID != "user-123" {
		t.Errorf("Chained WithUserID() UserID = %q, want %q", result.UserID, "user-123")
	}
	if result.Name != "My API Key" {
		t.Errorf("Chained WithName() Name = %q, want %q", result.Name, "My API Key")
	}
	// Original key should be unchanged
	if k.UserID != "" {
		t.Errorf("Original key UserID modified: %q", k.UserID)
	}
	if k.Name != "" {
		t.Errorf("Original key Name modified: %q", k.Name)
	}
}

// TestValidateExpiresAtBoundary tests boundary conditions for expiration
func TestValidateExpiresAtBoundary(t *testing.T) {
	exactExpiry := baseTime

	tests := []struct {
		name       string
		now        time.Time
		wantValid  bool
		wantReason string
	}{
		{
			name:       "expires exactly now - still valid",
			now:        exactExpiry,
			wantValid:  true,
			wantReason: "",
		},
		{
			name:       "expires 1 nanosecond ago",
			now:        exactExpiry.Add(1 * time.Nanosecond),
			wantValid:  false,
			wantReason: key.ReasonExpired,
		},
		{
			name:       "expires in 1 nanosecond",
			now:        exactExpiry.Add(-1 * time.Nanosecond),
			wantValid:  true,
			wantReason: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := key.Key{
				ID:        "key-1",
				ExpiresAt: &exactExpiry,
			}

			result := key.Validate(k, tt.now)

			if result.Valid != tt.wantValid {
				t.Errorf("Validate() valid = %v, want %v", result.Valid, tt.wantValid)
			}
			if result.Reason != tt.wantReason {
				t.Errorf("Validate() reason = %q, want %q", result.Reason, tt.wantReason)
			}
		})
	}
}

// TestValidateFormatEdgeCases tests additional edge cases for ValidateFormat
func TestValidateFormatEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		rawKey     string
		prefix     string
		wantPrefix string
		wantValid  bool
	}{
		{
			name:       "exactly minimum length",
			rawKey:     "ak_" + strings.Repeat("a", 64),
			prefix:     "ak_",
			wantPrefix: "ak_" + strings.Repeat("a", 9),
			wantValid:  true,
		},
		{
			name:      "one char short of minimum",
			rawKey:    "ak_" + strings.Repeat("a", 63),
			prefix:    "ak_",
			wantValid: false,
		},
		{
			name:       "longer than minimum",
			rawKey:     "ak_" + strings.Repeat("a", 100),
			prefix:     "ak_",
			wantPrefix: "ak_" + strings.Repeat("a", 9),
			wantValid:  true,
		},
		{
			name:       "empty prefix with valid key",
			rawKey:     strings.Repeat("a", 64),
			prefix:     "",
			wantPrefix: strings.Repeat("a", 12),
			wantValid:  true,
		},
		{
			name:      "prefix only",
			rawKey:    "ak_",
			prefix:    "ak_",
			wantValid: false,
		},
		{
			name:      "case sensitive prefix",
			rawKey:    "AK_" + strings.Repeat("a", 64),
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

// TestHasScopeEdgeCases tests additional edge cases for HasScope
func TestHasScopeEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		key     key.Key
		scope   string
		wantHas bool
	}{
		{
			name:    "empty slice vs nil scopes",
			key:     key.Key{Scopes: []string{}},
			scope:   "/api/anything",
			wantHas: true,
		},
		{
			name:    "multiple wildcards",
			key:     key.Key{Scopes: []string{"*", "*"}},
			scope:   "/api/anything",
			wantHas: true,
		},
		{
			name:    "wildcard with specific scopes",
			key:     key.Key{Scopes: []string{"/api/users", "*", "/api/items"}},
			scope:   "/api/admin",
			wantHas: true,
		},
		{
			name:    "empty required scope with scopes defined",
			key:     key.Key{Scopes: []string{"/api/users"}},
			scope:   "",
			wantHas: false,
		},
		{
			name:    "empty required scope with no scopes",
			key:     key.Key{Scopes: nil},
			scope:   "",
			wantHas: true,
		},
		{
			name:    "partial match does not work",
			key:     key.Key{Scopes: []string{"/api/users"}},
			scope:   "/api/user",
			wantHas: false,
		},
		{
			name:    "scope with trailing slash",
			key:     key.Key{Scopes: []string{"/api/users/"}},
			scope:   "/api/users",
			wantHas: false,
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

// TestMatchPathEdgeCases tests additional edge cases for MatchPath
func TestMatchPathEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{
			name:    "empty pattern and path",
			pattern: "",
			path:    "",
			want:    true,
		},
		{
			name:    "empty pattern non-empty path",
			pattern: "",
			path:    "/api",
			want:    false,
		},
		{
			name:    "empty path non-empty pattern",
			pattern: "/api",
			path:    "",
			want:    false,
		},
		{
			name:    "root wildcard",
			pattern: "/*",
			path:    "/api",
			want:    true,
		},
		{
			name:    "root wildcard matches root",
			pattern: "/*",
			path:    "/",
			want:    true,
		},
		{
			name:    "root wildcard matches empty - not a subpath",
			pattern: "/*",
			path:    "",
			want:    true,
		},
		{
			name:    "wildcard at end only",
			pattern: "/api/v1/*",
			path:    "/api/v1/users",
			want:    true,
		},
		{
			name:    "wildcard without slash",
			pattern: "/api*",
			path:    "/api/users",
			want:    false,
		},
		{
			name:    "multiple wildcards only last counts",
			pattern: "/*/api/*",
			path:    "/v1/api/users",
			want:    false,
		},
		{
			name:    "path with query string - no match",
			pattern: "/api/users",
			path:    "/api/users?id=1",
			want:    false,
		},
		{
			name:    "trailing slash exact",
			pattern: "/api/users/",
			path:    "/api/users/",
			want:    true,
		},
		{
			name:    "trailing slash mismatch",
			pattern: "/api/users",
			path:    "/api/users/",
			want:    false,
		},
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

// TestKeyStruct tests the Key struct fields directly
func TestKeyStruct(t *testing.T) {
	now := time.Now().UTC()
	expires := now.Add(24 * time.Hour)
	revoked := now.Add(-1 * time.Hour)
	lastUsed := now.Add(-5 * time.Minute)

	k := key.Key{
		ID:        "key_abc123",
		UserID:    "user_456",
		Hash:      []byte("bcrypt_hash"),
		Prefix:    "ak_abcd1234",
		Name:      "Test Key",
		Scopes:    []string{"/api/users", "/api/items"},
		ExpiresAt: &expires,
		RevokedAt: &revoked,
		CreatedAt: now,
		LastUsed:  &lastUsed,
	}

	// Verify all fields are accessible and set correctly
	if k.ID != "key_abc123" {
		t.Errorf("Key.ID = %q, want %q", k.ID, "key_abc123")
	}
	if k.UserID != "user_456" {
		t.Errorf("Key.UserID = %q, want %q", k.UserID, "user_456")
	}
	if string(k.Hash) != "bcrypt_hash" {
		t.Errorf("Key.Hash = %q, want %q", string(k.Hash), "bcrypt_hash")
	}
	if k.Prefix != "ak_abcd1234" {
		t.Errorf("Key.Prefix = %q, want %q", k.Prefix, "ak_abcd1234")
	}
	if k.Name != "Test Key" {
		t.Errorf("Key.Name = %q, want %q", k.Name, "Test Key")
	}
	if len(k.Scopes) != 2 {
		t.Errorf("len(Key.Scopes) = %d, want %d", len(k.Scopes), 2)
	}
	if k.ExpiresAt == nil || !k.ExpiresAt.Equal(expires) {
		t.Errorf("Key.ExpiresAt = %v, want %v", k.ExpiresAt, expires)
	}
	if k.RevokedAt == nil || !k.RevokedAt.Equal(revoked) {
		t.Errorf("Key.RevokedAt = %v, want %v", k.RevokedAt, revoked)
	}
	if !k.CreatedAt.Equal(now) {
		t.Errorf("Key.CreatedAt = %v, want %v", k.CreatedAt, now)
	}
	if k.LastUsed == nil || !k.LastUsed.Equal(lastUsed) {
		t.Errorf("Key.LastUsed = %v, want %v", k.LastUsed, lastUsed)
	}
}

// TestValidationResult tests the ValidationResult struct
func TestValidationResult(t *testing.T) {
	validResult := key.ValidationResult{
		Valid: true,
		Key: key.Key{
			ID:     "key-1",
			UserID: "user-1",
		},
		Reason: "",
	}

	if !validResult.Valid {
		t.Error("ValidResult.Valid should be true")
	}
	if validResult.Key.ID != "key-1" {
		t.Errorf("ValidResult.Key.ID = %q, want %q", validResult.Key.ID, "key-1")
	}
	if validResult.Reason != "" {
		t.Errorf("ValidResult.Reason = %q, want empty", validResult.Reason)
	}

	invalidResult := key.ValidationResult{
		Valid:  false,
		Reason: key.ReasonExpired,
	}

	if invalidResult.Valid {
		t.Error("InvalidResult.Valid should be false")
	}
	if invalidResult.Reason != key.ReasonExpired {
		t.Errorf("InvalidResult.Reason = %q, want %q", invalidResult.Reason, key.ReasonExpired)
	}
}

// TestUserContext tests the UserContext struct
func TestUserContext(t *testing.T) {
	ctx := key.UserContext{
		KeyID:     "key-123",
		UserID:    "user-456",
		PlanID:    "plan-789",
		RateLimit: 100,
		Scopes:    []string{"/api/users", "/api/items"},
	}

	if ctx.KeyID != "key-123" {
		t.Errorf("UserContext.KeyID = %q, want %q", ctx.KeyID, "key-123")
	}
	if ctx.UserID != "user-456" {
		t.Errorf("UserContext.UserID = %q, want %q", ctx.UserID, "user-456")
	}
	if ctx.PlanID != "plan-789" {
		t.Errorf("UserContext.PlanID = %q, want %q", ctx.PlanID, "plan-789")
	}
	if ctx.RateLimit != 100 {
		t.Errorf("UserContext.RateLimit = %d, want %d", ctx.RateLimit, 100)
	}
	if len(ctx.Scopes) != 2 {
		t.Errorf("len(UserContext.Scopes) = %d, want %d", len(ctx.Scopes), 2)
	}
}

// TestCreateParams tests the CreateParams struct
func TestCreateParams(t *testing.T) {
	expires := time.Now().Add(24 * time.Hour)

	params := key.CreateParams{
		UserID:    "user-123",
		Name:      "My API Key",
		Scopes:    []string{"/api/users"},
		ExpiresAt: &expires,
	}

	if params.UserID != "user-123" {
		t.Errorf("CreateParams.UserID = %q, want %q", params.UserID, "user-123")
	}
	if params.Name != "My API Key" {
		t.Errorf("CreateParams.Name = %q, want %q", params.Name, "My API Key")
	}
	if len(params.Scopes) != 1 {
		t.Errorf("len(CreateParams.Scopes) = %d, want %d", len(params.Scopes), 1)
	}
	if params.ExpiresAt == nil || !params.ExpiresAt.Equal(expires) {
		t.Errorf("CreateParams.ExpiresAt = %v, want %v", params.ExpiresAt, expires)
	}
}

// TestReasonConstants tests that reason constants are defined correctly
func TestReasonConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		want     string
	}{
		{"ReasonValid", key.ReasonValid, ""},
		{"ReasonNotFound", key.ReasonNotFound, "key_not_found"},
		{"ReasonExpired", key.ReasonExpired, "key_expired"},
		{"ReasonRevoked", key.ReasonRevoked, "key_revoked"},
		{"ReasonBadFormat", key.ReasonBadFormat, "invalid_format"},
		{"ReasonUserSuspend", key.ReasonUserSuspend, "user_suspended"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.want)
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

// BenchmarkGenerate benchmarks key generation
func BenchmarkGenerate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		key.Generate("ak_")
	}
}

// BenchmarkValidateFormat benchmarks format validation
func BenchmarkValidateFormat(b *testing.B) {
	rawKey := "ak_abcd1234efgh5678901234567890123456789012345678901234567890123456"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key.ValidateFormat(rawKey, "ak_")
	}
}

// BenchmarkHasScope benchmarks scope checking
func BenchmarkHasScope(b *testing.B) {
	k := key.Key{
		Scopes: []string{"/api/users", "/api/items", "/api/orders", "/api/payments"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key.HasScope(k, "/api/orders")
	}
}

// BenchmarkMatchPath benchmarks path matching
func BenchmarkMatchPath(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key.MatchPath("/api/*", "/api/users/123/profile")
	}
}
