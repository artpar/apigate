package tty

import (
	"context"
	"testing"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/schema"
)

func TestNew(t *testing.T) {
	c := New(nil)
	if c == nil {
		t.Fatal("New should return non-nil channel")
	}
	if c.modules == nil {
		t.Error("modules should be initialized")
	}
	if c.prompt != "apigate> " {
		t.Errorf("prompt = %q, want %q", c.prompt, "apigate> ")
	}
	if !c.showStats {
		t.Error("showStats should be true by default")
	}
}

func TestChannel_Name(t *testing.T) {
	c := &Channel{}
	if c.Name() != "tty" {
		t.Errorf("Name() = %q, want %q", c.Name(), "tty")
	}
}

func TestChannel_Register(t *testing.T) {
	c := New(nil)

	mod := convention.Derived{
		Source: schema.Module{Name: "test"},
		Plural: "tests",
	}

	err := c.Register(mod)
	if err != nil {
		t.Errorf("Register should not error: %v", err)
	}

	if _, exists := c.modules["test"]; !exists {
		t.Error("Module should be registered")
	}
}

func TestChannel_Start(t *testing.T) {
	c := New(nil)
	err := c.Start(context.Background())
	if err != nil {
		t.Errorf("Start should not error: %v", err)
	}
}

func TestChannel_Stop(t *testing.T) {
	c := New(nil)
	err := c.Stop(context.Background())
	if err != nil {
		t.Errorf("Stop should not error: %v", err)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatBytes(tt.input)
			if result != tt.expected {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCaptureStats(t *testing.T) {
	stats := captureStats()

	// MemStats should have non-zero Sys (memory from system)
	if stats.Sys == 0 {
		t.Error("captureStats() Sys should be non-zero")
	}
}

func TestChannel_ShowHelp(t *testing.T) {
	c := New(nil)

	// Register a module
	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
		Actions: []convention.DerivedAction{
			{Name: "list", Type: schema.ActionTypeList},
			{Name: "get", Type: schema.ActionTypeGet},
		},
	}
	c.Register(mod)

	// showHelp should not error
	err := c.showHelp([]string{})
	if err != nil {
		t.Errorf("showHelp() error: %v", err)
	}
}

func TestChannel_ShowHelp_WithModule(t *testing.T) {
	c := New(nil)

	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
		Fields: []convention.DerivedField{
			{Name: "id", Type: schema.FieldTypeString, Required: true},
			{Name: "email", Type: schema.FieldTypeString, Required: true},
		},
		Actions: []convention.DerivedAction{
			{Name: "list", Type: schema.ActionTypeList},
			{Name: "get", Type: schema.ActionTypeGet},
		},
	}
	c.Register(mod)

	err := c.showHelp([]string{"user"})
	if err != nil {
		t.Errorf("showHelp('user') error: %v", err)
	}
}

func TestChannel_ListModules(t *testing.T) {
	c := New(nil)

	// Register modules
	c.Register(convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
	})
	c.Register(convention.Derived{
		Source: schema.Module{Name: "item"},
		Plural: "items",
	})

	err := c.listModules()
	if err != nil {
		t.Errorf("listModules() error: %v", err)
	}
}

func TestChannel_ListModules_Empty(t *testing.T) {
	c := New(nil)

	err := c.listModules()
	if err != nil {
		t.Errorf("listModules() with no modules error: %v", err)
	}
}

func TestExecStats(t *testing.T) {
	stats := ExecStats{
		Duration: 100,
		MemAlloc: 1024,
		MemTotal: 2048,
		NumGC:    1,
	}

	if stats.Duration != 100 {
		t.Errorf("Duration = %v, want 100", stats.Duration)
	}
	if stats.MemAlloc != 1024 {
		t.Errorf("MemAlloc = %d, want 1024", stats.MemAlloc)
	}
}

func TestChannel_Execute_EmptyLine(t *testing.T) {
	c := New(nil)

	// Empty line should not error
	err := c.execute(context.Background(), "")
	if err != nil {
		t.Errorf("execute('') error: %v", err)
	}

	err = c.execute(context.Background(), "   ")
	if err != nil {
		t.Errorf("execute('   ') error: %v", err)
	}
}

func TestChannel_Execute_Help(t *testing.T) {
	c := New(nil)

	err := c.execute(context.Background(), "help")
	if err != nil {
		t.Errorf("execute('help') error: %v", err)
	}
}

