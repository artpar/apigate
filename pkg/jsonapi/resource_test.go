package jsonapi

import (
	"encoding/json"
	"testing"
)

func TestResourceBuilder(t *testing.T) {
	t.Run("NewResource creates builder with type and ID", func(t *testing.T) {
		builder := NewResource("users", "123")
		r := builder.Build()

		if r.Type != "users" {
			t.Errorf("Type = %v, want users", r.Type)
		}
		if r.ID != "123" {
			t.Errorf("ID = %v, want 123", r.ID)
		}
		if r.Attributes == nil {
			t.Error("Attributes should be initialized")
		}
	})

	t.Run("Attr adds attribute", func(t *testing.T) {
		r := NewResource("users", "1").
			Attr("name", "John").
			Attr("email", "john@example.com").
			Build()

		if r.Attributes["name"] != "John" {
			t.Errorf("name = %v, want John", r.Attributes["name"])
		}
		if r.Attributes["email"] != "john@example.com" {
			t.Errorf("email = %v, want john@example.com", r.Attributes["email"])
		}
	})

	t.Run("Attr initializes nil attributes", func(t *testing.T) {
		builder := &ResourceBuilder{resource: Resource{Type: "test", ID: "1"}}
		r := builder.Attr("key", "value").Build()

		if r.Attributes == nil {
			t.Error("Attr should initialize Attributes")
		}
		if r.Attributes["key"] != "value" {
			t.Error("Attr should set value")
		}
	})

	t.Run("Attrs adds multiple attributes", func(t *testing.T) {
		attrs := map[string]any{
			"name":  "John",
			"email": "john@example.com",
			"age":   30,
		}
		r := NewResource("users", "1").Attrs(attrs).Build()

		if len(r.Attributes) != 3 {
			t.Errorf("len(Attributes) = %d, want 3", len(r.Attributes))
		}
	})

	t.Run("Attrs skips id and type", func(t *testing.T) {
		attrs := map[string]any{
			"id":   "should-be-skipped",
			"type": "should-be-skipped",
			"name": "John",
		}
		r := NewResource("users", "1").Attrs(attrs).Build()

		if _, ok := r.Attributes["id"]; ok {
			t.Error("id should be skipped")
		}
		if _, ok := r.Attributes["type"]; ok {
			t.Error("type should be skipped")
		}
		if r.Attributes["name"] != "John" {
			t.Error("name should be set")
		}
	})

	t.Run("Attrs initializes nil attributes", func(t *testing.T) {
		builder := &ResourceBuilder{resource: Resource{Type: "test", ID: "1"}}
		r := builder.Attrs(map[string]any{"key": "value"}).Build()

		if r.Attributes == nil {
			t.Error("Attrs should initialize Attributes")
		}
	})

	t.Run("Relationship adds relationship", func(t *testing.T) {
		rel := Relationship{
			Data: ResourceIdentifier{Type: "posts", ID: "1"},
		}
		r := NewResource("users", "1").Relationship("author", rel).Build()

		if r.Relationships == nil {
			t.Fatal("Relationships should be initialized")
		}
		if _, ok := r.Relationships["author"]; !ok {
			t.Error("author relationship should exist")
		}
	})

	t.Run("Relationship initializes nil relationships", func(t *testing.T) {
		builder := &ResourceBuilder{resource: Resource{Type: "test", ID: "1"}}
		r := builder.Relationship("rel", Relationship{}).Build()

		if r.Relationships == nil {
			t.Error("Relationship should initialize Relationships")
		}
	})

	t.Run("BelongsTo adds to-one relationship", func(t *testing.T) {
		r := NewResource("posts", "1").BelongsTo("author", "users", "42").Build()

		if r.Relationships == nil {
			t.Fatal("Relationships should be set")
		}
		rel := r.Relationships["author"]
		id, ok := rel.Data.(ResourceIdentifier)
		if !ok {
			t.Fatal("Data should be ResourceIdentifier")
		}
		if id.Type != "users" || id.ID != "42" {
			t.Errorf("Relationship = %+v, want users/42", id)
		}
	})

	t.Run("BelongsTo skips empty ID", func(t *testing.T) {
		r := NewResource("posts", "1").BelongsTo("author", "users", "").Build()

		if r.Relationships != nil && len(r.Relationships) > 0 {
			t.Error("Empty ID should not create relationship")
		}
	})

	t.Run("HasMany adds to-many relationship", func(t *testing.T) {
		identifiers := []ResourceIdentifier{
			{Type: "tags", ID: "1"},
			{Type: "tags", ID: "2"},
		}
		r := NewResource("posts", "1").HasMany("tags", identifiers).Build()

		rel := r.Relationships["tags"]
		ids, ok := rel.Data.([]ResourceIdentifier)
		if !ok {
			t.Fatal("Data should be []ResourceIdentifier")
		}
		if len(ids) != 2 {
			t.Errorf("len(Data) = %d, want 2", len(ids))
		}
	})

	t.Run("HasManyIDs creates identifiers from IDs", func(t *testing.T) {
		r := NewResource("posts", "1").HasManyIDs("tags", "tags", []string{"1", "2", "3"}).Build()

		rel := r.Relationships["tags"]
		ids, ok := rel.Data.([]ResourceIdentifier)
		if !ok {
			t.Fatal("Data should be []ResourceIdentifier")
		}
		if len(ids) != 3 {
			t.Errorf("len(Data) = %d, want 3", len(ids))
		}
		if ids[0].Type != "tags" || ids[0].ID != "1" {
			t.Errorf("First identifier = %+v, want tags/1", ids[0])
		}
	})

	t.Run("Meta adds metadata", func(t *testing.T) {
		r := NewResource("users", "1").
			Meta("created_at", "2024-01-01").
			Meta("updated_at", "2024-01-02").
			Build()

		if r.Meta == nil {
			t.Fatal("Meta should be set")
		}
		if r.Meta["created_at"] != "2024-01-01" {
			t.Errorf("created_at = %v, want 2024-01-01", r.Meta["created_at"])
		}
	})

	t.Run("Meta initializes nil meta", func(t *testing.T) {
		builder := &ResourceBuilder{resource: Resource{Type: "test", ID: "1"}}
		r := builder.Meta("key", "value").Build()

		if r.Meta == nil {
			t.Error("Meta should initialize Meta map")
		}
	})

	t.Run("Link sets self link", func(t *testing.T) {
		r := NewResource("users", "1").Link("/api/users/1").Build()

		if r.Links == nil {
			t.Fatal("Links should be set")
		}
		if r.Links.Self != "/api/users/1" {
			t.Errorf("Self = %v, want /api/users/1", r.Links.Self)
		}
	})

	t.Run("ToIdentifier returns resource identifier", func(t *testing.T) {
		builder := NewResource("users", "123")
		id := builder.ToIdentifier()

		if id.Type != "users" {
			t.Errorf("Type = %v, want users", id.Type)
		}
		if id.ID != "123" {
			t.Errorf("ID = %v, want 123", id.ID)
		}
	})
}

