package convention

import (
	"testing"

	"github.com/artpar/apigate/core/schema"
)

// -----------------------------------------------------------------------------
// Derive function tests
// -----------------------------------------------------------------------------

func TestDerive_EmptyModule(t *testing.T) {
	mod := schema.Module{
		Name:   "user",
		Schema: map[string]schema.Field{},
	}

	d := Derive(mod)

	if d.Source.Name != "user" {
		t.Errorf("expected Source.Name = 'user', got %q", d.Source.Name)
	}
	if d.Plural != "users" {
		t.Errorf("expected Plural = 'users', got %q", d.Plural)
	}
	if d.Table != "users" {
		t.Errorf("expected Table = 'users', got %q", d.Table)
	}

	// Should have 3 implicit fields: id, created_at, updated_at
	if len(d.Fields) != 3 {
		t.Errorf("expected 3 implicit fields, got %d", len(d.Fields))
	}

	// Should have 5 implicit CRUD actions
	if len(d.Actions) != 5 {
		t.Errorf("expected 5 implicit actions, got %d", len(d.Actions))
	}

	// Should have id as lookup
	if len(d.Lookups) != 1 || d.Lookups[0] != "id" {
		t.Errorf("expected Lookups = ['id'], got %v", d.Lookups)
	}
}

func TestDerive_WithUserDefinedFields(t *testing.T) {
	required := true
	mod := schema.Module{
		Name: "product",
		Schema: map[string]schema.Field{
			"name": {
				Type:     schema.FieldTypeString,
				Required: &required,
			},
			"price": {
				Type: schema.FieldTypeFloat,
			},
			"category": {
				Type:   schema.FieldTypeEnum,
				Values: []string{"electronics", "clothing", "food"},
			},
		},
	}

	d := Derive(mod)

	// 3 user fields + 3 implicit fields (id, created_at, updated_at)
	if len(d.Fields) != 6 {
		t.Errorf("expected 6 fields, got %d", len(d.Fields))
	}

	// Check for specific fields
	foundName := false
	foundPrice := false
	foundCategory := false
	for _, f := range d.Fields {
		switch f.Name {
		case "name":
			foundName = true
			if !f.Required {
				t.Error("name field should be required")
			}
			if f.Type != schema.FieldTypeString {
				t.Errorf("name field type should be string, got %v", f.Type)
			}
		case "price":
			foundPrice = true
			if f.Required {
				t.Error("price field should not be required")
			}
			if f.Type != schema.FieldTypeFloat {
				t.Errorf("price field type should be float, got %v", f.Type)
			}
		case "category":
			foundCategory = true
			if len(f.Values) != 3 {
				t.Errorf("category should have 3 values, got %d", len(f.Values))
			}
		}
	}

	if !foundName {
		t.Error("name field not found")
	}
	if !foundPrice {
		t.Error("price field not found")
	}
	if !foundCategory {
		t.Error("category field not found")
	}
}

func TestDerive_WithLookupField(t *testing.T) {
	mod := schema.Module{
		Name: "user",
		Schema: map[string]schema.Field{
			"email": {
				Type:   schema.FieldTypeEmail,
				Unique: true,
				Lookup: true,
			},
		},
	}

	d := Derive(mod)

	// Should have id and email as lookups
	if len(d.Lookups) != 2 {
		t.Errorf("expected 2 lookups (id, email), got %d: %v", len(d.Lookups), d.Lookups)
	}

	// Check the email field properties
	for _, f := range d.Fields {
		if f.Name == "email" {
			if !f.Unique {
				t.Error("email field should be unique")
			}
			if !f.Lookup {
				t.Error("email field should be a lookup")
			}
			if f.Type != schema.FieldTypeEmail {
				t.Errorf("email field type should be email, got %v", f.Type)
			}
		}
	}
}

func TestDerive_WithRefField(t *testing.T) {
	mod := schema.Module{
		Name: "order",
		Schema: map[string]schema.Field{
			"user_id": {
				Type: schema.FieldTypeRef,
				To:   "user",
			},
		},
	}

	d := Derive(mod)

	for _, f := range d.Fields {
		if f.Name == "user_id" {
			if f.Type != schema.FieldTypeRef {
				t.Errorf("user_id field type should be ref, got %v", f.Type)
			}
			if f.Ref != "user" {
				t.Errorf("user_id field ref should be 'user', got %q", f.Ref)
			}
		}
	}
}

func TestDerive_WithSecretField(t *testing.T) {
	mod := schema.Module{
		Name: "user",
		Schema: map[string]schema.Field{
			"password": {
				Type: schema.FieldTypeSecret,
			},
		},
	}

	d := Derive(mod)

	for _, f := range d.Fields {
		if f.Name == "password" {
			if !f.Internal {
				t.Error("password (secret) field should be internal")
			}
			if f.SQLType != "BLOB" {
				t.Errorf("password field SQL type should be BLOB, got %q", f.SQLType)
			}
		}
	}

	// Secret fields should prompt for input in create action
	for _, a := range d.Actions {
		if a.Name == "create" {
			for _, input := range a.Input {
				if input.Name == "password" {
					if !input.Prompt {
						t.Error("password input should have Prompt=true")
					}
					if input.PromptText != "Enter password" {
						t.Errorf("expected prompt text 'Enter password', got %q", input.PromptText)
					}
				}
			}
		}
	}
}

func TestDerive_WithInternalField(t *testing.T) {
	mod := schema.Module{
		Name: "user",
		Schema: map[string]schema.Field{
			"metadata": {
				Type:     schema.FieldTypeJSON,
				Internal: true,
			},
		},
	}

	d := Derive(mod)

	for _, f := range d.Fields {
		if f.Name == "metadata" {
			if !f.Internal {
				t.Error("metadata field should be internal")
			}
		}
	}
}

