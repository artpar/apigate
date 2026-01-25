package jsonapi

import (
	"errors"
	"testing"
)

func TestErrorBuilder(t *testing.T) {
	t.Run("NewError creates builder with basic fields", func(t *testing.T) {
		err := NewError(404, "not_found", "Not Found").Build()

		if err.Status != "404" {
			t.Errorf("Status = %v, want 404", err.Status)
		}
		if err.Code != "not_found" {
			t.Errorf("Code = %v, want not_found", err.Code)
		}
		if err.Title != "Not Found" {
			t.Errorf("Title = %v, want Not Found", err.Title)
		}
	})

	t.Run("Detail sets detail message", func(t *testing.T) {
		err := NewError(400, "bad_request", "Bad Request").
			Detail("Invalid input").
			Build()

		if err.Detail != "Invalid input" {
			t.Errorf("Detail = %v, want Invalid input", err.Detail)
		}
	})

	t.Run("Detailf formats detail message", func(t *testing.T) {
		err := NewError(404, "not_found", "Not Found").
			Detailf("User with ID '%s' not found", "123").
			Build()

		if err.Detail != "User with ID '123' not found" {
			t.Errorf("Detail = %v, want formatted string", err.Detail)
		}
	})

	t.Run("ID sets error ID", func(t *testing.T) {
		err := NewError(500, "internal", "Error").
			ID("err-123").
			Build()

		if err.ID != "err-123" {
			t.Errorf("ID = %v, want err-123", err.ID)
		}
	})

	t.Run("Pointer sets source pointer", func(t *testing.T) {
		err := NewError(422, "validation", "Validation").
			Pointer("/data/attributes/email").
			Build()

		if err.Source == nil {
			t.Fatal("Source should not be nil")
		}
		if err.Source.Pointer != "/data/attributes/email" {
			t.Errorf("Pointer = %v, want /data/attributes/email", err.Source.Pointer)
		}
	})

	t.Run("Parameter sets source parameter", func(t *testing.T) {
		err := NewError(400, "bad_request", "Bad Request").
			Parameter("page").
			Build()

		if err.Source == nil {
			t.Fatal("Source should not be nil")
		}
		if err.Source.Parameter != "page" {
			t.Errorf("Parameter = %v, want page", err.Source.Parameter)
		}
	})

	t.Run("Header sets source header", func(t *testing.T) {
		err := NewError(401, "unauthorized", "Unauthorized").
			Header("Authorization").
			Build()

		if err.Source == nil {
			t.Fatal("Source should not be nil")
		}
		if err.Source.Header != "Authorization" {
			t.Errorf("Header = %v, want Authorization", err.Source.Header)
		}
	})

	t.Run("Meta adds metadata", func(t *testing.T) {
		err := NewError(429, "rate_limit", "Too Many Requests").
			Meta("retry_after", 60).
			Meta("limit", 100).
			Build()

		if err.Meta == nil {
			t.Fatal("Meta should not be nil")
		}
		if err.Meta["retry_after"] != 60 {
			t.Errorf("Meta[retry_after] = %v, want 60", err.Meta["retry_after"])
		}
		if err.Meta["limit"] != 100 {
			t.Errorf("Meta[limit] = %v, want 100", err.Meta["limit"])
		}
	})

	t.Run("AboutLink sets links.about", func(t *testing.T) {
		err := NewError(400, "bad_request", "Bad Request").
			AboutLink("https://docs.example.com/errors/bad_request").
			Build()

		if err.Links == nil {
			t.Fatal("Links should not be nil")
		}
		if err.Links.About != "https://docs.example.com/errors/bad_request" {
			t.Errorf("Links.About = %v, want docs URL", err.Links.About)
		}
	})

	t.Run("chaining multiple methods", func(t *testing.T) {
		err := NewError(422, "validation_error", "Validation Failed").
			ID("err-456").
			Detail("Email is invalid").
			Pointer("/data/attributes/email").
			Meta("field", "email").
			AboutLink("https://docs.example.com/validation").
			Build()

		if err.ID != "err-456" {
			t.Error("ID not set correctly")
		}
		if err.Detail != "Email is invalid" {
			t.Error("Detail not set correctly")
		}
		if err.Source == nil || err.Source.Pointer != "/data/attributes/email" {
			t.Error("Pointer not set correctly")
		}
		if err.Meta == nil || err.Meta["field"] != "email" {
			t.Error("Meta not set correctly")
		}
		if err.Links == nil || err.Links.About == "" {
			t.Error("Links not set correctly")
		}
	})
}

