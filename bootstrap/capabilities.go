// Package bootstrap - Capability system initialization.
package bootstrap

import (
	"github.com/artpar/apigate/core/capability"
	"github.com/artpar/apigate/core/capability/adapters"
	"github.com/artpar/apigate/domain/settings"
	"github.com/artpar/apigate/ports"
	"github.com/rs/zerolog"
)

// CapabilityConfig provides configuration for capability system initialization.
type CapabilityConfig struct {
	Settings settings.Settings
	Logger   zerolog.Logger

	// Optional pre-configured providers
	Cache       ports.CacheProvider
	Payment     ports.PaymentProvider
	Email       ports.EmailSender
	Hasher      ports.Hasher
}

// NewCapabilityContainer creates and initializes the capability container.
func NewCapabilityContainer(cfg CapabilityConfig) (*capability.Container, error) {
	container := capability.NewContainer()

	// Register cache provider
	if err := registerCacheProvider(container, cfg); err != nil {
		cfg.Logger.Warn().Err(err).Msg("failed to register cache provider")
	}

	// Register hasher provider
	if err := registerHasherProvider(container, cfg); err != nil {
		cfg.Logger.Warn().Err(err).Msg("failed to register hasher provider")
	}

	// Register email provider
	if err := registerEmailProvider(container, cfg); err != nil {
		cfg.Logger.Warn().Err(err).Msg("failed to register email provider")
	}

	// Register payment provider
	if err := registerPaymentProvider(container, cfg); err != nil {
		cfg.Logger.Warn().Err(err).Msg("failed to register payment provider")
	}

	// Register storage provider (memory for now)
	if err := registerStorageProvider(container, cfg); err != nil {
		cfg.Logger.Warn().Err(err).Msg("failed to register storage provider")
	}

	// Register queue provider (memory for now)
	if err := registerQueueProvider(container, cfg); err != nil {
		cfg.Logger.Warn().Err(err).Msg("failed to register queue provider")
	}

	// Register notification provider
	if err := registerNotificationProvider(container, cfg); err != nil {
		cfg.Logger.Warn().Err(err).Msg("failed to register notification provider")
	}

	cfg.Logger.Info().
		Int("count", len(container.ListCapabilities())).
		Msg("capability container initialized")

	return container, nil
}

func registerCacheProvider(container *capability.Container, cfg CapabilityConfig) error {
	var cacheProvider capability.CacheProvider

	if cfg.Cache != nil {
		// Wrap existing cache provider
		cacheProvider = adapters.WrapCache(cfg.Cache)
	} else {
		// Create default in-memory cache
		cacheProvider = adapters.NewMemoryCache("default")
		cfg.Logger.Debug().Msg("using in-memory cache provider")
	}

	return container.RegisterCache("default", cacheProvider, true)
}

func registerHasherProvider(container *capability.Container, cfg CapabilityConfig) error {
	var hasherProvider capability.HasherProvider

	if cfg.Hasher != nil {
		// Wrap existing hasher
		hasherProvider = adapters.WrapHasher("default", cfg.Hasher)
	} else {
		// We don't have a default hasher in adapters, skip if not provided
		cfg.Logger.Debug().Msg("no hasher provider configured")
		return nil
	}

	return container.RegisterHasher("default", hasherProvider, true)
}

func registerEmailProvider(container *capability.Container, cfg CapabilityConfig) error {
	if cfg.Email == nil {
		cfg.Logger.Debug().Msg("no email provider configured")
		return nil
	}

	emailProvider := adapters.WrapEmail("default", cfg.Email)
	return container.RegisterEmail("default", emailProvider, true)
}

func registerPaymentProvider(container *capability.Container, cfg CapabilityConfig) error {
	if cfg.Payment == nil {
		cfg.Logger.Debug().Msg("no payment provider configured")
		return nil
	}

	paymentProvider := adapters.WrapPayment(cfg.Payment)
	return container.RegisterPayment("default", paymentProvider, true)
}

func registerStorageProvider(container *capability.Container, cfg CapabilityConfig) error {
	// Default to in-memory storage
	storageProvider := adapters.NewMemoryStorage("default")
	cfg.Logger.Debug().Msg("using in-memory storage provider")
	return container.RegisterStorage("default", storageProvider, true)
}

func registerQueueProvider(container *capability.Container, cfg CapabilityConfig) error {
	// Default to in-memory queue
	queueProvider := adapters.NewMemoryQueue("default")
	cfg.Logger.Debug().Msg("using in-memory queue provider")
	return container.RegisterQueue("default", queueProvider, true)
}

func registerNotificationProvider(container *capability.Container, cfg CapabilityConfig) error {
	// Default to console notification (logs notifications)
	notifProvider := adapters.NewConsoleNotification("default")
	cfg.Logger.Debug().Msg("using console notification provider")
	return container.RegisterNotification("default", notifProvider, true)
}
