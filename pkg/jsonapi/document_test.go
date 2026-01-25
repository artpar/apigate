package jsonapi

import (
	"encoding/json"
	"testing"
)

func TestDocumentBuilder(t *testing.T) {
	t.Run("NewDocument creates empty builder", func(t *testing.T) {
		builder := NewDocument()
		if builder == nil {
			t.Fatal("NewDocument returned nil")
		}
		doc := builder.Build()
		if doc.Data != nil {
			t.Error("Expected nil data in empty document")
		}
	})

	t.Run("Data sets primary data", func(t *testing.T) {
		resource := Resource{Type: "users", ID: "1"}
		doc := NewDocument().Data(resource).Build()
		if doc.Data == nil {
			t.Error("Data should not be nil")
		}
	})

	t.Run("DataResource sets single resource", func(t *testing.T) {
		resource := Resource{Type: "users", ID: "1"}
		doc := NewDocument().DataResource(resource).Build()
		r, ok := doc.Data.(Resource)
		if !ok {
			t.Error("Data should be Resource type")
		}
		if r.ID != "1" {
			t.Errorf("Resource ID = %v, want 1", r.ID)
		}
	})

	t.Run("DataCollection sets resource array", func(t *testing.T) {
		resources := []Resource{
			{Type: "users", ID: "1"},
			{Type: "users", ID: "2"},
		}
		doc := NewDocument().DataCollection(resources).Build()
		r, ok := doc.Data.([]Resource)
		if !ok {
			t.Error("Data should be []Resource type")
		}
		if len(r) != 2 {
			t.Errorf("len(Data) = %d, want 2", len(r))
		}
	})

	t.Run("DataNull sets nil data", func(t *testing.T) {
		doc := NewDocument().DataNull().Build()
		if doc.Data != nil {
			t.Error("DataNull should set nil")
		}
	})

	t.Run("DataEmpty sets empty array", func(t *testing.T) {
		doc := NewDocument().DataEmpty().Build()
		r, ok := doc.Data.([]Resource)
		if !ok {
			t.Error("DataEmpty should set []Resource")
		}
		if len(r) != 0 {
			t.Error("DataEmpty should set empty array")
		}
	})

	t.Run("Errors sets errors and clears data", func(t *testing.T) {
		doc := NewDocument().
			Data("should be cleared").
			Errors(ErrNotFound("users")).
			Build()

		if doc.Data != nil {
			t.Error("Errors should clear Data")
		}
		if len(doc.Errors) != 1 {
			t.Errorf("len(Errors) = %d, want 1", len(doc.Errors))
		}
	})

	t.Run("Meta adds metadata", func(t *testing.T) {
		doc := NewDocument().
			Meta("count", 10).
			Meta("version", "1.0").
			Build()

		if doc.Meta == nil {
			t.Fatal("Meta should not be nil")
		}
		if doc.Meta["count"] != 10 {
			t.Errorf("Meta[count] = %v, want 10", doc.Meta["count"])
		}
		if doc.Meta["version"] != "1.0" {
			t.Errorf("Meta[version] = %v, want 1.0", doc.Meta["version"])
		}
	})

	t.Run("MetaAll sets all metadata", func(t *testing.T) {
		meta := Meta{"key1": "value1", "key2": "value2"}
		doc := NewDocument().MetaAll(meta).Build()

		if len(doc.Meta) != 2 {
			t.Errorf("len(Meta) = %d, want 2", len(doc.Meta))
		}
	})

	t.Run("Links sets top-level links", func(t *testing.T) {
		links := &Links{Self: "/users", Next: "/users?page=2"}
		doc := NewDocument().Links(links).Build()

		if doc.Links == nil {
			t.Fatal("Links should not be nil")
		}
		if doc.Links.Self != "/users" {
			t.Errorf("Links.Self = %v, want /users", doc.Links.Self)
		}
	})

	t.Run("Pagination adds meta and links", func(t *testing.T) {
		p := &Pagination{
			Total:   100,
			Page:    2,
			PerPage: 10,
			BaseURL: "/users",
		}
		doc := NewDocument().Pagination(p).Build()

		if doc.Meta == nil {
			t.Fatal("Pagination should set Meta")
		}
		if doc.Meta["total"] != int64(100) {
			t.Errorf("Meta[total] = %v, want 100", doc.Meta["total"])
		}
		if doc.Links == nil {
			t.Fatal("Pagination should set Links")
		}
	})

	t.Run("Pagination nil is safe", func(t *testing.T) {
		doc := NewDocument().Pagination(nil).Build()
		if doc.Meta != nil {
			t.Error("nil pagination should not set meta")
		}
	})

	t.Run("Include adds resources", func(t *testing.T) {
		r1 := Resource{Type: "posts", ID: "1"}
		r2 := Resource{Type: "posts", ID: "2"}
		doc := NewDocument().Include(r1, r2).Build()

		if len(doc.Included) != 2 {
			t.Errorf("len(Included) = %d, want 2", len(doc.Included))
		}
	})

	t.Run("IncludeAll adds resource slice", func(t *testing.T) {
		resources := []Resource{
			{Type: "posts", ID: "1"},
			{Type: "posts", ID: "2"},
		}
		doc := NewDocument().IncludeAll(resources).Build()

		if len(doc.Included) != 2 {
			t.Errorf("len(Included) = %d, want 2", len(doc.Included))
		}
	})

	t.Run("JSONAPI sets version", func(t *testing.T) {
		doc := NewDocument().JSONAPI().Build()

		if doc.JSONAPI == nil {
			t.Fatal("JSONAPI should not be nil")
		}
		if doc.JSONAPI.Version != Version {
			t.Errorf("JSONAPI.Version = %v, want %v", doc.JSONAPI.Version, Version)
		}
	})
}

