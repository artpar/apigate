package main

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/artpar/apigate/adapters/hasher"
	"github.com/artpar/apigate/adapters/sqlite"
	"github.com/artpar/apigate/ports"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Manage admin users",
	Long: `Manage APIGate admin users who can access the web UI.

Admin users have a password and can log into the web dashboard
to manage routes, upstreams, API keys, and view usage.

Examples:
  apigate admin list
  apigate admin create --email=admin@example.com
  apigate admin reset-password admin@example.com

For local dev without a config file, use --db to specify the database directly:
  apigate admin reset-password --db apigate.db admin@example.com
  apigate admin list --db apigate.db

Or set the APIGATE_DATABASE_PATH environment variable:
  APIGATE_DATABASE_PATH=apigate.db apigate admin list`,
}

var adminListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all admin users",
	RunE:  runAdminList,
}

var adminCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new admin user",
	Long: `Create a new admin user who can log into the web UI.

If --password is not provided, you will be prompted to enter it securely.

Examples:
  apigate admin create --email=admin@example.com
  apigate admin create --email=admin@example.com --password=secret`,
	RunE: runAdminCreate,
}

var adminResetPasswordCmd = &cobra.Command{
	Use:   "reset-password <email>",
	Short: "Reset an admin user's password",
	Long: `Reset the password for an existing admin user.

If --password is not provided, you will be prompted to enter it securely.

Examples:
  apigate admin reset-password admin@example.com
  apigate admin reset-password admin@example.com --password=newpassword`,
	Args: cobra.ExactArgs(1),
	RunE: runAdminResetPassword,
}

var adminDeleteCmd = &cobra.Command{
	Use:   "delete <email>",
	Short: "Delete an admin user",
	Args:  cobra.ExactArgs(1),
	RunE:  runAdminDelete,
}

var (
	adminEmail    string
	adminPassword string
)

func init() {
	rootCmd.AddCommand(adminCmd)

	adminCmd.AddCommand(adminListCmd)
	adminCmd.AddCommand(adminCreateCmd)
	adminCmd.AddCommand(adminResetPasswordCmd)
	adminCmd.AddCommand(adminDeleteCmd)

	// Add --db persistent flag to admin command (works with all subcommands)
	adminCmd.PersistentFlags().StringVar(&dbPath, "db", "", "database file path (bypasses config file)")

	adminCreateCmd.Flags().StringVar(&adminEmail, "email", "", "admin email (required)")
	adminCreateCmd.Flags().StringVar(&adminPassword, "password", "", "admin password (will prompt if not provided)")
	adminCreateCmd.MarkFlagRequired("email")

	adminResetPasswordCmd.Flags().StringVar(&adminPassword, "password", "", "new password (will prompt if not provided)")
}

func runAdminList(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	userStore := sqlite.NewUserStore(db)
	users, err := userStore.List(context.Background(), 1000, 0)
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	// Filter to only show users with passwords (admin users)
	var admins []ports.User
	for _, u := range users {
		if len(u.PasswordHash) > 0 {
			admins = append(admins, u)
		}
	}

	if len(admins) == 0 {
		fmt.Println("No admin users found.")
		fmt.Println()
		fmt.Println("Create an admin user with: apigate admin create --email=admin@example.com")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tEMAIL\tSTATUS\tCREATED")
	fmt.Fprintln(w, "--\t-----\t------\t-------")

	for _, u := range admins {
		created := u.CreatedAt.Format("2006-01-02 15:04")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", u.ID, u.Email, u.Status, created)
	}

	w.Flush()
	return nil
}

func runAdminCreate(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	userStore := sqlite.NewUserStore(db)

	// Check if email already exists
	existing, err := userStore.GetByEmail(context.Background(), adminEmail)
	if err == nil && existing.ID != "" {
		return fmt.Errorf("user with email %s already exists", adminEmail)
	}

	// Get password
	password := adminPassword
	if password == "" {
		password, err = promptPassword("Enter password: ")
		if err != nil {
			return err
		}
		confirm, err := promptPassword("Confirm password: ")
		if err != nil {
			return err
		}
		if password != confirm {
			return fmt.Errorf("passwords do not match")
		}
	}

	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	// Hash password
	h := getHasher()
	passwordHash, err := h.Hash(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now().UTC()
	user := ports.User{
		ID:           generateAdminID(),
		Email:        adminEmail,
		PasswordHash: passwordHash,
		PlanID:       "admin",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := userStore.Create(context.Background(), user); err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	fmt.Printf("%s Created admin user: %s\n", checkMark, user.Email)
	fmt.Printf("   ID: %s\n", user.ID)
	fmt.Println()
	fmt.Println("You can now log in at: http://localhost:8080/login")

	return nil
}

func runAdminResetPassword(cmd *cobra.Command, args []string) error {
	email := args[0]

	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	userStore := sqlite.NewUserStore(db)

	// Find user by email
	user, err := userStore.GetByEmail(context.Background(), email)
	if err != nil {
		return fmt.Errorf("user not found: %s", email)
	}

	if len(user.PasswordHash) == 0 {
		return fmt.Errorf("user %s is not an admin user (no password set)", email)
	}

	// Get new password
	password := adminPassword
	if password == "" {
		password, err = promptPassword("Enter new password: ")
		if err != nil {
			return err
		}
		confirm, err := promptPassword("Confirm new password: ")
		if err != nil {
			return err
		}
		if password != confirm {
			return fmt.Errorf("passwords do not match")
		}
	}

	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	// Hash password
	h := getHasher()
	passwordHash, err := h.Hash(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.PasswordHash = passwordHash
	user.UpdatedAt = time.Now().UTC()

	if err := userStore.Update(context.Background(), user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	fmt.Printf("%s Password reset for: %s\n", checkMark, email)
	return nil
}

func runAdminDelete(cmd *cobra.Command, args []string) error {
	email := args[0]

	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	userStore := sqlite.NewUserStore(db)

	// Find user by email
	user, err := userStore.GetByEmail(context.Background(), email)
	if err != nil {
		return fmt.Errorf("user not found: %s", email)
	}

	if len(user.PasswordHash) == 0 {
		return fmt.Errorf("user %s is not an admin user", email)
	}

	// Confirm deletion
	if !confirm(fmt.Sprintf("Delete admin user %s?", email)) {
		fmt.Println("Aborted.")
		return nil
	}

	if err := userStore.Delete(context.Background(), user.ID); err != nil {
		return fmt.Errorf("failed to delete admin user: %w", err)
	}

	fmt.Printf("%s Deleted admin user: %s\n", checkMark, email)
	return nil
}

func promptPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // Print newline after password input
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	return string(password), nil
}

func generateAdminID() string {
	return fmt.Sprintf("admin_%d", time.Now().UnixNano())
}

// getHasher returns a bcrypt hasher for password operations
func getHasher() ports.Hasher {
	return hasher.NewBcrypt(bcrypt.DefaultCost)
}
