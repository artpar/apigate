// Package oauth provides OAuth provider implementations.
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/artpar/apigate/domain/oauth"
	"github.com/artpar/apigate/ports"
)

const (
	googleAuthURL     = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL    = "https://oauth2.googleapis.com/token"
	googleUserInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
)

// GoogleProvider implements OAuth for Google.
type GoogleProvider struct {
	clientID     string
	clientSecret string
	scopes       []string
	hostedDomain string // Optional: restrict to specific Google Workspace domain
	httpClient   *http.Client
}

// GoogleConfig holds configuration for Google OAuth.
type GoogleConfig struct {
	ClientID     string
	ClientSecret string
	Scopes       []string
	HostedDomain string
}

// NewGoogleProvider creates a new Google OAuth provider.
func NewGoogleProvider(cfg GoogleConfig) *GoogleProvider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "email", "profile"}
	}

	return &GoogleProvider{
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		scopes:       scopes,
		hostedDomain: cfg.HostedDomain,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

// Name returns the provider name.
func (p *GoogleProvider) Name() string {
	return string(oauth.ProviderGoogle)
}

// GetAuthURL returns the authorization URL.
func (p *GoogleProvider) GetAuthURL(ctx context.Context, state, codeChallenge, nonce, redirectURI string) (string, error) {
	params := url.Values{
		"client_id":             {p.clientID},
		"redirect_uri":          {redirectURI},
		"response_type":         {"code"},
		"scope":                 {strings.Join(p.scopes, " ")},
		"state":                 {state},
		"access_type":           {"offline"}, // Request refresh token
		"prompt":                {"consent"}, // Force consent to get refresh token
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
	}

	if nonce != "" {
		params.Set("nonce", nonce)
	}

	if p.hostedDomain != "" {
		params.Set("hd", p.hostedDomain)
	}

	return googleAuthURL + "?" + params.Encode(), nil
}

// ExchangeCode exchanges an authorization code for tokens.
func (p *GoogleProvider) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (oauth.TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {p.clientID},
		"client_secret": {p.clientSecret},
		"code_verifier": {codeVerifier},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", googleTokenURL, strings.NewReader(data.Encode()))
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
func (p *GoogleProvider) GetUserProfile(ctx context.Context, accessToken string) (oauth.UserProfile, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", googleUserInfoURL, nil)
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

	var userInfo struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		VerifiedEmail bool   `json:"verified_email"`
		Name          string `json:"name"`
		GivenName     string `json:"given_name"`
		FamilyName    string `json:"family_name"`
		Picture       string `json:"picture"`
	}
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return oauth.UserProfile{}, fmt.Errorf("parse userinfo response: %w", err)
	}

	// Store raw data
	var rawData map[string]interface{}
	json.Unmarshal(body, &rawData)

	return oauth.UserProfile{
		ProviderUserID: userInfo.ID,
		Email:          userInfo.Email,
		EmailVerified:  userInfo.VerifiedEmail,
		Name:           userInfo.Name,
		GivenName:      userInfo.GivenName,
		FamilyName:     userInfo.FamilyName,
		AvatarURL:      userInfo.Picture,
		RawData:        rawData,
	}, nil
}

// RefreshToken refreshes an access token.
func (p *GoogleProvider) RefreshToken(ctx context.Context, refreshToken string) (oauth.TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {p.clientID},
		"client_secret": {p.clientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", googleTokenURL, strings.NewReader(data.Encode()))
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
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
		Scope       string `json:"scope"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return oauth.TokenResponse{}, fmt.Errorf("parse refresh response: %w", err)
	}

	return oauth.TokenResponse{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: refreshToken, // Keep existing refresh token
		TokenType:    tokenResp.TokenType,
		ExpiresIn:    tokenResp.ExpiresIn,
		Scope:        tokenResp.Scope,
	}, nil
}

// Ensure interface compliance.
var _ ports.OAuthProvider = (*GoogleProvider)(nil)
