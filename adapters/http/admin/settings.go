package admin

import (
	"net/http"
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
	cfg := h.config

	response := SettingsResponse{
		Server: ServerSettings{
			Host:         cfg.Server.Host,
			Port:         cfg.Server.Port,
			ReadTimeout:  cfg.Server.ReadTimeout.String(),
			WriteTimeout: cfg.Server.WriteTimeout.String(),
		},
		Upstream: UpstreamSettings{
			URL:     cfg.Upstream.URL,
			Timeout: cfg.Upstream.Timeout.String(),
		},
		Auth: AuthSettings{
			Mode:      cfg.Auth.Mode,
			KeyPrefix: cfg.Auth.KeyPrefix,
		},
		RateLimit: RateLimitSettings{
			Enabled:     cfg.RateLimit.Enabled,
			BurstTokens: cfg.RateLimit.BurstTokens,
			WindowSecs:  cfg.RateLimit.WindowSecs,
		},
		Usage: UsageSettings{
			Mode:          cfg.Usage.Mode,
			BatchSize:     cfg.Usage.BatchSize,
			FlushInterval: cfg.Usage.FlushInterval.String(),
		},
		Billing: BillingSettings{
			Mode: cfg.Billing.Mode,
		},
		Logging: LoggingSettings{
			Level:  cfg.Logging.Level,
			Format: cfg.Logging.Format,
		},
		Metrics: MetricsSettings{
			Enabled: cfg.Metrics.Enabled,
			Path:    cfg.Metrics.Path,
		},
		OpenAPI: OpenAPISettings{
			Enabled: cfg.OpenAPI.Enabled,
		},
	}

	writeJSON(w, http.StatusOK, response)
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
func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	// Settings updates require config file changes and reload
	// This will be implemented with the web UI in Phase 3
	writeError(w, http.StatusNotImplemented, "not_implemented",
		"Settings updates via API not yet available. Edit apigate.yaml and send SIGHUP to reload.")
}
