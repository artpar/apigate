package jsonapi

// DocumentBuilder provides a fluent API for building Document objects.
type DocumentBuilder struct {
	doc Document
}

// NewDocument creates a new DocumentBuilder.
func NewDocument() *DocumentBuilder {
	return &DocumentBuilder{
		doc: Document{},
	}
}

// Data sets the primary data of the document.
// Can be a Resource, []Resource, ResourceIdentifier, []ResourceIdentifier, or nil.
func (b *DocumentBuilder) Data(data any) *DocumentBuilder {
	b.doc.Data = data
	return b
}

// DataResource sets a single resource as the primary data.
func (b *DocumentBuilder) DataResource(r Resource) *DocumentBuilder {
	b.doc.Data = r
	return b
}

// DataCollection sets a collection of resources as the primary data.
func (b *DocumentBuilder) DataCollection(resources []Resource) *DocumentBuilder {
	b.doc.Data = resources
	return b
}

// DataNull sets the primary data to null (for empty to-one relationships).
func (b *DocumentBuilder) DataNull() *DocumentBuilder {
	b.doc.Data = nil
	return b
}

// DataEmpty sets the primary data to an empty array (for empty collections).
func (b *DocumentBuilder) DataEmpty() *DocumentBuilder {
	b.doc.Data = []Resource{}
	return b
}

// Errors sets the errors array. This is mutually exclusive with Data.
func (b *DocumentBuilder) Errors(errors ...Error) *DocumentBuilder {
	b.doc.Errors = errors
	b.doc.Data = nil // Errors and Data are mutually exclusive
	return b
}

// Meta adds a metadata entry to the document.
func (b *DocumentBuilder) Meta(key string, value any) *DocumentBuilder {
	if b.doc.Meta == nil {
		b.doc.Meta = make(Meta)
	}
	b.doc.Meta[key] = value
	return b
}

// MetaAll sets all metadata at once.
func (b *DocumentBuilder) MetaAll(meta Meta) *DocumentBuilder {
	b.doc.Meta = meta
	return b
}

// Links sets the top-level links.
func (b *DocumentBuilder) Links(links *Links) *DocumentBuilder {
	b.doc.Links = links
	return b
}

// Pagination adds pagination metadata and links.
func (b *DocumentBuilder) Pagination(p *Pagination) *DocumentBuilder {
	if p == nil {
		return b
	}

	// Add pagination metadata
	if b.doc.Meta == nil {
		b.doc.Meta = make(Meta)
	}
	b.doc.Meta["total"] = p.Total
	b.doc.Meta["page"] = p.Page
	b.doc.Meta["per_page"] = p.PerPage

	// Add pagination links
	b.doc.Links = p.Links()

	return b
}

// Include adds resources to the included section for compound documents.
func (b *DocumentBuilder) Include(resources ...Resource) *DocumentBuilder {
	b.doc.Included = append(b.doc.Included, resources...)
	return b
}

// IncludeAll adds all resources from a slice to the included section.
func (b *DocumentBuilder) IncludeAll(resources []Resource) *DocumentBuilder {
	b.doc.Included = append(b.doc.Included, resources...)
	return b
}

// JSONAPI sets the JSON:API version object.
func (b *DocumentBuilder) JSONAPI() *DocumentBuilder {
	b.doc.JSONAPI = &JSONAPI{Version: Version}
	return b
}

// Build returns the constructed Document.
func (b *DocumentBuilder) Build() Document {
	return b.doc
}

// NewSingleResourceDocument is a convenience function for creating a document with a single resource.
func NewSingleResourceDocument(r Resource) Document {
	return NewDocument().DataResource(r).Build()
}

// NewCollectionDocument is a convenience function for creating a document with a collection.
func NewCollectionDocument(resources []Resource, pagination *Pagination) Document {
	builder := NewDocument().DataCollection(resources)
	if pagination != nil {
		builder.Pagination(pagination)
	}
	return builder.Build()
}

// NewErrorDocument is a convenience function for creating an error document.
func NewErrorDocument(errors ...Error) Document {
	return NewDocument().Errors(errors...).Build()
}
