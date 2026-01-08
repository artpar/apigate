package cli

import (
	"context"
	"fmt"
	"testing"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/schema"
	"github.com/spf13/cobra"
)

func TestConvertInput(t *testing.T) {
	tests := []struct {
		name      string
		val       string
		fieldType schema.FieldType
		expected  any
	}{
		{
			name:      "string type",
			val:       "hello",
			fieldType: schema.FieldTypeString,
			expected:  "hello",
		},
		{
			name:      "int type valid",
			val:       "42",
			fieldType: schema.FieldTypeInt,
			expected:  42,
		},
		{
			name:      "int type negative",
			val:       "-10",
			fieldType: schema.FieldTypeInt,
			expected:  -10,
		},
		{
			name:      "float type valid",
			val:       "3.14",
			fieldType: schema.FieldTypeFloat,
			expected:  3.14,
		},
		{
			name:      "float type negative",
			val:       "-2.5",
			fieldType: schema.FieldTypeFloat,
			expected:  -2.5,
		},
		{
			name:      "bool type true",
			val:       "true",
			fieldType: schema.FieldTypeBool,
			expected:  true,
		},
		{
			name:      "bool type 1",
			val:       "1",
			fieldType: schema.FieldTypeBool,
			expected:  true,
		},
		{
			name:      "bool type yes",
			val:       "yes",
			fieldType: schema.FieldTypeBool,
			expected:  true,
		},
		{
			name:      "bool type false",
			val:       "false",
			fieldType: schema.FieldTypeBool,
			expected:  false,
		},
		{
			name:      "bool type 0",
			val:       "0",
			fieldType: schema.FieldTypeBool,
			expected:  false,
		},
		{
			name:      "unknown type returns string",
			val:       "test",
			fieldType: schema.FieldType("unknown"),
			expected:  "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertInput(tt.val, tt.fieldType)
			if got != tt.expected {
				t.Errorf("convertInput(%q, %q) = %v, want %v", tt.val, tt.fieldType, got, tt.expected)
			}
		})
	}
}

func TestChannel_Name(t *testing.T) {
	c := &Channel{}
	if c.Name() != "cli" {
		t.Errorf("Name() = %q, want %q", c.Name(), "cli")
	}
}

func TestNew(t *testing.T) {
	c := New(nil, nil)
	if c == nil {
		t.Error("New should return non-nil channel")
	}
	if c.modules == nil {
		t.Error("modules map should be initialized")
	}
}

func TestNew_WithRootCmd(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	c := New(rootCmd, nil)
	if c == nil {
		t.Error("New should return non-nil channel")
	}
	if c.rootCmd != rootCmd {
		t.Error("rootCmd should be set")
	}
	if c.formatters == nil {
		t.Error("formatters should be initialized")
	}
	if c.validator == nil {
		t.Error("validator should be initialized")
	}
}

func TestChannel_Start(t *testing.T) {
	c := New(nil, nil)
	err := c.Start(context.Background())
	if err != nil {
		t.Errorf("Start should return nil, got %v", err)
	}
}

func TestChannel_Stop(t *testing.T) {
	c := New(nil, nil)
	err := c.Stop(context.Background())
	if err != nil {
		t.Errorf("Stop should return nil, got %v", err)
	}
}

func TestChannel_Register_DisabledCLI(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	c := New(rootCmd, nil)

	// Module with CLI disabled
	mod := convention.Derived{
		Source: schema.Module{
			Name: "test",
			Channels: schema.Channels{
				CLI: schema.CLIChannel{
					Serve: schema.CLIServe{Enabled: false},
				},
			},
		},
	}

	err := c.Register(mod)
	if err != nil {
		t.Errorf("Register should return nil for disabled CLI, got %v", err)
	}

	// Verify module was not added
	if _, ok := c.modules["test"]; ok {
		t.Error("Module should not be added when CLI is disabled")
	}
}

func TestChannel_Register_NilRootCmd(t *testing.T) {
	c := New(nil, nil)

	// Module with CLI enabled
	mod := convention.Derived{
		Source: schema.Module{
			Name: "test",
			Channels: schema.Channels{
				CLI: schema.CLIChannel{
					Serve: schema.CLIServe{Enabled: true},
				},
			},
		},
	}

	err := c.Register(mod)
	if err != nil {
		t.Errorf("Register should return nil for nil rootCmd, got %v", err)
	}
}