func TestStatusCode(t *testing.T) {
	tests := []struct {
		status     string
		wantCode   int
	}{
		{"200", 200},
		{"400", 400},
		{"404", 404},
		{"500", 500},
		{"invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			err := Error{Status: tt.status}
			got := err.StatusCode()
			if got != tt.wantCode {
				t.Errorf("StatusCode() = %d, want %d", got, tt.wantCode)
			}
		})
	}
}

func TestErrBadRequest(t *testing.T) {
	err := ErrBadRequest("Invalid JSON")

	if err.Status != "400" {
		t.Errorf("Status = %v, want 400", err.Status)
	}
	if err.Code != "bad_request" {
		t.Errorf("Code = %v, want bad_request", err.Code)
	}
	if err.Detail != "Invalid JSON" {
		t.Errorf("Detail = %v, want Invalid JSON", err.Detail)
	}
}

func TestErrUnauthorized(t *testing.T) {
	t.Run("with detail", func(t *testing.T) {
		err := ErrUnauthorized("Invalid token")
		if err.Detail != "Invalid token" {
			t.Errorf("Detail = %v, want Invalid token", err.Detail)
		}
	})

	t.Run("without detail", func(t *testing.T) {
		err := ErrUnauthorized("")
		if err.Detail != "Authentication required" {
			t.Errorf("Detail = %v, want Authentication required", err.Detail)
		}
	})

	t.Run("status is 401", func(t *testing.T) {
		err := ErrUnauthorized("")
		if err.Status != "401" {
			t.Errorf("Status = %v, want 401", err.Status)
		}
	})
}

func TestErrForbidden(t *testing.T) {
	t.Run("with detail", func(t *testing.T) {
		err := ErrForbidden("Admin only")
		if err.Detail != "Admin only" {
			t.Errorf("Detail = %v, want Admin only", err.Detail)
		}
	})

	t.Run("without detail", func(t *testing.T) {
		err := ErrForbidden("")
		if err.Detail != "Access denied" {
			t.Errorf("Detail = %v, want Access denied", err.Detail)
		}
	})

	t.Run("status is 403", func(t *testing.T) {
		err := ErrForbidden("")
		if err.Status != "403" {
			t.Errorf("Status = %v, want 403", err.Status)
		}
	})
}

func TestErrNotFound(t *testing.T) {
	err := ErrNotFound("user")

	if err.Status != "404" {
		t.Errorf("Status = %v, want 404", err.Status)
	}
	if err.Code != "not_found" {
		t.Errorf("Code = %v, want not_found", err.Code)
	}
	if err.Detail != "The requested user was not found" {
		t.Errorf("Detail = %v, want formatted message", err.Detail)
	}
}

func TestErrNotFoundWithID(t *testing.T) {
	err := ErrNotFoundWithID("user", "123")

	if err.Status != "404" {
		t.Errorf("Status = %v, want 404", err.Status)
	}
	if err.Detail != "The user with ID '123' was not found" {
		t.Errorf("Detail = %v, want formatted message with ID", err.Detail)
	}
}

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

func TestErrConflict(t *testing.T) {
	err := ErrConflict("Resource already exists")

	if err.Status != "409" {
		t.Errorf("Status = %v, want 409", err.Status)
	}
	if err.Code != "conflict" {
		t.Errorf("Code = %v, want conflict", err.Code)
	}
	if err.Detail != "Resource already exists" {
		t.Errorf("Detail = %v, want Resource already exists", err.Detail)
	}
}