func TestChannel_Execute_Modules(t *testing.T) {
	c := New(nil)

	err := c.execute(context.Background(), "modules")
	if err != nil {
		t.Errorf("execute('modules') error: %v", err)
	}
}

func TestChannel_Execute_Exit(t *testing.T) {
	c := New(nil)
	c.running = true

	err := c.execute(context.Background(), "exit")
	if err != nil {
		t.Errorf("execute('exit') error: %v", err)
	}

	if c.running {
		t.Error("running should be false after exit")
	}
}

func TestChannel_Execute_Quit(t *testing.T) {
	c := New(nil)
	c.running = true

	err := c.execute(context.Background(), "quit")
	if err != nil {
		t.Errorf("execute('quit') error: %v", err)
	}

	if c.running {
		t.Error("running should be false after quit")
	}
}

func TestChannel_Execute_UnknownCommand(t *testing.T) {
	c := New(nil)

	// Unknown command should error
	err := c.execute(context.Background(), "unknowncommand")
	if err == nil {
		t.Error("execute('unknowncommand') should error")
	}
}

func TestChannel_Execute_Stats(t *testing.T) {
	c := New(nil)

	err := c.execute(context.Background(), "stats")
	if err != nil {
		t.Errorf("execute('stats') error: %v", err)
	}
}

func TestChannel_Execute_UnknownModule(t *testing.T) {
	c := New(nil)

	err := c.execute(context.Background(), "unknownmodule list")
	if err == nil {
		t.Error("execute('unknownmodule list') should error for unknown module")
	}
}

func TestChannel_Execute_ModuleHelp(t *testing.T) {
	c := New(nil)

	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
		Actions: []convention.DerivedAction{
			{Name: "list", Type: schema.ActionTypeList},
		},
	}
	c.Register(mod)

	// help <module> should work
	err := c.execute(context.Background(), "help user")
	if err != nil {
		t.Errorf("execute('help user') error: %v", err)
	}
}

func TestChannel_ListRecords_NoArgs(t *testing.T) {
	c := New(nil)

	err := c.listRecords(context.Background(), []string{})
	if err == nil {
		t.Error("listRecords with no args should error")
	}
}

func TestChannel_GetRecord_NoArgs(t *testing.T) {
	c := New(nil)

	err := c.getRecord(context.Background(), []string{})
	if err == nil {
		t.Error("getRecord with no args should error")
	}
}

func TestChannel_GetRecord_MissingID(t *testing.T) {
	c := New(nil)

	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
	}
	c.Register(mod)

	err := c.getRecord(context.Background(), []string{"user"})
	if err == nil {
		t.Error("getRecord without ID should error")
	}
}

func TestChannel_CreateRecord_NoArgs(t *testing.T) {
	c := New(nil)

	err := c.createRecord(context.Background(), []string{})
	if err == nil {
		t.Error("createRecord with no args should error")
	}
}

func TestChannel_UpdateRecord_NoArgs(t *testing.T) {
	c := New(nil)

	err := c.updateRecord(context.Background(), []string{})
	if err == nil {
		t.Error("updateRecord with no args should error")
	}
}