func TestDerive_WithDefaultValue(t *testing.T) {
	mod := schema.Module{
		Name: "setting",
		Schema: map[string]schema.Field{
			"enabled": {
				Type:    schema.FieldTypeBool,
				Default: true,
			},
		},
	}

	d := Derive(mod)

	for _, f := range d.Fields {
		if f.Name == "enabled" {
			if f.Default != true {
				t.Errorf("enabled field default should be true, got %v", f.Default)
			}
		}
	}

	// Fields with defaults should not be required in create inputs
	for _, a := range d.Actions {
		if a.Name == "create" {
			for _, input := range a.Input {
				if input.Name == "enabled" {
					if input.Required {
						t.Error("enabled input should not be required (has default)")
					}
					if input.Default != "true" {
						t.Errorf("expected default 'true', got %q", input.Default)
					}
				}
			}
		}
	}
}

func TestDerive_WithConstraints(t *testing.T) {
	mod := schema.Module{
		Name: "product",
		Schema: map[string]schema.Field{
			"price": {
				Type: schema.FieldTypeFloat,
				Constraints: []schema.Constraint{
					{Type: schema.ConstraintMin, Value: 0},
					{Type: schema.ConstraintMax, Value: 10000},
				},
			},
		},
	}

	d := Derive(mod)

	for _, f := range d.Fields {
		if f.Name == "price" {
			if len(f.Constraints) != 2 {
				t.Errorf("price field should have 2 constraints, got %d", len(f.Constraints))
			}
		}
	}
}

func TestDerive_WithDescription(t *testing.T) {
	mod := schema.Module{
		Name: "product",
		Schema: map[string]schema.Field{
			"name": {
				Type:        schema.FieldTypeString,
				Description: "The product name",
			},
		},
	}

	d := Derive(mod)

	for _, f := range d.Fields {
		if f.Name == "name" {
			if f.Description != "The product name" {
				t.Errorf("name field description should be 'The product name', got %q", f.Description)
			}
		}
	}
}

func TestDerive_WithCustomActions(t *testing.T) {
	mod := schema.Module{
		Name: "order",
		Schema: map[string]schema.Field{
			"status": {
				Type:   schema.FieldTypeEnum,
				Values: []string{"pending", "shipped", "delivered"},
			},
		},
		Actions: map[string]schema.Action{
			"ship": {
				Set:     map[string]string{"status": "shipped"},
				Auth:    "admin",
				Confirm: true,
			},
			"deliver": {
				Set:         map[string]string{"status": "delivered"},
				Description: "Mark order as delivered",
			},
		},
	}

	d := Derive(mod)

	// Should have 5 CRUD + 2 custom actions
	if len(d.Actions) != 7 {
		t.Errorf("expected 7 actions (5 CRUD + 2 custom), got %d", len(d.Actions))
	}

	foundShip := false
	foundDeliver := false
	for _, a := range d.Actions {
		switch a.Name {
		case "ship":
			foundShip = true
			if a.Type != schema.ActionTypeCustom {
				t.Errorf("ship action type should be custom, got %v", a.Type)
			}
			if a.Auth != "admin" {
				t.Errorf("ship action auth should be 'admin', got %q", a.Auth)
			}
			if !a.Confirm {
				t.Error("ship action should require confirmation")
			}
			if a.Set["status"] != "shipped" {
				t.Errorf("ship action should set status='shipped', got %q", a.Set["status"])
			}
			if a.Implicit {
				t.Error("ship action should not be implicit")
			}
		case "deliver":
			foundDeliver = true
			if a.Auth != "admin" {
				t.Errorf("deliver action auth should default to 'admin', got %q", a.Auth)
			}
			if a.Description != "Mark order as delivered" {
				t.Errorf("deliver action description mismatch: %q", a.Description)
			}
		}
	}

	if !foundShip {
		t.Error("ship action not found")
	}
	if !foundDeliver {
		t.Error("deliver action not found")
	}
}

func TestDerive_CustomActionWithInputs(t *testing.T) {
	mod := schema.Module{
		Name:   "user",
		Schema: map[string]schema.Field{},
		Actions: map[string]schema.Action{
			"change_password": {
				Input: []schema.ActionInput{
					{
						Name:     "new_password",
						Type:     "secret",
						Required: true,
						Prompt:   true,
					},
					{
						Field:   "email",
						Default: "default@example.com",
					},
				},
			},
		},
	}

	d := Derive(mod)

	var changePasswordAction *DerivedAction
	for i := range d.Actions {
		if d.Actions[i].Name == "change_password" {
			changePasswordAction = &d.Actions[i]
			break
		}
	}

	if changePasswordAction == nil {
		t.Fatal("change_password action not found")
	}

	if len(changePasswordAction.Input) != 2 {
		t.Fatalf("expected 2 inputs, got %d", len(changePasswordAction.Input))
	}

	// Check first input (new_password)
	passwordInput := changePasswordAction.Input[0]
	if passwordInput.Name != "new_password" {
		t.Errorf("expected input name 'new_password', got %q", passwordInput.Name)
	}
	if !passwordInput.Required {
		t.Error("new_password should be required")
	}
	if !passwordInput.Prompt {
		t.Error("new_password should have Prompt=true")
	}
	if passwordInput.Type != schema.FieldTypeSecret {
		t.Errorf("new_password type should be secret, got %v", passwordInput.Type)
	}

	// Check second input (uses Field, no Name)
	emailInput := changePasswordAction.Input[1]
	if emailInput.Name != "email" {
		t.Errorf("expected input name derived from field 'email', got %q", emailInput.Name)
	}
	if emailInput.Field != "email" {
		t.Errorf("expected input field 'email', got %q", emailInput.Field)
	}
}

