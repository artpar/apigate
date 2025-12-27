// Package config provides configuration loading and hot reload.
package config

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"
)

// Holder provides thread-safe access to configuration with hot reload support.
type Holder struct {
	mu       sync.RWMutex
	config   *Config
	path     string
	logger   zerolog.Logger
	watcher  *fsnotify.Watcher
	onChange []func(*Config)
	stopCh   chan struct{}
}

// NewHolder creates a new config holder and loads the initial configuration.
func NewHolder(path string, logger zerolog.Logger) (*Holder, error) {
	cfg, err := Load(path)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("absolute path: %w", err)
	}

	h := &Holder{
		config: cfg,
		path:   absPath,
		logger: logger,
		stopCh: make(chan struct{}),
	}

	return h, nil
}

// Get returns the current configuration (thread-safe).
func (h *Holder) Get() *Config {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.config
}

// Reload reloads the configuration from disk.
// Returns error if loading fails (keeps old config).
func (h *Holder) Reload() error {
	h.logger.Info().Str("path", h.path).Msg("reloading configuration")

	newCfg, err := Load(h.path)
	if err != nil {
		h.logger.Error().Err(err).Msg("config reload failed, keeping old config")
		return fmt.Errorf("reload config: %w", err)
	}

	h.mu.Lock()
	oldCfg := h.config
	h.config = newCfg
	h.mu.Unlock()

	// Log what changed
	h.logChanges(oldCfg, newCfg)

	// Notify listeners
	for _, fn := range h.onChange {
		fn(newCfg)
	}

	h.logger.Info().Msg("configuration reloaded successfully")
	return nil
}

// OnChange registers a callback to be called when config changes.
func (h *Holder) OnChange(fn func(*Config)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onChange = append(h.onChange, fn)
}

// WatchFile starts watching the config file for changes.
// Changes trigger automatic reload.
func (h *Holder) WatchFile() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	h.watcher = watcher

	// Watch the directory (more reliable for editors that do atomic saves)
	dir := filepath.Dir(h.path)
	if err := watcher.Add(dir); err != nil {
		watcher.Close()
		return fmt.Errorf("watch directory: %w", err)
	}

	go h.watchLoop()

	h.logger.Info().Str("path", h.path).Msg("watching config file for changes")
	return nil
}

// WatchSignals starts listening for SIGHUP to trigger reload.
func (h *Holder) WatchSignals() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP)

	go func() {
		for {
			select {
			case <-sigCh:
				h.logger.Info().Msg("received SIGHUP, reloading config")
				if err := h.Reload(); err != nil {
					h.logger.Error().Err(err).Msg("SIGHUP reload failed")
				}
			case <-h.stopCh:
				signal.Stop(sigCh)
				return
			}
		}
	}()

	h.logger.Info().Msg("listening for SIGHUP to reload config")
}

// Stop stops watching for file changes and signals.
func (h *Holder) Stop() {
	close(h.stopCh)
	if h.watcher != nil {
		h.watcher.Close()
	}
}

func (h *Holder) watchLoop() {
	filename := filepath.Base(h.path)

	for {
		select {
		case event, ok := <-h.watcher.Events:
			if !ok {
				return
			}

			// Only react to our config file
			if filepath.Base(event.Name) != filename {
				continue
			}

			// React to write or create (atomic save = create)
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				h.logger.Debug().
					Str("event", event.Op.String()).
					Str("file", event.Name).
					Msg("config file changed")

				if err := h.Reload(); err != nil {
					h.logger.Error().Err(err).Msg("file watch reload failed")
				}
			}

		case err, ok := <-h.watcher.Errors:
			if !ok {
				return
			}
			h.logger.Error().Err(err).Msg("file watcher error")

		case <-h.stopCh:
			return
		}
	}
}

func (h *Holder) logChanges(old, new *Config) {
	// Log significant changes
	if old.Logging.Level != new.Logging.Level {
		h.logger.Info().
			Str("old", old.Logging.Level).
			Str("new", new.Logging.Level).
			Msg("log level changed")
	}

	if len(old.Plans) != len(new.Plans) {
		h.logger.Info().
			Int("old", len(old.Plans)).
			Int("new", len(new.Plans)).
			Msg("plans count changed")
	}

	if old.RateLimit.BurstTokens != new.RateLimit.BurstTokens {
		h.logger.Info().
			Int("old", old.RateLimit.BurstTokens).
			Int("new", new.RateLimit.BurstTokens).
			Msg("burst tokens changed")
	}

	if len(old.Endpoints) != len(new.Endpoints) {
		h.logger.Info().
			Int("old", len(old.Endpoints)).
			Int("new", len(new.Endpoints)).
			Msg("endpoints count changed")
	}
}

// ReloadableFields returns which fields can be changed without restart.
func ReloadableFields() []string {
	return []string{
		"plans",
		"endpoints",
		"rate_limit.burst_tokens",
		"rate_limit.window_secs",
		"logging.level",
		"logging.format",
	}
}

// NonReloadableFields returns which fields require a restart.
func NonReloadableFields() []string {
	return []string{
		"server.host",
		"server.port",
		"upstream.url",
		"database.dsn",
		"auth.mode",
	}
}
