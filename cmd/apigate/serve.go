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
  - Connect to the database
  - Start proxying requests to the upstream API
  - Apply authentication, rate limiting, and usage metering

Examples:
  apigate serve
  apigate serve --config /etc/apigate/config.yaml
  apigate serve --no-hot-reload`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().BoolVar(&hotReload, "hot-reload", true, "enable hot reload of configuration")
}

func runServe(cmd *cobra.Command, args []string) error {
	// Check if config exists
	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		fmt.Println("No configuration found.")
		fmt.Println()
		fmt.Printf("Run 'apigate init' to create %s\n", cfgFile)
		fmt.Println("Or specify a config file with --config")
		return nil
	}

	// Create application
	var app *bootstrap.App
	var err error

	if hotReload {
		app, err = bootstrap.NewWithHotReload(cfgFile)
	} else {
		cfg, loadErr := config.Load(cfgFile)
		if loadErr != nil {
			return fmt.Errorf("error loading config: %w", loadErr)
		}
		app, err = bootstrap.New(cfg)
	}

	if err != nil {
		return fmt.Errorf("error initializing: %w", err)
	}

	// Run (blocks until shutdown)
	return app.Run()
}
