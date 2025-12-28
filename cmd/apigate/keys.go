package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/artpar/apigate/adapters/sqlite"
	"github.com/artpar/apigate/domain/key"
	"github.com/spf13/cobra"
)

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage API keys",
	Long: `Manage APIGate API keys.

Each user can have multiple API keys. Keys are used to authenticate
requests to your API.

Examples:
  apigate keys list
  apigate keys list --user=user_123
  apigate keys create --user=user_123
  apigate keys revoke key_abc123`,
}

var keysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List API keys",
	RunE:  runKeysList,
}

var keysCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new API key",
	RunE:  runKeysCreate,
}

var keysRevokeCmd = &cobra.Command{
	Use:   "revoke <key-id>",
	Short: "Revoke an API key",
	Args:  cobra.ExactArgs(1),
	RunE:  runKeysRevoke,
}

var (
	keyUserID string
	keyName   string
)

func init() {
	rootCmd.AddCommand(keysCmd)

	keysCmd.AddCommand(keysListCmd)
	keysCmd.AddCommand(keysCreateCmd)
	keysCmd.AddCommand(keysRevokeCmd)

	keysListCmd.Flags().StringVar(&keyUserID, "user", "", "filter by user ID")
	keysCreateCmd.Flags().StringVar(&keyUserID, "user", "", "user ID (required)")
	keysCreateCmd.Flags().StringVar(&keyName, "name", "", "key name (optional)")
	keysCreateCmd.MarkFlagRequired("user")
}

func runKeysList(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	keyStore := sqlite.NewKeyStore(db)

	var keys []key.Key
	if keyUserID != "" {
		keys, err = keyStore.ListByUser(context.Background(), keyUserID)
	} else {
		keys, err = keyStore.List(context.Background())
	}

	if err != nil {
		return fmt.Errorf("failed to list keys: %w", err)
	}

	if len(keys) == 0 {
		if keyUserID != "" {
			fmt.Printf("No keys found for user %s.\n", keyUserID)
		} else {
			fmt.Println("No API keys found.")
		}
		fmt.Println()
		fmt.Println("Create a key with: apigate keys create --user=<user-id>")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tPREFIX\tUSER\tSTATUS\tCREATED")
	fmt.Fprintln(w, "--\t------\t----\t------\t-------")

	for _, k := range keys {
		status := "active"
		if k.RevokedAt != nil {
			status = "revoked"
		}
		created := k.CreatedAt.Format("2006-01-02")
		fmt.Fprintf(w, "%s\t%s...\t%s\t%s\t%s\n", k.ID, k.Prefix, k.UserID, status, created)
	}

	w.Flush()
	return nil
}

func runKeysCreate(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	// Verify user exists
	userStore := sqlite.NewUserStore(db)
	_, err = userStore.Get(context.Background(), keyUserID)
	if err != nil {
		return fmt.Errorf("user not found: %s", keyUserID)
	}

	keyStore := sqlite.NewKeyStore(db)

	// Generate key
	rawKey, keyData := key.Generate("ak_")
	keyData = keyData.WithUserID(keyUserID)

	if err := keyStore.Create(context.Background(), keyData); err != nil {
		return fmt.Errorf("failed to create key: %w", err)
	}

	fmt.Printf("%s Created API key for user %s\n", checkMark, keyUserID)
	fmt.Println()
	fmt.Println("API Key (save this, shown once):")
	fmt.Printf("  %s\n", rawKey)
	fmt.Println()
	fmt.Printf("Key ID: %s\n", keyData.ID)

	return nil
}

func runKeysRevoke(cmd *cobra.Command, args []string) error {
	keyID := args[0]

	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	keyStore := sqlite.NewKeyStore(db)

	// Check if key exists
	k, err := keyStore.GetByID(context.Background(), keyID)
	if err != nil {
		return fmt.Errorf("key not found: %s", keyID)
	}

	if k.RevokedAt != nil {
		fmt.Printf("Key %s is already revoked.\n", keyID)
		return nil
	}

	// Confirm revocation
	if !confirm(fmt.Sprintf("Revoke key %s?", keyID)) {
		fmt.Println("Aborted.")
		return nil
	}

	now := time.Now()
	k.RevokedAt = &now

	if err := keyStore.Update(context.Background(), k); err != nil {
		return fmt.Errorf("failed to revoke key: %w", err)
	}

	fmt.Printf("%s Revoked key: %s\n", checkMark, keyID)
	return nil
}
