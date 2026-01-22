package jsonapi

import (
	"testing"
)

func TestErrMethodNotAllowed(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		allowed        []string
		wantStatus     string
		wantCode       string
		wantDetailHas  string
		wantMetaMethod string
	}{
		{
			name:           "with allowed methods",
			method:         "PATCH",
			allowed:        []string{"GET", "PUT", "DELETE"},
			wantStatus:     "405",
			wantCode:       "method_not_allowed",
			wantDetailHas:  "PATCH is not supported. Use one of: GET, PUT, DELETE",
			wantMetaMethod: "PATCH",
		},
		{
			name:           "single allowed method",
			method:         "POST",
			allowed:        []string{"GET"},
			wantStatus:     "405",
			wantCode:       "method_not_allowed",
			wantDetailHas:  "POST is not supported. Use one of: GET",
			wantMetaMethod: "POST",
		},
		{
			name:           "no allowed methods",
			method:         "DELETE",
			allowed:        []string{},
			wantStatus:     "405",
			wantCode:       "method_not_allowed",
			wantDetailHas:  "The DELETE method is not allowed for this resource",
			wantMetaMethod: "DELETE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ErrMethodNotAllowed(tt.method, tt.allowed)

			if err.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", err.Status, tt.wantStatus)
			}

			if err.Code != tt.wantCode {
				t.Errorf("Code = %v, want %v", err.Code, tt.wantCode)
			}

			if err.Detail != tt.wantDetailHas {
				t.Errorf("Detail = %v, want %v", err.Detail, tt.wantDetailHas)
			}

			if err.Meta == nil {
				t.Fatal("Meta is nil, want non-nil")
			}

			if method, ok := err.Meta["requested_method"].(string); !ok || method != tt.wantMetaMethod {
				t.Errorf("Meta[requested_method] = %v, want %v", method, tt.wantMetaMethod)
			}

			if len(tt.allowed) > 0 {
				allowedMethods, ok := err.Meta["allowed_methods"].([]string)
				if !ok {
					t.Errorf("Meta[allowed_methods] is not []string")
				} else if len(allowedMethods) != len(tt.allowed) {
					t.Errorf("Meta[allowed_methods] length = %d, want %d", len(allowedMethods), len(tt.allowed))
				}
			}
		})
	}
}

func TestErrMethodNotAllowed_StatusCode(t *testing.T) {
	err := ErrMethodNotAllowed("PATCH", []string{"GET", "PUT"})

	statusCode := err.StatusCode()
	if statusCode != 405 {
		t.Errorf("StatusCode() = %d, want 405", statusCode)
	}
}
