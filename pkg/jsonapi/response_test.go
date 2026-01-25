package jsonapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteDocument(t *testing.T) {
	t.Run("sets content type and status", func(t *testing.T) {
		w := httptest.NewRecorder()
		doc := NewDocument().DataResource(Resource{Type: "users", ID: "1"}).Build()

		WriteDocument(w, http.StatusOK, doc)

		if w.Header().Get("Content-Type") != ContentType {
			t.Errorf("Content-Type = %v, want %v", w.Header().Get("Content-Type"), ContentType)
		}
		if w.Code != http.StatusOK {
			t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("writes valid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		doc := NewDocument().DataResource(Resource{Type: "users", ID: "1"}).Build()

		WriteDocument(w, http.StatusOK, doc)

		var result Document
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Errorf("Invalid JSON: %v", err)
		}
	})
}

func TestWriteResource(t *testing.T) {
	w := httptest.NewRecorder()
	r := Resource{Type: "users", ID: "123", Attributes: map[string]any{"name": "John"}}

	WriteResource(w, http.StatusOK, r)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var doc Document
	if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}
}

func TestWriteCollection(t *testing.T) {
	t.Run("without pagination", func(t *testing.T) {
		w := httptest.NewRecorder()
		resources := []Resource{
			{Type: "users", ID: "1"},
			{Type: "users", ID: "2"},
		}

		WriteCollection(w, http.StatusOK, resources, nil)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("with pagination", func(t *testing.T) {
		w := httptest.NewRecorder()
		resources := []Resource{{Type: "users", ID: "1"}}
		pagination := NewPagination(100, 1, 10, "/users")

		WriteCollection(w, http.StatusOK, resources, pagination)

		var doc Document
		if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
			t.Fatalf("Invalid JSON: %v", err)
		}
		if doc.Meta == nil {
			t.Error("Pagination should add meta")
		}
		if doc.Links == nil {
			t.Error("Pagination should add links")
		}
	})
}

func TestWriteError(t *testing.T) {
	t.Run("writes single error", func(t *testing.T) {
		w := httptest.NewRecorder()

		WriteError(w, ErrNotFound("users"))

		if w.Code != http.StatusNotFound {
			t.Errorf("Status = %d, want %d", w.Code, http.StatusNotFound)
		}

		var doc Document
		if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
			t.Fatalf("Invalid JSON: %v", err)
		}
		if len(doc.Errors) != 1 {
			t.Errorf("len(Errors) = %d, want 1", len(doc.Errors))
		}
	})

	t.Run("writes multiple errors", func(t *testing.T) {
		w := httptest.NewRecorder()

		WriteError(w, ErrValidation("name", "is required"), ErrValidation("email", "is invalid"))

		// Status should be from first error (422)
		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("Status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
		}

		var doc Document
		json.Unmarshal(w.Body.Bytes(), &doc)
		if len(doc.Errors) != 2 {
			t.Errorf("len(Errors) = %d, want 2", len(doc.Errors))
		}
	})

	t.Run("handles empty errors", func(t *testing.T) {
		w := httptest.NewRecorder()

		WriteError(w)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Status = %d, want %d for empty errors", w.Code, http.StatusInternalServerError)
		}
	})
}

func TestWriteCreated(t *testing.T) {
	t.Run("with location header", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := Resource{Type: "users", ID: "123"}

		WriteCreated(w, r, "/users/123")

		if w.Code != http.StatusCreated {
			t.Errorf("Status = %d, want %d", w.Code, http.StatusCreated)
		}
		if w.Header().Get("Location") != "/users/123" {
			t.Errorf("Location = %v, want /users/123", w.Header().Get("Location"))
		}
	})

	t.Run("without location header", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := Resource{Type: "users", ID: "123"}

		WriteCreated(w, r, "")

		if w.Code != http.StatusCreated {
			t.Errorf("Status = %d, want %d", w.Code, http.StatusCreated)
		}
		if w.Header().Get("Location") != "" {
			t.Errorf("Location should be empty, got %v", w.Header().Get("Location"))
		}
	})
}

