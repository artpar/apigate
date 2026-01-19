package web

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/artpar/apigate/domain/oauth"
	"github.com/artpar/apigate/ports"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// OAuthStart initiates the OAuth flow for a provider.
func (h *Handler) OAuthStart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	providerName := chi.URLParam(r, "provider")

	if providerName == "" {
		http.Error(w, "Provider required", http.StatusBadRequest)
		return
	}

	// Get the OAuth provider
	provider, ok := h.oauthProviders[providerName]
	if !ok {
		http.Error(w, "Unknown OAuth provider", http.StatusBadRequest)
		return
	}

	if h.oauthStates == nil {
		http.Error(w, "OAuth not configured", http.StatusServiceUnavailable)
		return
	}

	// Generate state for CSRF protection
	stateBytes := make([]byte, 32)
	rand.Read(stateBytes)
	state := hex.EncodeToString(stateBytes)

	// Generate PKCE code verifier and challenge
	verifierBytes := make([]byte, 32)
	rand.Read(verifierBytes)
	codeVerifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	// S256 challenge
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	// Generate nonce for OIDC
	nonceBytes := make([]byte, 16)
	rand.Read(nonceBytes)
	nonce := hex.EncodeToString(nonceBytes)

	// Get redirect URI
	redirectURI := r.URL.Query().Get("redirect")
	if redirectURI == "" {
		redirectURI = "/dashboard"
	}

	// Build callback URL
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	callbackURL := scheme + "://" + r.Host + "/auth/oauth/" + providerName + "/callback"

	// Store state in database
	now := time.Now().UTC()
	oauthState := oauth.State{
		State:        state,
		Provider:     oauth.Provider(providerName),
		RedirectURI:  redirectURI,
		CodeVerifier: codeVerifier,
		Nonce:        nonce,
		CreatedAt:    now,
		ExpiresAt:    now.Add(10 * time.Minute), // State valid for 10 minutes
	}

	if err := h.oauthStates.Create(ctx, oauthState); err != nil {
		h.logger.Error().Err(err).Msg("Failed to create OAuth state")
		http.Error(w, "Failed to initiate OAuth", http.StatusInternalServerError)
		return
	}

	// Get authorization URL from provider
	authURL, err := provider.GetAuthURL(ctx, state, codeChallenge, nonce, callbackURL)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get OAuth auth URL")
		http.Error(w, "Failed to initiate OAuth", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, authURL, http.StatusFound)
}

