package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/artpar/apigate/adapters/hasher"
	"github.com/artpar/apigate/adapters/sqlite"
	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/domain/settings"
	"github.com/artpar/apigate/ports"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive setup wizard",
	Long: `Initialize APIGate with an interactive setup wizard.

This will:
  1. Ask for your upstream API URL
  2. Configure database location
  3. Create initial configuration file
  4. Create admin user (optional)
  5. Generate admin API key

Examples:
  apigate init
  apigate init --config /etc/apigate/config.yaml`,
	RunE: runInit,
}

var (
	initUpstream       string
	initDatabase       string
	initAdminEmail     string
	initAdminPassword  string
	initNonInteractive bool
)

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringVar(&initUpstream, "upstream", "", "upstream API URL")
	initCmd.Flags().StringVar(&initDatabase, "database", "apigate.db", "database file path")
	initCmd.Flags().StringVar(&initAdminEmail, "admin-email", "", "admin user email")
	initCmd.Flags().StringVar(&initAdminPassword, "admin-password", "", "admin user password (auto-generated if not provided)")
	initCmd.Flags().BoolVar(&initNonInteractive, "non-interactive", false, "run without prompts (requires --upstream)")
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Println("Welcome to APIGate!")
	fmt.Println()

	// Check if config already exists
	if _, err := os.Stat(cfgFile); err == nil {
		fmt.Printf("Configuration file already exists: %s\n", cfgFile)
		if !confirm("Overwrite?") {
			fmt.Println("Aborted.")
			return nil
		}
	}

	reader := bufio.NewReader(os.Stdin)

	// Get upstream URL
	upstream := initUpstream
	if upstream == "" {
		if initNonInteractive {
			return fmt.Errorf("--upstream is required in non-interactive mode")
		}
		upstream = prompt(reader, "Upstream API URL", "")
		if upstream == "" {
			return fmt.Errorf("upstream URL is required")
		}
	}

	// Get database location
	database := initDatabase
	if !initNonInteractive && initDatabase == "apigate.db" {
		database = prompt(reader, "Database location", "apigate.db")
	}

	// Create admin user?
	var adminEmail string
	var adminPassword string
	createAdmin := false
	if initAdminEmail != "" {
		adminEmail = initAdminEmail
		adminPassword = initAdminPassword
		createAdmin = true
	} else if !initNonInteractive {
		createAdmin = confirm("Create admin user?")
		if createAdmin {
			adminEmail = prompt(reader, "Admin email", "")
			if adminEmail == "" {
				return fmt.Errorf("admin email is required")
			}
			// Prompt for password
			adminPassword, _ = promptPassword("Admin password (leave empty to auto-generate)")
		}
	}

	// Generate password if not provided
	if createAdmin && adminPassword == "" {
		adminPassword = generatePassword()
	}

	// Generate config
	configContent := generateConfig(upstream, database)

	// Write config file
	if err := os.WriteFile(cfgFile, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	fmt.Printf("\n%s Generated %s\n", checkMark, cfgFile)

	// Create database and run migrations
	db, err := sqlite.Open(database)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}
	fmt.Printf("%s Created database %s\n", checkMark, database)

	// Save upstream URL and other settings to database
	settingsStore := sqlite.NewSettingsStore(db)
	ctx := context.Background()

	// Save upstream URL
	if err := settingsStore.Set(ctx, settings.KeyUpstreamURL, upstream, false); err != nil {
		return fmt.Errorf("failed to save upstream URL: %w", err)
	}

	// Enable portal by default
	if err := settingsStore.Set(ctx, settings.KeyPortalEnabled, "true", false); err != nil {
		return fmt.Errorf("failed to enable portal: %w", err)
	}

	fmt.Printf("%s Saved settings to database\n", checkMark)

	// Create admin user if requested
	if createAdmin && adminEmail != "" {
		apiKey, err := createAdminUser(db, adminEmail, adminPassword)
		if err != nil {
			return fmt.Errorf("failed to create admin user: %w", err)
		}
		fmt.Printf("%s Created admin user: %s\n", checkMark, adminEmail)
		fmt.Println()
		fmt.Println("Admin credentials (save these, shown once):")
		fmt.Printf("  Email:    %s\n", adminEmail)
		fmt.Printf("  Password: %s\n", adminPassword)
		fmt.Printf("  API Key:  %s\n", apiKey)
	}

	fmt.Println()
	fmt.Println("Run 'apigate serve' to start the proxy server.")
	fmt.Println()
	fmt.Println("Access points:")
	fmt.Println("  Admin Dashboard: http://localhost:8080/login")
	fmt.Println("  User Portal:     http://localhost:8080/portal/")
	fmt.Println("  API Proxy:       http://localhost:8080/ (requires API key)")

	return nil
}

func prompt(reader *bufio.Reader, label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("? %s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("? %s: ", label)
	}

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultVal
	}
	return input
}

func confirm(message string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("? %s [y/N]: ", message)
	input, _ := reader.ReadString('\n')
	input = strings.ToLower(strings.TrimSpace(input))
	return input == "y" || input == "yes"
}

func generateConfig(upstream, database string) string {
	return fmt.Sprintf(`# APIGate Configuration
# Generated by 'apigate init'

server:
  host: "0.0.0.0"
  port: 8080

upstream:
  url: "%s"
  timeout: 30s

database:
  driver: sqlite
  dsn: "%s"

auth:
  mode: local
  key_prefix: "ak_"

rate_limit:
  enabled: true
  burst_tokens: 10
  window_secs: 60

plans:
  - id: free
    name: "Free"
    rate_limit_per_minute: 60
    requests_per_month: 1000

  - id: pro
    name: "Pro"
    rate_limit_per_minute: 600
    requests_per_month: 100000

logging:
  level: info
  format: console

metrics:
  enabled: true

openapi:
  enabled: true
`, upstream, database)
}

func createAdminUser(db *sqlite.DB, email, password string) (string, error) {
	ctx := context.Background()

	// Create user store and key store
	userStore := sqlite.NewUserStore(db)
	keyStore := sqlite.NewKeyStore(db)

	// Hash the password
	h := hasher.NewBcrypt(10)
	passwordHash, err := h.Hash(password)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	// Create admin user with password
	user := ports.User{
		ID:           generateID(),
		Email:        email,
		PasswordHash: passwordHash,
		PlanID:       "free",
		Status:       "active",
	}

	if err := userStore.Create(ctx, user); err != nil {
		return "", fmt.Errorf("create user: %w", err)
	}

	// Generate and create API key
	rawKey, keyData := key.Generate("ak_")

	if err := keyStore.Create(ctx, keyData.WithUserID(user.ID)); err != nil {
		return "", fmt.Errorf("create key: %w", err)
	}

	return rawKey, nil
}

func generatePassword() string {
	bytes := make([]byte, 12)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:16]
}

func generateID() string {
	// Simple ID generation - in production would use UUID
	return fmt.Sprintf("user_%d", os.Getpid())
}
