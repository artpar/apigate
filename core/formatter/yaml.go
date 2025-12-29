package formatter

import (
	"fmt"
	"io"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/schema"
	"gopkg.in/yaml.v3"
)

// YAMLFormatter formats output as YAML.
type YAMLFormatter struct{}

// NewYAMLFormatter creates a new YAML formatter.
func NewYAMLFormatter() *YAMLFormatter {
	return &YAMLFormatter{}
}

// Name returns the formatter name.
func (f *YAMLFormatter) Name() string {
	return "yaml"
}

// Description returns the formatter description.
func (f *YAMLFormatter) Description() string {
	return "YAML output format"
}

// FormatList formats a list of records as YAML.
func (f *YAMLFormatter) FormatList(w io.Writer, mod convention.Derived, records []map[string]any, opts FormatOptions) error {
	// Filter columns if specified
	filtered := f.filterRecords(mod, records, opts.Columns)

	output := map[string]any{
		"module": mod.Source.Name,
		"count":  len(filtered),
		"data":   filtered,
	}

	return f.encode(w, output)
}

// FormatRecord formats a single record as YAML.
func (f *YAMLFormatter) FormatRecord(w io.Writer, mod convention.Derived, record map[string]any, opts FormatOptions) error {
	if record == nil {
		output := map[string]any{
			"module": mod.Source.Name,
			"data":   nil,
		}
		return f.encode(w, output)
	}

	// Filter columns if specified
	filtered := f.filterRecord(mod, record, opts.Columns)

	output := map[string]any{
		"module": mod.Source.Name,
		"data":   filtered,
	}

	return f.encode(w, output)
}

// FormatError formats an error as YAML.
func (f *YAMLFormatter) FormatError(w io.Writer, err error) error {
	output := map[string]any{
		"error": err.Error(),
	}
	return f.encode(w, output)
}

// encode writes YAML to the writer.
func (f *YAMLFormatter) encode(w io.Writer, data any) error {
	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)
	defer encoder.Close()
	return encoder.Encode(data)
}

// filterRecords filters a list of records to include only specified columns.
func (f *YAMLFormatter) filterRecords(mod convention.Derived, records []map[string]any, columns []string) []map[string]any {
	if len(columns) == 0 {
		return f.removeInternal(mod, records)
	}

	result := make([]map[string]any, len(records))
	for i, record := range records {
		result[i] = f.filterRecord(mod, record, columns)
	}
	return result
}

// filterRecord filters a single record.
func (f *YAMLFormatter) filterRecord(mod convention.Derived, record map[string]any, columns []string) map[string]any {
	if len(columns) == 0 {
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
func (f *YAMLFormatter) removeInternal(mod convention.Derived, records []map[string]any) []map[string]any {
	result := make([]map[string]any, len(records))
	for i, record := range records {
		result[i] = f.removeInternalSingle(mod, record)
	}
	return result
}

// removeInternalSingle removes internal fields from a single record.
func (f *YAMLFormatter) removeInternalSingle(mod convention.Derived, record map[string]any) map[string]any {
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
	if err := Register(NewYAMLFormatter()); err != nil {
		fmt.Printf("failed to register yaml formatter: %v\n", err)
	}
}
