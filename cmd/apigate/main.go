// Package main is the entry point for APIGate.
//
//	@title						APIGate - API Monetization Proxy
//	@version					1.0
//	@description				Self-hosted API monetization solution with authentication, rate limiting, usage metering, and billing.
//	@termsOfService				https://github.com/artpar/apigate
//
//	@contact.name				APIGate Support
//	@contact.url				https://github.com/artpar/apigate/issues
//
//	@license.name				MIT
//	@license.url				https://opensource.org/licenses/MIT
//
//	@host						localhost:8080
//	@BasePath					/
//
//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						X-API-Key
//	@description				API key for authentication
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Bearer token authentication (format: "Bearer {api_key}")
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/artpar/apigate/bootstrap"
	"github.com/artpar/apigate/config"
)

var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	// CLI flags
	configPath := flag.String("config", "apigate.yaml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version and exit")
	validate := flag.Bool("validate", false, "Validate configuration and exit")
	hotReload := flag.Bool("hot-reload", true, "Enable hot reload of configuration")
	flag.Parse()

	// Version command
	if *showVersion {
		fmt.Printf("apigate %s (commit: %s, built: %s)\n", version, commit, buildDate)
		os.Exit(0)
	}

	// Validate only mode
	if *validate {
		cfg, err := config.Load(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Configuration invalid: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Configuration valid\n")
		fmt.Printf("  Upstream: %s\n", cfg.Upstream.URL)
		fmt.Printf("  Auth mode: %s\n", cfg.Auth.Mode)
		fmt.Printf("  Plans: %d\n", len(cfg.Plans))
		os.Exit(0)
	}

	// Create application
	var app *bootstrap.App
	var err error

	if *hotReload {
		// Hot reload enabled - use NewWithHotReload
		app, err = bootstrap.NewWithHotReload(*configPath)
	} else {
		// Hot reload disabled - use legacy New
		cfg, loadErr := config.Load(*configPath)
		if loadErr != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", loadErr)
			os.Exit(1)
		}
		app, err = bootstrap.New(cfg)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing: %v\n", err)
		os.Exit(1)
	}

	// Run (blocks until shutdown)
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
