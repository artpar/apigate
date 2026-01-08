// Package bootstrap provides module hooks for business logic.
// Hooks run before/after actions to handle things like:
// - API key generation (before create)
// - Password hashing (before create/update)
// - Encryption (before create/update)
// - Router reload (after create/update/delete)
package bootstrap

import (
	"context"

	"github.com/artpar/apigate/adapters/hasher"
	"github.com/artpar/apigate/core/runtime"
	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/ports"
	"github.com/rs/zerolog"
)

// RouterReloader is an interface for components that can reload routing tables.
// This is implemented by the route/transform services.
type RouterReloader interface {
	Reload(ctx context.Context) error
}

// PlanReloader is an interface for components that can reload plan configurations.
// This is implemented by the bootstrap App to reload proxy service plans.
type PlanReloader interface {
	ReloadPlans(ctx context.Context) error
}

// routerReloader holds the router reloader instance (set after bootstrap).
var routerReloader RouterReloader

// planReloader holds the plan reloader instance (set after bootstrap).
var planReloader PlanReloader

// planStore holds the plan store instance for the clear_other_defaults function.
var planStore ports.PlanStore

// emailSender holds the email sender instance for email-related hooks.
var emailSender ports.EmailSender

// SetRouterReloader sets the router reloader for the reload_router function.
// This should be called after the route service is initialized.
func SetRouterReloader(r RouterReloader) {
	routerReloader = r
}

// SetPlanReloader sets the plan reloader for the reload_plans function.
// This should be called after the app is initialized.
func SetPlanReloader(r PlanReloader) {
	planReloader = r
}

// SetPlanStore sets the plan store for the clear_other_defaults function.
// This should be called after the plan store is initialized.
func SetPlanStore(s ports.PlanStore) {
	planStore = s
}

// SetEmailSender sets the email sender for email-related hooks.
// This should be called after the email sender is initialized.
func SetEmailSender(s ports.EmailSender) {
	emailSender = s
}

// TriggerPlanReload triggers a reload of plans in the proxy service.
// This can be called from other packages (like web setup) when plans are created directly.
func TriggerPlanReload(ctx context.Context) error {
	if planReloader == nil {
		return nil
	}
	return planReloader.ReloadPlans(ctx)
}

// RegisterHooks registers all module hooks with the runtime.
// This centralizes business logic that applies to module actions.
func RegisterHooks(rt *runtime.Runtime, logger zerolog.Logger) {
	// Register built-in functions for "call:" hooks
	registerBuiltinFunctions(rt, logger)

	// API Key module: generate key and hash before create
	rt.OnHook("api_key", "create", "before", apiKeyBeforeCreate(logger))

	// User module: hash password before set_password action
	rt.OnHook("user", "set_password", "before", userBeforeSetPassword(logger))

	logger.Info().Msg("module hooks registered")
}