func TestChannel_Register_WithModule(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	c := New(rootCmd, nil)

	// Module with CLI enabled and actions
	mod := convention.Derived{
		Source: schema.Module{
			Name: "user",
			Channels: schema.Channels{
				CLI: schema.CLIChannel{
					Serve: schema.CLIServe{Enabled: true},
				},
			},
		},
		Plural: "users",
		Actions: []convention.DerivedAction{
			{
				Type:        schema.ActionTypeList,
				Name:        "list",
				Description: "List users",
			},
			{
				Type:        schema.ActionTypeGet,
				Name:        "get",
				Description: "Get a user",
			},
			{
				Type:        schema.ActionTypeCreate,
				Name:        "create",
				Description: "Create a user",
			},
			{
				Type:        schema.ActionTypeUpdate,
				Name:        "update",
				Description: "Update a user",
			},
			{
				Type:        schema.ActionTypeDelete,
				Name:        "delete",
				Description: "Delete a user",
			},
		},
	}

	err := c.Register(mod)
	if err != nil {
		t.Errorf("Register should return nil, got %v", err)
	}

	// Verify module was added
	if _, ok := c.modules["user"]; !ok {
		t.Error("Module should be added")
	}

	// Verify module command was added to root
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "users" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Module command 'users' should be added to rootCmd")
	}
}

func TestChannel_buildActionCommand_List(t *testing.T) {
	c := New(nil, nil)
	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
	}
	action := convention.DerivedAction{
		Type:        schema.ActionTypeList,
		Name:        "list",
		Description: "List users",
	}

	cmd := c.buildActionCommand(mod, action)
	if cmd == nil {
		t.Error("buildActionCommand should return a command for list action")
	}
	if cmd.Use != "list" {
		t.Errorf("Command use should be 'list', got %q", cmd.Use)
	}
}

func TestChannel_buildActionCommand_Get(t *testing.T) {
	c := New(nil, nil)
	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
	}
	action := convention.DerivedAction{
		Type:        schema.ActionTypeGet,
		Name:        "get",
		Description: "Get a user",
	}

	cmd := c.buildActionCommand(mod, action)
	if cmd == nil {
		t.Error("buildActionCommand should return a command for get action")
	}
	if cmd.Use != "get <id-or-name>" {
		t.Errorf("Command use should be 'get <id-or-name>', got %q", cmd.Use)
	}
}

func TestChannel_buildActionCommand_Create(t *testing.T) {
	c := New(nil, nil)
	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
	}
	action := convention.DerivedAction{
		Type:        schema.ActionTypeCreate,
		Name:        "create",
		Description: "Create a user",
		Input: []convention.ActionInput{
			{Name: "name", Type: schema.FieldTypeString, Required: true},
			{Name: "email", Type: schema.FieldTypeString},
		},
	}

	cmd := c.buildActionCommand(mod, action)
	if cmd == nil {
		t.Error("buildActionCommand should return a command for create action")
	}
	if cmd.Use != "create" {
		t.Errorf("Command use should be 'create', got %q", cmd.Use)
	}
}

func TestChannel_buildActionCommand_Update(t *testing.T) {
	c := New(nil, nil)
	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
	}
	action := convention.DerivedAction{
		Type:        schema.ActionTypeUpdate,
		Name:        "update",
		Description: "Update a user",
		Input: []convention.ActionInput{
			{Name: "name", Type: schema.FieldTypeString},
		},
	}

	cmd := c.buildActionCommand(mod, action)
	if cmd == nil {
		t.Error("buildActionCommand should return a command for update action")
	}
	if cmd.Use != "update <id-or-name>" {
		t.Errorf("Command use should be 'update <id-or-name>', got %q", cmd.Use)
	}
}

func TestChannel_buildActionCommand_Delete(t *testing.T) {
	c := New(nil, nil)
	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
	}
	action := convention.DerivedAction{
		Type:        schema.ActionTypeDelete,
		Name:        "delete",
		Description: "Delete a user",
		Confirm:     true,
	}

	cmd := c.buildActionCommand(mod, action)
	if cmd == nil {
		t.Error("buildActionCommand should return a command for delete action")
	}
	if cmd.Use != "delete <id-or-name>" {
		t.Errorf("Command use should be 'delete <id-or-name>', got %q", cmd.Use)
	}

	// Check force flag exists
	f := cmd.Flags().Lookup("force")
	if f == nil {
		t.Error("Delete command should have 'force' flag")
	}
}

func TestChannel_buildActionCommand_Custom(t *testing.T) {
	c := New(nil, nil)
	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
	}
	action := convention.DerivedAction{
		Type:        schema.ActionTypeCustom,
		Name:        "activate",
		Description: "Activate a user",
		Input: []convention.ActionInput{
			{Name: "reason", Type: schema.FieldTypeString, Default: "default reason"},
		},
	}

	cmd := c.buildActionCommand(mod, action)
	if cmd == nil {
		t.Error("buildActionCommand should return a command for custom action")
	}
	if cmd.Use != "activate <id-or-name>" {
		t.Errorf("Command use should be 'activate <id-or-name>', got %q", cmd.Use)
	}
}

