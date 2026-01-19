package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/artpar/apigate/domain/oauth"
	"github.com/artpar/apigate/ports"
)

// OIDCProvider implements OAuth for any OpenID Connect provider.
type OIDCProvider struct {
	name         string
	clientID     string
	clientSecret string
	issuerURL    string
	scopes       []string
	httpClient   *http.Client

	// Cached discovery document
	mu        sync.RWMutex
	discovery *oidcDiscovery
	fetchedAt time.Time
}

// oidcDiscovery represents the OIDC discovery document.
type oidcDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserInfoEndpoint      string `json:"userinfo_endpoint"`
	JWKsURI               string `json:"jwks_uri"`
	EndSessionEndpoint    string `json:"end_session_endpoint"`
}

// OIDCConfig holds configuration for a generic OIDC provider.
type OIDCConfig struct {
	Name         string   // Display name
	ClientID     string
	ClientSecret string
	IssuerURL    string   // OIDC issuer URL (discovery will be at /.well-known/openid-configuration)
	Scopes       []string // Optional, defaults to ["openid", "email", "profile"]
}

// NewOIDCProvider creates a new OIDC provider.
func NewOIDCProvider(cfg OIDCConfig) *OIDCProvider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "email", "profile"}
	}

	name := cfg.Name
	if name == "" {
		name = "oidc"
	}

	return &OIDCProvider{
		name:         name,
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		issuerURL:    strings.TrimSuffix(cfg.IssuerURL, "/"),
		scopes:       scopes,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

// Name returns the provider name.
func (p *OIDCProvider) Name() string {
	return string(oauth.ProviderOIDC)
}

// getDiscovery fetches or returns cached OIDC discovery document.
func (p *OIDCProvider) getDiscovery(ctx context.Context) (*oidcDiscovery, error) {
	// Check cache (valid for 1 hour)
	p.mu.RLock()
	if p.discovery != nil && time.Since(p.fetchedAt) < time.Hour {
		discovery := p.discovery
		p.mu.RUnlock()
		return discovery, nil
	}
	p.mu.RUnlock()

	// Fetch discovery document
	discoveryURL := p.issuerURL + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, "GET", discoveryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create discovery request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("discovery request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discovery request failed: %s", string(body))
	}

	var discovery oidcDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		return nil, fmt.Errorf("parse discovery document: %w", err)
	}

	// Cache the discovery document
	p.mu.Lock()
	p.discovery = &discovery
	p.fetchedAt = time.Now()
	p.mu.Unlock()

	return &discovery, nil
}

// GetAuthURL returns the authorization URL.
func (p *OIDCProvider) GetAuthURL(ctx context.Context, state, codeChallenge, nonce, redirectURI string) (string, error) {
	discovery, err := p.getDiscovery(ctx)
	if err != nil {
		return "", fmt.Errorf("get discovery: %w", err)
	}

	params := url.Values{
		"client_id":             {p.clientID},
		"redirect_uri":          {redirectURI},
		"response_type":         {"code"},
		"scope":                 {strings.Join(p.scopes, " ")},
		"state":                 {state},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
	}

	if nonce != "" {
		params.Set("nonce", nonce)
	}

	return discovery.AuthorizationEndpoint + "?" + params.Encode(), nil
}

// ExchangeCode exchanges an authorization code for tokens.
func (p *OIDCProvider) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (oauth.TokenResponse, error) {
	discovery, err := p.getDiscovery(ctx)
	if err != nil {
		return oauth.TokenResponse{}, fmt.Errorf("get discovery: %w", err)
	}

	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {p.clientID},
		"client_secret": {p.clientSecret},
		"code_verifier": {codeVerifier},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", discovery.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return oauth.TokenResponse{}, fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return oauth.TokenResponse{}, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return oauth.TokenResponse{}, fmt.Errorf("read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}
		json.Unmarshal(body, &errResp)
		return oauth.TokenResponse{
			Error: fmt.Sprintf("%s: %s", errResp.Error, errResp.Description),
		}, nil
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return oauth.TokenResponse{}, fmt.Errorf("parse token response: %w", err)
	}

	return oauth.TokenResponse{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		IDToken:      tokenResp.IDToken,
		TokenType:    tokenResp.TokenType,
		ExpiresIn:    tokenResp.ExpiresIn,
		Scope:        tokenResp.Scope,
	}, nil
}

// GetUserProfile fetches the user profile using the access token.
func (p *OIDCProvider) GetUserProfile(ctx context.Context, accessToken string) (oauth.UserProfile, error) {
	discovery, err := p.getDiscovery(ctx)
	if err != nil {
		return oauth.UserProfile{}, fmt.Errorf("get discovery: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", discovery.UserInfoEndpoint, nil)
	if err != nil {
		return oauth.UserProfile{}, fmt.Errorf("create userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return oauth.UserProfile{}, fmt.Errorf("userinfo request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return oauth.UserProfile{}, fmt.Errorf("read userinfo response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return oauth.UserProfile{}, fmt.Errorf("userinfo request failed: %s", string(body))
	}

	// OIDC standard claims
	var claims struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		GivenName     string `json:"given_name"`
		FamilyName    string `json:"family_name"`
		Picture       string `json:"picture"`
		Nickname      string `json:"nickname"`
		PreferredUsername string `json:"preferred_username"`
	}
	if err := json.Unmarshal(body, &claims); err != nil {
		return oauth.UserProfile{}, fmt.Errorf("parse userinfo response: %w", err)
	}

	// Store raw data
	var rawData map[string]interface{}
	json.Unmarshal(body, &rawData)

	// Use preferred_username or nickname if name is empty
	name := claims.Name
	if name == "" {
		name = claims.PreferredUsername
	}
	if name == "" {
		name = claims.Nickname
	}

	return oauth.UserProfile{
		ProviderUserID: claims.Sub,
		Email:          claims.Email,
		EmailVerified:  claims.EmailVerified,
		Name:           name,
		GivenName:      claims.GivenName,
		FamilyName:     claims.FamilyName,
		AvatarURL:      claims.Picture,
		RawData:        rawData,
	}, nil
}

// RefreshToken refreshes an access token.
func (p *OIDCProvider) RefreshToken(ctx context.Context, refreshToken string) (oauth.TokenResponse, error) {
	discovery, err := p.getDiscovery(ctx)
	if err != nil {
		return oauth.TokenResponse{}, fmt.Errorf("get discovery: %w", err)
	}

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {p.clientID},
		"client_secret": {p.clientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", discovery.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return oauth.TokenResponse{}, fmt.Errorf("create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return oauth.TokenResponse{}, fmt.Errorf("refresh request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return oauth.TokenResponse{}, fmt.Errorf("read refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}
		json.Unmarshal(body, &errResp)
		return oauth.TokenResponse{
			Error: fmt.Sprintf("%s: %s", errResp.Error, errResp.Description),
		}, nil
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return oauth.TokenResponse{}, fmt.Errorf("parse refresh response: %w", err)
	}

	// If no new refresh token was issued, keep the old one
	if tokenResp.RefreshToken == "" {
		tokenResp.RefreshToken = refreshToken
	}

	return oauth.TokenResponse{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		IDToken:      tokenResp.IDToken,
		TokenType:    tokenResp.TokenType,
		ExpiresIn:    tokenResp.ExpiresIn,
		Scope:        tokenResp.Scope,
	}, nil
}

// Ensure interface compliance.
var _ ports.OAuthProvider = (*OIDCProvider)(nil)
