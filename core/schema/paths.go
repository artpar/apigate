package schema

import (
	"fmt"
	"strings"
)

// PathType represents the type of path/endpoint.
type PathType int

const (
	// PathTypeHTTP is an HTTP endpoint path.
	PathTypeHTTP PathType = iota

	// PathTypeCLI is a CLI command path.
	PathTypeCLI

	// PathTypeWebSocket is a WebSocket endpoint path.
	PathTypeWebSocket

	// PathTypeWebhook is a webhook endpoint path.
	PathTypeWebhook

	// PathTypeGRPC is a gRPC service path.
	PathTypeGRPC
)

// String returns the path type name.
func (p PathType) String() string {
	switch p {
	case PathTypeHTTP:
		return "http"
	case PathTypeCLI:
		return "cli"
	case PathTypeWebSocket:
		return "websocket"
	case PathTypeWebhook:
		return "webhook"
	case PathTypeGRPC:
		return "grpc"
	default:
		return "unknown"
	}
}

// PathClaim represents a module's claim on a path/endpoint.
type PathClaim struct {
	// Type of the path (HTTP, CLI, WebSocket, etc.)
	Type PathType

	// Path is the claimed path string.
	// For HTTP: "/users", "/users/{id}"
	// For CLI: "users", "users list"
	// For WebSocket: "/ws"
	Path string

	// Module that claims this path.
	Module string

	// Action that handles this path.
	Action string

	// Method for HTTP paths (GET, POST, etc.)
	Method string

	// Priority for conflict resolution (higher wins).
	Priority int

	// Description for documentation.
	Description string
}

// Key returns a unique key for this path claim.
func (p PathClaim) Key() string {
	switch p.Type {
	case PathTypeHTTP:
		return fmt.Sprintf("http:%s:%s", p.Method, p.Path)
	case PathTypeCLI:
		return fmt.Sprintf("cli:%s", p.Path)
	case PathTypeWebSocket:
		return fmt.Sprintf("ws:%s", p.Path)
	case PathTypeWebhook:
		return fmt.Sprintf("webhook:%s", p.Path)
	case PathTypeGRPC:
		return fmt.Sprintf("grpc:%s", p.Path)
	default:
		return fmt.Sprintf("%s:%s", p.Type, p.Path)
	}
}

// PathConflict represents a conflict between two path claims.
type PathConflict struct {
	// Type of the path.
	Type PathType

	// Path that is conflicted.
	Path string

	// Claims are the conflicting claims.
	Claims []PathClaim
}

// Error returns the conflict as an error string.
func (c PathConflict) Error() string {
	modules := make([]string, len(c.Claims))
	for i, claim := range c.Claims {
		modules[i] = claim.Module
	}
	return fmt.Sprintf("path conflict on %s %s: claimed by modules [%s]",
		c.Type, c.Path, strings.Join(modules, ", "))
}

// ExtractPaths extracts all path claims from a module.
func ExtractPaths(mod Module, plural string) []PathClaim {
	var claims []PathClaim

	// HTTP paths
	if mod.Channels.HTTP.Serve.Enabled {
		basePath := mod.Channels.HTTP.Serve.BasePath
		if basePath == "" {
			basePath = "/" + plural
		}

		// Implicit CRUD paths
		claims = append(claims, PathClaim{
			Type:   PathTypeHTTP,
			Path:   basePath,
			Module: mod.Name,
			Action: "list",
			Method: "GET",
		})
		claims = append(claims, PathClaim{
			Type:   PathTypeHTTP,
			Path:   basePath,
			Module: mod.Name,
			Action: "create",
			Method: "POST",
		})
		claims = append(claims, PathClaim{
			Type:   PathTypeHTTP,
			Path:   basePath + "/{id}",
			Module: mod.Name,
			Action: "get",
			Method: "GET",
		})
		claims = append(claims, PathClaim{
			Type:   PathTypeHTTP,
			Path:   basePath + "/{id}",
			Module: mod.Name,
			Action: "update",
			Method: "PATCH",
		})
		claims = append(claims, PathClaim{
			Type:   PathTypeHTTP,
			Path:   basePath + "/{id}",
			Module: mod.Name,
			Action: "delete",
			Method: "DELETE",
		})

		// Custom action paths
		for name := range mod.Actions {
			claims = append(claims, PathClaim{
				Type:   PathTypeHTTP,
				Path:   basePath + "/{id}/" + name,
				Module: mod.Name,
				Action: name,
				Method: "POST",
			})
		}

		// Explicit endpoint overrides
		for _, ep := range mod.Channels.HTTP.Serve.Endpoints {
			claims = append(claims, PathClaim{
				Type:   PathTypeHTTP,
				Path:   basePath + ep.Path,
				Module: mod.Name,
				Action: ep.Action,
				Method: ep.Method,
			})
		}
	}

	// CLI paths
	if mod.Channels.CLI.Serve.Enabled {
		command := mod.Channels.CLI.Serve.Command
		if command == "" {
			command = plural
		}

		// Implicit CRUD commands
		for _, action := range ImplicitActions() {
			claims = append(claims, PathClaim{
				Type:   PathTypeCLI,
				Path:   command + " " + action,
				Module: mod.Name,
				Action: action,
			})
		}

		// Custom action commands
		for name := range mod.Actions {
			claims = append(claims, PathClaim{
				Type:   PathTypeCLI,
				Path:   command + " " + name,
				Module: mod.Name,
				Action: name,
			})
		}
	}

	// WebSocket paths
	if mod.Channels.WebSocket.Serve.Enabled {
		path := mod.Channels.WebSocket.Serve.Path
		if path == "" {
			path = "/ws/" + plural
		}
		claims = append(claims, PathClaim{
			Type:   PathTypeWebSocket,
			Path:   path,
			Module: mod.Name,
		})
	}

	// Webhook paths
	if mod.Channels.Webhook.Serve.Enabled {
		path := mod.Channels.Webhook.Serve.Path
		if path == "" {
			path = "/webhooks/" + mod.Name
		}
		claims = append(claims, PathClaim{
			Type:   PathTypeWebhook,
			Path:   path,
			Module: mod.Name,
		})
	}

	return claims
}

// NormalizePath normalizes a path for comparison.
func NormalizePath(path string) string {
	// Remove trailing slashes
	path = strings.TrimSuffix(path, "/")
	// Normalize parameter syntax: {id}, :id, <id> all become {id}
	path = strings.ReplaceAll(path, ":id", "{id}")
	path = strings.ReplaceAll(path, "<id>", "{id}")
	return path
}