func TestChannel_buildActionCommand_Unknown(t *testing.T) {
	c := New(nil, nil)
	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
	}
	action := convention.DerivedAction{
		Type: schema.ActionType(99), // Unknown action type
		Name: "unknown",
	}

	cmd := c.buildActionCommand(mod, action)
	if cmd != nil {
		t.Error("buildActionCommand should return nil for unknown action type")
	}
}

func TestChannel_addOutputFlags(t *testing.T) {
	c := New(nil, nil)
	cmd := &cobra.Command{Use: "test"}

	c.addOutputFlags(cmd)

	// Check output flag
	f := cmd.Flags().Lookup("output")
	if f == nil {
		t.Error("addOutputFlags should add 'output' flag")
	}

	// Check no-header flag
	f = cmd.Flags().Lookup("no-header")
	if f == nil {
		t.Error("addOutputFlags should add 'no-header' flag")
	}

	// Check compact flag
	f = cmd.Flags().Lookup("compact")
	if f == nil {
		t.Error("addOutputFlags should add 'compact' flag")
	}
}

func TestChannel_getFormatter(t *testing.T) {
	c := New(nil, nil)
	cmd := &cobra.Command{Use: "test"}
	c.addOutputFlags(cmd)

	// Test default formatter
	f := c.getFormatter(cmd)
	if f == nil {
		t.Error("getFormatter should return a formatter")
	}

	// Test with json output
	cmd.Flags().Set("output", "json")
	f = c.getFormatter(cmd)
	if f == nil {
		t.Error("getFormatter should return a formatter for json")
	}

	// Test with invalid output (should return default)
	cmd.Flags().Set("output", "invalid")
	f = c.getFormatter(cmd)
	if f == nil {
		t.Error("getFormatter should return default formatter for invalid output")
	}
}

func TestChannel_getFormatOptions(t *testing.T) {
	c := New(nil, nil)
	cmd := &cobra.Command{Use: "test"}
	c.addOutputFlags(cmd)

	// Test default options
	opts := c.getFormatOptions(cmd)
	if opts.NoHeader {
		t.Error("NoHeader should be false by default")
	}
	if opts.Compact {
		t.Error("Compact should be false by default")
	}
	if opts.MaxWidth != 40 {
		t.Errorf("MaxWidth should be 40, got %d", opts.MaxWidth)
	}

	// Test with flags set
	cmd.Flags().Set("no-header", "true")
	cmd.Flags().Set("compact", "true")
	opts = c.getFormatOptions(cmd)
	if !opts.NoHeader {
		t.Error("NoHeader should be true when flag is set")
	}
	if !opts.Compact {
		t.Error("Compact should be true when flag is set")
	}
}

func TestChannel_buildListCommand_Flags(t *testing.T) {
	c := New(nil, nil)
	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
	}
	action := convention.DerivedAction{
		Type:        schema.ActionTypeList,
		Name:        "list",
		Description: "List users",
	}

	cmd := c.buildListCommand(mod, action)

	// Check limit flag
	f := cmd.Flags().Lookup("limit")
	if f == nil {
		t.Error("list command should have 'limit' flag")
	}
	if f.Shorthand != "l" {
		t.Errorf("limit flag shorthand should be 'l', got %q", f.Shorthand)
	}

	// Check offset flag
	f = cmd.Flags().Lookup("offset")
	if f == nil {
		t.Error("list command should have 'offset' flag")
	}
	if f.Shorthand != "o" {
		t.Errorf("offset flag shorthand should be 'o', got %q", f.Shorthand)
	}
}

func TestChannel_buildCreateCommand_Flags(t *testing.T) {
	c := New(nil, nil)
	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
	}
	action := convention.DerivedAction{
		Type:        schema.ActionTypeCreate,
		Name:        "create",
		Description: "Create a user",
		Input: []convention.ActionInput{
			{Name: "name", Type: schema.FieldTypeString, Required: true, Default: "John"},
			{Name: "email", Type: schema.FieldTypeString, Required: false},
		},
	}

	cmd := c.buildCreateCommand(mod, action)

	// Check name flag (required)
	f := cmd.Flags().Lookup("name")
	if f == nil {
		t.Error("create command should have 'name' flag")
	}
	if f.DefValue != "John" {
		t.Errorf("name flag default should be 'John', got %q", f.DefValue)
	}

	// Check email flag (optional)
	f = cmd.Flags().Lookup("email")
	if f == nil {
		t.Error("create command should have 'email' flag")
	}
}

