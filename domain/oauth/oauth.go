// Package oauth provides OAuth/OIDC value types and pure validation functions.
// This package has NO dependencies on I/O or external packages.
package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"time"
)

// Provider identifies an OAuth provider.
type Provider string

const (
	ProviderGoogle Provider = "google"
	ProviderGitHub Provider = "github"
	ProviderOIDC   Provider = "oidc"
)

// IsValid returns true if the provider is known.
func (p Provider) IsValid() bool {
	switch p {
	case ProviderGoogle, ProviderGitHub, ProviderOIDC:
		return true
	}
	return false
}

// DisplayName returns a human-readable name for the provider.
func (p Provider) DisplayName() string {
	switch p {
	case ProviderGoogle:
		return "Google"
	case ProviderGitHub:
		return "GitHub"
	case ProviderOIDC:
		return "OpenID Connect"
	default:
		return string(p)
	}
}

// Identity represents a linked OAuth identity (immutable value type).
type Identity struct {
	ID             string
	UserID         string
	Provider       Provider
	ProviderUserID string
	Email          string
	Name           string
	AvatarURL      string
	AccessToken    string
	RefreshToken   string
	TokenExpiresAt *time.Time
	RawData        map[string]interface{}
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// GenerateIdentityID creates a new identity ID.
func GenerateIdentityID() string {
	idBytes := make([]byte, 8)
	rand.Read(idBytes)
	return "oid_" + hex.EncodeToString(idBytes)
}

// WithID returns a copy of the identity with the ID set.
func (i Identity) WithID(id string) Identity {
	i.ID = id
	return i
}

// WithUserID returns a copy of the identity with the UserID set.
func (i Identity) WithUserID(userID string) Identity {
	i.UserID = userID
	return i
}

// WithTokens returns a copy of the identity with tokens updated.
func (i Identity) WithTokens(accessToken, refreshToken string, expiresAt *time.Time) Identity {
	i.AccessToken = accessToken
	i.RefreshToken = refreshToken
	i.TokenExpiresAt = expiresAt
	return i
}

// IsTokenExpired returns true if the access token has expired.
func (i Identity) IsTokenExpired() bool {
	if i.TokenExpiresAt == nil {
		return false // No expiration set
	}
	return time.Now().UTC().After(*i.TokenExpiresAt)
}

// State represents an OAuth state token for CSRF protection (immutable value type).
type State struct {
	State        string
	Provider     Provider
	RedirectURI  string
	CodeVerifier string
	Nonce        string
	ExpiresAt    time.Time
	CreatedAt    time.Time
}

// GenerateState creates a new OAuth state with random values.
// Returns the state struct to store in database.
func GenerateState(provider Provider, redirectURI string, ttl time.Duration) State {
	stateBytes := make([]byte, 32)
	rand.Read(stateBytes)
	state := hex.EncodeToString(stateBytes)

	codeVerifier := generateCodeVerifier()

	nonceBytes := make([]byte, 16)
	rand.Read(nonceBytes)
	nonce := hex.EncodeToString(nonceBytes)

	now := time.Now().UTC()
	return State{
		State:        state,
		Provider:     provider,
		RedirectURI:  redirectURI,
		CodeVerifier: codeVerifier,
		Nonce:        nonce,
		ExpiresAt:    now.Add(ttl),
		CreatedAt:    now,
	}
}

// IsExpired returns true if the state has expired.
func (s State) IsExpired() bool {
	return time.Now().UTC().After(s.ExpiresAt)
}

// IsValid returns true if the state is not expired.
func (s State) IsValid() bool {
	return !s.IsExpired()
}

// CodeChallenge returns the PKCE code challenge derived from the verifier.
func (s State) CodeChallenge() string {
	return generateCodeChallenge(s.CodeVerifier)
}

// PKCE (Proof Key for Code Exchange) functions

// generateCodeVerifier creates a random code verifier for PKCE.
func generateCodeVerifier() string {
	verifierBytes := make([]byte, 32)
	rand.Read(verifierBytes)
	return base64URLEncode(verifierBytes)
}

// generateCodeChallenge creates a code challenge from a verifier using SHA256.
func generateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64URLEncode(h[:])
}

// base64URLEncode encodes bytes to URL-safe base64 without padding.
func base64URLEncode(data []byte) string {
	encoded := base64.URLEncoding.EncodeToString(data)
	// Remove padding
	return strings.TrimRight(encoded, "=")
}

// UserProfile represents a user profile from an OAuth provider (value type).
type UserProfile struct {
	ProviderUserID string
	Email          string
	EmailVerified  bool
	Name           string
	GivenName      string
	FamilyName     string
	AvatarURL      string
	RawData        map[string]interface{}
}

// AuthResult represents the outcome of OAuth authentication (value type).
type AuthResult struct {
	Success     bool
	Identity    Identity // Populated if successful
	UserProfile UserProfile
	IsNewUser   bool   // True if user was created
	Error       string // Populated if not successful
}

// TokenResponse represents tokens from an OAuth provider (value type).
type TokenResponse struct {
	AccessToken  string
	RefreshToken string
	IDToken      string // For OIDC
	TokenType    string
	ExpiresIn    int // Seconds until expiration
	Scope        string
	Error        string
}

// ExpiresAt calculates the expiration time based on ExpiresIn.
func (t TokenResponse) ExpiresAt() *time.Time {
	if t.ExpiresIn <= 0 {
		return nil
	}
	expiresAt := time.Now().UTC().Add(time.Duration(t.ExpiresIn) * time.Second)
	return &expiresAt
}

// AuthURLParams contains parameters for building an OAuth authorization URL.
type AuthURLParams struct {
	ClientID     string
	RedirectURI  string
	Scope        string
	State        string
	CodeChallenge string
	Nonce        string // For OIDC
}

// BuildAuthURL constructs an OAuth authorization URL.
// This is a pure function that returns URL query parameters.
func BuildAuthURLParams(clientID, redirectURI, scope, state, codeChallenge, nonce string) map[string]string {
	params := map[string]string{
		"client_id":     clientID,
		"redirect_uri":  redirectURI,
		"response_type": "code",
		"scope":         scope,
		"state":         state,
	}

	if codeChallenge != "" {
		params["code_challenge"] = codeChallenge
		params["code_challenge_method"] = "S256"
	}

	if nonce != "" {
		params["nonce"] = nonce
	}

	return params
}

// ValidateCallback validates an OAuth callback request (pure function).
type CallbackValidation struct {
	Valid       bool
	Code        string
	State       string
	Error       string
	ErrorDesc   string
}

// ValidateCallback extracts and validates OAuth callback parameters.
func ValidateCallback(code, state, errorParam, errorDesc string) CallbackValidation {
	if errorParam != "" {
		return CallbackValidation{
			Valid:     false,
			Error:     errorParam,
			ErrorDesc: errorDesc,
		}
	}

	if code == "" {
		return CallbackValidation{
			Valid: false,
			Error: "missing_code",
		}
	}

	if state == "" {
		return CallbackValidation{
			Valid: false,
			Error: "missing_state",
		}
	}

	return CallbackValidation{
		Valid: true,
		Code:  code,
		State: state,
	}
}
