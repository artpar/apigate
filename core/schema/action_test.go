package schema

import (
	"testing"
)

func TestActionTypeString(t *testing.T) {
	tests := []struct {
		actionType ActionType
		want       string
	}{
		{ActionTypeList, "list"},
		{ActionTypeGet, "get"},
		{ActionTypeCreate, "create"},
		{ActionTypeUpdate, "update"},
		{ActionTypeDelete, "delete"},
		{ActionTypeCustom, "custom"},
		{ActionType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.actionType.String(); got != tt.want {
				t.Errorf("ActionType.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestImplicitActions(t *testing.T) {
	actions := ImplicitActions()

	expected := []string{"list", "get", "create", "update", "delete"}
	if len(actions) != len(expected) {
		t.Errorf("ImplicitActions() returned %d actions, want %d", len(actions), len(expected))
	}

	for i, action := range expected {
		if actions[i] != action {
			t.Errorf("ImplicitActions()[%d] = %q, want %q", i, actions[i], action)
		}
	}
}

func TestIsImplicit(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"list", true},
		{"get", true},
		{"create", true},
		{"update", true},
		{"delete", true},
		{"custom", false},
		{"activate", false},
		{"deactivate", false},
		{"", false},
		{"LIST", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsImplicit(tt.name); got != tt.want {
				t.Errorf("IsImplicit(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestActionTypeFromName(t *testing.T) {
	tests := []struct {
		name string
		want ActionType
	}{
		{"list", ActionTypeList},
		{"get", ActionTypeGet},
		{"create", ActionTypeCreate},
		{"update", ActionTypeUpdate},
		{"delete", ActionTypeDelete},
		{"custom", ActionTypeCustom},
		{"activate", ActionTypeCustom},
		{"", ActionTypeCustom},
		{"LIST", ActionTypeCustom}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ActionTypeFromName(tt.name); got != tt.want {
				t.Errorf("ActionTypeFromName(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestActionStruct(t *testing.T) {
	// Test that Action struct can hold all expected fields
	action := Action{
		Set:         map[string]string{"status": "active"},
		Input:       []ActionInput{{Field: "name", Required: true}},
		Output:      []ActionOutput{{Name: "id", Type: "string"}},
		Auth:        "admin",
		Confirm:     true,
		Before:      []string{"validate"},
		After:       []string{"notify"},
		Description: "Test action",
		Internal:    true,
	}

	if action.Set["status"] != "active" {
		t.Error("Action.Set not set correctly")
	}
	if len(action.Input) != 1 || action.Input[0].Field != "name" {
		t.Error("Action.Input not set correctly")
	}
	if len(action.Output) != 1 || action.Output[0].Name != "id" {
		t.Error("Action.Output not set correctly")
	}
	if action.Auth != "admin" {
		t.Error("Action.Auth not set correctly")
	}
	if !action.Confirm {
		t.Error("Action.Confirm not set correctly")
	}
	if len(action.Before) != 1 || action.Before[0] != "validate" {
		t.Error("Action.Before not set correctly")
	}
	if len(action.After) != 1 || action.After[0] != "notify" {
		t.Error("Action.After not set correctly")
	}
	if action.Description != "Test action" {
		t.Error("Action.Description not set correctly")
	}
	if !action.Internal {
		t.Error("Action.Internal not set correctly")
	}
}

func TestActionInputStruct(t *testing.T) {
	input := ActionInput{
		Field:       "email",
		Name:        "user_email",
		Type:        "email",
		To:          "user",
		Required:    true,
		Default:     "test@example.com",
		Prompt:      true,
		PromptText:  "Enter your email:",
		Description: "User email address",
	}

	if input.Field != "email" {
		t.Error("ActionInput.Field not set correctly")
	}
	if input.Name != "user_email" {
		t.Error("ActionInput.Name not set correctly")
	}
	if input.Type != "email" {
		t.Error("ActionInput.Type not set correctly")
	}
	if input.To != "user" {
		t.Error("ActionInput.To not set correctly")
	}
	if !input.Required {
		t.Error("ActionInput.Required not set correctly")
	}
	if input.Default != "test@example.com" {
		t.Error("ActionInput.Default not set correctly")
	}
	if !input.Prompt {
		t.Error("ActionInput.Prompt not set correctly")
	}
	if input.PromptText != "Enter your email:" {
		t.Error("ActionInput.PromptText not set correctly")
	}
	if input.Description != "User email address" {
		t.Error("ActionInput.Description not set correctly")
	}
}

func TestActionOutputStruct(t *testing.T) {
	output := ActionOutput{
		Name:        "result",
		Type:        "string",
		Description: "The result of the action",
	}

	if output.Name != "result" {
		t.Error("ActionOutput.Name not set correctly")
	}
	if output.Type != "string" {
		t.Error("ActionOutput.Type not set correctly")
	}
	if output.Description != "The result of the action" {
		t.Error("ActionOutput.Description not set correctly")
	}
}
