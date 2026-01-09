package web

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/artpar/apigate/ports"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// InvitesPage renders the admin invites management page.
func (h *Handler) InvitesPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	invites, err := h.invites.List(ctx, 100, 0)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list invites")
		invites = []ports.AdminInvite{}
	}

	// Get creator emails for display
	type InviteWithCreator struct {
		ports.AdminInvite
		CreatorEmail string
		IsExpired    bool
		IsUsed       bool
	}

	var invitesWithCreator []InviteWithCreator
	for _, inv := range invites {
		iwc := InviteWithCreator{
			AdminInvite: inv,
			IsExpired:   inv.ExpiresAt.Before(time.Now()),
			IsUsed:      inv.UsedAt != nil,
		}
		if user, err := h.users.Get(ctx, inv.CreatedBy); err == nil {
			iwc.CreatorEmail = user.Email
		}
		invitesWithCreator = append(invitesWithCreator, iwc)
	}

	data := struct {
		PageData
		Invites []InviteWithCreator
		Success string
		Error   string
	}{
		PageData: h.newPageData(ctx, "Admin Invites"),
		Invites:  invitesWithCreator,
		Success:  r.URL.Query().Get("success"),
		Error:    r.URL.Query().Get("error"),
	}

	h.render(w, "invites", data)
}

// InviteCreate creates a new admin invite.
func (h *Handler) InviteCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := getClaims(ctx)

	email := r.FormValue("email")
	if email == "" {
		http.Redirect(w, r, "/invites?error=email_required", http.StatusFound)
		return
	}

	// Check if user already exists
	if _, err := h.users.GetByEmail(ctx, email); err == nil {
		http.Redirect(w, r, "/invites?error=user_exists", http.StatusFound)
		return
	}

	// Generate random token (32 bytes = 64 hex chars)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		h.logger.Error().Err(err).Msg("Failed to generate invite token")
		http.Redirect(w, r, "/invites?error=internal", http.StatusFound)
		return
	}
	rawToken := hex.EncodeToString(tokenBytes)

	// Hash the token for storage
	hash := sha256.Sum256([]byte(rawToken))

	invite := ports.AdminInvite{
		ID:        uuid.New().String(),
		Email:     email,
		TokenHash: hash[:],
		CreatedBy: claims.UserID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(48 * time.Hour), // 48 hour expiry
	}

	if err := h.invites.Create(ctx, invite); err != nil {
		h.logger.Error().Err(err).Msg("Failed to create invite")
		http.Redirect(w, r, "/invites?error=internal", http.StatusFound)
		return
	}

	// Send invite email if email sender is configured
	if h.emailSender != nil {
		baseURL := getBaseURL(r)
		inviteURL := baseURL + "/admin/register/" + rawToken

		err := h.emailSender.Send(ctx, ports.EmailMessage{
			To:      email,
			Subject: "You're invited to APIGate",
			HTMLBody: `<h2>You've been invited to APIGate</h2>
<p>An administrator has invited you to join APIGate as an admin.</p>
<p><a href="` + inviteURL + `" style="display:inline-block;background:#4f46e5;color:white;padding:12px 24px;text-decoration:none;border-radius:4px;">Accept Invitation</a></p>
<p>This invitation expires in 48 hours.</p>
<p style="color:#666;font-size:12px;">If you didn't expect this invitation, you can ignore this email.</p>`,
			TextBody: "You've been invited to APIGate as an admin.\n\nAccept invitation: " + inviteURL + "\n\nThis invitation expires in 48 hours.",
		})
		if err != nil {
			h.logger.Error().Err(err).Str("email", email).Msg("Failed to send invite email")
		}
	}

	http.Redirect(w, r, "/invites?success=created", http.StatusFound)
}

