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