func TestChannel_DeleteRecord_NoArgs(t *testing.T) {
	c := New(nil)

	err := c.deleteRecord(context.Background(), []string{})
	if err == nil {
		t.Error("deleteRecord with no args should error")
	}
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected []string
	}{
		{
			name:     "simple args",
			line:     "list users",
			expected: []string{"list", "users"},
		},
		{
			name:     "empty line",
			line:     "",
			expected: nil,
		},
		{
			name:     "quoted string",
			line:     `create user name="John Doe"`,
			expected: []string{"create", "user", "name=John Doe"},
		},
		{
			name:     "single quotes",
			line:     "create user name='Jane Doe'",
			expected: []string{"create", "user", "name=Jane Doe"},
		},
		{
			name:     "multiple spaces",
			line:     "list    users",
			expected: []string{"list", "users"},
		},
		{
			name:     "tabs",
			line:     "list\tusers",
			expected: []string{"list", "users"},
		},
		{
			name:     "nested quotes in value",
			line:     `set value="he said 'hello'"`,
			expected: []string{"set", "value=he said 'hello'"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseArgs(tt.line)
			if len(got) != len(tt.expected) {
				t.Errorf("parseArgs(%q) = %v, want %v", tt.line, got, tt.expected)
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("parseArgs(%q)[%d] = %q, want %q", tt.line, i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestParseKeyValues(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected map[string]any
	}{
		{
			name:     "empty args",
			args:     []string{},
			expected: map[string]any{},
		},
		{
			name:     "single key-value",
			args:     []string{"name=John"},
			expected: map[string]any{"name": "John"},
		},
		{
			name:     "multiple key-values",
			args:     []string{"name=John", "age=30"},
			expected: map[string]any{"name": "John", "age": "30"},
		},
		{
			name:     "value with equals sign",
			args:     []string{"equation=a=b"},
			expected: map[string]any{"equation": "a=b"},
		},
		{
			name:     "skip malformed",
			args:     []string{"noequals"},
			expected: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseKeyValues(tt.args)
			if len(got) != len(tt.expected) {
				t.Errorf("parseKeyValues(%v) len = %d, want %d", tt.args, len(got), len(tt.expected))
			}
			for k, v := range tt.expected {
				if got[k] != v {
					t.Errorf("parseKeyValues(%v)[%s] = %v, want %v", tt.args, k, got[k], v)
				}
			}
		})
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name     string
		val      any
		expected string
	}{
		{
			name:     "nil",
			val:      nil,
			expected: "",
		},
		{
			name:     "string",
			val:      "hello",
			expected: "hello",
		},
		{
			name:     "bool true",
			val:      true,
			expected: "yes",
		},
		{
			name:     "bool false",
			val:      false,
			expected: "no",
		},
		{
			name:     "bytes",
			val:      []byte("data"),
			expected: "[binary]",
		},
		{
			name:     "number",
			val:      42,
			expected: "42",
		},
		{
			name:     "float",
			val:      3.14,
			expected: "3.14",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatValue(tt.val)
			if got != tt.expected {
				t.Errorf("formatValue(%v) = %q, want %q", tt.val, got, tt.expected)
			}
		})
	}
}

func TestChannel_ShowHelp_CustomActions(t *testing.T) {
	c := New(nil)

	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
		Actions: []convention.DerivedAction{
			{Name: "list", Type: schema.ActionTypeList},
			{Name: "get", Type: schema.ActionTypeGet},
			{Name: "activate", Type: schema.ActionTypeCustom, Description: "Activate a user"},
			{Name: "deactivate", Type: schema.ActionTypeCustom, Description: "Deactivate a user"},
		},
	}
	c.Register(mod)

	err := c.showHelp([]string{"user"})
	if err != nil {
		t.Errorf("showHelp('user') with custom actions error: %v", err)
	}
}

func TestChannel_Execute_HelpShorthand(t *testing.T) {
	c := New(nil)

	// Test 'h' shorthand
	err := c.execute(context.Background(), "h")
	if err != nil {
		t.Errorf("execute('h') error: %v", err)
	}

	// Test '?' shorthand
	err = c.execute(context.Background(), "?")
	if err != nil {
		t.Errorf("execute('?') error: %v", err)
	}
}

func TestChannel_Execute_QShorthand(t *testing.T) {
	c := New(nil)
	c.running = true

	err := c.execute(context.Background(), "q")
	if err != nil {
		t.Errorf("execute('q') error: %v", err)
	}

	if c.running {
		t.Error("running should be false after 'q'")
	}
}

func TestChannel_ListRecords_UnknownModule(t *testing.T) {
	c := New(nil)

	err := c.listRecords(context.Background(), []string{"unknownmodule"})
	if err == nil {
		t.Error("listRecords for unknown module should error")
	}
}

func TestChannel_GetRecord_UnknownModule(t *testing.T) {
	c := New(nil)

	err := c.getRecord(context.Background(), []string{"unknownmodule", "123"})
	if err == nil {
		t.Error("getRecord for unknown module should error")
	}
}

func TestChannel_CreateRecord_UnknownModule(t *testing.T) {
	c := New(nil)

	err := c.createRecord(context.Background(), []string{"unknownmodule", "name=test"})
	if err == nil {
		t.Error("createRecord for unknown module should error")
	}
}

func TestChannel_UpdateRecord_UnknownModule(t *testing.T) {
	c := New(nil)

	err := c.updateRecord(context.Background(), []string{"unknownmodule", "123", "name=test"})
	if err == nil {
		t.Error("updateRecord for unknown module should error")
	}
}

func TestChannel_DeleteRecord_UnknownModule(t *testing.T) {
	c := New(nil)

	err := c.deleteRecord(context.Background(), []string{"unknownmodule", "123"})
	if err == nil {
		t.Error("deleteRecord for unknown module should error")
	}
}

func TestChannel_ExecuteModuleCommand_NoArgs(t *testing.T) {
	c := New(nil)

	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
	}
	c.Register(mod)

	// Empty args should show help
	err := c.executeModuleCommand(context.Background(), mod, []string{})
	if err != nil {
		t.Errorf("executeModuleCommand with no args error: %v", err)
	}
}