func TestWriteNoContent(t *testing.T) {
	w := httptest.NewRecorder()

	WriteNoContent(w)

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if w.Body.Len() != 0 {
		t.Error("Body should be empty for NoContent")
	}
}

func TestWriteAccepted(t *testing.T) {
	w := httptest.NewRecorder()
	meta := Meta{"job_id": "abc123", "status": "queued"}

	WriteAccepted(w, meta)

	if w.Code != http.StatusAccepted {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusAccepted)
	}

	var doc Document
	json.Unmarshal(w.Body.Bytes(), &doc)
	if doc.Meta["job_id"] != "abc123" {
		t.Errorf("Meta[job_id] = %v, want abc123", doc.Meta["job_id"])
	}
}

func TestWriteMeta(t *testing.T) {
	w := httptest.NewRecorder()
	meta := Meta{"count": 42}

	WriteMeta(w, http.StatusOK, meta)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var doc Document
	json.Unmarshal(w.Body.Bytes(), &doc)
	if doc.Meta["count"] != float64(42) {
		t.Errorf("Meta[count] = %v, want 42", doc.Meta["count"])
	}
}

func TestConvenienceErrorWriters(t *testing.T) {
	tests := []struct {
		name       string
		write      func(w http.ResponseWriter)
		wantStatus int
	}{
		{"WriteBadRequest", func(w http.ResponseWriter) { WriteBadRequest(w, "invalid input") }, http.StatusBadRequest},
		{"WriteUnauthorized", func(w http.ResponseWriter) { WriteUnauthorized(w, "missing token") }, http.StatusUnauthorized},
		{"WriteForbidden", func(w http.ResponseWriter) { WriteForbidden(w, "access denied") }, http.StatusForbidden},
		{"WriteNotFound", func(w http.ResponseWriter) { WriteNotFound(w, "users") }, http.StatusNotFound},
		{"WriteConflict", func(w http.ResponseWriter) { WriteConflict(w, "already exists") }, http.StatusConflict},
		{"WriteValidationError", func(w http.ResponseWriter) { WriteValidationError(w, "email", "is invalid") }, http.StatusUnprocessableEntity},
		{"WriteInternalError", func(w http.ResponseWriter) { WriteInternalError(w, "something broke") }, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tt.write(w)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestWriteMethodNotAllowed(t *testing.T) {
	t.Run("sets Allow header", func(t *testing.T) {
		w := httptest.NewRecorder()

		WriteMethodNotAllowed(w, "DELETE", []string{"GET", "POST", "PUT"})

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
		}
		allow := w.Header().Get("Allow")
		if allow != "GET, POST, PUT" {
			t.Errorf("Allow = %v, want GET, POST, PUT", allow)
		}
	})

	t.Run("handles empty allowed methods", func(t *testing.T) {
		w := httptest.NewRecorder()

		WriteMethodNotAllowed(w, "DELETE", nil)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
		}
		if w.Header().Get("Allow") != "" {
			t.Errorf("Allow should be empty, got %v", w.Header().Get("Allow"))
		}
	})
}

func TestWriteErrorFromGo(t *testing.T) {
	w := httptest.NewRecorder()
	err := errors.New("something went wrong")

	WriteErrorFromGo(w, err)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var doc Document
	json.Unmarshal(w.Body.Bytes(), &doc)
	if len(doc.Errors) != 1 {
		t.Errorf("len(Errors) = %d, want 1", len(doc.Errors))
	}
}

func TestContentType(t *testing.T) {
	if ContentType != "application/vnd.api+json" {
		t.Errorf("ContentType = %v, want application/vnd.api+json", ContentType)
	}
}

func TestVersion(t *testing.T) {
	if Version != "1.1" {
		t.Errorf("Version = %v, want 1.1", Version)
	}
}
