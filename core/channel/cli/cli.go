// Package cli provides a CLI channel that generates commands from module definitions.
// It automatically creates list, get, create, update, delete, and custom action commands.
package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/formatter"
	"github.com/artpar/apigate/core/runtime"
	"github.com/artpar/apigate/core/schema"
	"github.com/artpar/apigate/core/validation"
	"github.com/spf13/cobra"
)

// Channel implements the CLI channel for modules.
type Channel struct {
	rootCmd    *cobra.Command
	runtime    *runtime.Runtime
	modules    map[string]convention.Derived
	formatters *formatter.Registry
	validator  *validation.Validator
}

// New creates a new CLI channel.
func New(rootCmd *cobra.Command, rt *runtime.Runtime) *Channel {
	return &Channel{
		rootCmd:    rootCmd,
		runtime:    rt,
		modules:    make(map[string]convention.Derived),
		formatters: formatter.DefaultRegistry,
		validator:  validation.New(make(map[string]convention.Derived)),
	}
}

// Name returns the channel name.
func (c *Channel) Name() string {
	return "cli"
}

// Register registers a module with the CLI channel.
func (c *Channel) Register(mod convention.Derived) error {
	// Check if CLI is enabled for this module
	if !mod.Source.Channels.CLI.Serve.Enabled {
		return nil
	}

	c.modules[mod.Source.Name] = mod

	// Update validator with all modules
	c.validator.UpdateModules(c.modules)

	// Create the module's root command
	moduleCmd := &cobra.Command{
		Use:   mod.Plural,
		Short: fmt.Sprintf("Manage %s", mod.Plural),
	}

	// Generate subcommands for each action
	for _, action := range mod.Actions {
		cmd := c.buildActionCommand(mod, action)
		if cmd != nil {
			moduleCmd.AddCommand(cmd)
		}
	}

	c.rootCmd.AddCommand(moduleCmd)
	return nil
}

// Start starts the CLI channel (no-op for CLI).
func (c *Channel) Start(ctx context.Context) error {
	return nil
}

// Stop stops the CLI channel (no-op for CLI).
func (c *Channel) Stop(ctx context.Context) error {
	return nil
}

// buildActionCommand creates a cobra command for an action.
func (c *Channel) buildActionCommand(mod convention.Derived, action convention.DerivedAction) *cobra.Command {
	switch action.Type {
	case schema.ActionTypeList:
		return c.buildListCommand(mod, action)
	case schema.ActionTypeGet:
		return c.buildGetCommand(mod, action)
	case schema.ActionTypeCreate:
		return c.buildCreateCommand(mod, action)
	case schema.ActionTypeUpdate:
		return c.buildUpdateCommand(mod, action)
	case schema.ActionTypeDelete:
		return c.buildDeleteCommand(mod, action)
	case schema.ActionTypeCustom:
		return c.buildCustomCommand(mod, action)
	default:
		return nil
	}
}

// buildListCommand creates a list command.
func (c *Channel) buildListCommand(mod convention.Derived, action convention.DerivedAction) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: action.Description,
		RunE: func(cmd *cobra.Command, args []string) error {
			limit, _ := cmd.Flags().GetInt("limit")
			offset, _ := cmd.Flags().GetInt("offset")

			result, err := c.runtime.Execute(cmd.Context(), mod.Source.Name, "list", runtime.ActionInput{
				Channel: "cli",
				Data: map[string]any{
					"limit":  limit,
					"offset": offset,
				},
			})
			if err != nil {
				return c.formatError(cmd, err)
			}

			return c.formatList(cmd, mod, result.List)
		},
	}

	cmd.Flags().IntP("limit", "l", 100, "Maximum number of records")
	cmd.Flags().IntP("offset", "o", 0, "Number of records to skip")
	c.addOutputFlags(cmd)

	return cmd
}

