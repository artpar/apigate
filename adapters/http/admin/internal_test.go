package admin

import (
	"testing"
	"time"

	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/domain/route"
)

func TestErrorType(t *testing.T) {
	err := errorType{
		code:    "test_error",
		message: "Test error message",
	}

	if err.Error() != "Test error message" {
		t.Errorf("Error() = %s, want Test error message", err.Error())
	}
}

func TestHeaderMatchesToDTO(t *testing.T) {
	tests := []struct {
		name   string
		input  []route.HeaderMatch
		expect int
	}{
		{"nil input", nil, 0},
		{"empty input", []route.HeaderMatch{}, 0},
		{"single header", []route.HeaderMatch{{Name: "X-Test", Value: "value", IsRegex: false, Required: true}}, 1},
		{"multiple headers", []route.HeaderMatch{
			{Name: "X-Test1", Value: "v1", IsRegex: false, Required: true},
			{Name: "X-Test2", Value: "v2", IsRegex: true, Required: false},
		}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := headerMatchesToDTO(tt.input)
			if tt.input == nil {
				if result != nil {
					t.Errorf("expected nil for nil input")
				}
				return
			}
			if len(result) != tt.expect {
				t.Errorf("len(result) = %d, want %d", len(result), tt.expect)
			}
			for i, h := range tt.input {
				if result[i].Name != h.Name {
					t.Errorf("Name = %s, want %s", result[i].Name, h.Name)
				}
				if result[i].Value != h.Value {
					t.Errorf("Value = %s, want %s", result[i].Value, h.Value)
				}
				if result[i].IsRegex != h.IsRegex {
					t.Errorf("IsRegex = %v, want %v", result[i].IsRegex, h.IsRegex)
				}
				if result[i].Required != h.Required {
					t.Errorf("Required = %v, want %v", result[i].Required, h.Required)
				}
			}
		})
	}
}

func TestDTOToHeaderMatches(t *testing.T) {
	tests := []struct {
		name   string
		input  []HeaderMatchDTO
		expect int
	}{
		{"nil input", nil, 0},
		{"empty input", []HeaderMatchDTO{}, 0},
		{"single header", []HeaderMatchDTO{{Name: "X-Test", Value: "value", IsRegex: false, Required: true}}, 1},
		{"multiple headers", []HeaderMatchDTO{
			{Name: "X-Test1", Value: "v1", IsRegex: false, Required: true},
			{Name: "X-Test2", Value: "v2", IsRegex: true, Required: false},
		}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dtoToHeaderMatches(tt.input)
			if tt.input == nil {
				if result != nil {
					t.Errorf("expected nil for nil input")
				}
				return
			}
			if len(result) != tt.expect {
				t.Errorf("len(result) = %d, want %d", len(result), tt.expect)
			}
			for i, h := range tt.input {
				if result[i].Name != h.Name {
					t.Errorf("Name = %s, want %s", result[i].Name, h.Name)
				}
				if result[i].Value != h.Value {
					t.Errorf("Value = %s, want %s", result[i].Value, h.Value)
				}
				if result[i].IsRegex != h.IsRegex {
					t.Errorf("IsRegex = %v, want %v", result[i].IsRegex, h.IsRegex)
				}
				if result[i].Required != h.Required {
					t.Errorf("Required = %v, want %v", result[i].Required, h.Required)
				}
			}
		})
	}
}

func TestTransformToDTO(t *testing.T) {
	tests := []struct {
		name  string
		input *route.Transform
	}{
		{"nil input", nil},
		{"empty transform", &route.Transform{}},
		{"full transform", &route.Transform{
			SetHeaders:    map[string]string{"X-Custom": "value"},
			DeleteHeaders: []string{"X-Remove"},
			BodyExpr:      "body expression",
			SetQuery:      map[string]string{"key": "value"},
			DeleteQuery:   []string{"oldkey"},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transformToDTO(tt.input)
			if tt.input == nil {
				if result != nil {
					t.Errorf("expected nil for nil input")
				}
				return
			}
			if result.BodyExpr != tt.input.BodyExpr {
				t.Errorf("BodyExpr = %s, want %s", result.BodyExpr, tt.input.BodyExpr)
			}
		})
	}
}

func TestDTOToTransform(t *testing.T) {
	tests := []struct {
		name  string
		input *TransformDTO
	}{
		{"nil input", nil},
		{"empty transform", &TransformDTO{}},
		{"full transform", &TransformDTO{
			SetHeaders:    map[string]string{"X-Custom": "value"},
			DeleteHeaders: []string{"X-Remove"},
			BodyExpr:      "body expression",
			SetQuery:      map[string]string{"key": "value"},
			DeleteQuery:   []string{"oldkey"},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dtoToTransform(tt.input)
			if tt.input == nil {
				if result != nil {
					t.Errorf("expected nil for nil input")
				}
				return
			}
			if result.BodyExpr != tt.input.BodyExpr {
				t.Errorf("BodyExpr = %s, want %s", result.BodyExpr, tt.input.BodyExpr)
			}
		})
	}
}

func TestKeyToResource(t *testing.T) {
	createdAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Test with nil optional fields
	result := keyToResource(key.Key{
		ID:        "key1",
		Name:      "Test Key",
		Prefix:    "test_",
		UserID:    "user1",
		CreatedAt: createdAt,
	})

	if result.ID != "key1" {
		t.Errorf("ID = %s, want key1", result.ID)
	}
	if result.Type != TypeKey {
		t.Errorf("Type = %s, want %s", result.Type, TypeKey)
	}
	if result.Attributes["name"] != "Test Key" {
		t.Errorf("name = %s, want Test Key", result.Attributes["name"])
	}
	if result.Attributes["prefix"] != "test_" {
		t.Errorf("prefix = %s, want test_", result.Attributes["prefix"])
	}
	// Optional fields should not be present when nil
	if _, ok := result.Attributes["revoked_at"]; ok {
		t.Error("revoked_at should not be present")
	}
	if _, ok := result.Attributes["expires_at"]; ok {
		t.Error("expires_at should not be present")
	}
	if _, ok := result.Attributes["last_used"]; ok {
		t.Error("last_used should not be present")
	}

	// Test with all optional fields set
	revokedAt := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	expiresAt := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	lastUsed := time.Date(2024, 5, 15, 0, 0, 0, 0, time.UTC)
	result2 := keyToResource(key.Key{
		ID:        "key2",
		Name:      "Revoked Key",
		Prefix:    "rev_",
		UserID:    "user2",
		CreatedAt: createdAt,
		RevokedAt: &revokedAt,
		ExpiresAt: &expiresAt,
		LastUsed:  &lastUsed,
	})

	if _, ok := result2.Attributes["revoked_at"]; !ok {
		t.Error("revoked_at should be present")
	}
	if _, ok := result2.Attributes["expires_at"]; !ok {
		t.Error("expires_at should be present")
	}
	if _, ok := result2.Attributes["last_used"]; !ok {
		t.Error("last_used should be present")
	}
}
