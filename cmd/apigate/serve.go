package main

import (
	"fmt"

	"github.com/artpar/apigate/bootstrap"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the API proxy server",
	Long: `Start the APIGate proxy server.

All configuration is loaded from the database after connection.
Only minimal bootstrap settings come from environment variables:

Environment variables:
  APIGATE_DATABASE_DSN  - Database path (default: apigate.db)
  APIGATE_SERVER_HOST   - Server host (default: from settings or 0.0.0.0)
  APIGATE_SERVER_PORT   - Server port (default: from settings or 8080)
  APIGATE_LOG_LEVEL     - Log level: debug, info, warn, error
  APIGATE_LOG_FORMAT    - Log format: json or console

All other settings are stored in the database and can be configured
via the admin UI or API:
  - Email provider settings (SMTP, etc.)
  - Payment provider settings (Stripe, Paddle, LemonSqueezy)
  - Portal settings
  - Rate limit settings
  - Upstream settings

Examples:
  apigate serve
  APIGATE_DATABASE_DSN=/data/apigate.db apigate serve
  APIGATE_LOG_LEVEL=debug APIGATE_LOG_FORMAT=console apigate serve`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	// Create application (config loaded from database)
	app, err := bootstrap.New()
	if err != nil {
		return fmt.Errorf("error initializing: %w", err)
	}

	// Run (blocks until shutdown)
	return app.Run()
}
