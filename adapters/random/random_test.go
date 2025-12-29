package random_test

import (
	"testing"

	"github.com/artpar/apigate/adapters/random"
)

func TestReal_Bytes(t *testing.T) {
	r := random.Real{}

	b, err := r.Bytes(32)
	if err != nil {
		t.Fatalf("Bytes failed: %v", err)
	}

	if len(b) != 32 {
		t.Errorf("expected 32 bytes, got %d", len(b))
	}
}

func TestReal_Bytes_Unique(t *testing.T) {
	r := random.Real{}

	b1, _ := r.Bytes(32)
	b2, _ := r.Bytes(32)

	if string(b1) == string(b2) {
		t.Error("random bytes should be different")
	}
}

func TestReal_String(t *testing.T) {
	r := random.Real{}

	s, err := r.String(16)
	if err != nil {
		t.Fatalf("String failed: %v", err)
	}

	if len(s) != 16 {
		t.Errorf("expected 16 chars, got %d", len(s))
	}

	// Should be hex
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("non-hex character: %c", c)
		}
	}
}

func TestReal_String_Unique(t *testing.T) {
	r := random.Real{}

	s1, _ := r.String(32)
	s2, _ := r.String(32)

	if s1 == s2 {
		t.Error("random strings should be different")
	}
}

func TestReal_String_OddLength(t *testing.T) {
	r := random.Real{}

	s, _ := r.String(17)
	if len(s) != 17 {
		t.Errorf("expected 17 chars, got %d", len(s))
	}
}

func TestFake_NewFake(t *testing.T) {
	f := random.NewFake()
	if f == nil {
		t.Fatal("expected non-nil")
	}
}

func TestFake_Bytes_Deterministic(t *testing.T) {
	f := random.NewFake()

	b1, _ := f.Bytes(8)
	b2, _ := f.Bytes(8)

	// Counter increments, so values differ but are predictable
	if b1[0] == b2[0] {
		t.Error("expected different bytes from counter")
	}
}

func TestFake_Bytes_WithValues(t *testing.T) {
	f := random.NewFake().WithValues(
		[]byte{100, 101, 102, 103, 104, 105, 106, 107}, // Use distinct values
		[]byte{200, 201, 202, 203},
	)

	b1, _ := f.Bytes(4)
	if b1[0] != 100 || b1[3] != 103 {
		t.Errorf("expected first preset values, got %v", b1)
	}

	b2, _ := f.Bytes(4)
	if b2[0] != 200 {
		t.Errorf("expected second preset value, got %v", b2)
	}

	// Third call should use deterministic fallback (counter-based)
	b3, _ := f.Bytes(4)
	// Counter starts at 1, so bytes will be [1, 2, 3, 4]
	if b3[0] != 1 {
		t.Errorf("expected deterministic fallback starting at 1, got %v", b3)
	}
}

func TestFake_Bytes_Padding(t *testing.T) {
	f := random.NewFake().WithValues([]byte{1, 2})

	// Request more bytes than preset value
	b, _ := f.Bytes(8)
	if len(b) != 8 {
		t.Errorf("expected 8 bytes, got %d", len(b))
	}
	if b[0] != 1 || b[1] != 2 {
		t.Error("first bytes should be from preset")
	}
}

func TestFake_String(t *testing.T) {
	f := random.NewFake()

	s, err := f.String(16)
	if err != nil {
		t.Fatalf("String failed: %v", err)
	}

	if len(s) != 16 {
		t.Errorf("expected 16 chars, got %d", len(s))
	}
}

func TestFake_Reset(t *testing.T) {
	f := random.NewFake().WithValues([]byte{1, 2, 3, 4})

	f.Bytes(4) // Use preset
	f.Bytes(4) // Use counter

	f.Reset()

	b, _ := f.Bytes(4)
	if b[0] != 1 {
		t.Error("expected preset value after Reset")
	}
}

func TestFake_ConcurrentAccess(t *testing.T) {
	f := random.NewFake()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				f.Bytes(32)
				f.String(16)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
	// Test passes if no race conditions
}