func TestChannel_buildUpdateCommand_Flags(t *testing.T) {
	c := New(nil, nil)
	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
	}
	action := convention.DerivedAction{
		Type:        schema.ActionTypeUpdate,
		Name:        "update",
		Description: "Update a user",
		Input: []convention.ActionInput{
			{Name: "name", Type: schema.FieldTypeString},
			{Name: "status", Type: schema.FieldTypeString},
		},
	}

	cmd := c.buildUpdateCommand(mod, action)

	// Check name flag
	f := cmd.Flags().Lookup("name")
	if f == nil {
		t.Error("update command should have 'name' flag")
	}

	// Check status flag
	f = cmd.Flags().Lookup("status")
	if f == nil {
		t.Error("update command should have 'status' flag")
	}
}

func TestChannel_Register_WithCustomAction(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	c := New(rootCmd, nil)

	mod := convention.Derived{
		Source: schema.Module{
			Name: "user",
			Channels: schema.Channels{
				CLI: schema.CLIChannel{
					Serve: schema.CLIServe{Enabled: true},
				},
			},
		},
		Plural: "users",
		Actions: []convention.DerivedAction{
			{
				Type:        schema.ActionTypeCustom,
				Name:        "deactivate",
				Description: "Deactivate a user",
			},
		},
	}

	err := c.Register(mod)
	if err != nil {
		t.Errorf("Register should return nil, got %v", err)
	}

	// Find the users command
	var usersCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "users" {
			usersCmd = cmd
			break
		}
	}

	if usersCmd == nil {
		t.Fatal("users command not found")
	}

	// Find the deactivate subcommand
	found := false
	for _, cmd := range usersCmd.Commands() {
		if cmd.Use == "deactivate <id-or-name>" {
			found = true
			break
		}
	}
	if !found {
		t.Error("deactivate subcommand should be added")
	}
}

func TestFormatPromptLabel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single word",
			input:    "name",
			expected: "Name",
		},
		{
			name:     "snake_case",
			input:    "first_name",
			expected: "First Name",
		},
		{
			name:     "multiple underscores",
			input:    "user_email_address",
			expected: "User Email Address",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "already capitalized",
			input:    "Email",
			expected: "Email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatPromptLabel(tt.input)
			if got != tt.expected {
				t.Errorf("formatPromptLabel(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNewPrompter(t *testing.T) {
	p := NewPrompter()
	if p == nil {
		t.Error("NewPrompter should return non-nil prompter")
	}
	if p.reader == nil {
		t.Error("reader should be initialized")
	}
}

func TestChannel_formatList(t *testing.T) {
	c := New(nil, nil)
	cmd := &cobra.Command{Use: "test"}
	c.addOutputFlags(cmd)

	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
		Fields: []convention.DerivedField{
			{Name: "id", Type: schema.FieldTypeUUID},
			{Name: "name", Type: schema.FieldTypeString},
		},
	}

	records := []map[string]any{
		{"id": "123", "name": "John"},
		{"id": "456", "name": "Jane"},
	}

	// Test with json output (should not error)
	cmd.Flags().Set("output", "json")
	err := c.formatList(cmd, mod, records)
	if err != nil {
		t.Errorf("formatList should not return error, got %v", err)
	}
}

func TestChannel_formatRecord(t *testing.T) {
	c := New(nil, nil)
	cmd := &cobra.Command{Use: "test"}
	c.addOutputFlags(cmd)

	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
		Fields: []convention.DerivedField{
			{Name: "id", Type: schema.FieldTypeUUID},
			{Name: "name", Type: schema.FieldTypeString},
		},
	}

	record := map[string]any{"id": "123", "name": "John"}

	// Test with json output
	cmd.Flags().Set("output", "json")
	err := c.formatRecord(cmd, mod, record)
	if err != nil {
		t.Errorf("formatRecord should not return error, got %v", err)
	}
}

func TestChannel_formatError(t *testing.T) {
	c := New(nil, nil)
	cmd := &cobra.Command{Use: "test"}
	c.addOutputFlags(cmd)

	testErr := fmt.Errorf("test error")
	err := c.formatError(cmd, testErr)
	if err != testErr {
		t.Errorf("formatError should return the same error, got %v", err)
	}
}

func TestChannel_getFormatter_EmptyOutput(t *testing.T) {
	c := New(nil, nil)
	cmd := &cobra.Command{Use: "test"}
	// No output flag set - should use default

	f := c.getFormatter(cmd)
	if f == nil {
		t.Error("getFormatter should return default formatter when output not set")
	}
}