func TestConvenienceDocumentFunctions(t *testing.T) {
	t.Run("NewSingleResourceDocument", func(t *testing.T) {
		r := Resource{Type: "users", ID: "1"}
		doc := NewSingleResourceDocument(r)

		res, ok := doc.Data.(Resource)
		if !ok {
			t.Fatal("Data should be Resource")
		}
		if res.ID != "1" {
			t.Errorf("ID = %v, want 1", res.ID)
		}
	})

	t.Run("NewCollectionDocument without pagination", func(t *testing.T) {
		resources := []Resource{
			{Type: "users", ID: "1"},
			{Type: "users", ID: "2"},
		}
		doc := NewCollectionDocument(resources, nil)

		res, ok := doc.Data.([]Resource)
		if !ok {
			t.Fatal("Data should be []Resource")
		}
		if len(res) != 2 {
			t.Errorf("len(Data) = %d, want 2", len(res))
		}
	})

	t.Run("NewCollectionDocument with pagination", func(t *testing.T) {
		resources := []Resource{
			{Type: "users", ID: "1"},
		}
		p := &Pagination{Total: 100, Page: 1, PerPage: 10, BaseURL: "/users"}
		doc := NewCollectionDocument(resources, p)

		if doc.Meta == nil {
			t.Error("Pagination should add meta")
		}
		if doc.Links == nil {
			t.Error("Pagination should add links")
		}
	})

	t.Run("NewErrorDocument", func(t *testing.T) {
		err1 := ErrNotFoundWithID("users", "1")
		err2 := ErrBadRequest("Name is required")
		doc := NewErrorDocument(err1, err2)

		if len(doc.Errors) != 2 {
			t.Errorf("len(Errors) = %d, want 2", len(doc.Errors))
		}
		if doc.Data != nil {
			t.Error("Error document should have nil data")
		}
	})
}

func TestDocumentJSON(t *testing.T) {
	t.Run("document serializes correctly", func(t *testing.T) {
		doc := NewDocument().
			DataResource(Resource{Type: "users", ID: "1", Attributes: map[string]any{"name": "John"}}).
			Meta("count", 1).
			JSONAPI().
			Build()

		data, err := json.Marshal(doc)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var result Document
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if result.JSONAPI == nil {
			t.Error("JSONAPI should not be nil after unmarshal")
		}
	})

	t.Run("empty document serializes", func(t *testing.T) {
		doc := NewDocument().Build()
		data, err := json.Marshal(doc)
		if err != nil {
			t.Fatalf("Failed to marshal empty document: %v", err)
		}
		if len(data) == 0 {
			t.Error("Empty document should produce some JSON")
		}
	})
}
