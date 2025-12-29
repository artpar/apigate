package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/artpar/apigate/adapters/hasher"
	"github.com/artpar/apigate/adapters/sqlite"
	"github.com/artpar/apigate/config"
	"github.com/artpar/apigate/ports"
	"github.com/spf13/cobra"
)

var usersCmd = &cobra.Command{
	Use:        "users",
	Short:      "Manage users",
	Deprecated: "use 'apigate mod users' instead. Run 'apigate migration-guide' for details.",
	Long: `Manage APIGate users.

Users are the developers who consume your API. Each user can have
multiple API keys and is assigned to a plan.

Examples:
  apigate users list
  apigate users create --email=dev@example.com --plan=free
  apigate users delete user_123

NOTE: This command is deprecated. Use 'apigate mod users' instead.`,
}

var usersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users",
	RunE:  runUsersList,
}

var usersCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new user",
	RunE:  runUsersCreate,
}

var usersDeleteCmd = &cobra.Command{
	Use:   "delete <user-id>",
	Short: "Delete a user",
	Args:  cobra.ExactArgs(1),
	RunE:  runUsersDelete,
}

var usersGetCmd = &cobra.Command{
	Use:   "get <user-id-or-email>",
	Short: "Get user details",
	Args:  cobra.ExactArgs(1),
	RunE:  runUsersGet,
}

var usersActivateCmd = &cobra.Command{
	Use:   "activate <user-id-or-email>",
	Short: "Activate a user",
	Args:  cobra.ExactArgs(1),
	RunE:  runUsersActivate,
}

var usersDeactivateCmd = &cobra.Command{
	Use:   "deactivate <user-id-or-email>",
	Short: "Deactivate a user",
	Args:  cobra.ExactArgs(1),
	RunE:  runUsersDeactivate,
}

var usersSetPasswordCmd = &cobra.Command{
	Use:   "set-password <user-id-or-email>",
	Short: "Set or reset a user's password",
	Args:  cobra.ExactArgs(1),
	RunE:  runUsersSetPassword,
}

var (
	userEmail    string
	userPlan     string
	userPassword string
	userName     string
)

func init() {
	rootCmd.AddCommand(usersCmd)

	usersCmd.AddCommand(usersListCmd)
	usersCmd.AddCommand(usersCreateCmd)
	usersCmd.AddCommand(usersDeleteCmd)
	usersCmd.AddCommand(usersGetCmd)
	usersCmd.AddCommand(usersActivateCmd)
	usersCmd.AddCommand(usersDeactivateCmd)
	usersCmd.AddCommand(usersSetPasswordCmd)

	usersCreateCmd.Flags().StringVar(&userEmail, "email", "", "user email (required)")
	usersCreateCmd.Flags().StringVar(&userName, "name", "", "user name")
	usersCreateCmd.Flags().StringVar(&userPlan, "plan", "free", "plan ID")
	usersCreateCmd.Flags().StringVar(&userPassword, "password", "", "user password (optional, will prompt if not provided)")
	usersCreateCmd.MarkFlagRequired("email")

	usersSetPasswordCmd.Flags().StringVar(&userPassword, "password", "", "new password (will prompt if not provided)")
}