// buildGetCommand creates a get command.
func (c *Channel) buildGetCommand(mod convention.Derived, action convention.DerivedAction) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <id-or-name>",
		Short: action.Description,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := c.runtime.Execute(cmd.Context(), mod.Source.Name, "get", runtime.ActionInput{
				Lookup:  args[0],
				Channel: "cli",
			})
			if err != nil {
				return c.formatError(cmd, err)
			}

			return c.formatRecord(cmd, mod, result.Data)
		},
	}

	c.addOutputFlags(cmd)

	return cmd
}

// buildCreateCommand creates a create command.
func (c *Channel) buildCreateCommand(mod convention.Derived, action convention.DerivedAction) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: action.Description,
		RunE: func(cmd *cobra.Command, args []string) error {
			data := make(map[string]any)

			// Collect values from flags
			for _, input := range action.Input {
				if val, err := cmd.Flags().GetString(input.Name); err == nil && val != "" {
					data[input.Name] = convertInput(val, input.Type)
				} else if input.Required {
					return fmt.Errorf("required field %q not provided", input.Name)
				}
			}

			// Client-side validation
			validationResult := c.validator.ValidateCreate(mod.Source.Name, data)
			if !validationResult.Valid {
				return c.formatError(cmd, fmt.Errorf("%s", validationResult.Error()))
			}

			result, err := c.runtime.Execute(cmd.Context(), mod.Source.Name, "create", runtime.ActionInput{
				Data:    data,
				Channel: "cli",
			})
			if err != nil {
				return c.formatError(cmd, err)
			}

			outputFmt, _ := cmd.Flags().GetString("output")
			if outputFmt == "json" || outputFmt == "yaml" {
				return c.formatRecord(cmd, mod, result.Data)
			}
			fmt.Printf("Created %s: %s\n", mod.Source.Name, result.ID)
			return nil
		},
	}

	// Add flags for each input
	for _, input := range action.Input {
		usage := input.Name
		if input.Required {
			usage += " (required)"
		}
		defaultStr, _ := input.Default.(string)
		cmd.Flags().String(input.Name, defaultStr, usage)
		if input.Required {
			cmd.MarkFlagRequired(input.Name)
		}
	}
	c.addOutputFlags(cmd)

	return cmd
}

// buildUpdateCommand creates an update command.
func (c *Channel) buildUpdateCommand(mod convention.Derived, action convention.DerivedAction) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id-or-name>",
		Short: action.Description,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data := make(map[string]any)

			// Collect values from flags that were set
			for _, input := range action.Input {
				if cmd.Flags().Changed(input.Name) {
					val, _ := cmd.Flags().GetString(input.Name)
					data[input.Name] = convertInput(val, input.Type)
				}
			}

			if len(data) == 0 {
				return fmt.Errorf("no fields to update")
			}

			// Client-side validation
			validationResult := c.validator.ValidateUpdate(mod.Source.Name, data)
			if !validationResult.Valid {
				return c.formatError(cmd, fmt.Errorf("%s", validationResult.Error()))
			}

			result, err := c.runtime.Execute(cmd.Context(), mod.Source.Name, "update", runtime.ActionInput{
				Lookup:  args[0],
				Data:    data,
				Channel: "cli",
			})
			if err != nil {
				return c.formatError(cmd, err)
			}

			outputFmt, _ := cmd.Flags().GetString("output")
			if outputFmt == "json" || outputFmt == "yaml" {
				return c.formatRecord(cmd, mod, result.Data)
			}
			fmt.Printf("Updated %s: %s\n", mod.Source.Name, args[0])
			return nil
		},
	}

	// Add flags for each input
	for _, input := range action.Input {
		cmd.Flags().String(input.Name, "", input.Name)
	}
	c.addOutputFlags(cmd)

	return cmd
}

