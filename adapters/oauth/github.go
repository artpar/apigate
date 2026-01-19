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
	githubAuthURL     = "https://github.com/login/oauth/authorize"
	githubTokenURL    = "https://github.com/login/oauth/access_token"
	githubUserURL     = "https://api.github.com/user"
	githubUserOrgsURL = "https://api.github.com/user/orgs"
)

// GitHubProvider implements OAuth for GitHub.
type GitHubProvider struct {
	clientID          string
	clientSecret      string
	scopes            []string
	allowedOrgs       []string // Optional: restrict to specific organizations
	allowPrivateEmail bool
	httpClient        *http.Client
}

// GitHubConfig holds configuration for GitHub OAuth.
type GitHubConfig struct {
	ClientID          string
	ClientSecret      string
	Scopes            []string
	AllowedOrgs       []string
	AllowPrivateEmail bool
}

// NewGitHubProvider creates a new GitHub OAuth provider.
func NewGitHubProvider(cfg GitHubConfig) *GitHubProvider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"read:user", "user:email"}
	}

	return &GitHubProvider{
		clientID:          cfg.ClientID,
		clientSecret:      cfg.ClientSecret,
		scopes:            scopes,
		allowedOrgs:       cfg.AllowedOrgs,
		allowPrivateEmail: cfg.AllowPrivateEmail,
		httpClient:        &http.Client{Timeout: 10 * time.Second},
	}
}

// Name returns the provider name.
func (p *GitHubProvider) Name() string {
	return string(oauth.ProviderGitHub)
}

// GetAuthURL returns the authorization URL.
func (p *GitHubProvider) GetAuthURL(ctx context.Context, state, codeChallenge, nonce, redirectURI string) (string, error) {
	params := url.Values{
		"client_id":    {p.clientID},
		"redirect_uri": {redirectURI},
		"scope":        {strings.Join(p.scopes, " ")},
		"state":        {state},
	}

	// GitHub doesn't support PKCE yet, but we include it for future compatibility
	// and to maintain consistent API across providers

	return githubAuthURL + "?" + params.Encode(), nil
}

