package oauth

import (
	"strings"
	"testing"
	"time"
)

func TestProvider_IsValid(t *testing.T) {
	tests := []struct {
		provider Provider
		want     bool
	}{
		{ProviderGoogle, true},
		{ProviderGitHub, true},
		{ProviderOIDC, true},
		{Provider("invalid"), false},
		{Provider(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			got := tt.provider.IsValid()
			if got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvider_DisplayName(t *testing.T) {
	tests := []struct {
		provider Provider
		want     string
	}{
		{ProviderGoogle, "Google"},
		{ProviderGitHub, "GitHub"},
		{ProviderOIDC, "OpenID Connect"},
		{Provider("custom"), "custom"},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			got := tt.provider.DisplayName()
			if got != tt.want {
				t.Errorf("DisplayName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateIdentityID(t *testing.T) {
	id := GenerateIdentityID()
	if !strings.HasPrefix(id, "oid_") {
		t.Errorf("GenerateIdentityID() = %v, want prefix oid_", id)
	}
	if len(id) != 20 { // oid_ (4) + 16 hex chars
		t.Errorf("GenerateIdentityID() length = %d, want 20", len(id))
	}

	// Ensure unique
	id2 := GenerateIdentityID()
	if id == id2 {
		t.Error("GenerateIdentityID() should generate unique IDs")
	}
}

func TestIdentity_WithID(t *testing.T) {
	i := Identity{Email: "test@example.com"}
	i2 := i.WithID("oid_123")

	if i2.ID != "oid_123" {
		t.Errorf("WithID() ID = %v, want oid_123", i2.ID)
	}
	if i2.Email != "test@example.com" {
		t.Error("WithID() should preserve Email")
	}
	if i.ID != "" {
		t.Error("WithID() should not modify original")
	}
}

func TestIdentity_WithUserID(t *testing.T) {
	i := Identity{Email: "test@example.com"}
	i2 := i.WithUserID("usr_123")

	if i2.UserID != "usr_123" {
		t.Errorf("WithUserID() UserID = %v, want usr_123", i2.UserID)
	}
	if i.UserID != "" {
		t.Error("WithUserID() should not modify original")
	}
}

func TestIdentity_WithTokens(t *testing.T) {
	i := Identity{Email: "test@example.com"}
	expires := time.Now().Add(time.Hour)
	i2 := i.WithTokens("access", "refresh", &expires)

	if i2.AccessToken != "access" {
		t.Errorf("WithTokens() AccessToken = %v, want access", i2.AccessToken)
	}
	if i2.RefreshToken != "refresh" {
		t.Errorf("WithTokens() RefreshToken = %v, want refresh", i2.RefreshToken)
	}
	if i2.TokenExpiresAt == nil || !i2.TokenExpiresAt.Equal(expires) {
		t.Errorf("WithTokens() TokenExpiresAt = %v, want %v", i2.TokenExpiresAt, expires)
	}
	if i.AccessToken != "" {
		t.Error("WithTokens() should not modify original")
	}
}

func TestIdentity_IsTokenExpired(t *testing.T) {
	now := time.Now().UTC()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{"nil expiration", nil, false},
		{"expired", &past, true},
		{"not expired", &future, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := Identity{TokenExpiresAt: tt.expiresAt}
			got := i.IsTokenExpired()
			if got != tt.want {
				t.Errorf("IsTokenExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateState(t *testing.T) {
	state := GenerateState(ProviderGoogle, "https://example.com/callback", 10*time.Minute)

	if state.State == "" {
		t.Error("GenerateState() State should not be empty")
	}
	if len(state.State) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("GenerateState() State length = %d, want 64", len(state.State))
	}
	if state.Provider != ProviderGoogle {
		t.Errorf("GenerateState() Provider = %v, want google", state.Provider)
	}
	if state.RedirectURI != "https://example.com/callback" {
		t.Errorf("GenerateState() RedirectURI = %v, want https://example.com/callback", state.RedirectURI)
	}
	if state.CodeVerifier == "" {
		t.Error("GenerateState() CodeVerifier should not be empty")
	}
	if state.Nonce == "" {
		t.Error("GenerateState() Nonce should not be empty")
	}
	if state.ExpiresAt.Before(time.Now().UTC()) {
		t.Error("GenerateState() ExpiresAt should be in the future")
	}
}

func TestState_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{"expired yesterday", time.Now().UTC().Add(-24 * time.Hour), true},
		{"expires tomorrow", time.Now().UTC().Add(24 * time.Hour), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := State{ExpiresAt: tt.expiresAt}
			got := s.IsExpired()
			if got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestState_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{"expired", time.Now().UTC().Add(-time.Hour), false},
		{"not expired", time.Now().UTC().Add(time.Hour), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := State{ExpiresAt: tt.expiresAt}
			got := s.IsValid()
			if got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestState_CodeChallenge(t *testing.T) {
	state := GenerateState(ProviderGoogle, "https://example.com", 10*time.Minute)
	challenge := state.CodeChallenge()

	if challenge == "" {
		t.Error("CodeChallenge() should not be empty")
	}
	// Base64 URL encoding without padding
	if strings.Contains(challenge, "=") {
		t.Error("CodeChallenge() should not contain padding")
	}
	// Deterministic: same verifier should give same challenge
	challenge2 := state.CodeChallenge()
	if challenge != challenge2 {
		t.Error("CodeChallenge() should be deterministic")
	}
}

func TestGenerateCodeVerifier(t *testing.T) {
	verifier := generateCodeVerifier()
	if verifier == "" {
		t.Error("generateCodeVerifier() should not be empty")
	}
	// Base64 URL encoding without padding
	if strings.Contains(verifier, "=") {
		t.Error("generateCodeVerifier() should not contain padding")
	}
	// Unique
	verifier2 := generateCodeVerifier()
	if verifier == verifier2 {
		t.Error("generateCodeVerifier() should generate unique verifiers")
	}
}

func TestGenerateCodeChallenge(t *testing.T) {
	verifier := "test_verifier"
	challenge := generateCodeChallenge(verifier)

	if challenge == "" {
		t.Error("generateCodeChallenge() should not be empty")
	}
	// Deterministic
	challenge2 := generateCodeChallenge(verifier)
	if challenge != challenge2 {
		t.Error("generateCodeChallenge() should be deterministic")
	}
	// Different verifier = different challenge
	challenge3 := generateCodeChallenge("different_verifier")
	if challenge == challenge3 {
		t.Error("generateCodeChallenge() should produce different challenges for different verifiers")
	}
}

func TestBase64URLEncode(t *testing.T) {
	data := []byte("test data")
	encoded := base64URLEncode(data)

	if encoded == "" {
		t.Error("base64URLEncode() should not be empty")
	}
	// No padding
	if strings.Contains(encoded, "=") {
		t.Error("base64URLEncode() should not contain padding")
	}
	// URL safe (no + or /)
	if strings.ContainsAny(encoded, "+/") {
		t.Error("base64URLEncode() should be URL safe")
	}
}

func TestTokenResponse_ExpiresAt(t *testing.T) {
	tests := []struct {
		name      string
		expiresIn int
		wantNil   bool
	}{
		{"zero expires in", 0, true},
		{"negative expires in", -1, true},
		{"positive expires in", 3600, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := TokenResponse{ExpiresIn: tt.expiresIn}
			got := tr.ExpiresAt()
			if tt.wantNil && got != nil {
				t.Errorf("ExpiresAt() = %v, want nil", got)
			}
			if !tt.wantNil {
				if got == nil {
					t.Error("ExpiresAt() should not be nil")
				} else {
					expectedTime := time.Now().UTC().Add(time.Duration(tt.expiresIn) * time.Second)
					// Allow 1 second tolerance
					if got.Sub(expectedTime) > time.Second || expectedTime.Sub(*got) > time.Second {
						t.Errorf("ExpiresAt() = %v, want approximately %v", got, expectedTime)
					}
				}
			}
		})
	}
}

func TestBuildAuthURLParams(t *testing.T) {
	t.Run("basic params", func(t *testing.T) {
		params := BuildAuthURLParams("client_id", "https://example.com/callback", "openid email", "state123", "", "")

		if params["client_id"] != "client_id" {
			t.Errorf("client_id = %v, want client_id", params["client_id"])
		}
		if params["redirect_uri"] != "https://example.com/callback" {
			t.Errorf("redirect_uri = %v, want https://example.com/callback", params["redirect_uri"])
		}
		if params["response_type"] != "code" {
			t.Errorf("response_type = %v, want code", params["response_type"])
		}
		if params["scope"] != "openid email" {
			t.Errorf("scope = %v, want openid email", params["scope"])
		}
		if params["state"] != "state123" {
			t.Errorf("state = %v, want state123", params["state"])
		}
		if _, ok := params["code_challenge"]; ok {
			t.Error("code_challenge should not be present when empty")
		}
		if _, ok := params["nonce"]; ok {
			t.Error("nonce should not be present when empty")
		}
	})

	t.Run("with PKCE", func(t *testing.T) {
		params := BuildAuthURLParams("client_id", "https://example.com", "openid", "state", "challenge123", "")

		if params["code_challenge"] != "challenge123" {
			t.Errorf("code_challenge = %v, want challenge123", params["code_challenge"])
		}
		if params["code_challenge_method"] != "S256" {
			t.Errorf("code_challenge_method = %v, want S256", params["code_challenge_method"])
		}
	})

	t.Run("with nonce", func(t *testing.T) {
		params := BuildAuthURLParams("client_id", "https://example.com", "openid", "state", "", "nonce123")

		if params["nonce"] != "nonce123" {
			t.Errorf("nonce = %v, want nonce123", params["nonce"])
		}
	})
}

func TestValidateCallback(t *testing.T) {
	tests := []struct {
		name      string
		code      string
		state     string
		errorP    string
		errorDesc string
		wantValid bool
		wantError string
	}{
		{
			name:      "valid callback",
			code:      "auth_code",
			state:     "state123",
			wantValid: true,
		},
		{
			name:      "error from provider",
			errorP:    "access_denied",
			errorDesc: "User denied access",
			wantValid: false,
			wantError: "access_denied",
		},
		{
			name:      "missing code",
			state:     "state123",
			wantValid: false,
			wantError: "missing_code",
		},
		{
			name:      "missing state",
			code:      "auth_code",
			wantValid: false,
			wantError: "missing_state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateCallback(tt.code, tt.state, tt.errorP, tt.errorDesc)
			if result.Valid != tt.wantValid {
				t.Errorf("ValidateCallback() Valid = %v, want %v", result.Valid, tt.wantValid)
			}
			if tt.wantError != "" && result.Error != tt.wantError {
				t.Errorf("ValidateCallback() Error = %v, want %v", result.Error, tt.wantError)
			}
			if tt.wantValid {
				if result.Code != tt.code {
					t.Errorf("ValidateCallback() Code = %v, want %v", result.Code, tt.code)
				}
				if result.State != tt.state {
					t.Errorf("ValidateCallback() State = %v, want %v", result.State, tt.state)
				}
			}
		})
	}
}