// OAuthCallback handles the OAuth callback from the provider.
func (h *Handler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	providerName := chi.URLParam(r, "provider")

	// Check for error from provider
	if errCode := r.URL.Query().Get("error"); errCode != "" {
		errDesc := r.URL.Query().Get("error_description")
		h.logger.Warn().Str("error", errCode).Str("description", errDesc).Msg("OAuth error")
		http.Redirect(w, r, "/login?error="+errCode, http.StatusFound)
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		http.Error(w, "Missing code or state", http.StatusBadRequest)
		return
	}

	// Get the OAuth provider
	provider, ok := h.oauthProviders[providerName]
	if !ok {
		http.Error(w, "Unknown OAuth provider", http.StatusBadRequest)
		return
	}

	if h.oauthStates == nil || h.oauthIdentities == nil {
		http.Error(w, "OAuth not configured", http.StatusServiceUnavailable)
		return
	}

	// Validate state
	oauthState, err := h.oauthStates.Get(ctx, state)
	if err != nil {
		h.logger.Warn().Err(err).Msg("Invalid OAuth state")
		http.Redirect(w, r, "/login?error=invalid_state", http.StatusFound)
		return
	}

	// Check if state has expired
	if time.Now().UTC().After(oauthState.ExpiresAt) {
		h.oauthStates.Delete(ctx, state)
		http.Redirect(w, r, "/login?error=state_expired", http.StatusFound)
		return
	}

	// Delete state (single use)
	h.oauthStates.Delete(ctx, state)

	// Build callback URL
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	callbackURL := scheme + "://" + r.Host + "/auth/oauth/" + providerName + "/callback"

	// Exchange code for tokens
	tokenResp, err := provider.ExchangeCode(ctx, code, oauthState.CodeVerifier, callbackURL)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to exchange OAuth code")
		http.Redirect(w, r, "/login?error=exchange_failed", http.StatusFound)
		return
	}

	if tokenResp.Error != "" {
		h.logger.Warn().Str("error", tokenResp.Error).Msg("OAuth token error")
		http.Redirect(w, r, "/login?error=token_error", http.StatusFound)
		return
	}

	// Get user profile from provider
	profile, err := provider.GetUserProfile(ctx, tokenResp.AccessToken)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get OAuth user profile")
		http.Redirect(w, r, "/login?error=profile_failed", http.StatusFound)
		return
	}

	// Try to find existing OAuth identity
	identity, err := h.oauthIdentities.GetByProviderUser(ctx, oauth.Provider(providerName), profile.ProviderUserID)
	if err == nil {
		// Found existing identity - update tokens and log in
		identity.AccessToken = tokenResp.AccessToken
		identity.RefreshToken = tokenResp.RefreshToken
		if tokenResp.ExpiresIn > 0 {
			expiresAt := time.Now().UTC().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
			identity.TokenExpiresAt = &expiresAt
		}
		identity.UpdatedAt = time.Now().UTC()

		h.oauthIdentities.Update(ctx, identity)

		// Get the user and log in
		u, err := h.users.Get(ctx, identity.UserID)
		if err != nil {
			h.logger.Error().Err(err).Msg("Failed to get user for OAuth identity")
			http.Redirect(w, r, "/login?error=user_not_found", http.StatusFound)
			return
		}

		h.loginUser(w, r, u, oauthState.RedirectURI)
		return
	}

	// No existing identity - try to link or create user
	var u ports.User
	foundUser := false

	// Try to find user by email if we have a verified email
	if profile.Email != "" && profile.EmailVerified && h.users != nil {
		existingUser, err := h.users.GetByEmail(ctx, profile.Email)
		if err == nil {
			u = existingUser
			foundUser = true
		}
	}

	// If no user found by email, check if registration is allowed
	if !foundUser {
		// Check if OAuth registration is enabled
		allowRegistration := true // Default to true, could be checked from settings
		if h.settings != nil {
			setting, err := h.settings.Get(ctx, "oauth.allow_registration")
			if err == nil && setting.Value == "false" {
				allowRegistration = false
			}
		}

		if !allowRegistration {
			http.Redirect(w, r, "/login?error=registration_disabled", http.StatusFound)
			return
		}

		// Create new user
		name := profile.Name
		if name == "" {
			name = strings.Split(profile.Email, "@")[0]
		}

		now := time.Now().UTC()
		newUser := ports.User{
			ID:        uuid.New().String(),
			Email:     profile.Email,
			Name:      name,
			Status:    "active",
			CreatedAt: now,
			UpdatedAt: now,
		}

		if h.users != nil {
			if err := h.users.Create(ctx, newUser); err != nil {
				h.logger.Error().Err(err).Msg("Failed to create user from OAuth")
				http.Redirect(w, r, "/login?error=user_creation_failed", http.StatusFound)
				return
			}
		}
		u = newUser
	}

	// Create OAuth identity
	now := time.Now().UTC()
	newIdentity := oauth.Identity{
		ID:             oauth.GenerateIdentityID(),
		UserID:         u.ID,
		Provider:       oauth.Provider(providerName),
		ProviderUserID: profile.ProviderUserID,
		Email:          profile.Email,
		Name:           profile.Name,
		AvatarURL:      profile.AvatarURL,
		AccessToken:    tokenResp.AccessToken,
		RefreshToken:   tokenResp.RefreshToken,
		RawData:        profile.RawData,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if tokenResp.ExpiresIn > 0 {
		expiresAt := now.Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		newIdentity.TokenExpiresAt = &expiresAt
	}

	if err := h.oauthIdentities.Create(ctx, newIdentity); err != nil {
		h.logger.Error().Err(err).Msg("Failed to create OAuth identity")
		// Log the user in anyway if they exist
		if foundUser {
			h.loginUser(w, r, u, oauthState.RedirectURI)
			return
		}
		http.Redirect(w, r, "/login?error=identity_creation_failed", http.StatusFound)
		return
	}

	h.loginUser(w, r, u, oauthState.RedirectURI)
}

// loginUser logs in a user and redirects them.
func (h *Handler) loginUser(w http.ResponseWriter, r *http.Request, u ports.User, redirectURI string) {
	// Generate JWT token
	// OAuth users are treated as customers (not admins) by default
	role := "customer"

	token, expiresAt, err := h.tokens.GenerateToken(u.ID, u.Email, role)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to generate token")
		http.Redirect(w, r, "/login?error=token_generation_failed", http.StatusFound)
		return
	}

	// Set JWT cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	// Also set apigate_session cookie for module WebUI compatibility
	setModuleSessionCookie(w, u.ID, u.Email, u.Name, expiresAt)

	// Redirect to the original destination or dashboard
	if redirectURI == "" || !strings.HasPrefix(redirectURI, "/") {
		redirectURI = "/dashboard"
	}

	http.Redirect(w, r, redirectURI, http.StatusFound)
}