// ExchangeCode exchanges an authorization code for tokens.
func (p *GitHubProvider) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (oauth.TokenResponse, error) {
	data := url.Values{
		"client_id":     {p.clientID},
		"client_secret": {p.clientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", githubTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return oauth.TokenResponse{}, fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return oauth.TokenResponse{}, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return oauth.TokenResponse{}, fmt.Errorf("read token response: %w", err)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		Scope        string `json:"scope"`
		Error        string `json:"error"`
		ErrorDesc    string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return oauth.TokenResponse{}, fmt.Errorf("parse token response: %w", err)
	}

	if tokenResp.Error != "" {
		return oauth.TokenResponse{
			Error: fmt.Sprintf("%s: %s", tokenResp.Error, tokenResp.ErrorDesc),
		}, nil
	}

	return oauth.TokenResponse{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
		Scope:        tokenResp.Scope,
		ExpiresIn:    0, // GitHub tokens don't expire by default
	}, nil
}

// GetUserProfile fetches the user profile using the access token.
func (p *GitHubProvider) GetUserProfile(ctx context.Context, accessToken string) (oauth.UserProfile, error) {
	// Fetch user info
	req, err := http.NewRequestWithContext(ctx, "GET", githubUserURL, nil)
	if err != nil {
		return oauth.UserProfile{}, fmt.Errorf("create user request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return oauth.UserProfile{}, fmt.Errorf("user request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return oauth.UserProfile{}, fmt.Errorf("read user response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return oauth.UserProfile{}, fmt.Errorf("user request failed: %s", string(body))
	}

	var userInfo struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return oauth.UserProfile{}, fmt.Errorf("parse user response: %w", err)
	}

	// If email is empty or we need to allow private email, fetch from emails endpoint
	email := userInfo.Email
	emailVerified := email != ""
	if email == "" || p.allowPrivateEmail {
		fetchedEmail, verified, err := p.fetchPrimaryEmail(ctx, accessToken)
		if err == nil && fetchedEmail != "" {
			email = fetchedEmail
			emailVerified = verified
		}
	}

	// Check organization membership if required
	if len(p.allowedOrgs) > 0 {
		orgs, err := p.fetchUserOrgs(ctx, accessToken)
		if err != nil {
			return oauth.UserProfile{}, fmt.Errorf("fetch user orgs: %w", err)
		}

		allowed := false
		for _, org := range orgs {
			for _, allowedOrg := range p.allowedOrgs {
				if strings.EqualFold(org, allowedOrg) {
					allowed = true
					break
				}
			}
			if allowed {
				break
			}
		}
		if !allowed {
			return oauth.UserProfile{}, fmt.Errorf("user is not a member of allowed organizations")
		}
	}

	// Store raw data
	var rawData map[string]interface{}
	json.Unmarshal(body, &rawData)

	name := userInfo.Name
	if name == "" {
		name = userInfo.Login
	}

	return oauth.UserProfile{
		ProviderUserID: fmt.Sprintf("%d", userInfo.ID),
		Email:          email,
		EmailVerified:  emailVerified,
		Name:           name,
		AvatarURL:      userInfo.AvatarURL,
		RawData:        rawData,
	}, nil
}

// fetchPrimaryEmail fetches the user's primary email from GitHub API.
func (p *GitHubProvider) fetchPrimaryEmail(ctx context.Context, accessToken string) (string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("emails request failed with status %d", resp.StatusCode)
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, err
	}

	if err := json.Unmarshal(body, &emails); err != nil {
		return "", false, err
	}

	// Find primary verified email
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, true, nil
		}
	}

	// Fall back to any verified email
	for _, e := range emails {
		if e.Verified {
			return e.Email, true, nil
		}
	}

	// Fall back to primary email even if not verified
	for _, e := range emails {
		if e.Primary {
			return e.Email, false, nil
		}
	}

	return "", false, nil
}

// fetchUserOrgs fetches the user's organizations.
func (p *GitHubProvider) fetchUserOrgs(ctx context.Context, accessToken string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", githubUserOrgsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("orgs request failed with status %d", resp.StatusCode)
	}

	var orgs []struct {
		Login string `json:"login"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(body, &orgs); err != nil {
		return nil, err
	}

	result := make([]string, len(orgs))
	for i, org := range orgs {
		result[i] = org.Login
	}
	return result, nil
}

// RefreshToken refreshes an access token.
// GitHub tokens don't expire by default, so this is a no-op.
func (p *GitHubProvider) RefreshToken(ctx context.Context, refreshToken string) (oauth.TokenResponse, error) {
	// GitHub OAuth tokens don't expire and don't support refresh
	// If a refresh token was provided (from GitHub Apps), try to use it
	if refreshToken == "" {
		return oauth.TokenResponse{
			Error: "GitHub OAuth tokens do not support refresh",
		}, nil
	}

	data := url.Values{
		"client_id":     {p.clientID},
		"client_secret": {p.clientSecret},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", githubTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return oauth.TokenResponse{}, fmt.Errorf("create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return oauth.TokenResponse{}, fmt.Errorf("refresh request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return oauth.TokenResponse{}, fmt.Errorf("read refresh response: %w", err)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		Scope        string `json:"scope"`
		ExpiresIn    int    `json:"expires_in"`
		Error        string `json:"error"`
		ErrorDesc    string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return oauth.TokenResponse{}, fmt.Errorf("parse refresh response: %w", err)
	}

	if tokenResp.Error != "" {
		return oauth.TokenResponse{
			Error: fmt.Sprintf("%s: %s", tokenResp.Error, tokenResp.ErrorDesc),
		}, nil
	}

	return oauth.TokenResponse{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
		Scope:        tokenResp.Scope,
		ExpiresIn:    tokenResp.ExpiresIn,
	}, nil
}

// Ensure interface compliance.
var _ ports.OAuthProvider = (*GitHubProvider)(nil)
