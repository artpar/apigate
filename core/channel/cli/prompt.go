// Package cli provides interactive prompting for CLI input.
package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/artpar/apigate/core/convention"
	"github.com/artpar/apigate/core/schema"
	"golang.org/x/term"
)

// Prompter handles interactive CLI input.
type Prompter struct {
	reader *bufio.Reader
}

// NewPrompter creates a new prompter.
func NewPrompter() *Prompter {
	return &Prompter{
		reader: bufio.NewReader(os.Stdin),
	}
}

// PromptForFields prompts for missing required fields.
// Returns a map of field name to value.
func (p *Prompter) PromptForFields(mod convention.Derived, inputs []convention.ActionInput, existing map[string]any) (map[string]any, error) {
	result := make(map[string]any)

	// Copy existing values
	for k, v := range existing {
		result[k] = v
	}

	// Find field schema by name
	fieldMap := make(map[string]convention.DerivedField)
	for _, f := range mod.Fields {
		fieldMap[f.Name] = f
	}

	// Prompt for each missing required input
	for _, input := range inputs {
		// Skip if already provided
		if _, ok := result[input.Name]; ok {
			continue
		}

		// Skip if not required and not marked for prompting
		if !input.Required && !input.Prompt {
			continue
		}

		field, hasField := fieldMap[input.Name]

		// Determine prompt text
		promptText := input.PromptText
		if promptText == "" {
			promptText = formatPromptLabel(input.Name)
		}

		// Add type hint if enum
		if hasField && field.Type == schema.FieldTypeEnum && len(field.Values) > 0 {
			promptText += fmt.Sprintf(" [%s]", strings.Join(field.Values, "/"))
		}

		// Add required indicator
		if input.Required {
			promptText += " (required)"
		}

		promptText += ": "

		// Prompt based on field type
		var value string
		var err error

		if hasField && field.Type == schema.FieldTypeSecret {
			value, err = p.PromptSecret(promptText)
		} else {
			value, err = p.Prompt(promptText)
		}

		if err != nil {
			return nil, err
		}

		// Use default if empty and available
		if value == "" && input.Default != "" {
			value = input.Default
		}

		// Validate required
		if input.Required && value == "" {
			return nil, fmt.Errorf("field %q is required", input.Name)
		}

		if value != "" {
			result[input.Name] = convertInput(value, input.Type)
		}
	}

	return result, nil
}

// Prompt displays a prompt and reads a line of input.
func (p *Prompter) Prompt(prompt string) (string, error) {
	fmt.Print(prompt)
	line, err := p.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// PromptSecret displays a prompt and reads secret input (no echo).
func (p *Prompter) PromptSecret(prompt string) (string, error) {
	fmt.Print(prompt)

	// Check if we're in a terminal
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		password, err := term.ReadPassword(fd)
		fmt.Println() // New line after password
		if err != nil {
			return "", err
		}
		return string(password), nil
	}

	// Fallback for non-terminal (e.g., piped input)
	return p.Prompt("")
}

// Confirm prompts for yes/no confirmation.
func (p *Prompter) Confirm(prompt string) (bool, error) {
	response, err := p.Prompt(prompt + " [y/N]: ")
	if err != nil {
		return false, err
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes", nil
}

// PromptSelect prompts user to select from a list of options.
func (p *Prompter) PromptSelect(prompt string, options []string) (string, error) {
	fmt.Println(prompt)
	for i, opt := range options {
		fmt.Printf("  %d. %s\n", i+1, opt)
	}

	for {
		response, err := p.Prompt("Enter number or value: ")
		if err != nil {
			return "", err
		}

		// Check if it's a number
		var idx int
		if _, err := fmt.Sscanf(response, "%d", &idx); err == nil {
			if idx >= 1 && idx <= len(options) {
				return options[idx-1], nil
			}
			fmt.Println("Invalid selection. Try again.")
			continue
		}

		// Check if it matches an option
		for _, opt := range options {
			if strings.EqualFold(response, opt) {
				return opt, nil
			}
		}

		fmt.Println("Invalid selection. Try again.")
	}
}

// formatPromptLabel formats a field name as a prompt label.
func formatPromptLabel(name string) string {
	// Convert snake_case to Title Case
	words := strings.Split(name, "_")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

// DefaultPrompter is the global prompter instance.
var DefaultPrompter = NewPrompter()
