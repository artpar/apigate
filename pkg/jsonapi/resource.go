package jsonapi

// ResourceBuilder provides a fluent API for building Resource objects.
type ResourceBuilder struct {
	resource Resource
}

// NewResource creates a new ResourceBuilder with the given type and ID.
func NewResource(resourceType, id string) *ResourceBuilder {
	return &ResourceBuilder{
		resource: Resource{
			Type:       resourceType,
			ID:         id,
			Attributes: make(map[string]any),
		},
	}
}

// Attr adds an attribute to the resource.
func (b *ResourceBuilder) Attr(key string, value any) *ResourceBuilder {
	if b.resource.Attributes == nil {
		b.resource.Attributes = make(map[string]any)
	}
	b.resource.Attributes[key] = value
	return b
}

// Attrs adds multiple attributes to the resource.
func (b *ResourceBuilder) Attrs(attrs map[string]any) *ResourceBuilder {
	if b.resource.Attributes == nil {
		b.resource.Attributes = make(map[string]any)
	}
	for k, v := range attrs {
		// Skip id and type as they're top-level fields
		if k == "id" || k == "type" {
			continue
		}
		b.resource.Attributes[k] = v
	}
	return b
}

// Relationship adds a relationship to the resource.
func (b *ResourceBuilder) Relationship(name string, rel Relationship) *ResourceBuilder {
	if b.resource.Relationships == nil {
		b.resource.Relationships = make(map[string]Relationship)
	}
	b.resource.Relationships[name] = rel
	return b
}

// BelongsTo adds a to-one relationship.
func (b *ResourceBuilder) BelongsTo(name, relType, relID string) *ResourceBuilder {
	if relID == "" {
		// Don't add empty relationships
		return b
	}
	return b.Relationship(name, Relationship{
		Data: ResourceIdentifier{Type: relType, ID: relID},
	})
}

// HasMany adds a to-many relationship.
func (b *ResourceBuilder) HasMany(name string, identifiers []ResourceIdentifier) *ResourceBuilder {
	return b.Relationship(name, Relationship{
		Data: identifiers,
	})
}

// HasManyIDs is a convenience method for adding a to-many relationship with just IDs.
func (b *ResourceBuilder) HasManyIDs(name, relType string, ids []string) *ResourceBuilder {
	identifiers := make([]ResourceIdentifier, len(ids))
	for i, id := range ids {
		identifiers[i] = ResourceIdentifier{Type: relType, ID: id}
	}
	return b.HasMany(name, identifiers)
}

// Meta adds metadata to the resource.
func (b *ResourceBuilder) Meta(key string, value any) *ResourceBuilder {
	if b.resource.Meta == nil {
		b.resource.Meta = make(Meta)
	}
	b.resource.Meta[key] = value
	return b
}

// Link sets the self link for the resource.
func (b *ResourceBuilder) Link(self string) *ResourceBuilder {
	b.resource.Links = &ResourceLinks{Self: self}
	return b
}

// Build returns the constructed Resource.
func (b *ResourceBuilder) Build() Resource {
	return b.resource
}

// ToIdentifier returns a ResourceIdentifier for this resource.
func (b *ResourceBuilder) ToIdentifier() ResourceIdentifier {
	return ResourceIdentifier{
		Type: b.resource.Type,
		ID:   b.resource.ID,
	}
}

// ResourceFromMap creates a Resource from a map, useful for dynamic data.
// The map should contain an "id" key. The resourceType is provided explicitly.
func ResourceFromMap(resourceType string, data map[string]any) Resource {
	id := ""
	if v, ok := data["id"].(string); ok {
		id = v
	}

	rb := NewResource(resourceType, id)

	for k, v := range data {
		if k == "id" || k == "type" {
			continue
		}
		rb.Attr(k, v)
	}

	return rb.Build()
}

// ResourcesFromMaps creates a slice of Resources from a slice of maps.
func ResourcesFromMaps(resourceType string, data []map[string]any) []Resource {
	resources := make([]Resource, len(data))
	for i, item := range data {
		resources[i] = ResourceFromMap(resourceType, item)
	}
	return resources
}
