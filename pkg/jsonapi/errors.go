package jsonapi

import (
	"fmt"
	"strconv"
)

// ErrorBuilder provides a fluent API for building Error objects.
type ErrorBuilder struct {
	err Error
}

// NewError creates a new ErrorBuilder with the given status, code, and title.
func NewError(status int, code, title string) *ErrorBuilder {
	return &ErrorBuilder{
		err: Error{
			Status: strconv.Itoa(status),
			Code:   code,
			Title:  title,
		},
	}
}

// Detail sets the error detail message.
func (b *ErrorBuilder) Detail(detail string) *ErrorBuilder {
	b.err.Detail = detail
	return b
}

// Detailf sets the error detail message with formatting.
func (b *ErrorBuilder) Detailf(format string, args ...any) *ErrorBuilder {
	b.err.Detail = fmt.Sprintf(format, args...)
	return b
}

// ID sets the error ID.
func (b *ErrorBuilder) ID(id string) *ErrorBuilder {
	b.err.ID = id
	return b
}

// Pointer sets the JSON pointer to the source of the error.
// Example: "/data/attributes/email"
func (b *ErrorBuilder) Pointer(pointer string) *ErrorBuilder {
	if b.err.Source == nil {
		b.err.Source = &ErrorSource{}
	}
	b.err.Source.Pointer = pointer
	return b
}

// Parameter sets the query parameter that caused the error.
func (b *ErrorBuilder) Parameter(param string) *ErrorBuilder {
	if b.err.Source == nil {
		b.err.Source = &ErrorSource{}
	}
	b.err.Source.Parameter = param
	return b
}

// Header sets the header that caused the error.
func (b *ErrorBuilder) Header(header string) *ErrorBuilder {
	if b.err.Source == nil {
		b.err.Source = &ErrorSource{}
	}
	b.err.Source.Header = header
	return b
}

// Meta adds metadata to the error.
func (b *ErrorBuilder) Meta(key string, value any) *ErrorBuilder {
	if b.err.Meta == nil {
		b.err.Meta = make(Meta)
	}
	b.err.Meta[key] = value
	return b
}

// AboutLink sets the about link for more information about the error.
func (b *ErrorBuilder) AboutLink(url string) *ErrorBuilder {
	if b.err.Links == nil {
		b.err.Links = &ErrorLinks{}
	}
	b.err.Links.About = url
	return b
}

// Build returns the constructed Error.
func (b *ErrorBuilder) Build() Error {
	return b.err
}

// StatusCode returns the HTTP status code as an int.
func (e Error) StatusCode() int {
	code, _ := strconv.Atoi(e.Status)
	return code
}

// Common error constructors

// ErrBadRequest creates a 400 Bad Request error.
func ErrBadRequest(detail string) Error {
	return NewError(400, "bad_request", "Bad Request").Detail(detail).Build()
}

// ErrUnauthorized creates a 401 Unauthorized error.
func ErrUnauthorized(detail string) Error {
	if detail == "" {
		detail = "Authentication required"
	}
	return NewError(401, "unauthorized", "Unauthorized").Detail(detail).Build()
}

// ErrForbidden creates a 403 Forbidden error.
func ErrForbidden(detail string) Error {
	if detail == "" {
		detail = "Access denied"
	}
	return NewError(403, "forbidden", "Forbidden").Detail(detail).Build()
}

// ErrNotFound creates a 404 Not Found error.
func ErrNotFound(resourceType string) Error {
	return NewError(404, "not_found", "Not Found").
		Detailf("The requested %s was not found", resourceType).
		Build()
}

// ErrNotFoundWithID creates a 404 Not Found error with resource ID.
func ErrNotFoundWithID(resourceType, id string) Error {
	return NewError(404, "not_found", "Not Found").
		Detailf("The %s with ID '%s' was not found", resourceType, id).
		Build()
}

// ErrMethodNotAllowed creates a 405 Method Not Allowed error.
func ErrMethodNotAllowed(method string) Error {
	return NewError(405, "method_not_allowed", "Method Not Allowed").
		Detailf("The %s method is not allowed for this resource", method).
		Build()
}

