package jsonapi

import (
	"encoding/json"
	"net/http"
)

// WriteDocument writes a JSON:API document to the response.
func WriteDocument(w http.ResponseWriter, status int, doc Document) {
	w.Header().Set("Content-Type", ContentType)
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(doc)
}

// WriteResource writes a single resource response.
func WriteResource(w http.ResponseWriter, status int, r Resource) {
	WriteDocument(w, status, NewSingleResourceDocument(r))
}

// WriteCollection writes a collection response with optional pagination.
func WriteCollection(w http.ResponseWriter, status int, resources []Resource, pagination *Pagination) {
	WriteDocument(w, status, NewCollectionDocument(resources, pagination))
}

// WriteError writes an error response with one or more errors.
// The HTTP status is derived from the first error's status field.
func WriteError(w http.ResponseWriter, errs ...Error) {
	if len(errs) == 0 {
		WriteDocument(w, http.StatusInternalServerError, NewErrorDocument(ErrInternal("")))
		return
	}

	status := errs[0].StatusCode()
	if status == 0 {
		status = http.StatusInternalServerError
	}

	WriteDocument(w, status, NewErrorDocument(errs...))
}

// WriteCreated writes a 201 Created response with the resource and optional Location header.
func WriteCreated(w http.ResponseWriter, r Resource, location string) {
	if location != "" {
		w.Header().Set("Location", location)
	}
	WriteResource(w, http.StatusCreated, r)
}

// WriteNoContent writes a 204 No Content response (typically for DELETE).
func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// WriteAccepted writes a 202 Accepted response (for async operations).
func WriteAccepted(w http.ResponseWriter, meta Meta) {
	doc := NewDocument().MetaAll(meta).Build()
	WriteDocument(w, http.StatusAccepted, doc)
}

// WriteMeta writes a response with only metadata (no data).
func WriteMeta(w http.ResponseWriter, status int, meta Meta) {
	doc := NewDocument().MetaAll(meta).Build()
	WriteDocument(w, status, doc)
}

// WriteBadRequest is a convenience for 400 errors.
func WriteBadRequest(w http.ResponseWriter, detail string) {
	WriteError(w, ErrBadRequest(detail))
}

// WriteUnauthorized is a convenience for 401 errors.
func WriteUnauthorized(w http.ResponseWriter, detail string) {
	WriteError(w, ErrUnauthorized(detail))
}

// WriteForbidden is a convenience for 403 errors.
func WriteForbidden(w http.ResponseWriter, detail string) {
	WriteError(w, ErrForbidden(detail))
}

// WriteNotFound is a convenience for 404 errors.
func WriteNotFound(w http.ResponseWriter, resourceType string) {
	WriteError(w, ErrNotFound(resourceType))
}

// WriteMethodNotAllowed is a convenience for 405 errors.
// It sets the Allow header per RFC 7231 and includes allowed methods in the error body.
func WriteMethodNotAllowed(w http.ResponseWriter, method string, allowedMethods []string) {
	// Set Allow header per RFC 7231
	if len(allowedMethods) > 0 {
		allowHeader := ""
		for i, m := range allowedMethods {
			if i > 0 {
				allowHeader += ", "
			}
			allowHeader += m
		}
		w.Header().Set("Allow", allowHeader)
	}
	WriteError(w, ErrMethodNotAllowed(method, allowedMethods))
}

// WriteConflict is a convenience for 409 errors.
func WriteConflict(w http.ResponseWriter, detail string) {
	WriteError(w, ErrConflict(detail))
}

// WriteValidationError is a convenience for 422 validation errors.
func WriteValidationError(w http.ResponseWriter, field, message string) {
	WriteError(w, ErrValidation(field, message))
}

// WriteInternalError is a convenience for 500 errors.
func WriteInternalError(w http.ResponseWriter, detail string) {
	WriteError(w, ErrInternal(detail))
}

// WriteErrorFromGo converts a Go error to a JSON:API error response.
func WriteErrorFromGo(w http.ResponseWriter, err error) {
	WriteError(w, ErrFromError(err))
}
