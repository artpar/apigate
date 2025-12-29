package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/artpar/apigate/adapters/sqlite"
	"github.com/spf13/cobra"
)

var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Manage settings",
	Long: `Manage APIGate settings stored in the database.

Settings control various aspects of APIGate behavior.

Examples:
  apigate settings list
  apigate settings get upstream_url
  apigate settings set upstream_url https://api.example.com`,
}

var settingsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all settings",
	RunE:  runSettingsList,
}

var settingsGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a setting value",
	Args:  cobra.ExactArgs(1),
	RunE:  runSettingsGet,
}

var settingsSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a setting value",
	Args:  cobra.ExactArgs(2),
	RunE:  runSettingsSet,
}

var settingsDeleteCmd = &cobra.Command{
	Use:   "delete <key>",
	Short: "Delete a setting",
	Args:  cobra.ExactArgs(1),
	RunE:  runSettingsDelete,
}

var settingEncrypted bool

func init() {
	rootCmd.AddCommand(settingsCmd)

	settingsCmd.AddCommand(settingsListCmd)
	settingsCmd.AddCommand(settingsGetCmd)
	settingsCmd.AddCommand(settingsSetCmd)
	settingsCmd.AddCommand(settingsDeleteCmd)

	settingsSetCmd.Flags().BoolVar(&settingEncrypted, "encrypted", false, "store value encrypted")
}

func runSettingsList(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	settingsStore := sqlite.NewSettingsStore(db)
	settings, err := settingsStore.GetAll(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list settings: %w", err)
	}

	if len(settings) == 0 {
		fmt.Println("No settings found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tVALUE")
	fmt.Fprintln(w, "---\t-----")

	for key, value := range settings {
		// Truncate long values for display
		displayValue := value
		if len(displayValue) > 50 {
			displayValue = displayValue[:47] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\n", key, displayValue)
	}

	w.Flush()
	return nil
}

func runSettingsGet(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	settingsStore := sqlite.NewSettingsStore(db)
	setting, err := settingsStore.Get(context.Background(), args[0])
	if err != nil {
		return fmt.Errorf("setting not found: %s", args[0])
	}

	fmt.Println(setting.Value)
	return nil
}

func runSettingsSet(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	settingsStore := sqlite.NewSettingsStore(db)
	if err := settingsStore.Set(context.Background(), args[0], args[1], settingEncrypted); err != nil {
		return fmt.Errorf("failed to set setting: %w", err)
	}

	fmt.Printf("%s Set %s\n", checkMark, args[0])
	return nil
}

func runSettingsDelete(cmd *cobra.Command, args []string) error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	settingsStore := sqlite.NewSettingsStore(db)
	if err := settingsStore.Delete(context.Background(), args[0]); err != nil {
		return fmt.Errorf("failed to delete setting: %w", err)
	}

	fmt.Printf("%s Deleted %s\n", checkMark, args[0])
	return nil
}