func TestErrValidation(t *testing.T) {
	err := ErrValidation("email", "must be a valid email")

	if err.Status != "422" {
		t.Errorf("Status = %v, want 422", err.Status)
	}
	if err.Code != "validation_error" {
		t.Errorf("Code = %v, want validation_error", err.Code)
	}
	if err.Detail != "must be a valid email" {
		t.Errorf("Detail = %v, want must be a valid email", err.Detail)
	}
	if err.Source == nil || err.Source.Pointer != "/data/attributes/email" {
		t.Errorf("Pointer = %v, want /data/attributes/email", err.Source)
	}
}

func TestErrValidationRequired(t *testing.T) {
	err := ErrValidationRequired("name")

	if err.Detail != "name is required" {
		t.Errorf("Detail = %v, want name is required", err.Detail)
	}
	if err.Source == nil || err.Source.Pointer != "/data/attributes/name" {
		t.Errorf("Pointer should point to name field")
	}
}

func TestErrValidationInvalid(t *testing.T) {
	err := ErrValidationInvalid("age", "must be positive")

	if err.Detail != "age is invalid: must be positive" {
		t.Errorf("Detail = %v, want formatted message", err.Detail)
	}
}

func TestErrRateLimited(t *testing.T) {
	t.Run("with detail", func(t *testing.T) {
		err := ErrRateLimited("100 requests per minute")
		if err.Detail != "100 requests per minute" {
			t.Errorf("Detail = %v, want 100 requests per minute", err.Detail)
		}
	})

	t.Run("without detail", func(t *testing.T) {
		err := ErrRateLimited("")
		if err.Detail != "Rate limit exceeded" {
			t.Errorf("Detail = %v, want Rate limit exceeded", err.Detail)
		}
	})

	t.Run("status is 429", func(t *testing.T) {
		err := ErrRateLimited("")
		if err.Status != "429" {
			t.Errorf("Status = %v, want 429", err.Status)
		}
	})
}

func TestErrInternal(t *testing.T) {
	t.Run("with detail", func(t *testing.T) {
		err := ErrInternal("Database connection failed")
		if err.Detail != "Database connection failed" {
			t.Errorf("Detail = %v, want Database connection failed", err.Detail)
		}
	})

	t.Run("without detail", func(t *testing.T) {
		err := ErrInternal("")
		if err.Detail != "An internal error occurred" {
			t.Errorf("Detail = %v, want An internal error occurred", err.Detail)
		}
	})

	t.Run("status is 500", func(t *testing.T) {
		err := ErrInternal("")
		if err.Status != "500" {
			t.Errorf("Status = %v, want 500", err.Status)
		}
	})
}

func TestErrNotImplemented(t *testing.T) {
	err := ErrNotImplemented("Batch processing")

	if err.Status != "501" {
		t.Errorf("Status = %v, want 501", err.Status)
	}
	if err.Code != "not_implemented" {
		t.Errorf("Code = %v, want not_implemented", err.Code)
	}
	if err.Detail != "Batch processing is not implemented" {
		t.Errorf("Detail = %v, want Batch processing is not implemented", err.Detail)
	}
}

func TestErrServiceUnavailable(t *testing.T) {
	t.Run("with detail", func(t *testing.T) {
		err := ErrServiceUnavailable("Server maintenance")
		if err.Detail != "Server maintenance" {
			t.Errorf("Detail = %v, want Server maintenance", err.Detail)
		}
	})

	t.Run("without detail", func(t *testing.T) {
		err := ErrServiceUnavailable("")
		if err.Detail != "Service temporarily unavailable" {
			t.Errorf("Detail = %v, want Service temporarily unavailable", err.Detail)
		}
	})

	t.Run("status is 503", func(t *testing.T) {
		err := ErrServiceUnavailable("")
		if err.Status != "503" {
			t.Errorf("Status = %v, want 503", err.Status)
		}
	})
}