// ErrConflict creates a 409 Conflict error.
func ErrConflict(detail string) Error {
	return NewError(409, "conflict", "Conflict").Detail(detail).Build()
}

// ErrValidation creates a 422 Unprocessable Entity error for validation failures.
func ErrValidation(field, message string) Error {
	return NewError(422, "validation_error", "Validation Failed").
		Detail(message).
		Pointer("/data/attributes/" + field).
		Build()
}

// ErrValidationRequired creates a validation error for a required field.
func ErrValidationRequired(field string) Error {
	return ErrValidation(field, fmt.Sprintf("%s is required", field))
}

// ErrValidationInvalid creates a validation error for an invalid field value.
func ErrValidationInvalid(field, reason string) Error {
	return ErrValidation(field, fmt.Sprintf("%s is invalid: %s", field, reason))
}

// ErrRateLimited creates a 429 Too Many Requests error.
func ErrRateLimited(detail string) Error {
	if detail == "" {
		detail = "Rate limit exceeded"
	}
	return NewError(429, "rate_limit_exceeded", "Too Many Requests").Detail(detail).Build()
}

// ErrInternal creates a 500 Internal Server Error.
func ErrInternal(detail string) Error {
	if detail == "" {
		detail = "An internal error occurred"
	}
	return NewError(500, "internal_error", "Internal Server Error").Detail(detail).Build()
}

// ErrNotImplemented creates a 501 Not Implemented error.
func ErrNotImplemented(feature string) Error {
	return NewError(501, "not_implemented", "Not Implemented").
		Detailf("%s is not implemented", feature).
		Build()
}

// ErrServiceUnavailable creates a 503 Service Unavailable error.
func ErrServiceUnavailable(detail string) Error {
	if detail == "" {
		detail = "Service temporarily unavailable"
	}
	return NewError(503, "service_unavailable", "Service Unavailable").Detail(detail).Build()
}

// ErrFromError creates a JSON:API Error from a standard Go error.
func ErrFromError(err error) Error {
	if err == nil {
		return ErrInternal("")
	}
	return ErrInternal(err.Error())
}

// -----------------------------------------------------------------------------
// Metering API Errors
// -----------------------------------------------------------------------------

// ErrInvalidEventType creates a 422 error for invalid event type.
func ErrInvalidEventType(eventType string) Error {
	return NewError(422, "invalid_event_type", "Invalid Event Type").
		Detailf("Event type '%s' is not recognized", eventType).
		Build()
}

// ErrDuplicateEvent creates a 409 error for duplicate event ID.
func ErrDuplicateEvent(eventID string) Error {
	return NewError(409, "duplicate_event", "Duplicate Event").
		Detailf("Event with ID '%s' has already been processed", eventID).
		Build()
}

// ErrUserNotFoundForMeter creates a 422 error when user_id doesn't exist.
func ErrUserNotFoundForMeter(userID string) Error {
	return NewError(422, "user_not_found", "User Not Found").
		Detailf("User with ID '%s' does not exist", userID).
		Pointer("/data/attributes/user_id").
		Build()
}

// ErrInvalidQuantity creates a 422 error for invalid quantity.
func ErrInvalidQuantity(quantity float64) Error {
	return NewError(422, "invalid_quantity", "Invalid Quantity").
		Detailf("Quantity must be greater than 0, got %f", quantity).
		Pointer("/data/attributes/quantity").
		Build()
}

// ErrInvalidTimestamp creates a 422 error for invalid timestamp.
func ErrInvalidTimestamp(reason string) Error {
	return NewError(422, "invalid_timestamp", "Invalid Timestamp").
		Detail(reason).
		Pointer("/data/attributes/timestamp").
		Build()
}

// ErrInsufficientScope creates a 403 error for missing API key scope.
func ErrInsufficientScope(requiredScope string) Error {
	return NewError(403, "insufficient_scope", "Insufficient Scope").
		Detailf("API key requires '%s' scope to perform this action", requiredScope).
		Build()
}