// InviteDelete deletes an admin invite.
func (h *Handler) InviteDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	if err := h.invites.Delete(ctx, id); err != nil {
		h.logger.Error().Err(err).Str("id", id).Msg("Failed to delete invite")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// For HTMX requests, return empty to remove the row
	if r.Header.Get("HX-Request") == "true" {
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/invites?success=deleted", http.StatusFound)
}

// AdminRegisterPage renders the admin registration form for invited users.
func (h *Handler) AdminRegisterPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token := chi.URLParam(r, "token")

	// Hash token to look up invite
	hash := sha256.Sum256([]byte(token))
	invite, err := h.invites.GetByTokenHash(ctx, hash[:])
	if err != nil {
		h.renderAdminRegisterError(w, r, "Invalid or expired invitation link.", "")
		return
	}

	// Check if already used
	if invite.UsedAt != nil {
		h.renderAdminRegisterError(w, r, "This invitation has already been used.", "")
		return
	}

	// Check if expired
	if invite.ExpiresAt.Before(time.Now()) {
		h.renderAdminRegisterError(w, r, "This invitation has expired. Please request a new one.", "")
		return
	}

	data := struct {
		PageData
		Email string
		Token string
		Error string
	}{
		PageData: h.newPageData(ctx, "Complete Registration"),
		Email:    invite.Email,
		Token:    token,
	}

	h.render(w, "admin_register", data)
}

// AdminRegisterSubmit handles admin registration from invite.
func (h *Handler) AdminRegisterSubmit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token := chi.URLParam(r, "token")

	// Validate invite
	hash := sha256.Sum256([]byte(token))
	invite, err := h.invites.GetByTokenHash(ctx, hash[:])
	if err != nil {
		h.renderAdminRegisterError(w, r, "Invalid or expired invitation link.", "")
		return
	}

	if invite.UsedAt != nil {
		h.renderAdminRegisterError(w, r, "This invitation has already been used.", "")
		return
	}

	if invite.ExpiresAt.Before(time.Now()) {
		h.renderAdminRegisterError(w, r, "This invitation has expired. Please request a new one.", "")
		return
	}

	// Validate form
	name := r.FormValue("name")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	if name == "" {
		h.renderAdminRegisterFormError(w, r, "Name is required.", invite.Email, token)
		return
	}

	if len(password) < 8 {
		h.renderAdminRegisterFormError(w, r, "Password must be at least 8 characters.", invite.Email, token)
		return
	}

	if password != confirmPassword {
		h.renderAdminRegisterFormError(w, r, "Passwords do not match.", invite.Email, token)
		return
	}

	// Hash password
	passwordHash, err := h.hasher.Hash(password)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to hash password")
		h.renderAdminRegisterFormError(w, r, "Registration failed. Please try again.", invite.Email, token)
		return
	}

	// Create user
	user := ports.User{
		ID:           uuid.New().String(),
		Email:        invite.Email,
		Name:         name,
		PasswordHash: passwordHash,
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := h.users.Create(ctx, user); err != nil {
		h.logger.Error().Err(err).Msg("Failed to create user")
		h.renderAdminRegisterFormError(w, r, "Registration failed. Please try again.", invite.Email, token)
		return
	}

	// Mark invite as used
	if err := h.invites.MarkUsed(ctx, invite.ID, time.Now()); err != nil {
		h.logger.Error().Err(err).Msg("Failed to mark invite as used")
	}

	// Redirect to login with success message
	http.Redirect(w, r, "/login?success=registered", http.StatusFound)
}

func (h *Handler) renderAdminRegisterError(w http.ResponseWriter, r *http.Request, errMsg, email string) {
	data := struct {
		PageData
		Email      string
		Token      string
		Error      string
		LinkError  bool
	}{
		PageData:  h.newPageData(r.Context(), "Registration"),
		Email:     email,
		Error:     errMsg,
		LinkError: true,
	}
	h.render(w, "admin_register", data)
}

func (h *Handler) renderAdminRegisterFormError(w http.ResponseWriter, r *http.Request, errMsg, email, token string) {
	data := struct {
		PageData
		Email     string
		Token     string
		Error     string
		LinkError bool
	}{
		PageData: h.newPageData(r.Context(), "Complete Registration"),
		Email:    email,
		Token:    token,
		Error:    errMsg,
	}
	h.render(w, "admin_register", data)
}

// getBaseURL returns the base URL from the request.
func getBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
