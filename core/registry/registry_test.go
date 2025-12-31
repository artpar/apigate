package registry

import (
	"testing"

	"github.com/artpar/apigate/core/schema"
)

// Helper function to create a simple test module
func makeTestModule(name string) schema.Module {
	return schema.Module{
		Name: name,
		Schema: map[string]schema.Field{
			"id":   {Type: schema.FieldTypeString},
			"name": {Type: schema.FieldTypeString},
		},
		Channels: schema.Channels{
			HTTP: schema.HTTPChannel{
				Serve: schema.HTTPServe{
					Enabled:  true,
					BasePath: "/api/" + name,
				},
			},
		},
	}
}

func TestNew(t *testing.T) {
	r := New()
	if r == nil {
		t.Fatal("New() returned nil")
	}
	if r.modules == nil {
		t.Error("modules map not initialized")
	}
	if r.paths == nil {
		t.Error("paths map not initialized")
	}
	if r.tables == nil {
		t.Error("tables map not initialized")
	}
}

func TestRegistry_Register(t *testing.T) {
	r := New()
	mod := makeTestModule("user")

	err := r.Register(mod)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Check module was registered
	derived, ok := r.Get("user")
	if !ok {
		t.Error("Get() should find registered module")
	}
	if derived.Source.Name != "user" {
		t.Errorf("Get().Source.Name = %s, want user", derived.Source.Name)
	}
}

func TestRegistry_Register_DuplicateName(t *testing.T) {
	r := New()
	mod := makeTestModule("user")

	err := r.Register(mod)
	if err != nil {
		t.Fatalf("First Register() error = %v", err)
	}

	// Attempt to register again
	err = r.Register(mod)
	if err == nil {
		t.Error("Second Register() should fail with duplicate name")
	}
}

func TestRegistry_Unregister(t *testing.T) {
	r := New()
	mod := makeTestModule("user")

	err := r.Register(mod)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	err = r.Unregister("user")
	if err != nil {
		t.Fatalf("Unregister() error = %v", err)
	}

	// Check module was removed
	_, ok := r.Get("user")
	if ok {
		t.Error("Get() should not find unregistered module")
	}
}

func TestRegistry_Unregister_NotFound(t *testing.T) {
	r := New()

	err := r.Unregister("nonexistent")
	if err == nil {
		t.Error("Unregister() should fail for non-existent module")
	}
}

func TestRegistry_Get_NotFound(t *testing.T) {
	r := New()

	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("Get() should return false for non-existent module")
	}
}

func TestRegistry_List(t *testing.T) {
	r := New()

	// Register multiple modules
	modules := []string{"user", "plan", "key"}
	for _, name := range modules {
		mod := makeTestModule(name)
		if err := r.Register(mod); err != nil {
			t.Fatalf("Register(%s) error = %v", name, err)
		}
	}

	list := r.List()
	if len(list) != 3 {
		t.Errorf("List() returned %d modules, want 3", len(list))
	}

	// Check that list is sorted by name
	for i := 1; i < len(list); i++ {
		if list[i-1].Source.Name >= list[i].Source.Name {
			t.Error("List() should be sorted by name")
		}
	}
}

func TestRegistry_All(t *testing.T) {
	r := New()

	// Register multiple modules
	modules := []string{"user", "plan", "key"}
	for _, name := range modules {
		mod := makeTestModule(name)
		if err := r.Register(mod); err != nil {
			t.Fatalf("Register(%s) error = %v", name, err)
		}
	}

	all := r.All()
	if len(all) != 3 {
		t.Errorf("All() returned %d modules, want 3", len(all))
	}

	// Check specific modules are present
	for _, name := range modules {
		if _, ok := all[name]; !ok {
			t.Errorf("All() should contain module %s", name)
		}
	}
}

