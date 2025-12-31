package schema

import (
	"testing"
)

func TestModuleIsCapability(t *testing.T) {
	tests := []struct {
		name     string
		module   Module
		expected bool
	}{
		{
			name:     "regular module (no capability)",
			module:   Module{Name: "user"},
			expected: false,
		},
		{
			name:     "capability module",
			module:   Module{Capability: "payment"},
			expected: true,
		},
		{
			name:     "empty capability string",
			module:   Module{Name: "user", Capability: ""},
			expected: false,
		},
		{
			name:     "both name and capability set",
			module:   Module{Name: "stripe_payment", Capability: "payment"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.module.IsCapability(); got != tt.expected {
				t.Errorf("Module.IsCapability() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestModuleStruct(t *testing.T) {
	// Test that Module struct can hold all expected fields
	mod := Module{
		Name:        "user",
		Capability:  "",
		Description: "User module",
		Schema: map[string]Field{
			"email": {Type: FieldTypeEmail},
			"name":  {Type: FieldTypeString},
		},
		Actions: map[string]Action{
			"activate": {Set: map[string]string{"status": "active"}},
		},
		Channels: Channels{
			HTTP: HTTPChannel{
				Serve: HTTPServe{Enabled: true},
			},
		},
		Hooks: map[string][]Hook{
			"after_create": {{Emit: "user.created"}},
		},
		Meta: ModuleMeta{
			Version:     "1.0.0",
			Description: "User management module",
			Depends:     []string{"auth"},
		},
	}

	if mod.Name != "user" {
		t.Error("Module.Name not set correctly")
	}
	if mod.IsCapability() {
		t.Error("Module should not be a capability")
	}
	if mod.Description != "User module" {
		t.Error("Module.Description not set correctly")
	}
	if len(mod.Schema) != 2 {
		t.Error("Module.Schema not set correctly")
	}
	if len(mod.Actions) != 1 {
		t.Error("Module.Actions not set correctly")
	}
	if !mod.Channels.HTTP.Serve.Enabled {
		t.Error("Module.Channels not set correctly")
	}
	if len(mod.Hooks) != 1 {
		t.Error("Module.Hooks not set correctly")
	}
	if mod.Meta.Version != "1.0.0" {
		t.Error("Module.Meta.Version not set correctly")
	}
}

func TestModuleMetaStruct(t *testing.T) {
	meta := ModuleMeta{
		Version:     "2.0.0",
		Description: "Test module",
		Depends:     []string{"auth", "billing"},
		Implements:  []string{"payment", "subscription"},
		Requires: map[string]ModuleRequirement{
			"payment": {
				Capability:  "payment",
				Required:    true,
				Description: "Payment provider",
				Default:     "stripe",
			},
		},
		Icon:        "user-icon",
		DisplayName: "User Management",
		Plural:      "users",
	}

	if meta.Version != "2.0.0" {
		t.Error("ModuleMeta.Version not set correctly")
	}
	if meta.Description != "Test module" {
		t.Error("ModuleMeta.Description not set correctly")
	}
	if len(meta.Depends) != 2 {
		t.Error("ModuleMeta.Depends not set correctly")
	}
	if len(meta.Implements) != 2 {
		t.Error("ModuleMeta.Implements not set correctly")
	}
	if len(meta.Requires) != 1 {
		t.Error("ModuleMeta.Requires not set correctly")
	}
	if meta.Requires["payment"].Capability != "payment" {
		t.Error("ModuleMeta.Requires[payment].Capability not set correctly")
	}
	if !meta.Requires["payment"].Required {
		t.Error("ModuleMeta.Requires[payment].Required not set correctly")
	}
	if meta.Icon != "user-icon" {
		t.Error("ModuleMeta.Icon not set correctly")
	}
	if meta.DisplayName != "User Management" {
		t.Error("ModuleMeta.DisplayName not set correctly")
	}
	if meta.Plural != "users" {
		t.Error("ModuleMeta.Plural not set correctly")
	}
}

func TestModuleRequirementStruct(t *testing.T) {
	req := ModuleRequirement{
		Capability:  "storage",
		Required:    true,
		Description: "File storage provider",
		Default:     "s3",
	}

	if req.Capability != "storage" {
		t.Error("ModuleRequirement.Capability not set correctly")
	}
	if !req.Required {
		t.Error("ModuleRequirement.Required not set correctly")
	}
	if req.Description != "File storage provider" {
		t.Error("ModuleRequirement.Description not set correctly")
	}
	if req.Default != "s3" {
		t.Error("ModuleRequirement.Default not set correctly")
	}
}

func TestHookStruct(t *testing.T) {
	tests := []struct {
		name string
		hook Hook
	}{
		{
			name: "emit shorthand",
			hook: Hook{Emit: "user.created"},
		},
		{
			name: "call shorthand",
			hook: Hook{Call: "send_welcome_email"},
		},
		{
			name: "email hook",
			hook: Hook{
				Type:     "email",
				Template: "welcome",
				To:       "{{.email}}",
			},
		},
		{
			name: "emit explicit",
			hook: Hook{
				Type:  "emit",
				Event: "user.activated",
			},
		},
		{
			name: "webhook hook",
			hook: Hook{
				Type:   "webhook",
				URL:    "https://example.com/webhook",
				Method: "POST",
				Body:   map[string]string{"user_id": "{{.id}}"},
			},
		},
		{
			name: "conditional hook",
			hook: Hook{
				Emit: "premium.activated",
				When: "status == 'premium'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the struct can hold the values without error
			h := tt.hook
			_ = h
		})
	}
}

func TestHookEmitShorthand(t *testing.T) {
	hook := Hook{Emit: "user.created"}

	if hook.Emit != "user.created" {
		t.Errorf("Hook.Emit = %q, want %q", hook.Emit, "user.created")
	}
	if hook.Call != "" {
		t.Error("Hook.Call should be empty for emit hook")
	}
}

func TestHookCallShorthand(t *testing.T) {
	hook := Hook{Call: "send_notification"}

	if hook.Call != "send_notification" {
		t.Errorf("Hook.Call = %q, want %q", hook.Call, "send_notification")
	}
	if hook.Emit != "" {
		t.Error("Hook.Emit should be empty for call hook")
	}
}

func TestHookWebhook(t *testing.T) {
	hook := Hook{
		Type:   "webhook",
		URL:    "https://api.example.com/callback",
		Method: "POST",
		Body: map[string]string{
			"event": "{{.event}}",
			"data":  "{{.data}}",
		},
		When: "status == 'active'",
	}

	if hook.Type != "webhook" {
		t.Error("Hook.Type not set correctly")
	}
	if hook.URL != "https://api.example.com/callback" {
		t.Error("Hook.URL not set correctly")
	}
	if hook.Method != "POST" {
		t.Error("Hook.Method not set correctly")
	}
	if len(hook.Body) != 2 {
		t.Error("Hook.Body not set correctly")
	}
	if hook.When != "status == 'active'" {
		t.Error("Hook.When not set correctly")
	}
}

func TestHookEmail(t *testing.T) {
	hook := Hook{
		Type:     "email",
		Template: "welcome_email",
		To:       "{{.email}}",
	}

	if hook.Type != "email" {
		t.Error("Hook.Type not set correctly")
	}
	if hook.Template != "welcome_email" {
		t.Error("Hook.Template not set correctly")
	}
	if hook.To != "{{.email}}" {
		t.Error("Hook.To not set correctly")
	}
}
