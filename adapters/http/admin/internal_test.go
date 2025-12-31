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

func TestKeyToResponse(t *testing.T) {
	createdAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Test with nil optional fields
	result := keyToResponse(key.Key{
		ID:        "key1",
		Name:      "Test Key",
		Prefix:    "test_",
		UserID:    "user1",
		CreatedAt: createdAt,
	})

	if result.ID != "key1" {
		t.Errorf("ID = %s, want key1", result.ID)
	}
	if result.Name != "Test Key" {
		t.Errorf("Name = %s, want Test Key", result.Name)
	}
	if result.Prefix != "test_" {
		t.Errorf("Prefix = %s, want test_", result.Prefix)
	}
	if result.RevokedAt != nil {
		t.Errorf("RevokedAt should be nil")
	}
	if result.ExpiresAt != nil {
		t.Errorf("ExpiresAt should be nil")
	}
	if result.LastUsed != nil {
		t.Errorf("LastUsed should be nil")
	}

	// Test with all optional fields set
	revokedAt := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	expiresAt := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	lastUsed := time.Date(2024, 5, 15, 0, 0, 0, 0, time.UTC)
	result2 := keyToResponse(key.Key{
		ID:        "key2",
		Name:      "Revoked Key",
		Prefix:    "rev_",
		UserID:    "user2",
		CreatedAt: createdAt,
		RevokedAt: &revokedAt,
		ExpiresAt: &expiresAt,
		LastUsed:  &lastUsed,
	})

	if result2.RevokedAt == nil {
		t.Errorf("RevokedAt should not be nil")
	}
	if result2.ExpiresAt == nil {
		t.Errorf("ExpiresAt should not be nil")
	}
	if result2.LastUsed == nil {
		t.Errorf("LastUsed should not be nil")
	}
}
