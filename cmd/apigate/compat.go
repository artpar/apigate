// Package main provides CLI compatibility layer for deprecated commands.
// These aliases provide backward compatibility while users migrate to the
// module-based command system.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// deprecationWarning prints a warning about deprecated commands.
func deprecationWarning(oldCmd, newCmd string) {
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  WARNING: '%s' is deprecated.\n", oldCmd)
	fmt.Fprintf(os.Stderr, "  Use '%s' instead.\n", newCmd)
	fmt.Fprintf(os.Stderr, "\n")
}

// DeprecatedCommand creates a command that shows a deprecation warning
// and provides guidance on the new command.
type DeprecatedCommand struct {
	OldUse      string // e.g., "users"
	NewUse      string // e.g., "mod users"
	Module      string // e.g., "user"
	Description string
}

// createDeprecatedAlias creates a deprecated alias command that warns users.
func createDeprecatedAlias(dc DeprecatedCommand) *cobra.Command {
	return &cobra.Command{
		Use:        dc.OldUse,
		Short:      fmt.Sprintf("[DEPRECATED] %s - use '%s' instead", dc.Description, dc.NewUse),
		Deprecated: fmt.Sprintf("use '%s' instead", dc.NewUse),
		Hidden:     false, // Show in help but mark deprecated
		Run: func(cmd *cobra.Command, args []string) {
			deprecationWarning(
				fmt.Sprintf("apigate %s", dc.OldUse),
				fmt.Sprintf("apigate %s", dc.NewUse),
			)

			fmt.Println("Available commands in the new system:")
			fmt.Println()
			fmt.Printf("  apigate %s list               List all %s\n", dc.NewUse, dc.Module+"s")
			fmt.Printf("  apigate %s get <id>           Get a specific %s\n", dc.NewUse, dc.Module)
			fmt.Printf("  apigate %s create             Create a new %s\n", dc.NewUse, dc.Module)
			fmt.Printf("  apigate %s update <id>        Update a %s\n", dc.NewUse, dc.Module)
			fmt.Printf("  apigate %s delete <id>        Delete a %s\n", dc.NewUse, dc.Module)
			fmt.Println()
			fmt.Printf("Run 'apigate %s --help' for more information.\n", dc.NewUse)
		},
	}
}

// createDeprecatedSubcommand creates a deprecated subcommand alias.
func createDeprecatedSubcommand(parentOld, parentNew, action, module string) *cobra.Command {
	cmd := &cobra.Command{
		Use:        action,
		Short:      fmt.Sprintf("[DEPRECATED] Use 'apigate %s %s' instead", parentNew, action),
		Deprecated: fmt.Sprintf("use 'apigate %s %s' instead", parentNew, action),
		DisableFlagParsing: true, // Pass all args through
		Run: func(cmd *cobra.Command, args []string) {
			oldCmd := fmt.Sprintf("apigate %s %s", parentOld, action)
			newCmd := fmt.Sprintf("apigate %s %s", parentNew, action)
			deprecationWarning(oldCmd, newCmd)

			// Show the equivalent new command
			if len(args) > 0 {
				fmt.Printf("Equivalent command: apigate %s %s %s\n", parentNew, action, strings.Join(args, " "))
			} else {
				fmt.Printf("Equivalent command: apigate %s %s\n", parentNew, action)
			}
			fmt.Println()
			fmt.Printf("Run 'apigate %s %s --help' for usage.\n", parentNew, action)
		},
	}
	return cmd
}

