package main

import (
	"context"
	"os"

	"github.com/artpar/apigate/adapters/sqlite"
	"github.com/artpar/apigate/bootstrap"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var modCmd *cobra.Command
var moduleRuntime *bootstrap.ModuleRuntime

func init() {
	modCmd = &cobra.Command{
		Use:   "mod",
		Short: "Module-based management commands",
		Long: `Access declarative module CRUD operations.

These commands use the module system to manage entities:
  apigate mod users list
  apigate mod plans get free
  apigate mod upstreams create --name "API" --base_url "https://api.example.com"

Available subcommands are generated from loaded modules.`,
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			// Flush and close analytics when command completes
			if moduleRuntime != nil {
				moduleRuntime.Stop(context.Background())
			}
		},
	}
	rootCmd.AddCommand(modCmd)

	// Try to initialize modules at startup for help text
	// This is best-effort - errors are silently ignored
	tryInitModules()
}

func tryInitModules() {
	// Setup quiet logger
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)

	// Get database path
	dsn := os.Getenv("APIGATE_DATABASE_DSN")
	if dsn == "" {
		dsn = "apigate.db"
	}

	// Check if database exists - if not, skip silently
	if _, err := os.Stat(dsn); os.IsNotExist(err) {
		return
	}

	// Open database
	db, err := sqlite.Open(dsn)
	if err != nil {
		return
	}

	// Create module runtime with modCmd as root
	mr, err := bootstrap.NewModuleRuntime(db.DB, modCmd, logger, bootstrap.ModuleConfig{})
	if err != nil {
		db.Close()
		return
	}

	// Load modules
	ctx := context.Background()
	if err := mr.LoadModules(ctx, bootstrap.ModuleConfig{
		EmbeddedModules: bootstrap.CoreModules(),
	}); err != nil {
		db.Close()
		return
	}

	moduleRuntime = mr
}