func TestRegistry_LookupPath(t *testing.T) {
	r := New()
	mod := makeTestModule("user")

	if err := r.Register(mod); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Try to lookup a path - the actual paths depend on convention.Derive
	paths := r.GetHTTPPaths()
	if len(paths) > 0 {
		// We have at least one HTTP path to test
		path := paths[0]
		module, action, ok := r.LookupPath(schema.PathTypeHTTP, path.Method, path.Path)
		if !ok {
			t.Errorf("LookupPath() should find path %s", path.Path)
		}
		if module == "" {
			t.Error("LookupPath() should return module name")
		}
		if action == "" {
			t.Error("LookupPath() should return action name")
		}
	}
}

func TestRegistry_LookupPath_NotFound(t *testing.T) {
	r := New()

	module, action, ok := r.LookupPath(schema.PathTypeHTTP, "GET", "/nonexistent")
	if ok {
		t.Error("LookupPath() should return false for non-existent path")
	}
	if module != "" || action != "" {
		t.Error("LookupPath() should return empty strings for non-existent path")
	}
}

func TestRegistry_GetHTTPPaths(t *testing.T) {
	r := New()
	mod := makeTestModule("user")

	if err := r.Register(mod); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	paths := r.GetHTTPPaths()
	// Paths depend on convention.Derive, just check type is correct
	for _, p := range paths {
		if p.Type != schema.PathTypeHTTP {
			t.Errorf("GetHTTPPaths() returned path of type %s", p.Type)
		}
	}
}

func TestRegistry_GetCLIPaths(t *testing.T) {
	r := New()

	// Create module with CLI channels
	mod := schema.Module{
		Name: "user",
		Schema: map[string]schema.Field{
			"id": {Type: schema.FieldTypeString},
		},
		Channels: schema.Channels{
			CLI: schema.CLIChannel{
				Serve: schema.CLIServe{
					Enabled: true,
					Command: "users",
				},
			},
		},
	}

	if err := r.Register(mod); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	paths := r.GetCLIPaths()
	for _, p := range paths {
		if p.Type != schema.PathTypeCLI {
			t.Errorf("GetCLIPaths() returned path of type %s", p.Type)
		}
	}
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		pattern string
		key     string
		want    bool
	}{
		{"api/users", "api/users", true},
		{"api/users/{id}", "api/users/123", true},
		{"api/users/{id}/keys", "api/users/123/keys", true},
		{"api/users", "api/plans", false},
		{"api/users/{id}", "api/users", false},
		{"api/users/{id}", "api/users/123/extra", false},
		{"{module}/list", "users/list", true},
		{"{module}/{action}", "users/create", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.key, func(t *testing.T) {
			got := matchPattern(tt.pattern, tt.key)
			if got != tt.want {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.pattern, tt.key, got, tt.want)
			}
		})
	}
}

func TestConflictError_Error(t *testing.T) {
	err := &ConflictError{
		Conflicts: []schema.PathConflict{
			{
				Type: schema.PathTypeHTTP,
				Path: "/api/users",
				Claims: []schema.PathClaim{
					{Module: "user", Action: "list"},
					{Module: "profile", Action: "list"},
				},
			},
		},
	}

	msg := err.Error()
	if msg == "" {
		t.Error("ConflictError.Error() should return non-empty message")
	}
}

func TestConflictError_HasConflicts(t *testing.T) {
	noConflicts := &ConflictError{Conflicts: nil}
	if noConflicts.HasConflicts() {
		t.Error("HasConflicts() should return false for empty conflicts")
	}

	hasConflicts := &ConflictError{
		Conflicts: []schema.PathConflict{{Path: "/test"}},
	}
	if !hasConflicts.HasConflicts() {
		t.Error("HasConflicts() should return true for non-empty conflicts")
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := New()

	// Concurrent reads and writes
	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			r.List()
			r.All()
			r.GetHTTPPaths()
			r.GetCLIPaths()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			mod := makeTestModule("module" + string(rune('A'+i)))
			r.Register(mod)
		}
		done <- true
	}()

	<-done
	<-done
	// Test passes if no race conditions detected
}
