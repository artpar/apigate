package formatter

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/schema"
)

// TableFormatter formats output as aligned text tables.
type TableFormatter struct{}

// NewTableFormatter creates a new table formatter.
func NewTableFormatter() *TableFormatter {
	return &TableFormatter{}
}

// Name returns the formatter name.
func (f *TableFormatter) Name() string {
	return "table"
}

// Description returns the formatter description.
func (f *TableFormatter) Description() string {
	return "Aligned text table output"
}

// FormatList formats a list of records as a table.
func (f *TableFormatter) FormatList(w io.Writer, mod convention.Derived, records []map[string]any, opts FormatOptions) error {
	if len(records) == 0 {
		fmt.Fprintln(w, "No records found.")
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	// Determine columns
	columns := f.resolveColumns(mod, opts.Columns)

	// Print header
	if !opts.NoHeader {
		var headers []string
		for _, col := range columns {
			headers = append(headers, strings.ToUpper(col))
		}
		fmt.Fprintln(tw, strings.Join(headers, "\t"))
	}

	// Print rows
	for _, record := range records {
		var values []string
		for _, col := range columns {
			val := f.formatValue(record[col], opts.MaxWidth)
			values = append(values, val)
		}
		fmt.Fprintln(tw, strings.Join(values, "\t"))
	}

	return tw.Flush()
}

// FormatRecord formats a single record as key-value pairs.
func (f *TableFormatter) FormatRecord(w io.Writer, mod convention.Derived, record map[string]any, opts FormatOptions) error {
	if record == nil {
		fmt.Fprintln(w, "Record not found.")
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	// Determine columns
	columns := f.resolveColumns(mod, opts.Columns)

	for _, col := range columns {
		label := f.formatLabel(col)
		val := f.formatValue(record[col], 0) // No truncation for detail view
		fmt.Fprintf(tw, "%s:\t%s\n", label, val)
	}

	return tw.Flush()
}

// FormatError formats an error message.
func (f *TableFormatter) FormatError(w io.Writer, err error) error {
	fmt.Fprintf(w, "Error: %s\n", err.Error())
	return nil
}

// resolveColumns determines which columns to display.
func (f *TableFormatter) resolveColumns(mod convention.Derived, requested []string) []string {
	if len(requested) > 0 {
		return requested
	}

	// Default: all non-internal, non-secret fields
	var columns []string
	for _, field := range mod.Fields {
		if !field.Internal && field.Type != schema.FieldTypeSecret {
			columns = append(columns, field.Name)
		}
	}
	return columns
}

// formatLabel formats a field name as a label.
func (f *TableFormatter) formatLabel(name string) string {
	// Convert snake_case to Title Case
	words := strings.Split(name, "_")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

// formatValue formats a value for display.
func (f *TableFormatter) formatValue(val any, maxWidth int) string {
	if val == nil {
		return "-"
	}

	var str string
	switch v := val.(type) {
	case string:
		str = v
	case bool:
		if v {
			str = "yes"
		} else {
			str = "no"
		}
	case []byte:
		str = "[binary]"
	case float64:
		// Check if it's a whole number
		if v == float64(int64(v)) {
			str = fmt.Sprintf("%d", int64(v))
		} else {
			str = fmt.Sprintf("%.2f", v)
		}
	default:
		b, _ := json.Marshal(v)
		str = string(b)
	}

	// Truncate if needed
	if maxWidth > 0 && len(str) > maxWidth {
		str = str[:maxWidth-3] + "..."
	}

	return str
}

func init() {
	Register(NewTableFormatter())
}
