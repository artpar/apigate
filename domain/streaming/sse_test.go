package streaming

import (
	"testing"
)

func TestParseSSEEvents(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   []SSEEvent
	}{
		{
			name:  "empty input",
			input: "",
			want:  nil,
		},
		{
			name:  "single event",
			input: "data: hello\n\n",
			want: []SSEEvent{
				{Data: "hello"},
			},
		},
		{
			name:  "event with type",
			input: "event: message\ndata: hello\n\n",
			want: []SSEEvent{
				{Event: "message", Data: "hello"},
			},
		},
		{
			name:  "multiple events",
			input: "data: first\n\ndata: second\n\n",
			want: []SSEEvent{
				{Data: "first"},
				{Data: "second"},
			},
		},
		{
			name:  "multi-line data",
			input: "data: line1\ndata: line2\n\n",
			want: []SSEEvent{
				{Data: "line1\nline2"},
			},
		},
		{
			name:  "event with id",
			input: "id: 123\ndata: hello\n\n",
			want: []SSEEvent{
				{ID: "123", Data: "hello"},
			},
		},
		{
			name:  "comment ignored",
			input: ": this is a comment\ndata: hello\n\n",
			want: []SSEEvent{
				{Data: "hello"},
			},
		},
		{
			name:  "json data",
			input: "data: {\"usage\":{\"tokens\":100}}\n\n",
			want: []SSEEvent{
				{Data: `{"usage":{"tokens":100}}`},
			},
		},
		{
			name:  "no trailing newline",
			input: "data: hello",
			want: []SSEEvent{
				{Data: "hello"},
			},
		},
		{
			name:  "typical LLM stream",
			input: "data: {\"text\":\"Hello\"}\n\ndata: {\"text\":\" world\"}\n\ndata: {\"usage\":{\"tokens\":50}}\n\n",
			want: []SSEEvent{
				{Data: `{"text":"Hello"}`},
				{Data: `{"text":" world"}`},
				{Data: `{"usage":{"tokens":50}}`},
			},
		},
		{
			name:  "space after colon",
			input: "data: hello\n\n",
			want: []SSEEvent{
				{Data: "hello"},
			},
		},
		{
			name:  "no space after colon",
			input: "data:hello\n\n",
			want: []SSEEvent{
				{Data: "hello"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSSEEvents([]byte(tt.input))

			if len(got) != len(tt.want) {
				t.Errorf("ParseSSEEvents() returned %d events, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i].Event != tt.want[i].Event {
					t.Errorf("Event[%d].Event = %q, want %q", i, got[i].Event, tt.want[i].Event)
				}
				if got[i].Data != tt.want[i].Data {
					t.Errorf("Event[%d].Data = %q, want %q", i, got[i].Data, tt.want[i].Data)
				}
				if got[i].ID != tt.want[i].ID {
					t.Errorf("Event[%d].ID = %q, want %q", i, got[i].ID, tt.want[i].ID)
				}
			}
		})
	}
}

func TestParseSSELastEvent(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  *SSEEvent
	}{
		{
			name:  "empty",
			input: "",
			want:  nil,
		},
		{
			name:  "single event",
			input: "data: hello\n\n",
			want:  &SSEEvent{Data: "hello"},
		},
		{
			name:  "multiple events returns last",
			input: "data: first\n\ndata: second\n\ndata: last\n\n",
			want:  &SSEEvent{Data: "last"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSSELastEvent([]byte(tt.input))

			if tt.want == nil {
				if got != nil {
					t.Errorf("ParseSSELastEvent() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Errorf("ParseSSELastEvent() = nil, want %v", tt.want)
				return
			}

			if got.Data != tt.want.Data {
				t.Errorf("ParseSSELastEvent().Data = %q, want %q", got.Data, tt.want.Data)
			}
		})
	}
}

func TestExtractSSELastData(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty",
			input: "",
			want:  "",
		},
		{
			name:  "single event",
			input: "data: {\"tokens\":100}\n\n",
			want:  `{"tokens":100}`,
		},
		{
			name:  "multiple events returns last data",
			input: "data: {\"chunk\":1}\n\ndata: {\"chunk\":2}\n\ndata: {\"usage\":{\"tokens\":50}}\n\n",
			want:  `{"usage":{"tokens":50}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractSSELastData([]byte(tt.input))
			if got != tt.want {
				t.Errorf("ExtractSSELastData() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "empty",
			input: "",
			want:  nil,
		},
		{
			name:  "single line",
			input: "hello",
			want:  []string{"hello"},
		},
		{
			name:  "multiple lines",
			input: "line1\nline2\nline3",
			want:  []string{"line1", "line2", "line3"},
		},
		{
			name:  "windows line endings",
			input: "line1\r\nline2\r\nline3",
			want:  []string{"line1", "line2", "line3"},
		},
		{
			name:  "empty lines preserved",
			input: "line1\n\nline3",
			want:  []string{"line1", "", "line3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitLines([]byte(tt.input))

			if len(got) != len(tt.want) {
				t.Errorf("SplitLines() = %v, want %v", got, tt.want)
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("SplitLines()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
