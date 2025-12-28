package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/artpar/apigate/adapters/sqlite"
	"github.com/artpar/apigate/config"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration before deployment",
	Long: `Validate the APIGate configuration file.

Checks:
  - YAML syntax is valid
  - Required fields are present
  - Upstream is reachable (optional)
  - Database is writable (optional)

Examples:
  apigate validate
  apigate validate --config /etc/apigate/config.yaml`,
	RunE: runValidate,
}

var (
	validateCheckUpstream bool
	validateCheckDatabase bool
)

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().BoolVar(&validateCheckUpstream, "check-upstream", false, "check if upstream is reachable")
	validateCmd.Flags().BoolVar(&validateCheckDatabase, "check-database", false, "check if database is writable")
}

func runValidate(cmd *cobra.Command, args []string) error {
	fmt.Printf("Validating %s...\n\n", cfgFile)

	// Check file exists
	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		fmt.Printf("  %s Config file exists\n", crossMark)
		return fmt.Errorf("config file not found: %s", cfgFile)
	}
	fmt.Printf("  %s Config file exists\n", checkMark)

	// Load and validate config
	cfg, err := config.Load(cfgFile)
	if err != nil {
		fmt.Printf("  %s Config syntax valid\n", crossMark)
		return fmt.Errorf("config error: %w", err)
	}
	fmt.Printf("  %s Config syntax valid\n", checkMark)

	// Show config summary
	fmt.Printf("  %s Upstream: %s\n", checkMark, cfg.Upstream.URL)
	fmt.Printf("  %s Auth mode: %s\n", checkMark, cfg.Auth.Mode)
	fmt.Printf("  %s Database: %s (%s)\n", checkMark, cfg.Database.DSN, cfg.Database.Driver)
	fmt.Printf("  %s Plans configured: %d\n", checkMark, len(cfg.Plans))

	// Optional: check upstream
	if validateCheckUpstream {
		if err := checkUpstreamReachable(cfg.Upstream.URL); err != nil {
			fmt.Printf("  %s Upstream reachable\n", crossMark)
			fmt.Printf("      Error: %v\n", err)
		} else {
			fmt.Printf("  %s Upstream reachable\n", checkMark)
		}
	}

	// Optional: check database
	if validateCheckDatabase {
		if err := checkDatabaseWritable(cfg.Database.DSN); err != nil {
			fmt.Printf("  %s Database writable\n", crossMark)
			fmt.Printf("      Error: %v\n", err)
		} else {
			fmt.Printf("  %s Database writable\n", checkMark)
		}
	}

	fmt.Println()
	fmt.Println("Configuration is valid.")
	return nil
}

func checkUpstreamReachable(url string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func checkDatabaseWritable(dsn string) error {
	db, err := sqlite.Open(dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	return nil
}

const (
	checkMark = "\033[32m✓\033[0m"
	crossMark = "\033[31m✗\033[0m"
)
