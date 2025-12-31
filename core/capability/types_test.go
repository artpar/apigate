package capability_test

import (
	"testing"

	"github.com/artpar/apigate/core/capability"
)

// =============================================================================
// Capability Type Tests
// =============================================================================

func TestCapabilityType_String(t *testing.T) {
	tests := []struct {
		cap  capability.Type
		want string
	}{
		{capability.Payment, "payment"},
		{capability.Email, "email"},
		{capability.Cache, "cache"},
		{capability.Storage, "storage"},
		{capability.Auth, "auth"},
		{capability.Queue, "queue"},
		{capability.Notification, "notification"},
		{capability.Hasher, "hasher"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.cap.String(); got != tt.want {
				t.Errorf("Type.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseCapabilityType(t *testing.T) {
	tests := []struct {
		input   string
		want    capability.Type
		wantErr bool
	}{
		{"payment", capability.Payment, false},
		{"email", capability.Email, false},
		{"cache", capability.Cache, false},
		{"storage", capability.Storage, false},
		{"auth", capability.Auth, false},
		{"queue", capability.Queue, false},
		{"notification", capability.Notification, false},
		{"hasher", capability.Hasher, false},
		{"unknown", capability.Unknown, true},
		{"", capability.Unknown, true},
		// Custom capabilities should be allowed
		{"reconciliation", capability.Custom, false},
		{"analytics", capability.Custom, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := capability.ParseType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// For custom types, we allow any string
			if !tt.wantErr && tt.want != capability.Custom && got != tt.want {
				t.Errorf("ParseType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCapabilityType_IsBuiltin(t *testing.T) {
	builtins := []capability.Type{
		capability.Payment,
		capability.Email,
		capability.Cache,
		capability.Storage,
		capability.Auth,
		capability.Queue,
		capability.Notification,
		capability.Hasher,
	}

	for _, cap := range builtins {
		if !cap.IsBuiltin() {
			t.Errorf("%s should be builtin", cap)
		}
	}

	if capability.Custom.IsBuiltin() {
		t.Error("Custom should not be builtin")
	}
}

// =============================================================================
// Provider Info Tests
// =============================================================================

func TestProviderInfo_Validate(t *testing.T) {
	tests := []struct {
		name    string
		info    capability.ProviderInfo
		wantErr bool
	}{
		{
			name: "valid provider",
			info: capability.ProviderInfo{
				Name:        "stripe_prod",
				Module:      "payment_stripe",
				Capability:  capability.Payment,
				Enabled:     true,
				IsDefault:   true,
				Description: "Production Stripe",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			info: capability.ProviderInfo{
				Module:     "payment_stripe",
				Capability: capability.Payment,
			},
			wantErr: true,
		},
		{
			name: "missing module",
			info: capability.ProviderInfo{
				Name:       "stripe_prod",
				Capability: capability.Payment,
			},
			wantErr: true,
		},
		{
			name: "invalid capability",
			info: capability.ProviderInfo{
				Name:       "stripe_prod",
				Module:     "payment_stripe",
				Capability: capability.Unknown,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.info.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ProviderInfo.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// =============================================================================
// Additional Type Tests for Coverage
// =============================================================================

func TestCapabilityType_IsValid(t *testing.T) {
	tests := []struct {
		cap  capability.Type
		want bool
	}{
		{capability.Payment, true},
		{capability.Email, true},
		{capability.Cache, true},
		{capability.Storage, true},
		{capability.Auth, true},
		{capability.Queue, true},
		{capability.Notification, true},
		{capability.Hasher, true},
		{capability.Custom, true},
		{capability.Unknown, false},
		{capability.Type(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.cap), func(t *testing.T) {
			if got := tt.cap.IsValid(); got != tt.want {
				t.Errorf("Type.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProviderInfo_CapabilityKey(t *testing.T) {
	tests := []struct {
		name string
		info capability.ProviderInfo
		want string
	}{
		{
			name: "builtin capability",
			info: capability.ProviderInfo{
				Name:       "stripe_prod",
				Module:     "payment_stripe",
				Capability: capability.Payment,
			},
			want: "payment",
		},
		{
			name: "custom capability",
			info: capability.ProviderInfo{
				Name:             "recon_main",
				Module:           "reconciliation_default",
				Capability:       capability.Custom,
				CustomCapability: "reconciliation",
			},
			want: "reconciliation",
		},
		{
			name: "email capability",
			info: capability.ProviderInfo{
				Name:       "smtp_main",
				Module:     "email_smtp",
				Capability: capability.Email,
			},
			want: "email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.CapabilityKey(); got != tt.want {
				t.Errorf("ProviderInfo.CapabilityKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProviderInfo_Validate_CustomCapabilityMissing(t *testing.T) {
	info := capability.ProviderInfo{
		Name:             "recon_main",
		Module:           "reconciliation_default",
		Capability:       capability.Custom,
		CustomCapability: "", // Missing custom capability name
	}

	err := info.Validate()
	if err == nil {
		t.Error("Validate() should return error when custom capability name is missing")
	}
}

func TestProviderInfo_Validate_CustomCapabilityValid(t *testing.T) {
	info := capability.ProviderInfo{
		Name:             "recon_main",
		Module:           "reconciliation_default",
		Capability:       capability.Custom,
		CustomCapability: "reconciliation",
	}

	err := info.Validate()
	if err != nil {
		t.Errorf("Validate() should not error for valid custom capability, got %v", err)
	}
}

func TestProviderInfo_Validate_MultipleErrors(t *testing.T) {
	info := capability.ProviderInfo{
		Name:       "",
		Module:     "",
		Capability: capability.Unknown,
	}

	err := info.Validate()
	if err == nil {
		t.Fatal("Validate() should return error for multiple validation failures")
	}

	// Error message should contain multiple issues
	errStr := err.Error()
	if errStr == "" {
		t.Error("Error message should not be empty")
	}
}

func TestParseType_CaseInsensitive(t *testing.T) {
	tests := []struct {
		input string
		want  capability.Type
	}{
		{"PAYMENT", capability.Payment},
		{"Payment", capability.Payment},
		{"EMAIL", capability.Email},
		{"Email", capability.Email},
		{"CACHE", capability.Cache},
		{"Cache", capability.Cache},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := capability.ParseType(tt.input)
			if err != nil {
				t.Errorf("ParseType(%s) error = %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("ParseType(%s) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseType_TrimWhitespace(t *testing.T) {
	tests := []struct {
		input string
		want  capability.Type
	}{
		{"  payment  ", capability.Payment},
		{"\temail\t", capability.Email},
		{" cache ", capability.Cache},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := capability.ParseType(tt.input)
			if err != nil {
				t.Errorf("ParseType(%s) error = %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("ParseType(%s) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseType_ReservedWords(t *testing.T) {
	tests := []string{
		"unknown",
		"custom",
		"UNKNOWN",
		"CUSTOM",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := capability.ParseType(input)
			if err == nil {
				t.Errorf("ParseType(%s) should return error for reserved word", input)
			}
		})
	}
}

func TestCapabilityType_Unknown(t *testing.T) {
	if capability.Unknown.String() != "" {
		t.Errorf("Unknown.String() = %q, want empty string", capability.Unknown.String())
	}

	if capability.Unknown.IsValid() {
		t.Error("Unknown.IsValid() should be false")
	}

	if capability.Unknown.IsBuiltin() {
		t.Error("Unknown.IsBuiltin() should be false")
	}
}

func TestCapabilityType_Custom(t *testing.T) {
	if capability.Custom.String() != "custom" {
		t.Errorf("Custom.String() = %q, want 'custom'", capability.Custom.String())
	}

	if !capability.Custom.IsValid() {
		t.Error("Custom.IsValid() should be true")
	}

	if capability.Custom.IsBuiltin() {
		t.Error("Custom.IsBuiltin() should be false")
	}
}

func TestCapabilityType_AllBuiltins(t *testing.T) {
	builtins := []struct {
		cap  capability.Type
		name string
	}{
		{capability.Payment, "payment"},
		{capability.Email, "email"},
		{capability.Cache, "cache"},
		{capability.Storage, "storage"},
		{capability.Auth, "auth"},
		{capability.Queue, "queue"},
		{capability.Notification, "notification"},
		{capability.Hasher, "hasher"},
	}

	for _, tt := range builtins {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cap.String() != tt.name {
				t.Errorf("%s.String() = %q, want %q", tt.name, tt.cap.String(), tt.name)
			}
			if !tt.cap.IsValid() {
				t.Errorf("%s.IsValid() should be true", tt.name)
			}
			if !tt.cap.IsBuiltin() {
				t.Errorf("%s.IsBuiltin() should be true", tt.name)
			}
		})
	}
}

func TestProviderInfo_Config(t *testing.T) {
	info := capability.ProviderInfo{
		Name:       "stripe_prod",
		Module:     "payment_stripe",
		Capability: capability.Payment,
		Config: map[string]any{
			"api_key":    "sk_test_xxx",
			"webhook_id": "whk_123",
		},
	}

	if info.Config == nil {
		t.Error("Config should not be nil")
	}

	if info.Config["api_key"] != "sk_test_xxx" {
		t.Errorf("Config[api_key] = %v, want sk_test_xxx", info.Config["api_key"])
	}
}
