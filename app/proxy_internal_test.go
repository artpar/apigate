package app

import (
	"testing"

	"github.com/artpar/apigate/domain/key"
)

func TestReasonToMessage(t *testing.T) {
	tests := []struct {
		name     string
		reason   string
		expected string
	}{
		{
			name:     "expired key",
			reason:   key.ReasonExpired,
			expected: "API key has expired",
		},
		{
			name:     "revoked key",
			reason:   key.ReasonRevoked,
			expected: "API key has been revoked",
		},
		{
			name:     "key not found",
			reason:   key.ReasonNotFound,
			expected: "API key not found",
		},
		{
			name:     "unknown reason",
			reason:   "unknown",
			expected: "Invalid API key",
		},
		{
			name:     "empty reason",
			reason:   "",
			expected: "Invalid API key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reasonToMessage(tt.reason)
			if got != tt.expected {
				t.Errorf("reasonToMessage(%q) = %q, want %q", tt.reason, got, tt.expected)
			}
		})
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{9, "9"},
		{10, "10"},
		{99, "99"},
		{100, "100"},
		{12345, "12345"},
		{-1, "-1"},
		{-10, "-10"},
		{-12345, "-12345"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := itoa(tt.input)
			if got != tt.expected {
				t.Errorf("itoa(%d) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestBytesContains(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		substr   []byte
		expected bool
	}{
		{
			name:     "contains substring",
			data:     []byte("hello world"),
			substr:   []byte("world"),
			expected: true,
		},
		{
			name:     "substring at start",
			data:     []byte("hello world"),
			substr:   []byte("hello"),
			expected: true,
		},
		{
			name:     "does not contain substring",
			data:     []byte("hello world"),
			substr:   []byte("foo"),
			expected: false,
		},
		{
			name:     "empty data",
			data:     []byte{},
			substr:   []byte("foo"),
			expected: false,
		},
		{
			name:     "empty substring",
			data:     []byte("hello world"),
			substr:   []byte{},
			expected: true,
		},
		{
			name:     "substring longer than data",
			data:     []byte("hi"),
			substr:   []byte("hello"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bytesContains(tt.data, tt.substr)
			if got != tt.expected {
				t.Errorf("bytesContains() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "contains exact case",
			s:        "Hello World",
			substr:   "World",
			expected: true,
		},
		{
			name:     "contains different case",
			s:        "Hello World",
			substr:   "world",
			expected: true,
		},
		{
			name:     "contains uppercase search",
			s:        "hello world",
			substr:   "WORLD",
			expected: true,
		},
		{
			name:     "does not contain",
			s:        "Hello World",
			substr:   "foo",
			expected: false,
		},
		{
			name:     "empty string",
			s:        "",
			substr:   "foo",
			expected: false,
		},
		{
			name:     "empty search",
			s:        "Hello World",
			substr:   "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsIgnoreCase(tt.s, tt.substr)
			if got != tt.expected {
				t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.expected)
			}
		})
	}
}
