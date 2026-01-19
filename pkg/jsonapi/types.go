// Package jsonapi provides JSON:API specification compliant response types and builders.
// See https://jsonapi.org for the full specification.
package jsonapi

// Document represents a JSON:API top-level document.
// A document MUST contain at least one of: data, errors, or meta.
type Document struct {
	Data     any        `json:"data,omitempty"`
	Errors   []Error    `json:"errors,omitempty"`
	Meta     Meta       `json:"meta,omitempty"`
	Links    *Links     `json:"links,omitempty"`
	Included []Resource `json:"included,omitempty"`
	JSONAPI  *JSONAPI   `json:"jsonapi,omitempty"`
}

// Resource represents a JSON:API resource object.
type Resource struct {
	Type          string                  `json:"type"`
	ID            string                  `json:"id"`
	Attributes    map[string]any          `json:"attributes,omitempty"`
	Relationships map[string]Relationship `json:"relationships,omitempty"`
	Links         *ResourceLinks          `json:"links,omitempty"`
	Meta          Meta                    `json:"meta,omitempty"`
}

// ResourceIdentifier represents a resource linkage (type + id only).
type ResourceIdentifier struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Meta Meta   `json:"meta,omitempty"`
}

// Relationship represents a relationship to one or more resources.
type Relationship struct {
	Data  any    `json:"data"`            // ResourceIdentifier, []ResourceIdentifier, or nil
	Links *Links `json:"links,omitempty"` // related, self links
	Meta  Meta   `json:"meta,omitempty"`
}

// Links represents pagination and navigation links.
type Links struct {
	Self    string `json:"self,omitempty"`
	Related string `json:"related,omitempty"`
	First   string `json:"first,omitempty"`
	Last    string `json:"last,omitempty"`
	Prev    string `json:"prev,omitempty"`
	Next    string `json:"next,omitempty"`
}

// ResourceLinks represents links within a resource object.
type ResourceLinks struct {
	Self string `json:"self,omitempty"`
}

// Error represents a JSON:API error object.
type Error struct {
	ID     string       `json:"id,omitempty"`
	Links  *ErrorLinks  `json:"links,omitempty"`
	Status string       `json:"status"`
	Code   string       `json:"code"`
	Title  string       `json:"title"`
	Detail string       `json:"detail,omitempty"`
	Source *ErrorSource `json:"source,omitempty"`
	Meta   Meta         `json:"meta,omitempty"`
}

// ErrorLinks represents links within an error object.
type ErrorLinks struct {
	About string `json:"about,omitempty"`
	Type  string `json:"type,omitempty"`
}

// ErrorSource indicates the source of an error.
type ErrorSource struct {
	Pointer   string `json:"pointer,omitempty"`   // JSON pointer to offending field
	Parameter string `json:"parameter,omitempty"` // Query parameter that caused error
	Header    string `json:"header,omitempty"`    // Header that caused error
}

// Meta represents arbitrary metadata.
type Meta map[string]any

// JSONAPI represents the JSON:API version object.
type JSONAPI struct {
	Version string `json:"version"`
	Meta    Meta   `json:"meta,omitempty"`
}

// ContentType is the JSON:API media type.
const ContentType = "application/vnd.api+json"

// Version is the JSON:API specification version.
const Version = "1.1"