// RegisterDeprecationAliases registers deprecated command aliases.
// Call this after module runtime is initialized to avoid conflicts.
func RegisterDeprecationAliases(root *cobra.Command) {
	// Define deprecated command mappings
	// The old commands (users, plans, routes, keys) map to mod subcommands
	deprecatedCommands := []struct {
		old     string
		new     string
		module  string
		desc    string
		actions []string
	}{
		{
			old:     "users-legacy",
			new:     "mod users",
			module:  "user",
			desc:    "Manage users",
			actions: []string{"list", "get", "create", "update", "delete", "activate", "deactivate"},
		},
		{
			old:     "plans-legacy",
			new:     "mod plans",
			module:  "plan",
			desc:    "Manage plans",
			actions: []string{"list", "get", "create", "update", "delete"},
		},
		{
			old:     "routes-legacy",
			new:     "mod routes",
			module:  "route",
			desc:    "Manage routes",
			actions: []string{"list", "get", "create", "update", "delete"},
		},
		{
			old:     "keys-legacy",
			new:     "mod keys",
			module:  "key",
			desc:    "Manage API keys",
			actions: []string{"list", "get", "create", "update", "delete", "rotate"},
		},
	}

	for _, dc := range deprecatedCommands {
		// Create parent deprecated command
		parentCmd := createDeprecatedAlias(DeprecatedCommand{
			OldUse:      dc.old,
			NewUse:      dc.new,
			Module:      dc.module,
			Description: dc.desc,
		})

		// Add deprecated subcommands
		for _, action := range dc.actions {
			subCmd := createDeprecatedSubcommand(dc.old, dc.new, action, dc.module)
			parentCmd.AddCommand(subCmd)
		}

		root.AddCommand(parentCmd)
	}
}

// AddDeprecationNotice adds a deprecation notice to an existing command.
// Use this to mark existing commands as deprecated while keeping them functional.
func AddDeprecationNotice(cmd *cobra.Command, newCommand string) {
	originalRun := cmd.Run
	originalRunE := cmd.RunE

	cmd.Deprecated = fmt.Sprintf("use '%s' instead", newCommand)

	// Wrap the run function to show warning
	if originalRunE != nil {
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			deprecationWarning(cmd.CommandPath(), newCommand)
			return originalRunE(cmd, args)
		}
	} else if originalRun != nil {
		cmd.Run = func(cmd *cobra.Command, args []string) {
			deprecationWarning(cmd.CommandPath(), newCommand)
			originalRun(cmd, args)
		}
	}
}

// MigrationGuide prints a guide for migrating from old to new commands.
var migrationGuideCmd = &cobra.Command{
	Use:   "migration-guide",
	Short: "Show CLI migration guide",
	Long:  "Display a guide for migrating from deprecated CLI commands to the new module-based system.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("CLI Migration Guide")
		fmt.Println("==================")
		fmt.Println()
		fmt.Println("APIGate is migrating to a unified module-based CLI.")
		fmt.Println("The old commands still work but will be removed in a future version.")
		fmt.Println()
		fmt.Println("Command Mapping:")
		fmt.Println()
		fmt.Println("  Old Command                    New Command")
		fmt.Println("  -----------                    -----------")
		fmt.Println("  apigate users list          -> apigate mod users list")
		fmt.Println("  apigate users create        -> apigate mod users create")
		fmt.Println("  apigate users get <id>      -> apigate mod users get <id>")
		fmt.Println("  apigate users delete <id>   -> apigate mod users delete <id>")
		fmt.Println("  apigate users activate      -> apigate mod users activate <id>")
		fmt.Println("  apigate users deactivate    -> apigate mod users deactivate <id>")
		fmt.Println()
		fmt.Println("  apigate plans list          -> apigate mod plans list")
		fmt.Println("  apigate plans create        -> apigate mod plans create")
		fmt.Println()
		fmt.Println("  apigate routes list         -> apigate mod routes list")
		fmt.Println("  apigate routes create       -> apigate mod routes create")
		fmt.Println()
		fmt.Println("  apigate keys list           -> apigate mod keys list")
		fmt.Println("  apigate keys create         -> apigate mod keys create")
		fmt.Println()
		fmt.Println("New Features in Module CLI:")
		fmt.Println()
		fmt.Println("  - Output formats: --output table|json|yaml")
		fmt.Println("  - Client-side validation before sending to server")
		fmt.Println("  - Interactive prompts for required fields")
		fmt.Println("  - Consistent interface across all modules")
		fmt.Println()
		fmt.Println("Run 'apigate mod --help' for the full list of available modules.")
	},
}

func init() {
	rootCmd.AddCommand(migrationGuideCmd)
}