// registerBuiltinFunctions registers functions that can be called via "call:" hooks.
func registerBuiltinFunctions(rt *runtime.Runtime, logger zerolog.Logger) {
	// reload_router - reloads the routing table after route/upstream changes
	rt.RegisterFunction("reload_router", func(ctx context.Context, event runtime.HookEvent) error {
		logger.Info().
			Str("module", event.Module).
			Str("action", event.Action).
			Msg("reload_router hook triggered")
		if routerReloader == nil {
			return nil
		}
		return routerReloader.Reload(ctx)
	})

	// send_verification_email - sends email verification after user creation
	rt.RegisterFunction("send_verification_email", func(ctx context.Context, event runtime.HookEvent) error {
		if emailSender == nil {
			logger.Debug().Msg("send_verification_email: email sender not set, skipping")
			return nil
		}

		email, _ := event.Data["email"].(string)
		if email == "" {
			logger.Debug().Msg("send_verification_email: no email in event data")
			return nil
		}

		name, _ := event.Data["name"].(string)
		token, _ := event.Meta["verification_token"].(string)

		if token == "" {
			logger.Debug().Str("email", email).Msg("send_verification_email: no token provided, skipping")
			return nil
		}

		if err := emailSender.SendVerification(ctx, email, name, token); err != nil {
			logger.Error().Err(err).Str("email", email).Msg("failed to send verification email")
			return err
		}

		logger.Info().Str("email", email).Msg("verification email sent")
		return nil
	})

	// clear_other_defaults - clears is_default on other plans when setting default
	rt.RegisterFunction("clear_other_defaults", func(ctx context.Context, event runtime.HookEvent) error {
		if planStore == nil {
			logger.Warn().Msg("clear_other_defaults: plan store not set, skipping")
			return nil
		}

		// Get the ID of the plan that's being set as default
		planID, ok := event.Data["id"].(string)
		if !ok || planID == "" {
			logger.Debug().Msg("clear_other_defaults: no plan ID in event data")
			return nil
		}

		// Clear is_default on all other plans
		if err := planStore.ClearOtherDefaults(ctx, planID); err != nil {
			logger.Error().Err(err).Str("except_id", planID).Msg("failed to clear other defaults")
			return err
		}

		logger.Info().Str("except_id", planID).Msg("cleared is_default on other plans")
		return nil
	})

	// sync_to_stripe - syncs plan to Stripe after create/update
	rt.RegisterFunction("sync_to_stripe", func(ctx context.Context, event runtime.HookEvent) error {
		logger.Debug().
			Str("module", event.Module).
			Msg("sync_to_stripe called (not yet implemented)")
		// TODO: Integrate with payment adapter
		return nil
	})

	// reload_plans - reloads plans into proxy service after plan changes
	rt.RegisterFunction("reload_plans", func(ctx context.Context, event runtime.HookEvent) error {
		logger.Info().
			Str("module", event.Module).
			Str("action", event.Action).
			Msg("reload_plans hook triggered")
		if planReloader == nil {
			logger.Warn().Msg("plan reloader not set, skipping reload")
			return nil
		}
		return planReloader.ReloadPlans(ctx)
	})

	logger.Debug().
		Int("count", 5).
		Msg("built-in functions registered")
}

// apiKeyBeforeCreate generates a secure API key and hash.
// The raw key is stored in Meta for one-time display to the user.
func apiKeyBeforeCreate(logger zerolog.Logger) runtime.HookHandler {
	return func(ctx context.Context, event runtime.HookEvent) error {
		// Generate the API key with "ak_" prefix
		rawKey, k := key.Generate("ak_")

		// Set the hash in the data (will be stored in database)
		// Keep as []byte for proper BLOB storage - bcrypt hashes are ASCII-safe
		event.Data["hash"] = k.Hash

		// Set the prefix for lookup
		event.Data["prefix"] = k.Prefix

		// Store the raw key in Meta for one-time display
		// This is the ONLY time the user sees the raw key
		event.Meta["raw_key"] = rawKey

		logger.Info().
			Str("prefix", k.Prefix).
			Msg("generated API key")

		return nil
	}
}

// userBeforeSetPassword hashes the password before storing.
// Takes the "password" input and converts it to "password_hash".
func userBeforeSetPassword(logger zerolog.Logger) runtime.HookHandler {
	h := hasher.NewBcrypt(10) // Cost of 10 is a good balance

	return func(ctx context.Context, event runtime.HookEvent) error {
		password, ok := event.Data["password"].(string)
		if !ok || password == "" {
			return nil // Let validation handle missing password
		}

		// Hash the password
		hash, err := h.Hash(password)
		if err != nil {
			logger.Error().Err(err).Msg("failed to hash password")
			return err
		}

		// Remove the plaintext password and set the hash
		delete(event.Data, "password")
		event.Data["password_hash"] = string(hash)

		logger.Debug().Msg("password hashed for set_password action")
		return nil
	}
}
