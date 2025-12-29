package idgen_test

import (
	"regexp"
	"testing"

	"github.com/artpar/apigate/adapters/idgen"
)

func TestUUID_New(t *testing.T) {
	g := idgen.UUID{}

	id := g.New()
	if id == "" {
		t.Error("expected non-empty ID")
	}

	// UUID v4 format: 8-4-4-4-12 hex chars
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !uuidRegex.MatchString(id) {
		t.Errorf("ID %s doesn't match UUID v4 format", id)
	}
}

func TestUUID_New_Unique(t *testing.T) {
	g := idgen.UUID{}

	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := g.New()
		if seen[id] {
			t.Errorf("duplicate ID generated: %s", id)
		}
		seen[id] = true
	}
}

func TestSequential_New(t *testing.T) {
	g := idgen.NewSequential("test_")

	id := g.New()
	if id != "test_1" {
		t.Errorf("first ID = %s, want test_1", id)
	}

	id = g.New()
	if id != "test_2" {
		t.Errorf("second ID = %s, want test_2", id)
	}

	id = g.New()
	if id != "test_3" {
		t.Errorf("third ID = %s, want test_3", id)
	}
}

func TestSequential_New_NoPrefix(t *testing.T) {
	g := idgen.NewSequential("")

	id := g.New()
	if id != "1" {
		t.Errorf("ID = %s, want 1", id)
	}
}

func TestSequential_New_CustomPrefix(t *testing.T) {
	g := idgen.NewSequential("user-")

	id := g.New()
	if id != "user-1" {
		t.Errorf("ID = %s, want user-1", id)
	}
}

func TestSequential_Reset(t *testing.T) {
	g := idgen.NewSequential("id_")

	g.New() // 1
	g.New() // 2
	g.New() // 3

	g.Reset()

	id := g.New()
	if id != "id_1" {
		t.Errorf("after reset ID = %s, want id_1", id)
	}
}

func TestSequential_LargeNumbers(t *testing.T) {
	g := idgen.NewSequential("n_")

	// Generate many IDs
	for i := 0; i < 1000; i++ {
		g.New()
	}

	id := g.New()
	if id != "n_1001" {
		t.Errorf("ID = %s, want n_1001", id)
	}
}

func TestSequential_ConcurrentAccess(t *testing.T) {
	g := idgen.NewSequential("concurrent_")

	done := make(chan bool)
	ids := make(chan string, 1000)

	// Generate IDs concurrently
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				ids <- g.New()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	close(ids)

	// Check all IDs are unique
	seen := make(map[string]bool)
	for id := range ids {
		if seen[id] {
			t.Errorf("duplicate ID: %s", id)
		}
		seen[id] = true
	}

	if len(seen) != 1000 {
		t.Errorf("expected 1000 unique IDs, got %d", len(seen))
	}
}

func TestSequential_Zero(t *testing.T) {
	g := idgen.NewSequential("z_")
	g.Reset()

	// After reset, counter is 0, so first ID should be 1
	id := g.New()
	if id != "z_1" {
		t.Errorf("ID = %s, want z_1", id)
	}
}