func TestDerive_CustomActionDefaultDescription(t *testing.T) {
	mod := schema.Module{
		Name:   "user",
		Schema: map[string]schema.Field{},
		Actions: map[string]schema.Action{
			"activate": {},
		},
	}

	d := Derive(mod)

	for _, a := range d.Actions {
		if a.Name == "activate" {
			// Should have auto-generated description
			if a.Description == "" {
				t.Error("activate action should have auto-generated description")
			}
		}
	}
}

// -----------------------------------------------------------------------------
// ImplicitFields tests
// -----------------------------------------------------------------------------

func TestDerive_ImplicitFieldsProperties(t *testing.T) {
	mod := schema.Module{
		Name:   "item",
		Schema: map[string]schema.Field{},
	}

	d := Derive(mod)

	var idField, createdAtField, updatedAtField *DerivedField
	for i := range d.Fields {
		switch d.Fields[i].Name {
		case "id":
			idField = &d.Fields[i]
		case "created_at":
			createdAtField = &d.Fields[i]
		case "updated_at":
			updatedAtField = &d.Fields[i]
		}
	}

	// Check id field
	if idField == nil {
		t.Fatal("id field not found")
	}
	if idField.Type != schema.FieldTypeUUID {
		t.Errorf("id field type should be uuid, got %v", idField.Type)
	}
	if idField.SQLType != "TEXT" {
		t.Errorf("id field SQL type should be TEXT, got %q", idField.SQLType)
	}
	if !idField.Unique {
		t.Error("id field should be unique")
	}
	if idField.Required {
		t.Error("id field should not be required (auto-generated)")
	}
	if !idField.Lookup {
		t.Error("id field should be a lookup")
	}
	if !idField.Implicit {
		t.Error("id field should be implicit")
	}

	// Check created_at field
	if createdAtField == nil {
		t.Fatal("created_at field not found")
	}
	if createdAtField.Type != schema.FieldTypeTimestamp {
		t.Errorf("created_at type should be timestamp, got %v", createdAtField.Type)
	}
	if !createdAtField.Implicit {
		t.Error("created_at should be implicit")
	}

	// Check updated_at field
	if updatedAtField == nil {
		t.Fatal("updated_at field not found")
	}
	if updatedAtField.Type != schema.FieldTypeTimestamp {
		t.Errorf("updated_at type should be timestamp, got %v", updatedAtField.Type)
	}
	if !updatedAtField.Implicit {
		t.Error("updated_at should be implicit")
	}
}

// -----------------------------------------------------------------------------
// ImplicitActions tests
// -----------------------------------------------------------------------------

func TestDerive_ImplicitActionsProperties(t *testing.T) {
	mod := schema.Module{
		Name: "item",
		Schema: map[string]schema.Field{
			"name": {Type: schema.FieldTypeString},
		},
	}

	d := Derive(mod)

	actionMap := make(map[string]DerivedAction)
	for _, a := range d.Actions {
		actionMap[a.Name] = a
	}

	// List action
	listAction, ok := actionMap["list"]
	if !ok {
		t.Fatal("list action not found")
	}
	if listAction.Type != schema.ActionTypeList {
		t.Errorf("list action type should be ActionTypeList, got %v", listAction.Type)
	}
	if listAction.Auth != "admin" {
		t.Errorf("list action auth should be 'admin', got %q", listAction.Auth)
	}
	if !listAction.Implicit {
		t.Error("list action should be implicit")
	}

	// Get action
	getAction, ok := actionMap["get"]
	if !ok {
		t.Fatal("get action not found")
	}
	if getAction.Type != schema.ActionTypeGet {
		t.Errorf("get action type should be ActionTypeGet, got %v", getAction.Type)
	}

	// Create action
	createAction, ok := actionMap["create"]
	if !ok {
		t.Fatal("create action not found")
	}
	if createAction.Type != schema.ActionTypeCreate {
		t.Errorf("create action type should be ActionTypeCreate, got %v", createAction.Type)
	}

	// Update action
	updateAction, ok := actionMap["update"]
	if !ok {
		t.Fatal("update action not found")
	}
	if updateAction.Type != schema.ActionTypeUpdate {
		t.Errorf("update action type should be ActionTypeUpdate, got %v", updateAction.Type)
	}

	// Delete action
	deleteAction, ok := actionMap["delete"]
	if !ok {
		t.Fatal("delete action not found")
	}
	if deleteAction.Type != schema.ActionTypeDelete {
		t.Errorf("delete action type should be ActionTypeDelete, got %v", deleteAction.Type)
	}
	if !deleteAction.Confirm {
		t.Error("delete action should require confirmation")
	}
}

func TestDerive_CreateActionInputs(t *testing.T) {
	required := true
	mod := schema.Module{
		Name: "product",
		Schema: map[string]schema.Field{
			"name": {
				Type:     schema.FieldTypeString,
				Required: &required,
			},
			"price": {
				Type:    schema.FieldTypeFloat,
				Default: 0.0,
			},
		},
	}

	d := Derive(mod)

	var createAction *DerivedAction
	for i := range d.Actions {
		if d.Actions[i].Name == "create" {
			createAction = &d.Actions[i]
			break
		}
	}

	if createAction == nil {
		t.Fatal("create action not found")
	}

	// Create should have inputs for user-defined fields only
	inputMap := make(map[string]ActionInput)
	for _, input := range createAction.Input {
		inputMap[input.Name] = input
	}

	// Name should be required
	if nameInput, ok := inputMap["name"]; ok {
		if !nameInput.Required {
			t.Error("name input should be required")
		}
	} else {
		t.Error("name input not found in create action")
	}

	// Price should not be required (has default)
	if priceInput, ok := inputMap["price"]; ok {
		if priceInput.Required {
			t.Error("price input should not be required (has default)")
		}
	} else {
		t.Error("price input not found in create action")
	}

	// id, created_at, updated_at should NOT be in inputs
	if _, ok := inputMap["id"]; ok {
		t.Error("id should not be in create inputs")
	}
	if _, ok := inputMap["created_at"]; ok {
		t.Error("created_at should not be in create inputs")
	}
	if _, ok := inputMap["updated_at"]; ok {
		t.Error("updated_at should not be in create inputs")
	}
}

