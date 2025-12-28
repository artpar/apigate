package main

import (
	"fmt"
	"os"

	"github.com/artpar/apigate/bootstrap"
	"github.com/artpar/apigate/config"
	"github.com/spf13/cobra"
)

var (
	hotReload bool
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the API proxy server",
	Long: `Start the APIGate proxy server.

The server will:
  - Load configuration from apigate.yaml (or --config)
  - Or load configuration from APIGATE_* environment variables
  - Connect to the database
  - Start proxying requests to the upstream API
  - Apply authentication, rate limiting, and usage metering

Environment variables (for Docker deployments):
  APIGATE_UPSTREAM_URL      - Upstream API URL (required)
  APIGATE_DATABASE_DSN      - Database path (default: apigate.db)
  APIGATE_SERVER_PORT       - Server port (default: 8080)
  APIGATE_AUTH_MODE         - Auth mode: local or remote
  APIGATE_LOG_LEVEL         - Log level: debug, info, warn, error
  APIGATE_ADMIN_EMAIL       - Admin email for first-run bootstrap

Examples:
  apigate serve
  apigate serve --config /etc/apigate/config.yaml
  apigate serve --hot-reload=false

  # Docker (env vars only):
  APIGATE_UPSTREAM_URL=https://api.example.com apigate serve`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().BoolVar(&hotReload, "hot-reload", true, "enable hot reload of configuration")
}

func runServe(cmd *cobra.Command, args []string) error {
	hasConfigFile := false
	if _, err := os.Stat(cfgFile); err == nil {
		hasConfigFile = true
	}

	hasEnvConfig := config.HasEnvConfig()

	// No configuration at all
	if !hasConfigFile && !hasEnvConfig {
		fmt.Println("No configuration found.")
		fmt.Println()
		fmt.Printf("Option 1: Run 'apigate init' to create %s\n", cfgFile)
		fmt.Println("Option 2: Set APIGATE_UPSTREAM_URL environment variable")
		fmt.Println()
		fmt.Println("Example (env vars):")
		fmt.Println("  APIGATE_UPSTREAM_URL=https://api.example.com apigate serve")
		return nil
	}

	// Create application
	var app *bootstrap.App
	var err error

	if hasConfigFile && hotReload {
		// Hot reload only works with config file
		app, err = bootstrap.NewWithHotReload(cfgFile)
	} else {
		// Load config (file with env overrides, or env-only)
		cfg, loadErr := config.LoadWithFallback(cfgFile)
		if loadErr != nil {
			return fmt.Errorf("error loading config: %w", loadErr)
		}

		if !hasConfigFile {
			fmt.Println("Running with environment variables (no config file)")
		}

		app, err = bootstrap.New(cfg)
	}

	if err != nil {
		return fmt.Errorf("error initializing: %w", err)
	}

	// Run (blocks until shutdown)
	return app.Run()
}