func TestErrFromError(t *testing.T) {
	t.Run("with error", func(t *testing.T) {
		goErr := errors.New("connection refused")
		err := ErrFromError(goErr)

		if err.Status != "500" {
			t.Errorf("Status = %v, want 500", err.Status)
		}
		if err.Detail != "connection refused" {
			t.Errorf("Detail = %v, want connection refused", err.Detail)
		}
	})

	t.Run("with nil error", func(t *testing.T) {
		err := ErrFromError(nil)

		if err.Status != "500" {
			t.Errorf("Status = %v, want 500", err.Status)
		}
		if err.Detail != "An internal error occurred" {
			t.Errorf("Detail = %v, want An internal error occurred", err.Detail)
		}
	})
}

// Metering API Error Tests

func TestErrInvalidEventType(t *testing.T) {
	err := ErrInvalidEventType("unknown_event")

	if err.Status != "422" {
		t.Errorf("Status = %v, want 422", err.Status)
	}
	if err.Code != "invalid_event_type" {
		t.Errorf("Code = %v, want invalid_event_type", err.Code)
	}
	if err.Detail != "Event type 'unknown_event' is not recognized" {
		t.Errorf("Detail = %v, want formatted message", err.Detail)
	}
}

func TestErrDuplicateEvent(t *testing.T) {
	err := ErrDuplicateEvent("evt-123")

	if err.Status != "409" {
		t.Errorf("Status = %v, want 409", err.Status)
	}
	if err.Code != "duplicate_event" {
		t.Errorf("Code = %v, want duplicate_event", err.Code)
	}
	if err.Detail != "Event with ID 'evt-123' has already been processed" {
		t.Errorf("Detail = %v, want formatted message", err.Detail)
	}
}

func TestErrUserNotFoundForMeter(t *testing.T) {
	err := ErrUserNotFoundForMeter("user-456")

	if err.Status != "422" {
		t.Errorf("Status = %v, want 422", err.Status)
	}
	if err.Code != "user_not_found" {
		t.Errorf("Code = %v, want user_not_found", err.Code)
	}
	if err.Source == nil || err.Source.Pointer != "/data/attributes/user_id" {
		t.Error("Pointer should point to user_id")
	}
}

func TestErrInvalidQuantity(t *testing.T) {
	err := ErrInvalidQuantity(-5.5)

	if err.Status != "422" {
		t.Errorf("Status = %v, want 422", err.Status)
	}
	if err.Code != "invalid_quantity" {
		t.Errorf("Code = %v, want invalid_quantity", err.Code)
	}
	if err.Source == nil || err.Source.Pointer != "/data/attributes/quantity" {
		t.Error("Pointer should point to quantity")
	}
}

func TestErrInvalidTimestamp(t *testing.T) {
	err := ErrInvalidTimestamp("Timestamp cannot be in the future")

	if err.Status != "422" {
		t.Errorf("Status = %v, want 422", err.Status)
	}
	if err.Code != "invalid_timestamp" {
		t.Errorf("Code = %v, want invalid_timestamp", err.Code)
	}
	if err.Detail != "Timestamp cannot be in the future" {
		t.Errorf("Detail = %v, want Timestamp cannot be in the future", err.Detail)
	}
	if err.Source == nil || err.Source.Pointer != "/data/attributes/timestamp" {
		t.Error("Pointer should point to timestamp")
	}
}

func TestErrInsufficientScope(t *testing.T) {
	err := ErrInsufficientScope("meter:write")

	if err.Status != "403" {
		t.Errorf("Status = %v, want 403", err.Status)
	}
	if err.Code != "insufficient_scope" {
		t.Errorf("Code = %v, want insufficient_scope", err.Code)
	}
	if err.Detail != "API key requires 'meter:write' scope to perform this action" {
		t.Errorf("Detail = %v, want formatted message", err.Detail)
	}
}
