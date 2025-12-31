package schema

import (
	"strings"
	"testing"
)

func TestPathTypeString(t *testing.T) {
	tests := []struct {
		pathType PathType
		expected string
	}{
		{PathTypeHTTP, "http"},
		{PathTypeCLI, "cli"},
		{PathTypeWebSocket, "websocket"},
		{PathTypeWebhook, "webhook"},
		{PathTypeGRPC, "grpc"},
		{PathType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.pathType.String(); got != tt.expected {
				t.Errorf("PathType.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPathClaimKey(t *testing.T) {
	tests := []struct {
		name     string
		claim    PathClaim
		expected string
	}{
		{
			name: "HTTP GET",
			claim: PathClaim{
				Type:   PathTypeHTTP,
				Method: "GET",
				Path:   "/users",
			},
			expected: "http:GET:/users",
		},
		{
			name: "HTTP POST",
			claim: PathClaim{
				Type:   PathTypeHTTP,
				Method: "POST",
				Path:   "/users",
			},
			expected: "http:POST:/users",
		},
		{
			name: "CLI",
			claim: PathClaim{
				Type: PathTypeCLI,
				Path: "users list",
			},
			expected: "cli:users list",
		},
		{
			name: "WebSocket",
			claim: PathClaim{
				Type: PathTypeWebSocket,
				Path: "/ws/users",
			},
			expected: "ws:/ws/users",
		},
		{
			name: "Webhook",
			claim: PathClaim{
				Type: PathTypeWebhook,
				Path: "/webhooks/user",
			},
			expected: "webhook:/webhooks/user",
		},
		{
			name: "gRPC",
			claim: PathClaim{
				Type: PathTypeGRPC,
				Path: "UserService",
			},
			expected: "grpc:UserService",
		},
		{
			name: "Unknown",
			claim: PathClaim{
				Type: PathType(99),
				Path: "/test",
			},
			expected: "unknown:/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.claim.Key(); got != tt.expected {
				t.Errorf("PathClaim.Key() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPathConflictError(t *testing.T) {
	conflict := PathConflict{
		Type: PathTypeHTTP,
		Path: "/users",
		Claims: []PathClaim{
			{Module: "user"},
			{Module: "account"},
		},
	}

	errStr := conflict.Error()

	if !strings.Contains(errStr, "path conflict") {
		t.Error("PathConflict.Error() should contain 'path conflict'")
	}
	if !strings.Contains(errStr, "http") {
		t.Error("PathConflict.Error() should contain path type 'http'")
	}
	if !strings.Contains(errStr, "/users") {
		t.Error("PathConflict.Error() should contain path '/users'")
	}
	if !strings.Contains(errStr, "user") {
		t.Error("PathConflict.Error() should contain module 'user'")
	}
	if !strings.Contains(errStr, "account") {
		t.Error("PathConflict.Error() should contain module 'account'")
	}
}

func TestExtractPaths_HTTPEnabled(t *testing.T) {
	mod := Module{
		Name: "user",
		Schema: map[string]Field{
			"name": {Type: FieldTypeString},
		},
		Channels: Channels{
			HTTP: HTTPChannel{
				Serve: HTTPServe{
					Enabled: true,
				},
			},
		},
	}

	claims := ExtractPaths(mod, "users")

	// Should have 5 implicit CRUD paths
	expectedPaths := map[string]string{
		"GET:/users":        "list",
		"POST:/users":       "create",
		"GET:/users/{id}":   "get",
		"PATCH:/users/{id}": "update",
		"DELETE:/users/{id}": "delete",
	}

	for _, claim := range claims {
		if claim.Type != PathTypeHTTP {
			continue
		}
		key := claim.Method + ":" + claim.Path
		expectedAction, ok := expectedPaths[key]
		if !ok {
			continue // might be a custom action
		}
		if claim.Action != expectedAction {
			t.Errorf("Path %s has action %q, want %q", key, claim.Action, expectedAction)
		}
		if claim.Module != "user" {
			t.Errorf("Path %s has module %q, want %q", key, claim.Module, "user")
		}
	}
}

func TestExtractPaths_HTTPCustomBasePath(t *testing.T) {
	mod := Module{
		Name: "user",
		Schema: map[string]Field{
			"name": {Type: FieldTypeString},
		},
		Channels: Channels{
			HTTP: HTTPChannel{
				Serve: HTTPServe{
					Enabled:  true,
					BasePath: "/api/v1/users",
				},
			},
		},
	}

	claims := ExtractPaths(mod, "users")

	foundList := false
	for _, claim := range claims {
		if claim.Type == PathTypeHTTP && claim.Action == "list" {
			foundList = true
			if claim.Path != "/api/v1/users" {
				t.Errorf("List path = %q, want %q", claim.Path, "/api/v1/users")
			}
		}
	}

	if !foundList {
		t.Error("Should have list action path")
	}
}

func TestExtractPaths_HTTPCustomActions(t *testing.T) {
	mod := Module{
		Name: "user",
		Schema: map[string]Field{
			"status": {Type: FieldTypeString},
		},
		Actions: map[string]Action{
			"activate":   {Set: map[string]string{"status": "active"}},
			"deactivate": {Set: map[string]string{"status": "inactive"}},
		},
		Channels: Channels{
			HTTP: HTTPChannel{
				Serve: HTTPServe{
					Enabled: true,
				},
			},
		},
	}

	claims := ExtractPaths(mod, "users")

	foundActivate := false
	foundDeactivate := false
	for _, claim := range claims {
		if claim.Type == PathTypeHTTP {
			if claim.Action == "activate" {
				foundActivate = true
				if claim.Path != "/users/{id}/activate" {
					t.Errorf("Activate path = %q, want %q", claim.Path, "/users/{id}/activate")
				}
				if claim.Method != "POST" {
					t.Errorf("Activate method = %q, want %q", claim.Method, "POST")
				}
			}
			if claim.Action == "deactivate" {
				foundDeactivate = true
			}
		}
	}

	if !foundActivate {
		t.Error("Should have activate action path")
	}
	if !foundDeactivate {
		t.Error("Should have deactivate action path")
	}
}

func TestExtractPaths_HTTPExplicitEndpoints(t *testing.T) {
	mod := Module{
		Name: "user",
		Schema: map[string]Field{
			"name": {Type: FieldTypeString},
		},
		Channels: Channels{
			HTTP: HTTPChannel{
				Serve: HTTPServe{
					Enabled: true,
					Endpoints: []HTTPEndpoint{
						{Action: "search", Method: "GET", Path: "/search"},
					},
				},
			},
		},
	}

	claims := ExtractPaths(mod, "users")

	foundSearch := false
	for _, claim := range claims {
		if claim.Type == PathTypeHTTP && claim.Action == "search" {
			foundSearch = true
			if claim.Path != "/users/search" {
				t.Errorf("Search path = %q, want %q", claim.Path, "/users/search")
			}
			if claim.Method != "GET" {
				t.Errorf("Search method = %q, want %q", claim.Method, "GET")
			}
		}
	}

	if !foundSearch {
		t.Error("Should have search endpoint path")
	}
}

func TestExtractPaths_CLIEnabled(t *testing.T) {
	mod := Module{
		Name: "user",
		Schema: map[string]Field{
			"name": {Type: FieldTypeString},
		},
		Channels: Channels{
			CLI: CLIChannel{
				Serve: CLIServe{
					Enabled: true,
				},
			},
		},
	}

	claims := ExtractPaths(mod, "users")

	expectedActions := []string{"list", "get", "create", "update", "delete"}
	foundActions := make(map[string]bool)

	for _, claim := range claims {
		if claim.Type == PathTypeCLI {
			foundActions[claim.Action] = true
			expectedPath := "users " + claim.Action
			if claim.Path != expectedPath {
				t.Errorf("CLI path for %s = %q, want %q", claim.Action, claim.Path, expectedPath)
			}
		}
	}

	for _, action := range expectedActions {
		if !foundActions[action] {
			t.Errorf("Missing CLI action: %s", action)
		}
	}
}

func TestExtractPaths_CLICustomCommand(t *testing.T) {
	mod := Module{
		Name: "user",
		Schema: map[string]Field{
			"name": {Type: FieldTypeString},
		},
		Channels: Channels{
			CLI: CLIChannel{
				Serve: CLIServe{
					Enabled: true,
					Command: "accounts",
				},
			},
		},
	}

	claims := ExtractPaths(mod, "users")

	foundList := false
	for _, claim := range claims {
		if claim.Type == PathTypeCLI && claim.Action == "list" {
			foundList = true
			if claim.Path != "accounts list" {
				t.Errorf("CLI list path = %q, want %q", claim.Path, "accounts list")
			}
		}
	}

	if !foundList {
		t.Error("Should have CLI list action")
	}
}

func TestExtractPaths_CLICustomActions(t *testing.T) {
	mod := Module{
		Name: "user",
		Schema: map[string]Field{
			"status": {Type: FieldTypeString},
		},
		Actions: map[string]Action{
			"activate": {Set: map[string]string{"status": "active"}},
		},
		Channels: Channels{
			CLI: CLIChannel{
				Serve: CLIServe{
					Enabled: true,
				},
			},
		},
	}

	claims := ExtractPaths(mod, "users")

	foundActivate := false
	for _, claim := range claims {
		if claim.Type == PathTypeCLI && claim.Action == "activate" {
			foundActivate = true
			if claim.Path != "users activate" {
				t.Errorf("CLI activate path = %q, want %q", claim.Path, "users activate")
			}
		}
	}

	if !foundActivate {
		t.Error("Should have CLI activate action")
	}
}

func TestExtractPaths_WebSocketEnabled(t *testing.T) {
	mod := Module{
		Name: "user",
		Schema: map[string]Field{
			"name": {Type: FieldTypeString},
		},
		Channels: Channels{
			WebSocket: WebSocketChannel{
				Serve: WebSocketServe{
					Enabled: true,
				},
			},
		},
	}

	claims := ExtractPaths(mod, "users")

	foundWS := false
	for _, claim := range claims {
		if claim.Type == PathTypeWebSocket {
			foundWS = true
			if claim.Path != "/ws/users" {
				t.Errorf("WebSocket path = %q, want %q", claim.Path, "/ws/users")
			}
			if claim.Module != "user" {
				t.Errorf("WebSocket module = %q, want %q", claim.Module, "user")
			}
		}
	}

	if !foundWS {
		t.Error("Should have WebSocket path")
	}
}

func TestExtractPaths_WebSocketCustomPath(t *testing.T) {
	mod := Module{
		Name: "user",
		Schema: map[string]Field{
			"name": {Type: FieldTypeString},
		},
		Channels: Channels{
			WebSocket: WebSocketChannel{
				Serve: WebSocketServe{
					Enabled: true,
					Path:    "/realtime",
				},
			},
		},
	}

	claims := ExtractPaths(mod, "users")

	foundWS := false
	for _, claim := range claims {
		if claim.Type == PathTypeWebSocket {
			foundWS = true
			if claim.Path != "/realtime" {
				t.Errorf("WebSocket path = %q, want %q", claim.Path, "/realtime")
			}
		}
	}

	if !foundWS {
		t.Error("Should have WebSocket path")
	}
}

func TestExtractPaths_WebhookEnabled(t *testing.T) {
	mod := Module{
		Name: "user",
		Schema: map[string]Field{
			"name": {Type: FieldTypeString},
		},
		Channels: Channels{
			Webhook: WebhookChannel{
				Serve: WebhookServe{
					Enabled: true,
				},
			},
		},
	}

	claims := ExtractPaths(mod, "users")

	foundWebhook := false
	for _, claim := range claims {
		if claim.Type == PathTypeWebhook {
			foundWebhook = true
			if claim.Path != "/webhooks/user" {
				t.Errorf("Webhook path = %q, want %q", claim.Path, "/webhooks/user")
			}
			if claim.Module != "user" {
				t.Errorf("Webhook module = %q, want %q", claim.Module, "user")
			}
		}
	}

	if !foundWebhook {
		t.Error("Should have Webhook path")
	}
}

func TestExtractPaths_WebhookCustomPath(t *testing.T) {
	mod := Module{
		Name: "user",
		Schema: map[string]Field{
			"name": {Type: FieldTypeString},
		},
		Channels: Channels{
			Webhook: WebhookChannel{
				Serve: WebhookServe{
					Enabled: true,
					Path:    "/hooks/incoming",
				},
			},
		},
	}

	claims := ExtractPaths(mod, "users")

	foundWebhook := false
	for _, claim := range claims {
		if claim.Type == PathTypeWebhook {
			foundWebhook = true
			if claim.Path != "/hooks/incoming" {
				t.Errorf("Webhook path = %q, want %q", claim.Path, "/hooks/incoming")
			}
		}
	}

	if !foundWebhook {
		t.Error("Should have Webhook path")
	}
}

func TestExtractPaths_NoChannelsEnabled(t *testing.T) {
	mod := Module{
		Name: "user",
		Schema: map[string]Field{
			"name": {Type: FieldTypeString},
		},
	}

	claims := ExtractPaths(mod, "users")

	if len(claims) != 0 {
		t.Errorf("Expected 0 claims for module with no channels enabled, got %d", len(claims))
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/users/", "/users"},
		{"/users", "/users"},
		{"/users/:id", "/users/{id}"},
		{"/users/<id>", "/users/{id}"},
		{"/users/{id}", "/users/{id}"},
		{"/users/:id/posts/:id", "/users/{id}/posts/{id}"},
		{"/users/<id>/posts/<id>", "/users/{id}/posts/{id}"},
		{"", ""},
		{"/", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := NormalizePath(tt.input); got != tt.expected {
				t.Errorf("NormalizePath(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestPathClaimStruct(t *testing.T) {
	claim := PathClaim{
		Type:        PathTypeHTTP,
		Path:        "/users",
		Module:      "user",
		Action:      "list",
		Method:      "GET",
		Priority:    10,
		Description: "List all users",
	}

	if claim.Type != PathTypeHTTP {
		t.Error("PathClaim.Type not set correctly")
	}
	if claim.Path != "/users" {
		t.Error("PathClaim.Path not set correctly")
	}
	if claim.Module != "user" {
		t.Error("PathClaim.Module not set correctly")
	}
	if claim.Action != "list" {
		t.Error("PathClaim.Action not set correctly")
	}
	if claim.Method != "GET" {
		t.Error("PathClaim.Method not set correctly")
	}
	if claim.Priority != 10 {
		t.Error("PathClaim.Priority not set correctly")
	}
	if claim.Description != "List all users" {
		t.Error("PathClaim.Description not set correctly")
	}
}

func TestPathConflictStruct(t *testing.T) {
	conflict := PathConflict{
		Type: PathTypeHTTP,
		Path: "/users",
		Claims: []PathClaim{
			{Module: "user", Action: "list"},
			{Module: "account", Action: "list"},
		},
	}

	if conflict.Type != PathTypeHTTP {
		t.Error("PathConflict.Type not set correctly")
	}
	if conflict.Path != "/users" {
		t.Error("PathConflict.Path not set correctly")
	}
	if len(conflict.Claims) != 2 {
		t.Error("PathConflict.Claims not set correctly")
	}
}