// buildDeleteCommand creates a delete command.
func (c *Channel) buildDeleteCommand(mod convention.Derived, action convention.DerivedAction) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id-or-name>",
		Short: action.Description,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")
			if !force && action.Confirm {
				fmt.Printf("Are you sure you want to delete %s %s? (use --force to confirm)\n", mod.Source.Name, args[0])
				return nil
			}

			_, err := c.runtime.Execute(cmd.Context(), mod.Source.Name, "delete", runtime.ActionInput{
				Lookup:  args[0],
				Channel: "cli",
			})
			if err != nil {
				return err
			}

			fmt.Printf("Deleted %s: %s\n", mod.Source.Name, args[0])
			return nil
		},
	}

	cmd.Flags().BoolP("force", "f", false, "Force delete without confirmation")

	return cmd
}

// buildCustomCommand creates a custom action command.
func (c *Channel) buildCustomCommand(mod convention.Derived, action convention.DerivedAction) *cobra.Command {
	cmd := &cobra.Command{
		Use:   action.Name + " <id-or-name>",
		Short: action.Description,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data := make(map[string]any)

			// Collect values from flags
			for _, input := range action.Input {
				if cmd.Flags().Changed(input.Name) {
					val, _ := cmd.Flags().GetString(input.Name)
					data[input.Name] = convertInput(val, input.Type)
				}
			}

			result, err := c.runtime.Execute(cmd.Context(), mod.Source.Name, action.Name, runtime.ActionInput{
				Lookup:  args[0],
				Data:    data,
				Channel: "cli",
			})
			if err != nil {
				return err
			}

			fmt.Printf("%s completed for %s: %s\n", strings.Title(action.Name), mod.Source.Name, result.ID)
			return nil
		},
	}

	// Add flags for each input
	for _, input := range action.Input {
		defaultStr, _ := input.Default.(string)
		cmd.Flags().String(input.Name, defaultStr, input.Name)
	}

	return cmd
}

// addOutputFlags adds common output format flags to a command.
func (c *Channel) addOutputFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("output", "O", "table", "Output format: "+strings.Join(c.formatters.List(), ", "))
	cmd.Flags().Bool("no-header", false, "Disable header row (table format)")
	cmd.Flags().Bool("compact", false, "Compact output (json/yaml)")
}

// getFormatter returns the formatter for the current command.
func (c *Channel) getFormatter(cmd *cobra.Command) formatter.Formatter {
	outputFmt, _ := cmd.Flags().GetString("output")
	if outputFmt == "" {
		outputFmt = "table"
	}

	f, ok := c.formatters.Get(outputFmt)
	if !ok {
		return c.formatters.Default()
	}
	return f
}

// getFormatOptions builds format options from command flags.
func (c *Channel) getFormatOptions(cmd *cobra.Command) formatter.FormatOptions {
	noHeader, _ := cmd.Flags().GetBool("no-header")
	compact, _ := cmd.Flags().GetBool("compact")

	return formatter.FormatOptions{
		NoHeader: noHeader,
		Compact:  compact,
		MaxWidth: 40,
	}
}

// formatList formats and outputs a list of records.
func (c *Channel) formatList(cmd *cobra.Command, mod convention.Derived, records []map[string]any) error {
	f := c.getFormatter(cmd)
	opts := c.getFormatOptions(cmd)
	return f.FormatList(os.Stdout, mod, records, opts)
}

// formatRecord formats and outputs a single record.
func (c *Channel) formatRecord(cmd *cobra.Command, mod convention.Derived, record map[string]any) error {
	f := c.getFormatter(cmd)
	opts := c.getFormatOptions(cmd)
	return f.FormatRecord(os.Stdout, mod, record, opts)
}

// formatError formats and outputs an error.
func (c *Channel) formatError(cmd *cobra.Command, err error) error {
	f := c.getFormatter(cmd)
	f.FormatError(os.Stderr, err)
	return err
}

// convertInput converts a string input to the appropriate type.
func convertInput(val string, fieldType schema.FieldType) any {
	switch fieldType {
	case schema.FieldTypeInt:
		var i int
		fmt.Sscanf(val, "%d", &i)
		return i
	case schema.FieldTypeFloat:
		var f float64
		fmt.Sscanf(val, "%f", &f)
		return f
	case schema.FieldTypeBool:
		return val == "true" || val == "1" || val == "yes"
	default:
		return val
	}
}