func TestDerive_UpdateActionInputs(t *testing.T) {
	required := true
	mod := schema.Module{
		Name: "product",
		Schema: map[string]schema.Field{
			"name": {
				Type:     schema.FieldTypeString,
				Required: &required,
			},
		},
	}

	d := Derive(mod)

	var updateAction *DerivedAction
	for i := range d.Actions {
		if d.Actions[i].Name == "update" {
			updateAction = &d.Actions[i]
			break
		}
	}

	if updateAction == nil {
		t.Fatal("update action not found")
	}

	// Update inputs should never be required
	for _, input := range updateAction.Input {
		if input.Name == "name" {
			if input.Required {
				t.Error("update inputs should never be required")
			}
		}
	}
}

func TestDerive_ActionOutputFields(t *testing.T) {
	mod := schema.Module{
		Name: "product",
		Schema: map[string]schema.Field{
			"name":  {Type: schema.FieldTypeString},
			"price": {Type: schema.FieldTypeFloat},
		},
	}

	d := Derive(mod)

	var getAction *DerivedAction
	for i := range d.Actions {
		if d.Actions[i].Name == "get" {
			getAction = &d.Actions[i]
			break
		}
	}

	if getAction == nil {
		t.Fatal("get action not found")
	}

	// Output should include id and user-defined fields
	outputMap := make(map[string]bool)
	for _, o := range getAction.Output {
		outputMap[o] = true
	}

	if !outputMap["id"] {
		t.Error("get output should include 'id'")
	}
	if !outputMap["name"] {
		t.Error("get output should include 'name'")
	}
	if !outputMap["price"] {
		t.Error("get output should include 'price'")
	}
}

func TestDerive_RefFieldsNotInEditableFields(t *testing.T) {
	mod := schema.Module{
		Name: "order",
		Schema: map[string]schema.Field{
			"user_id": {
				Type: schema.FieldTypeRef,
				To:   "user",
			},
			"total": {
				Type: schema.FieldTypeFloat,
			},
		},
	}

	d := Derive(mod)

	var listAction *DerivedAction
	for i := range d.Actions {
		if d.Actions[i].Name == "list" {
			listAction = &d.Actions[i]
			break
		}
	}

	if listAction == nil {
		t.Fatal("list action not found")
	}

	// Ref fields should still be in output (listable)
	outputMap := make(map[string]bool)
	for _, o := range listAction.Output {
		outputMap[o] = true
	}

	if !outputMap["user_id"] {
		t.Error("list output should include 'user_id' (ref field)")
	}
	if !outputMap["total"] {
		t.Error("list output should include 'total'")
	}
}

// -----------------------------------------------------------------------------
// toString tests
// -----------------------------------------------------------------------------

