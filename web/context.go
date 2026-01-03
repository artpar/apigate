package web

import (
	"context"

	"github.com/artpar/apigate/adapters/auth"
	"github.com/artpar/apigate/core/terminology"
)

type ctxKey string

const claimsKey ctxKey = "claims"

// withClaims adds JWT claims to the context.
func withClaims(ctx context.Context, claims *auth.Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// getClaims retrieves JWT claims from context.
func getClaims(ctx context.Context) *auth.Claims {
	claims, ok := ctx.Value(claimsKey).(*auth.Claims)
	if !ok {
		return nil
	}
	return claims
}

// PageData holds common data for all pages.
type PageData struct {
	Title       string
	User        *UserInfo
	CurrentPath string
	Flash       *FlashMessage
	Config      *ConfigInfo
	Labels      terminology.Labels // UI labels for metering units
}

// UserInfo represents the logged-in user.
type UserInfo struct {
	ID    string
	Email string
	Role  string
}

// FlashMessage represents a one-time notification.
type FlashMessage struct {
	Type    string // "success", "error", "warning", "info"
	Message string
}

// ConfigInfo represents config for templates.
type ConfigInfo struct {
	UpstreamURL string
	Version     string
}

// newPageData creates base page data from request context.
func (h *Handler) newPageData(ctx context.Context, title string) PageData {
	data := PageData{
		Title: title,
		Config: &ConfigInfo{
			UpstreamURL: h.appSettings.UpstreamURL,
			Version:     "dev",
		},
		Labels: terminology.Default(),
	}

	// Load terminology labels from settings
	if h.settings != nil {
		if setting, err := h.settings.Get(ctx, "metering.unit"); err == nil && setting.Value != "" {
			data.Labels = terminology.ForUnit(setting.Value)
		}
	}

	if claims := getClaims(ctx); claims != nil {
		data.User = &UserInfo{
			ID:    claims.UserID,
			Email: claims.Email,
			Role:  claims.Role,
		}
	}

	return data
}
