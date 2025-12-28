package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/artpar/apigate/adapters/sqlite"
	"github.com/artpar/apigate/config"
	"github.com/artpar/apigate/ports"
	"github.com/spf13/cobra"
)

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "Manage users",
	Long: `Manage APIGate users.

Users are the developers who consume your API. Each user can have
multiple API keys and is assigned to a plan.

Examples:
  apigate users list
  apigate users create --email=dev@example.com --plan=free
  apigate users delete user_123`,
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

var (
	userEmail  string
	userPlan   string
)

func init() {
	rootCmd.AddCommand(usersCmd)

	usersCmd.AddCommand(usersListCmd)
	usersCmd.AddCommand(usersCreateCmd)
	usersCmd.AddCommand(usersDeleteCmd)

	usersCreateCmd.Flags().StringVar(&userEmail, "email", "", "user email (required)")
	usersCreateCmd.Flags().StringVar(&userPlan, "plan", "free", "plan ID")
	usersCreateCmd.MarkFlagRequired("email")
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
		PlanID: userPlan,
		Status: "active",
	}

	if err := userStore.Create(context.Background(), user); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	fmt.Printf("%s Created user: %s\n", checkMark, user.ID)
	fmt.Printf("   Email: %s\n", user.Email)
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