func TestResourceFromMap(t *testing.T) {
	t.Run("creates resource from map", func(t *testing.T) {
		data := map[string]any{
			"id":    "1",
			"name":  "John",
			"email": "john@example.com",
		}
		r := ResourceFromMap("users", data)

		if r.Type != "users" {
			t.Errorf("Type = %v, want users", r.Type)
		}
		if r.ID != "1" {
			t.Errorf("ID = %v, want 1", r.ID)
		}
		if r.Attributes["name"] != "John" {
			t.Errorf("name = %v, want John", r.Attributes["name"])
		}
	})

	t.Run("skips id and type in attributes", func(t *testing.T) {
		data := map[string]any{
			"id":   "1",
			"type": "should-be-skipped",
			"name": "John",
		}
		r := ResourceFromMap("users", data)

		if _, ok := r.Attributes["id"]; ok {
			t.Error("id should not be in attributes")
		}
		if _, ok := r.Attributes["type"]; ok {
			t.Error("type should not be in attributes")
		}
	})

	t.Run("handles missing id", func(t *testing.T) {
		data := map[string]any{
			"name": "John",
		}
		r := ResourceFromMap("users", data)

		if r.ID != "" {
			t.Errorf("ID = %v, want empty string", r.ID)
		}
	})

	t.Run("handles non-string id", func(t *testing.T) {
		data := map[string]any{
			"id":   123, // int instead of string
			"name": "John",
		}
		r := ResourceFromMap("users", data)

		if r.ID != "" {
			t.Errorf("ID = %v, want empty string for non-string id", r.ID)
		}
	})
}

func TestResourcesFromMaps(t *testing.T) {
	t.Run("creates resources from maps", func(t *testing.T) {
		data := []map[string]any{
			{"id": "1", "name": "John"},
			{"id": "2", "name": "Jane"},
			{"id": "3", "name": "Bob"},
		}
		resources := ResourcesFromMaps("users", data)

		if len(resources) != 3 {
			t.Fatalf("len(resources) = %d, want 3", len(resources))
		}

		if resources[0].ID != "1" {
			t.Errorf("First ID = %v, want 1", resources[0].ID)
		}
		if resources[1].Attributes["name"] != "Jane" {
			t.Errorf("Second name = %v, want Jane", resources[1].Attributes["name"])
		}
	})

	t.Run("handles empty slice", func(t *testing.T) {
		resources := ResourcesFromMaps("users", []map[string]any{})

		if len(resources) != 0 {
			t.Errorf("len(resources) = %d, want 0", len(resources))
		}
	})
}

func TestResourceJSON(t *testing.T) {
	t.Run("resource serializes correctly", func(t *testing.T) {
		r := NewResource("users", "1").
			Attr("name", "John").
			Attr("email", "john@example.com").
			BelongsTo("team", "teams", "42").
			Meta("version", 1).
			Link("/api/users/1").
			Build()

		data, err := json.Marshal(r)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var result Resource
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if result.Type != "users" {
			t.Errorf("Type = %v, want users", result.Type)
		}
		if result.ID != "1" {
			t.Errorf("ID = %v, want 1", result.ID)
		}
	})
}

func TestResourceIdentifier(t *testing.T) {
	t.Run("serializes correctly", func(t *testing.T) {
		id := ResourceIdentifier{
			Type: "users",
			ID:   "123",
			Meta: Meta{"source": "api"},
		}

		data, err := json.Marshal(id)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var result ResourceIdentifier
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if result.Type != "users" || result.ID != "123" {
			t.Errorf("Identifier = %+v, want users/123", result)
		}
	})
}
