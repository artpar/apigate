package admin

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"
)

// DoctorResponse represents the system health check response.
type DoctorResponse struct {
	Status     string            `json:"status"` // "healthy", "degraded", "unhealthy"
	Timestamp  string            `json:"timestamp"`
	Version    string            `json:"version"`
	Checks     []HealthCheck     `json:"checks"`
	System     SystemInfo        `json:"system"`
	Statistics StatisticsInfo    `json:"statistics"`
}

// HealthCheck represents a single health check result.
type HealthCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "pass", "warn", "fail"
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}

// SystemInfo represents system information.
type SystemInfo struct {
	GoVersion    string `json:"go_version"`
	NumCPU       int    `json:"num_cpu"`
	NumGoroutine int    `json:"num_goroutine"`
	MemAlloc     string `json:"mem_alloc"`
	MemSys       string `json:"mem_sys"`
	Uptime       string `json:"uptime,omitempty"`
}

// StatisticsInfo represents usage statistics.
type StatisticsInfo struct {
	TotalUsers    int   `json:"total_users"`
	TotalKeys     int   `json:"total_keys"`
	ActiveSessions int  `json:"active_sessions"`
}

var startTime = time.Now()

// Doctor performs a comprehensive system health check.
//
//	@Summary		System health check
//	@Description	Comprehensive health check with diagnostics
//	@Tags			Admin - System
//	@Produce		json
//	@Success		200	{object}	DoctorResponse	"Health check results"
//	@Security		AdminAuth
//	@Router			/admin/doctor [get]
func (h *Handler) Doctor(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	response := DoctorResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   "dev",
		Checks:    []HealthCheck{},
	}

	// 1. Database check
	dbCheck := h.checkDatabase(ctx)
	response.Checks = append(response.Checks, dbCheck)

	// 2. Upstream check
	upstreamCheck := h.checkUpstream(ctx)
	response.Checks = append(response.Checks, upstreamCheck)

	// 3. Config validation
	configCheck := h.checkConfig()
	response.Checks = append(response.Checks, configCheck)

	// 4. Memory check
	memCheck := h.checkMemory()
	response.Checks = append(response.Checks, memCheck)

	// Determine overall status
	hasWarn := false
	hasFail := false
	for _, check := range response.Checks {
		switch check.Status {
		case "warn":
			hasWarn = true
		case "fail":
			hasFail = true
		}
	}

	if hasFail {
		response.Status = "unhealthy"
	} else if hasWarn {
		response.Status = "degraded"
	}

	// System info
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	response.System = SystemInfo{
		GoVersion:    runtime.Version(),
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
		MemAlloc:     formatBytes(memStats.Alloc),
		MemSys:       formatBytes(memStats.Sys),
		Uptime:       time.Since(startTime).Round(time.Second).String(),
	}

	// Statistics
	userCount, _ := h.users.Count(ctx)

	// Count total keys
	totalKeys := 0
	users, _ := h.users.List(ctx, 1000, 0)
	for _, u := range users {
		keys, _ := h.keys.ListByUser(ctx, u.ID)
		totalKeys += len(keys)
	}

	response.Statistics = StatisticsInfo{
		TotalUsers:     userCount,
		TotalKeys:      totalKeys,
		ActiveSessions: len(h.sessions.sessions),
	}

	// Set appropriate status code
	statusCode := http.StatusOK
	if response.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	writeJSON(w, statusCode, response)
}

func (h *Handler) checkDatabase(ctx context.Context) HealthCheck {
	check := HealthCheck{
		Name:   "database",
		Status: "pass",
	}

	start := time.Now()

	// Try to count users as a simple query
	_, err := h.users.Count(ctx)
	check.Latency = time.Since(start).String()

	if err != nil {
		check.Status = "fail"
		check.Message = fmt.Sprintf("Database query failed: %v", err)
	} else {
		check.Message = "Database connection healthy"
	}

	return check
}

func (h *Handler) checkUpstream(ctx context.Context) HealthCheck {
	check := HealthCheck{
		Name:   "upstream",
		Status: "pass",
	}

	// We don't have direct access to upstream here, so just report config
	if h.config.Upstream.URL == "" {
		check.Status = "fail"
		check.Message = "No upstream URL configured"
	} else {
		check.Message = fmt.Sprintf("Upstream configured: %s", h.config.Upstream.URL)
	}

	return check
}

func (h *Handler) checkConfig() HealthCheck {
	check := HealthCheck{
		Name:   "config",
		Status: "pass",
	}

	issues := []string{}

	// Check for common config issues
	if h.config.Upstream.URL == "" {
		issues = append(issues, "upstream.url not set")
	}

	if len(h.config.Plans) == 0 {
		issues = append(issues, "no plans configured")
	}

	if !h.config.RateLimit.Enabled {
		issues = append(issues, "rate limiting disabled")
	}

	if len(issues) > 0 {
		check.Status = "warn"
		check.Message = fmt.Sprintf("Config warnings: %v", issues)
	} else {
		check.Message = "Configuration valid"
	}

	return check
}

func (h *Handler) checkMemory() HealthCheck {
	check := HealthCheck{
		Name:   "memory",
		Status: "pass",
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Warn if using more than 500MB
	if memStats.Alloc > 500*1024*1024 {
		check.Status = "warn"
		check.Message = fmt.Sprintf("High memory usage: %s", formatBytes(memStats.Alloc))
	} else {
		check.Message = fmt.Sprintf("Memory usage: %s", formatBytes(memStats.Alloc))
	}

	return check
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
