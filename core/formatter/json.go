package formatter

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/schema"
)

// JSONFormatter formats output as JSON.
type JSONFormatter struct{}

// NewJSONFormatter creates a new JSON formatter.
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

// Name returns the formatter name.
func (f *JSONFormatter) Name() string {
	return "json"
}

// Description returns the formatter description.
func (f *JSONFormatter) Description() string {
	return "JSON output format"
}

// FormatList formats a list of records as JSON.
func (f *JSONFormatter) FormatList(w io.Writer, mod convention.Derived, records []map[string]any, opts FormatOptions) error {
	// Filter columns if specified
	filtered := f.filterRecords(mod, records, opts.Columns)

	output := map[string]any{
		"module": mod.Source.Name,
		"count":  len(filtered),
		"data":   filtered,
	}

	return f.encode(w, output, opts.Compact)
}

// FormatRecord formats a single record as JSON.
func (f *JSONFormatter) FormatRecord(w io.Writer, mod convention.Derived, record map[string]any, opts FormatOptions) error {
	if record == nil {
		output := map[string]any{
			"module": mod.Source.Name,
			"data":   nil,
		}
		return f.encode(w, output, opts.Compact)
	}

	// Filter columns if specified
	filtered := f.filterRecord(mod, record, opts.Columns)

	output := map[string]any{
		"module": mod.Source.Name,
		"data":   filtered,
	}

	return f.encode(w, output, opts.Compact)
}

// FormatError formats an error as JSON.
func (f *JSONFormatter) FormatError(w io.Writer, err error) error {
	output := map[string]any{
		"error": err.Error(),
	}
	return f.encode(w, output, false)
}

// encode writes JSON to the writer.
func (f *JSONFormatter) encode(w io.Writer, data any, compact bool) error {
	encoder := json.NewEncoder(w)
	if !compact {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(data)
}

// filterRecords filters a list of records to include only specified columns.
func (f *JSONFormatter) filterRecords(mod convention.Derived, records []map[string]any, columns []string) []map[string]any {
	if len(columns) == 0 {
		// Return all non-internal fields
		return f.removeInternal(mod, records)
	}

	result := make([]map[string]any, len(records))
	for i, record := range records {
		result[i] = f.filterRecord(mod, record, columns)
	}
	return result
}

// filterRecord filters a single record.
func (f *JSONFormatter) filterRecord(mod convention.Derived, record map[string]any, columns []string) map[string]any {
	if len(columns) == 0 {
		// Return all non-internal fields
		return f.removeInternalSingle(mod, record)
	}

	result := make(map[string]any)
	for _, col := range columns {
		if val, ok := record[col]; ok {
			result[col] = val
		}
	}
	return result
}

// removeInternal removes internal fields from records.
func (f *JSONFormatter) removeInternal(mod convention.Derived, records []map[string]any) []map[string]any {
	result := make([]map[string]any, len(records))
	for i, record := range records {
		result[i] = f.removeInternalSingle(mod, record)
	}
	return result
}

// removeInternalSingle removes internal fields from a single record.
func (f *JSONFormatter) removeInternalSingle(mod convention.Derived, record map[string]any) map[string]any {
	// Build set of internal fields
	internal := make(map[string]bool)
	for _, field := range mod.Fields {
		if field.Internal || field.Type == schema.FieldTypeSecret {
			internal[field.Name] = true
		}
	}

	result := make(map[string]any)
	for k, v := range record {
		if !internal[k] {
			result[k] = v
		}
	}
	return result
}

func init() {
	if err := Register(NewJSONFormatter()); err != nil {
		fmt.Printf("failed to register json formatter: %v\n", err)
	}
}