func TestChannel_ExecuteModuleCommand_UnknownAction(t *testing.T) {
	c := New(nil)

	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
		Actions: []convention.DerivedAction{
			{Name: "list", Type: schema.ActionTypeList},
		},
	}
	c.Register(mod)

	err := c.executeModuleCommand(context.Background(), mod, []string{"unknownaction"})
	if err == nil {
		t.Error("executeModuleCommand with unknown action should error")
	}
}

func TestChannel_Execute_ListShorthand(t *testing.T) {
	c := New(nil)

	// 'ls' with no module should error
	err := c.execute(context.Background(), "ls")
	if err == nil {
		t.Error("execute('ls') without module should error")
	}
}

func TestChannel_Execute_ModuleNewAlias(t *testing.T) {
	c := New(nil)

	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
	}
	c.Register(mod)

	// 'user new' should call createRecord (will error without enough args)
	err := c.executeModuleCommand(context.Background(), mod, []string{"new"})
	if err == nil {
		t.Error("executeModuleCommand('new') without args should error")
	}
}

func TestChannel_Execute_ModuleSetAlias(t *testing.T) {
	c := New(nil)

	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
	}
	c.Register(mod)

	// 'user set' should call updateRecord (will error without enough args)
	err := c.executeModuleCommand(context.Background(), mod, []string{"set"})
	if err == nil {
		t.Error("executeModuleCommand('set') without args should error")
	}
}

func TestChannel_StatsToggle(t *testing.T) {
	c := New(nil)
	original := c.showStats

	err := c.execute(context.Background(), "stats")
	if err != nil {
		t.Errorf("execute('stats') error: %v", err)
	}

	if c.showStats == original {
		t.Error("stats command should toggle showStats")
	}

	// Toggle back
	err = c.execute(context.Background(), "stats")
	if err != nil {
		t.Errorf("execute('stats') error: %v", err)
	}

	if c.showStats != original {
		t.Error("stats command should toggle showStats back")
	}
}

func TestChannel_ExecuteModuleCommand_GetNoID(t *testing.T) {
	c := New(nil)

	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
	}
	c.Register(mod)

	// 'user get' with no ID should error
	err := c.executeModuleCommand(context.Background(), mod, []string{"get"})
	if err == nil {
		t.Error("executeModuleCommand('get') without ID should error")
	}
}

func TestChannel_ExecuteModuleCommand_ShowAlias(t *testing.T) {
	c := New(nil)

	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
	}
	c.Register(mod)

	// 'user show' with no ID should error
	err := c.executeModuleCommand(context.Background(), mod, []string{"show"})
	if err == nil {
		t.Error("executeModuleCommand('show') without ID should error")
	}
}

func TestChannel_ExecuteModuleCommand_DeleteNoID(t *testing.T) {
	c := New(nil)

	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
	}
	c.Register(mod)

	// 'user delete' with no ID should error
	err := c.executeModuleCommand(context.Background(), mod, []string{"delete"})
	if err == nil {
		t.Error("executeModuleCommand('delete') without ID should error")
	}
}

func TestChannel_ExecuteModuleCommand_RmAlias(t *testing.T) {
	c := New(nil)

	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
	}
	c.Register(mod)

	// 'user rm' with no ID should error
	err := c.executeModuleCommand(context.Background(), mod, []string{"rm"})
	if err == nil {
		t.Error("executeModuleCommand('rm') without ID should error")
	}
}

func TestChannel_ExecuteModuleCommand_CustomActionNoID(t *testing.T) {
	c := New(nil)

	mod := convention.Derived{
		Source: schema.Module{Name: "user"},
		Plural: "users",
		Actions: []convention.DerivedAction{
			{Name: "activate", Type: schema.ActionTypeCustom},
		},
	}
	c.Register(mod)

	// 'user activate' with no ID should error
	err := c.executeModuleCommand(context.Background(), mod, []string{"activate"})
	if err == nil {
		t.Error("executeModuleCommand('activate') without ID should error")
	}
}