func TestToString(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"string", "hello", "hello"},
		{"int", 42, "42"},
		{"int64", int64(123456789), "123456789"},
		{"float64", 3.14, "3.14"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"nil", nil, ""},
		{"struct", struct{}{}, ""},
		{"slice", []int{1, 2}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toString(tt.input)
			if result != tt.expected {
				t.Errorf("toString(%v) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// deriveCustomActionInputs tests
// -----------------------------------------------------------------------------

func TestDeriveCustomActionInputs_Empty(t *testing.T) {
	result := deriveCustomActionInputs(nil)
	if result != nil {
		t.Errorf("expected nil for empty inputs, got %v", result)
	}

	result = deriveCustomActionInputs([]schema.ActionInput{})
	if result != nil {
		t.Errorf("expected nil for empty slice, got %v", result)
	}
}

func TestDeriveCustomActionInputs_WithNameAndField(t *testing.T) {
	inputs := []schema.ActionInput{
		{
			Name:        "custom_name",
			Field:       "target_field",
			Type:        "string",
			Required:    true,
			Default:     "default_value",
			Prompt:      true,
			PromptText:  "Enter value",
			Description: "A custom input",
		},
	}

	result := deriveCustomActionInputs(inputs)

	if len(result) != 1 {
		t.Fatalf("expected 1 input, got %d", len(result))
	}

	r := result[0]
	if r.Name != "custom_name" {
		t.Errorf("Name should be 'custom_name', got %q", r.Name)
	}
	if r.Field != "target_field" {
		t.Errorf("Field should be 'target_field', got %q", r.Field)
	}
	if r.Type != schema.FieldTypeString {
		t.Errorf("Type should be string, got %v", r.Type)
	}
	if !r.Required {
		t.Error("Required should be true")
	}
	if r.Default != "default_value" {
		t.Errorf("Default should be 'default_value', got %v", r.Default)
	}
	if !r.Prompt {
		t.Error("Prompt should be true")
	}
	if r.PromptText != "Enter value" {
		t.Errorf("PromptText should be 'Enter value', got %q", r.PromptText)
	}
	if r.Description != "A custom input" {
		t.Errorf("Description should be 'A custom input', got %q", r.Description)
	}
}

func TestDeriveCustomActionInputs_NameFallsBackToField(t *testing.T) {
	inputs := []schema.ActionInput{
		{
			Field: "my_field",
		},
	}

	result := deriveCustomActionInputs(inputs)

	if len(result) != 1 {
		t.Fatalf("expected 1 input, got %d", len(result))
	}

	if result[0].Name != "my_field" {
		t.Errorf("Name should fall back to 'my_field', got %q", result[0].Name)
	}
}

func TestDeriveCustomActionInputs_WithTo(t *testing.T) {
	inputs := []schema.ActionInput{
		{
			Name: "user_ref",
			Type: "ref",
			To:   "user",
		},
	}

	result := deriveCustomActionInputs(inputs)

	if len(result) != 1 {
		t.Fatalf("expected 1 input, got %d", len(result))
	}

	if result[0].To != "user" {
		t.Errorf("To should be 'user', got %q", result[0].To)
	}
}

// -----------------------------------------------------------------------------
// deriveLookups tests
// -----------------------------------------------------------------------------

func TestDeriveLookups_NoLookups(t *testing.T) {
	fields := []DerivedField{
		{Name: "name", Lookup: false},
		{Name: "description", Lookup: false},
	}

	lookups := deriveLookups(fields)

	if len(lookups) != 0 {
		t.Errorf("expected 0 lookups, got %d", len(lookups))
	}
}

func TestDeriveLookups_MultipleLookups(t *testing.T) {
	fields := []DerivedField{
		{Name: "id", Lookup: true},
		{Name: "email", Lookup: true},
		{Name: "name", Lookup: false},
		{Name: "slug", Lookup: true},
	}

	lookups := deriveLookups(fields)

	if len(lookups) != 3 {
		t.Errorf("expected 3 lookups, got %d", len(lookups))
	}

	lookupMap := make(map[string]bool)
	for _, l := range lookups {
		lookupMap[l] = true
	}

	if !lookupMap["id"] {
		t.Error("id should be in lookups")
	}
	if !lookupMap["email"] {
		t.Error("email should be in lookups")
	}
	if !lookupMap["slug"] {
		t.Error("slug should be in lookups")
	}
	if lookupMap["name"] {
		t.Error("name should not be in lookups")
	}
}

// -----------------------------------------------------------------------------
// SQLType tests (via deriveFields)
// -----------------------------------------------------------------------------

func TestDerive_SQLTypes(t *testing.T) {
	mod := schema.Module{
		Name: "test",
		Schema: map[string]schema.Field{
			"string_field":    {Type: schema.FieldTypeString},
			"int_field":       {Type: schema.FieldTypeInt},
			"float_field":     {Type: schema.FieldTypeFloat},
			"bool_field":      {Type: schema.FieldTypeBool},
			"timestamp_field": {Type: schema.FieldTypeTimestamp},
			"email_field":     {Type: schema.FieldTypeEmail},
			"uuid_field":      {Type: schema.FieldTypeUUID},
			"json_field":      {Type: schema.FieldTypeJSON},
			"bytes_field":     {Type: schema.FieldTypeBytes},
			"secret_field":    {Type: schema.FieldTypeSecret},
			"strings_field":   {Type: schema.FieldTypeStrings},
			"ints_field":      {Type: schema.FieldTypeInts},
		},
	}

	d := Derive(mod)

	sqlTypes := make(map[string]string)
	for _, f := range d.Fields {
		sqlTypes[f.Name] = f.SQLType
	}

	testCases := []struct {
		field       string
		expectedSQL string
	}{
		{"string_field", "TEXT"},
		{"int_field", "INTEGER"},
		{"float_field", "REAL"},
		{"bool_field", "INTEGER"},
		{"timestamp_field", "TEXT"},
		{"email_field", "TEXT"},
		{"uuid_field", "TEXT"},
		{"json_field", "TEXT"},
		{"bytes_field", "BLOB"},
		{"secret_field", "BLOB"},
		{"strings_field", "TEXT"},
		{"ints_field", "TEXT"},
	}

	for _, tc := range testCases {
		t.Run(tc.field, func(t *testing.T) {
			if sqlTypes[tc.field] != tc.expectedSQL {
				t.Errorf("%s SQL type should be %q, got %q", tc.field, tc.expectedSQL, sqlTypes[tc.field])
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Edge cases
// -----------------------------------------------------------------------------

func TestDerive_IrregularPluralModule(t *testing.T) {
	mod := schema.Module{
		Name:   "person",
		Schema: map[string]schema.Field{},
	}

	d := Derive(mod)

	if d.Plural != "people" {
		t.Errorf("expected Plural = 'people', got %q", d.Plural)
	}
	if d.Table != "people" {
		t.Errorf("expected Table = 'people', got %q", d.Table)
	}
}

func TestDerive_ActionDescriptions(t *testing.T) {
	mod := schema.Module{
		Name:   "product",
		Schema: map[string]schema.Field{},
	}

	d := Derive(mod)

	actionDescriptions := make(map[string]string)
	for _, a := range d.Actions {
		actionDescriptions[a.Name] = a.Description
	}

	expectedDescriptions := map[string]string{
		"list":   "List all products",
		"get":    "Get product details",
		"create": "Create a new product",
		"update": "Update product",
		"delete": "Delete product",
	}

	for action, expected := range expectedDescriptions {
		if actionDescriptions[action] != expected {
			t.Errorf("%s description should be %q, got %q", action, expected, actionDescriptions[action])
		}
	}
}

func TestDerive_SecretFieldUpdatePrompt(t *testing.T) {
	mod := schema.Module{
		Name: "user",
		Schema: map[string]schema.Field{
			"password": {Type: schema.FieldTypeSecret},
		},
	}

	d := Derive(mod)

	var updateAction *DerivedAction
	for i := range d.Actions {
		if d.Actions[i].Name == "update" {
			updateAction = &d.Actions[i]
			break
		}
	}

	if updateAction == nil {
		t.Fatal("update action not found")
	}

	for _, input := range updateAction.Input {
		if input.Name == "password" {
			if !input.Prompt {
				t.Error("password input in update should have Prompt=true")
			}
			if input.PromptText != "Enter new password" {
				t.Errorf("expected prompt 'Enter new password', got %q", input.PromptText)
			}
		}
	}
}

func TestDerive_InternalFieldsNotInOutput(t *testing.T) {
	mod := schema.Module{
		Name: "user",
		Schema: map[string]schema.Field{
			"name": {Type: schema.FieldTypeString},
			"password_hash": {
				Type:     schema.FieldTypeBytes,
				Internal: true,
			},
		},
	}

	d := Derive(mod)

	var listAction *DerivedAction
	for i := range d.Actions {
		if d.Actions[i].Name == "list" {
			listAction = &d.Actions[i]
			break
		}
	}

	if listAction == nil {
		t.Fatal("list action not found")
	}

	for _, output := range listAction.Output {
		if output == "password_hash" {
			t.Error("internal field 'password_hash' should not be in list output")
		}
	}
}

func TestDerive_AllFieldTypes(t *testing.T) {
	mod := schema.Module{
		Name: "test",
		Schema: map[string]schema.Field{
			"duration_field": {Type: schema.FieldTypeDuration},
			"url_field":      {Type: schema.FieldTypeURL},
			"enum_field":     {Type: schema.FieldTypeEnum, Values: []string{"a", "b"}},
			"ref_field":      {Type: schema.FieldTypeRef, To: "other"},
		},
	}

	d := Derive(mod)

	fieldMap := make(map[string]DerivedField)
	for _, f := range d.Fields {
		fieldMap[f.Name] = f
	}

	if fieldMap["duration_field"].Type != schema.FieldTypeDuration {
		t.Error("duration field type mismatch")
	}
	if fieldMap["url_field"].Type != schema.FieldTypeURL {
		t.Error("url field type mismatch")
	}
	if fieldMap["enum_field"].Type != schema.FieldTypeEnum {
		t.Error("enum field type mismatch")
	}
	if len(fieldMap["enum_field"].Values) != 2 {
		t.Error("enum field should have 2 values")
	}
	if fieldMap["ref_field"].Type != schema.FieldTypeRef {
		t.Error("ref field type mismatch")
	}
	if fieldMap["ref_field"].Ref != "other" {
		t.Error("ref field Ref should be 'other'")
	}
}

// -----------------------------------------------------------------------------
// Source preservation tests
// -----------------------------------------------------------------------------

func TestDerive_PreservesSourceModule(t *testing.T) {
	mod := schema.Module{
		Name:        "user",
		Description: "User module",
		Schema:      map[string]schema.Field{},
		Meta: schema.ModuleMeta{
			Version:     "1.0.0",
			Description: "Test module",
		},
	}

	d := Derive(mod)

	if d.Source.Name != "user" {
		t.Error("Source.Name not preserved")
	}
	if d.Source.Description != "User module" {
		t.Error("Source.Description not preserved")
	}
	if d.Source.Meta.Version != "1.0.0" {
		t.Error("Source.Meta.Version not preserved")
	}
}

func TestDerive_FieldSourcePreserved(t *testing.T) {
	required := true
	mod := schema.Module{
		Name: "item",
		Schema: map[string]schema.Field{
			"name": {
				Type:     schema.FieldTypeString,
				Required: &required,
			},
		},
	}

	d := Derive(mod)

	for _, f := range d.Fields {
		if f.Name == "name" {
			if f.Source == nil {
				t.Error("user-defined field should have Source set")
			}
			if f.Source.Type != schema.FieldTypeString {
				t.Error("Source.Type not preserved")
			}
		}
		if f.Name == "id" {
			if f.Source != nil {
				t.Error("implicit field should have nil Source")
			}
		}
	}
}

func TestDerive_ActionSourcePreserved(t *testing.T) {
	mod := schema.Module{
		Name:   "order",
		Schema: map[string]schema.Field{},
		Actions: map[string]schema.Action{
			"ship": {
				Auth:    "admin",
				Confirm: true,
			},
		},
	}

	d := Derive(mod)

	for _, a := range d.Actions {
		if a.Name == "ship" {
			if a.Source == nil {
				t.Error("custom action should have Source set")
			}
			if a.Source.Auth != "admin" {
				t.Error("Source.Auth not preserved")
			}
		}
		if a.Name == "list" {
			if a.Source != nil {
				t.Error("implicit action should have nil Source")
			}
		}
	}
}

// -----------------------------------------------------------------------------
// Pluralize function tests (plural.go)
// -----------------------------------------------------------------------------

func TestPluralize_EmptyString(t *testing.T) {
	result := Pluralize("")
	if result != "" {
		t.Errorf("Pluralize('') should return '', got %q", result)
	}
}

func TestPluralize_RegularWords(t *testing.T) {
	tests := []struct {
		singular string
		plural   string
	}{
		{"user", "users"},
		{"product", "products"},
		{"item", "items"},
		{"order", "orders"},
	}

	for _, tt := range tests {
		t.Run(tt.singular, func(t *testing.T) {
			result := Pluralize(tt.singular)
			if result != tt.plural {
				t.Errorf("Pluralize(%q) = %q, expected %q", tt.singular, result, tt.plural)
			}
		})
	}
}

func TestPluralize_WordsEndingInS(t *testing.T) {
	tests := []struct {
		singular string
		plural   string
	}{
		{"bus", "buses"},
		{"class", "classes"},
		{"gas", "gases"},
	}

	for _, tt := range tests {
		t.Run(tt.singular, func(t *testing.T) {
			result := Pluralize(tt.singular)
			if result != tt.plural {
				t.Errorf("Pluralize(%q) = %q, expected %q", tt.singular, result, tt.plural)
			}
		})
	}
}

func TestPluralize_WordsEndingInX(t *testing.T) {
	tests := []struct {
		singular string
		plural   string
	}{
		{"box", "boxes"},
		{"tax", "taxes"},
		{"fox", "foxes"},
	}

	for _, tt := range tests {
		t.Run(tt.singular, func(t *testing.T) {
			result := Pluralize(tt.singular)
			if result != tt.plural {
				t.Errorf("Pluralize(%q) = %q, expected %q", tt.singular, result, tt.plural)
			}
		})
	}
}

func TestPluralize_WordsEndingInZ(t *testing.T) {
	tests := []struct {
		singular string
		plural   string
	}{
		{"quiz", "quizes"},
		{"buzz", "buzzes"},
	}

	for _, tt := range tests {
		t.Run(tt.singular, func(t *testing.T) {
			result := Pluralize(tt.singular)
			if result != tt.plural {
				t.Errorf("Pluralize(%q) = %q, expected %q", tt.singular, result, tt.plural)
			}
		})
	}
}

func TestPluralize_WordsEndingInCh(t *testing.T) {
	tests := []struct {
		singular string
		plural   string
	}{
		{"church", "churches"},
		{"watch", "watches"},
		{"match", "matches"},
	}

	for _, tt := range tests {
		t.Run(tt.singular, func(t *testing.T) {
			result := Pluralize(tt.singular)
			if result != tt.plural {
				t.Errorf("Pluralize(%q) = %q, expected %q", tt.singular, result, tt.plural)
			}
		})
	}
}

func TestPluralize_WordsEndingInSh(t *testing.T) {
	tests := []struct {
		singular string
		plural   string
	}{
		{"dish", "dishes"},
		{"brush", "brushes"},
		{"wish", "wishes"},
	}

	for _, tt := range tests {
		t.Run(tt.singular, func(t *testing.T) {
			result := Pluralize(tt.singular)
			if result != tt.plural {
				t.Errorf("Pluralize(%q) = %q, expected %q", tt.singular, result, tt.plural)
			}
		})
	}
}

func TestPluralize_WordsEndingInConsonantY(t *testing.T) {
	tests := []struct {
		singular string
		plural   string
	}{
		{"category", "categories"},
		{"company", "companies"},
		{"city", "cities"},
		{"baby", "babies"},
	}

	for _, tt := range tests {
		t.Run(tt.singular, func(t *testing.T) {
			result := Pluralize(tt.singular)
			if result != tt.plural {
				t.Errorf("Pluralize(%q) = %q, expected %q", tt.singular, result, tt.plural)
			}
		})
	}
}

func TestPluralize_WordsEndingInVowelY(t *testing.T) {
	tests := []struct {
		singular string
		plural   string
	}{
		{"key", "keys"},
		{"day", "days"},
		{"boy", "boys"},
		{"toy", "toys"},
	}

	for _, tt := range tests {
		t.Run(tt.singular, func(t *testing.T) {
			result := Pluralize(tt.singular)
			if result != tt.plural {
				t.Errorf("Pluralize(%q) = %q, expected %q", tt.singular, result, tt.plural)
			}
		})
	}
}

func TestPluralize_WordsEndingInF(t *testing.T) {
	tests := []struct {
		singular string
		plural   string
	}{
		{"leaf", "leaves"},
		{"loaf", "loaves"},
		{"shelf", "shelves"},
	}

	for _, tt := range tests {
		t.Run(tt.singular, func(t *testing.T) {
			result := Pluralize(tt.singular)
			if result != tt.plural {
				t.Errorf("Pluralize(%q) = %q, expected %q", tt.singular, result, tt.plural)
			}
		})
	}
}

func TestPluralize_WordsEndingInFe(t *testing.T) {
	tests := []struct {
		singular string
		plural   string
	}{
		{"wife", "wives"},
		{"knife", "knives"},
		{"life", "lives"},
	}

	for _, tt := range tests {
		t.Run(tt.singular, func(t *testing.T) {
			result := Pluralize(tt.singular)
			if result != tt.plural {
				t.Errorf("Pluralize(%q) = %q, expected %q", tt.singular, result, tt.plural)
			}
		})
	}
}

func TestPluralize_IrregularPlurals(t *testing.T) {
	tests := []struct {
		singular string
		plural   string
	}{
		{"person", "people"},
		{"man", "men"},
		{"woman", "women"},
		{"child", "children"},
		{"foot", "feet"},
		{"tooth", "teeth"},
		{"goose", "geese"},
		{"mouse", "mice"},
		{"ox", "oxen"},
		{"index", "indices"},
		{"matrix", "matrices"},
		{"vertex", "vertices"},
		{"analysis", "analyses"},
		{"crisis", "crises"},
		{"thesis", "theses"},
		{"datum", "data"},
		{"medium", "media"},
		{"schema", "schemas"},
		{"status", "statuses"},
	}

	for _, tt := range tests {
		t.Run(tt.singular, func(t *testing.T) {
			result := Pluralize(tt.singular)
			if result != tt.plural {
				t.Errorf("Pluralize(%q) = %q, expected %q", tt.singular, result, tt.plural)
			}
		})
	}
}

func TestPluralize_CapitalizedIrregular(t *testing.T) {
	result := Pluralize("Person")
	if result != "People" {
		t.Errorf("Pluralize('Person') = %q, expected 'People'", result)
	}
}

func TestPluralize_SingleCharY(t *testing.T) {
	// Single char 'y' should just add 's'
	result := Pluralize("y")
	if result != "ys" {
		t.Errorf("Pluralize('y') = %q, expected 'ys'", result)
	}
}

// -----------------------------------------------------------------------------
// Singularize function tests (plural.go)
// -----------------------------------------------------------------------------

func TestSingularize_EmptyString(t *testing.T) {
	result := Singularize("")
	if result != "" {
		t.Errorf("Singularize('') should return '', got %q", result)
	}
}

func TestSingularize_RegularWords(t *testing.T) {
	tests := []struct {
		plural   string
		singular string
	}{
		{"users", "user"},
		{"products", "product"},
		{"items", "item"},
		{"orders", "order"},
	}

	for _, tt := range tests {
		t.Run(tt.plural, func(t *testing.T) {
			result := Singularize(tt.plural)
			if result != tt.singular {
				t.Errorf("Singularize(%q) = %q, expected %q", tt.plural, result, tt.singular)
			}
		})
	}
}

func TestSingularize_WordsEndingInIes(t *testing.T) {
	tests := []struct {
		plural   string
		singular string
	}{
		{"categories", "category"},
		{"companies", "company"},
		{"cities", "city"},
	}

	for _, tt := range tests {
		t.Run(tt.plural, func(t *testing.T) {
			result := Singularize(tt.plural)
			if result != tt.singular {
				t.Errorf("Singularize(%q) = %q, expected %q", tt.plural, result, tt.singular)
			}
		})
	}
}

func TestSingularize_WordsEndingInVes(t *testing.T) {
	tests := []struct {
		plural   string
		singular string
	}{
		{"leaves", "leaf"},
		{"loaves", "loaf"},
		{"knives", "knif"},
		{"lives", "lif"},
	}

	for _, tt := range tests {
		t.Run(tt.plural, func(t *testing.T) {
			result := Singularize(tt.plural)
			if result != tt.singular {
				t.Errorf("Singularize(%q) = %q, expected %q", tt.plural, result, tt.singular)
			}
		})
	}
}

func TestSingularize_WordsEndingInSes(t *testing.T) {
	tests := []struct {
		plural   string
		singular string
	}{
		{"buses", "bus"},
		{"classes", "class"},
	}

	for _, tt := range tests {
		t.Run(tt.plural, func(t *testing.T) {
			result := Singularize(tt.plural)
			if result != tt.singular {
				t.Errorf("Singularize(%q) = %q, expected %q", tt.plural, result, tt.singular)
			}
		})
	}
}

func TestSingularize_WordsEndingInXes(t *testing.T) {
	tests := []struct {
		plural   string
		singular string
	}{
		{"boxes", "box"},
		{"taxes", "tax"},
	}

	for _, tt := range tests {
		t.Run(tt.plural, func(t *testing.T) {
			result := Singularize(tt.plural)
			if result != tt.singular {
				t.Errorf("Singularize(%q) = %q, expected %q", tt.plural, result, tt.singular)
			}
		})
	}
}

func TestSingularize_WordsEndingInZes(t *testing.T) {
	result := Singularize("quizes")
	if result != "quiz" {
		t.Errorf("Singularize('quizes') = %q, expected 'quiz'", result)
	}
}

func TestSingularize_WordsEndingInChes(t *testing.T) {
	tests := []struct {
		plural   string
		singular string
	}{
		{"churches", "church"},
		{"watches", "watch"},
	}

	for _, tt := range tests {
		t.Run(tt.plural, func(t *testing.T) {
			result := Singularize(tt.plural)
			if result != tt.singular {
				t.Errorf("Singularize(%q) = %q, expected %q", tt.plural, result, tt.singular)
			}
		})
	}
}

func TestSingularize_WordsEndingInShes(t *testing.T) {
	tests := []struct {
		plural   string
		singular string
	}{
		{"dishes", "dish"},
		{"brushes", "brush"},
	}

	for _, tt := range tests {
		t.Run(tt.plural, func(t *testing.T) {
			result := Singularize(tt.plural)
			if result != tt.singular {
				t.Errorf("Singularize(%q) = %q, expected %q", tt.plural, result, tt.singular)
			}
		})
	}
}

func TestSingularize_IrregularPlurals(t *testing.T) {
	tests := []struct {
		plural   string
		singular string
	}{
		{"people", "person"},
		{"men", "man"},
		{"women", "woman"},
		{"children", "child"},
		{"feet", "foot"},
		{"teeth", "tooth"},
		{"geese", "goose"},
		{"mice", "mouse"},
		{"oxen", "ox"},
		{"indices", "index"},
		{"matrices", "matrix"},
		{"vertices", "vertex"},
		{"analyses", "analysis"},
		{"crises", "crisis"},
		{"theses", "thesis"},
		{"data", "datum"},
		{"media", "medium"},
		{"schemas", "schema"},
		{"statuses", "status"},
	}

	for _, tt := range tests {
		t.Run(tt.plural, func(t *testing.T) {
			result := Singularize(tt.plural)
			if result != tt.singular {
				t.Errorf("Singularize(%q) = %q, expected %q", tt.plural, result, tt.singular)
			}
		})
	}
}

func TestSingularize_CapitalizedIrregular(t *testing.T) {
	result := Singularize("People")
	if result != "Person" {
		t.Errorf("Singularize('People') = %q, expected 'Person'", result)
	}
}

func TestSingularize_WordsEndingInSS(t *testing.T) {
	// Words ending in 'ss' should not have 's' removed
	result := Singularize("class")
	if result != "class" {
		t.Errorf("Singularize('class') = %q, expected 'class'", result)
	}
}

func TestSingularize_AlreadySingular(t *testing.T) {
	// Word not ending in 's' should return as-is
	result := Singularize("item")
	if result != "item" {
		t.Errorf("Singularize('item') = %q, expected 'item'", result)
	}
}

// -----------------------------------------------------------------------------
// isVowel function tests (plural.go)
// -----------------------------------------------------------------------------

func TestIsVowel(t *testing.T) {
	vowels := []rune{'a', 'e', 'i', 'o', 'u', 'A', 'E', 'I', 'O', 'U'}
	for _, v := range vowels {
		if !isVowel(v) {
			t.Errorf("isVowel(%c) should return true", v)
		}
	}

	consonants := []rune{'b', 'c', 'd', 'f', 'g', 'h', 'j', 'k', 'l', 'm', 'n', 'p', 'q', 'r', 's', 't', 'v', 'w', 'x', 'y', 'z'}
	for _, c := range consonants {
		if isVowel(c) {
			t.Errorf("isVowel(%c) should return false", c)
		}
	}
}
