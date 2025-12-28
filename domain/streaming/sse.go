// Package streaming provides utilities for handling streaming protocols.
package streaming

import (
	"bufio"
	"bytes"
	"strings"
)

// SSEEvent represents a single Server-Sent Event.
// This is a pure data structure - no behavior, just values.
type SSEEvent struct {
	Event string `json:"event,omitempty"` // Event type (from "event:" field)
	Data  string `json:"data"`            // Event data (from "data:" field(s))
	ID    string `json:"id,omitempty"`    // Event ID (from "id:" field)
	Retry int    `json:"retry,omitempty"` // Retry timeout in ms (from "retry:" field)
}

// ParseSSEEvents parses raw SSE data into a slice of events.
// This is a pure function - takes bytes, returns structured data.
// Follows the SSE specification (https://html.spec.whatwg.org/multipage/server-sent-events.html).
func ParseSSEEvents(data []byte) []SSEEvent {
	if len(data) == 0 {
		return nil
	}

	var events []SSEEvent
	var current SSEEvent
	var dataLines []string

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()

		// Empty line = end of event
		if line == "" {
			if len(dataLines) > 0 || current.Event != "" || current.ID != "" {
				current.Data = strings.Join(dataLines, "\n")
				events = append(events, current)
				current = SSEEvent{}
				dataLines = nil
			}
			continue
		}

		// Skip comments
		if strings.HasPrefix(line, ":") {
			continue
		}

		// Parse field
		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			// Field with no value
			continue
		}

		field := line[:colonIdx]
		value := line[colonIdx+1:]

		// Remove leading space from value (SSE spec)
		if len(value) > 0 && value[0] == ' ' {
			value = value[1:]
		}

		switch field {
		case "event":
			current.Event = value
		case "data":
			dataLines = append(dataLines, value)
		case "id":
			current.ID = value
		case "retry":
			// Parse retry as int, ignore if invalid
			var retry int
			if _, err := parseRetry(value); err == nil {
				current.Retry = retry
			}
		}
	}

	// Handle final event if no trailing newline
	if len(dataLines) > 0 || current.Event != "" || current.ID != "" {
		current.Data = strings.Join(dataLines, "\n")
		events = append(events, current)
	}

	return events
}

// ParseSSELastEvent returns just the last complete SSE event.
// Useful when only the final event contains usage data.
func ParseSSELastEvent(data []byte) *SSEEvent {
	events := ParseSSEEvents(data)
	if len(events) == 0 {
		return nil
	}
	return &events[len(events)-1]
}

// ExtractSSEData extracts all data fields from SSE, concatenated.
// Returns just the data portions, ignoring event types and IDs.
func ExtractSSEData(data []byte) string {
	events := ParseSSEEvents(data)
	if len(events) == 0 {
		return ""
	}

	var parts []string
	for _, e := range events {
		if e.Data != "" {
			parts = append(parts, e.Data)
		}
	}
	return strings.Join(parts, "\n")
}

// ExtractSSELastData returns the data from the last complete event.
// This is the most common case for metering - final event has usage info.
func ExtractSSELastData(data []byte) string {
	event := ParseSSELastEvent(data)
	if event == nil {
		return ""
	}
	return event.Data
}

// parseRetry parses the retry field value.
func parseRetry(s string) (int, error) {
	var v int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil // Invalid, return 0
		}
		v = v*10 + int(c-'0')
	}
	return v, nil
}

// SplitLines splits data into lines, handling both \n and \r\n.
// Returns empty strings for empty lines (preserves structure).
func SplitLines(data []byte) []string {
	if len(data) == 0 {
		return nil
	}

	s := string(data)
	// Normalize line endings
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	return strings.Split(s, "\n")
}

// SplitLinesNonEmpty splits data into non-empty lines.
func SplitLinesNonEmpty(data []byte) []string {
	lines := SplitLines(data)
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}
