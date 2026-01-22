package admin

import (
	"net/http"

	"github.com/artpar/apigate/pkg/jsonapi"
)

// SettingsResponse represents current settings.
type SettingsResponse struct {
	Server    ServerSettings    `json:"server"`
	Upstream  UpstreamSettings  `json:"upstream"`
	Auth      AuthSettings      `json:"auth"`
	RateLimit RateLimitSettings `json:"rate_limit"`
	Usage     UsageSettings     `json:"usage"`
	Billing   BillingSettings   `json:"billing"`
	Logging   LoggingSettings   `json:"logging"`
	Metrics   MetricsSettings   `json:"metrics"`
	OpenAPI   OpenAPISettings   `json:"openapi"`
}

// ServerSettings represents server configuration.
type ServerSettings struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	ReadTimeout  string `json:"read_timeout"`
	WriteTimeout string `json:"write_timeout"`
}

// UpstreamSettings represents upstream configuration.
type UpstreamSettings struct {
	URL     string `json:"url"`
	Timeout string `json:"timeout"`
}

// AuthSettings represents auth configuration.
type AuthSettings struct {
	Mode      string `json:"mode"`
	KeyPrefix string `json:"key_prefix"`
}

// RateLimitSettings represents rate limit configuration.
type RateLimitSettings struct {
	Enabled     bool `json:"enabled"`
	BurstTokens int  `json:"burst_tokens"`
	WindowSecs  int  `json:"window_secs"`
}

// UsageSettings represents usage tracking configuration.
type UsageSettings struct {
	Mode          string `json:"mode"`
	BatchSize     int    `json:"batch_size"`
	FlushInterval string `json:"flush_interval"`
}

// BillingSettings represents billing configuration.
type BillingSettings struct {
	Mode string `json:"mode"`
}

// LoggingSettings represents logging configuration.
type LoggingSettings struct {
	Level  string `json:"level"`
	Format string `json:"format"`
}

// MetricsSettings represents metrics configuration.
type MetricsSettings struct {
	Enabled bool   `json:"enabled"`
	Path    string `json:"path"`
}

// OpenAPISettings represents OpenAPI configuration.
type OpenAPISettings struct {
	Enabled bool `json:"enabled"`
}

// GetSettings returns current settings.
//
//	@Summary		Get settings
//	@Description	Get current configuration settings
//	@Tags			Admin - Settings
//	@Produce		json
//	@Success		200	{object}	SettingsResponse	"Current settings"
//	@Security		AdminAuth
//	@Router			/admin/settings [get]
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	// Settings are now stored in the database via the settings domain.
	// This endpoint returns placeholder values. Use the web UI for settings management.
	response := SettingsResponse{
		Server: ServerSettings{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  "30s",
			WriteTimeout: "60s",
		},
		Upstream: UpstreamSettings{
			URL:     "(configured via database)",
			Timeout: "30s",
		},
		Auth: AuthSettings{
			Mode:      "local",
			KeyPrefix: "ak_",
		},
		RateLimit: RateLimitSettings{
			Enabled:     true,
			BurstTokens: 5,
			WindowSecs:  60,
		},
		Usage: UsageSettings{
			Mode:          "async",
			BatchSize:     100,
			FlushInterval: "1s",
		},
		Billing: BillingSettings{
			Mode: "none",
		},
		Logging: LoggingSettings{
			Level:  "info",
			Format: "json",
		},
		Metrics: MetricsSettings{
			Enabled: false,
			Path:    "/metrics",
		},
		OpenAPI: OpenAPISettings{
			Enabled: false,
		},
	}

	// Return as JSON:API meta response (settings aren't a typical resource)
	jsonapi.WriteMeta(w, http.StatusOK, jsonapi.Meta{
		"server":     response.Server,
		"upstream":   response.Upstream,
		"auth":       response.Auth,
		"rate_limit": response.RateLimit,
		"usage":      response.Usage,
		"billing":    response.Billing,
		"logging":    response.Logging,
		"metrics":    response.Metrics,
		"openapi":    response.OpenAPI,
	})
}

// UpdateSettings updates settings.
//
//	@Summary		Update settings
//	@Description	Update configuration settings (some require restart)
//	@Tags			Admin - Settings
//	@Accept			json
//	@Produce		json
//	@Param			request	body		map[string]interface{}	true	"Settings to update"
//	@Success		501		{object}	ErrorResponse			"Not yet implemented"
//	@Security		AdminAuth
//	@Router			/admin/settings [put]
//	@Router			/admin/settings [patch]
func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	// Settings are now managed via the database settings domain.
	// Use the web UI settings page for configuration changes.
	jsonapi.WriteError(w, jsonapi.ErrNotImplemented("Settings updates via API"))
}