// OAuthUnlink handles unlinking an OAuth identity from the current user.
func (h *Handler) OAuthUnlink(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := getClaims(ctx)
	if claims == nil {
		h.redirectToLogin(w, r)
		return
	}

	providerName := chi.URLParam(r, "provider")
	if providerName == "" {
		http.Error(w, "Provider required", http.StatusBadRequest)
		return
	}

	if h.oauthIdentities == nil {
		http.Error(w, "OAuth not configured", http.StatusServiceUnavailable)
		return
	}

	// Find the OAuth identity for this user and provider
	identity, err := h.oauthIdentities.GetByUserAndProvider(ctx, claims.UserID, oauth.Provider(providerName))
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			http.Error(w, "OAuth identity not found", http.StatusNotFound)
			return
		}
		h.logger.Error().Err(err).Msg("Failed to get OAuth identity")
		http.Error(w, "Failed to unlink OAuth", http.StatusInternalServerError)
		return
	}

	// Check that user has another way to log in (password or another OAuth)
	if h.users != nil {
		u, err := h.users.Get(ctx, claims.UserID)
		if err == nil && len(u.PasswordHash) == 0 {
			// User has no password, check for other OAuth identities
			identities, err := h.oauthIdentities.ListByUser(ctx, claims.UserID)
			if err != nil || len(identities) <= 1 {
				http.Error(w, "Cannot unlink - you would have no way to log in", http.StatusBadRequest)
				return
			}
		}
	}

	// Delete the identity
	if err := h.oauthIdentities.Delete(ctx, identity.ID); err != nil {
		h.logger.Error().Err(err).Msg("Failed to delete OAuth identity")
		http.Error(w, "Failed to unlink OAuth", http.StatusInternalServerError)
		return
	}

	// For HTMX requests, return success
	if r.Header.Get("HX-Request") == "true" {
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/settings?success=OAuth+unlinked", http.StatusFound)
}

// OAuthLink initiates OAuth linking for an authenticated user.
// This is similar to OAuthStart but stores the user ID to link to.
func (h *Handler) OAuthLink(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := getClaims(ctx)
	if claims == nil {
		h.redirectToLogin(w, r)
		return
	}

	providerName := chi.URLParam(r, "provider")
	if providerName == "" {
		http.Error(w, "Provider required", http.StatusBadRequest)
		return
	}

	// Get the OAuth provider
	provider, ok := h.oauthProviders[providerName]
	if !ok {
		http.Error(w, "Unknown OAuth provider", http.StatusBadRequest)
		return
	}

	if h.oauthStates == nil {
		http.Error(w, "OAuth not configured", http.StatusServiceUnavailable)
		return
	}

	// Check if already linked
	if h.oauthIdentities != nil {
		_, err := h.oauthIdentities.GetByUserAndProvider(ctx, claims.UserID, oauth.Provider(providerName))
		if err == nil {
			http.Redirect(w, r, "/settings?error=already_linked", http.StatusFound)
			return
		}
	}

	// Generate state with link marker
	stateBytes := make([]byte, 32)
	rand.Read(stateBytes)
	state := "link_" + claims.UserID + "_" + hex.EncodeToString(stateBytes)

	// Generate PKCE
	verifierBytes := make([]byte, 32)
	rand.Read(verifierBytes)
	codeVerifier := base64.RawURLEncoding.EncodeToString(verifierBytes)
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	// Generate nonce
	nonceBytes := make([]byte, 16)
	rand.Read(nonceBytes)
	nonce := hex.EncodeToString(nonceBytes)

	// Build callback URL
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	callbackURL := scheme + "://" + r.Host + "/auth/oauth/" + providerName + "/callback"

	// Store state
	now := time.Now().UTC()
	oauthState := oauth.State{
		State:        state,
		Provider:     oauth.Provider(providerName),
		RedirectURI:  "/settings",
		CodeVerifier: codeVerifier,
		Nonce:        nonce,
		CreatedAt:    now,
		ExpiresAt:    now.Add(10 * time.Minute),
	}

	if err := h.oauthStates.Create(ctx, oauthState); err != nil {
		h.logger.Error().Err(err).Msg("Failed to create OAuth state")
		http.Error(w, "Failed to initiate OAuth", http.StatusInternalServerError)
		return
	}

	// Get authorization URL
	authURL, err := provider.GetAuthURL(ctx, state, codeChallenge, nonce, callbackURL)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get OAuth auth URL")
		http.Error(w, "Failed to initiate OAuth", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, authURL, http.StatusFound)
}

// GetEnabledOAuthProviders returns the list of enabled OAuth providers for display.
func (h *Handler) GetEnabledOAuthProviders() []string {
	if h.oauthProviders == nil {
		return nil
	}

	providers := make([]string, 0, len(h.oauthProviders))
	for name := range h.oauthProviders {
		providers = append(providers, name)
	}
	return providers
}