func runUsersList(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	userStore := sqlite.NewUserStore(db)
	users, err := userStore.List(context.Background(), 1000, 0) // Get up to 1000 users
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	if len(users) == 0 {
		fmt.Println("No users found.")
		fmt.Println()
		fmt.Println("Create a user with: apigate users create --email=dev@example.com")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tEMAIL\tPLAN\tSTATUS")
	fmt.Fprintln(w, "--\t-----\t----\t------")

	for _, u := range users {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", u.ID, u.Email, u.PlanID, u.Status)
	}

	w.Flush()
	return nil
}

func runUsersCreate(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	userStore := sqlite.NewUserStore(db)

	user := ports.User{
		ID:     fmt.Sprintf("user_%d", os.Getpid()),
		Email:  userEmail,
		Name:   userName,
		PlanID: userPlan,
		Status: "active",
	}

	// Handle password if provided or prompt for it
	if userPassword != "" {
		h := hasher.NewBcrypt(10)
		hash, err := h.Hash(userPassword)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		user.PasswordHash = hash
	}

	if err := userStore.Create(context.Background(), user); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	fmt.Printf("%s Created user: %s\n", checkMark, user.ID)
	fmt.Printf("   Email: %s\n", user.Email)
	if user.Name != "" {
		fmt.Printf("   Name:  %s\n", user.Name)
	}
	fmt.Printf("   Plan:  %s\n", user.PlanID)
	fmt.Println()
	fmt.Println("Create an API key with: apigate keys create --user=" + user.ID)

	return nil
}

func runUsersDelete(cmd *cobra.Command, args []string) error {
	userID := args[0]

	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	userStore := sqlite.NewUserStore(db)

	// Check if user exists
	_, err = userStore.Get(context.Background(), userID)
	if err != nil {
		return fmt.Errorf("user not found: %s", userID)
	}

	// Confirm deletion
	if !confirm(fmt.Sprintf("Delete user %s?", userID)) {
		fmt.Println("Aborted.")
		return nil
	}

	if err := userStore.Delete(context.Background(), userID); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	fmt.Printf("%s Deleted user: %s\n", checkMark, userID)
	return nil
}

func openDatabase() (*sqlite.DB, error) {
	// Load config to get database path
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	db, err := sqlite.Open(cfg.Database.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return db, nil
}

// getUserByIDOrEmail retrieves a user by ID or email address
func getUserByIDOrEmail(userStore *sqlite.UserStore, identifier string) (ports.User, error) {
	ctx := context.Background()

	// If it contains @, treat as email
	if strings.Contains(identifier, "@") {
		return userStore.GetByEmail(ctx, identifier)
	}

	// Otherwise treat as ID
	return userStore.Get(ctx, identifier)
}

func runUsersGet(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	userStore := sqlite.NewUserStore(db)
	user, err := getUserByIDOrEmail(userStore, args[0])
	if err != nil {
		return fmt.Errorf("user not found: %s", args[0])
	}

	fmt.Printf("ID:      %s\n", user.ID)
	fmt.Printf("Email:   %s\n", user.Email)
	if user.Name != "" {
		fmt.Printf("Name:    %s\n", user.Name)
	}
	fmt.Printf("Plan:    %s\n", user.PlanID)
	fmt.Printf("Status:  %s\n", user.Status)
	fmt.Printf("Created: %s\n", user.CreatedAt.Format("2006-01-02 15:04:05"))
	if !user.UpdatedAt.IsZero() {
		fmt.Printf("Updated: %s\n", user.UpdatedAt.Format("2006-01-02 15:04:05"))
	}
	if user.StripeID != "" {
		fmt.Printf("Stripe:  %s\n", user.StripeID)
	}
	hasPassword := len(user.PasswordHash) > 0
	fmt.Printf("Password: %v\n", hasPassword)

	return nil
}

func runUsersActivate(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	userStore := sqlite.NewUserStore(db)
	user, err := getUserByIDOrEmail(userStore, args[0])
	if err != nil {
		return fmt.Errorf("user not found: %s", args[0])
	}

	if user.Status == "active" {
		fmt.Printf("User %s is already active\n", user.Email)
		return nil
	}

	user.Status = "active"
	if err := userStore.Update(context.Background(), user); err != nil {
		return fmt.Errorf("failed to activate user: %w", err)
	}

	fmt.Printf("%s Activated user: %s (%s)\n", checkMark, user.Email, user.ID)
	return nil
}

func runUsersDeactivate(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	userStore := sqlite.NewUserStore(db)
	user, err := getUserByIDOrEmail(userStore, args[0])
	if err != nil {
		return fmt.Errorf("user not found: %s", args[0])
	}

	if user.Status == "suspended" {
		fmt.Printf("User %s is already deactivated\n", user.Email)
		return nil
	}

	user.Status = "suspended"
	if err := userStore.Update(context.Background(), user); err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	fmt.Printf("%s Deactivated user: %s (%s)\n", checkMark, user.Email, user.ID)
	return nil
}

func runUsersSetPassword(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	userStore := sqlite.NewUserStore(db)
	user, err := getUserByIDOrEmail(userStore, args[0])
	if err != nil {
		return fmt.Errorf("user not found: %s", args[0])
	}

	password := userPassword
	if password == "" {
		var err error
		password, err = promptPassword("New password: ")
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		if password == "" {
			return fmt.Errorf("password cannot be empty")
		}
	}

	h := hasher.NewBcrypt(10)
	hash, err := h.Hash(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.PasswordHash = hash
	if err := userStore.Update(context.Background(), user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	fmt.Printf("%s Password updated for user: %s (%s)\n", checkMark, user.Email, user.ID)
	return nil
}
