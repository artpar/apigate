// Package openapi provides OpenAPI 3.0 specification generation.
// This file generates OpenAPI specs from database route configurations.
package openapi

import (
	"regexp"
	"sort"
	"strings"

	"github.com/artpar/apigate/domain/route"
)

// RouteGenerator generates OpenAPI specs from route configurations.
type RouteGenerator struct {
	routes    []route.Route
	upstreams map[string]route.Upstream
	info      Info
	servers   []Server
}

// NewRouteGenerator creates a new route-based OpenAPI generator.
func NewRouteGenerator(routes []route.Route, upstreams map[string]route.Upstream) *RouteGenerator {
	return &RouteGenerator{
		routes:    routes,
		upstreams: upstreams,
		info: Info{
			Title:       "Proxied APIs",
			Version:     "1.0.0",
			Description: "Auto-generated documentation for proxied API routes",
		},
	}
}

// SetInfo sets the API info.
func (g *RouteGenerator) SetInfo(info Info) {
	g.info = info
}

// AddServer adds a server URL.
func (g *RouteGenerator) AddServer(url, description string) {
	g.servers = append(g.servers, Server{
		URL:         url,
		Description: description,
	})
}

// Generate creates the OpenAPI specification from routes.
func (g *RouteGenerator) Generate() *Spec {
	spec := &Spec{
		OpenAPI: "3.0.3",
		Info:    g.info,
		Servers: g.servers,
		Paths:   make(map[string]PathItem),
		Components: Components{
			Schemas:         make(map[string]*Schema),
			SecuritySchemes: make(map[string]SecurityScheme),
		},
		Tags: []Tag{
			{
				Name:        "Proxied APIs",
				Description: "Dynamically configured proxy routes",
			},
		},
	}

	// Collect security schemes from upstreams
	g.collectSecuritySchemes(spec)

	// Sort routes by priority (highest first) for consistent output
	sortedRoutes := make([]route.Route, len(g.routes))
	copy(sortedRoutes, g.routes)
	sort.Slice(sortedRoutes, func(i, j int) bool {
		if sortedRoutes[i].Priority != sortedRoutes[j].Priority {
			return sortedRoutes[i].Priority > sortedRoutes[j].Priority
		}
		return sortedRoutes[i].PathPattern < sortedRoutes[j].PathPattern
	})

	// Generate paths for each enabled route
	for _, r := range sortedRoutes {
		if !r.Enabled {
			continue
		}
		g.generateRoutePath(spec, r)
	}

	return spec
}

// collectSecuritySchemes collects unique security schemes from upstreams.
func (g *RouteGenerator) collectSecuritySchemes(spec *Spec) {
	// Always add API key auth (used by APIGate itself)
	spec.Components.SecuritySchemes["apiKey"] = SecurityScheme{
		Type:        "apiKey",
		In:          "header",
		Name:        "X-API-Key",
		Description: "API key authentication for APIGate",
	}

	// Collect from upstreams
	for _, upstream := range g.upstreams {
		if !upstream.Enabled {
			continue
		}

		switch upstream.AuthType {
		case route.AuthBearer:
			schemeName := sanitizeSchemeName(upstream.Name) + "_bearer"
			spec.Components.SecuritySchemes[schemeName] = SecurityScheme{
				Type:         "http",
				Scheme:       "bearer",
				Description:  "Bearer authentication for " + upstream.Name,
			}
		case route.AuthBasic:
			schemeName := sanitizeSchemeName(upstream.Name) + "_basic"
			spec.Components.SecuritySchemes[schemeName] = SecurityScheme{
				Type:        "http",
				Scheme:      "basic",
				Description: "Basic authentication for " + upstream.Name,
			}
		case route.AuthHeader:
			if upstream.AuthHeader != "" {
				schemeName := sanitizeSchemeName(upstream.Name) + "_header"
				spec.Components.SecuritySchemes[schemeName] = SecurityScheme{
					Type:        "apiKey",
					In:          "header",
					Name:        upstream.AuthHeader,
					Description: "Header authentication for " + upstream.Name,
				}
			}
		}
	}
}

// generateRoutePath adds a route to the spec.
func (g *RouteGenerator) generateRoutePath(spec *Spec, r route.Route) {
	openAPIPath, pathParams := convertPathPattern(r.PathPattern, r.MatchType)

	// Get or create PathItem
	pathItem := spec.Paths[openAPIPath]

	// Build parameters
	params := make([]Parameter, 0, len(pathParams)+len(r.Headers))

	// Add path parameters
	for _, paramName := range pathParams {
		params = append(params, Parameter{
			Name:        paramName,
			In:          "path",
			Required:    true,
			Description: "Path parameter: " + paramName,
			Schema:      &Schema{Type: "string"},
		})
	}

	// Add header parameters from route's header matching conditions
	for _, header := range r.Headers {
		params = append(params, Parameter{
			Name:        header.Name,
			In:          "header",
			Required:    header.Required,
			Description: "Header parameter",
			Schema:      &Schema{Type: "string"},
		})
	}

	// Build security requirements
	security := []SecurityRequirement{{"apiKey": {}}}

	// Add upstream-specific security if applicable
	if upstream, ok := g.upstreams[r.UpstreamID]; ok && upstream.Enabled {
		switch upstream.AuthType {
		case route.AuthBearer:
			schemeName := sanitizeSchemeName(upstream.Name) + "_bearer"
			security = append(security, SecurityRequirement{schemeName: {}})
		case route.AuthBasic:
			schemeName := sanitizeSchemeName(upstream.Name) + "_basic"
			security = append(security, SecurityRequirement{schemeName: {}})
		case route.AuthHeader:
			if upstream.AuthHeader != "" {
				schemeName := sanitizeSchemeName(upstream.Name) + "_header"
				security = append(security, SecurityRequirement{schemeName: {}})
			}
		}
	}

	// Determine which methods to document
	methods := r.Methods
	if len(methods) == 0 {
		// If no methods specified, route accepts all
		methods = []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
	}

	// Build description
	description := r.Description
	if description == "" {
		description = "Proxied route: " + r.Name
	}

	// Add protocol info to description
	if r.Protocol != "" && r.Protocol != route.ProtocolHTTP {
		description += " (Protocol: " + string(r.Protocol) + ")"
	}

	// Get upstream name for tags
	upstreamName := "Proxied APIs"
	if upstream, ok := g.upstreams[r.UpstreamID]; ok {
		upstreamName = upstream.Name
	}

	// Create operations for each method
	for _, method := range methods {
		method = strings.ToUpper(method)
		operationID := generateOperationID(r.Name, method)

		op := &Operation{
			Tags:        []string{upstreamName},
			Summary:     r.Name,
			Description: description,
			OperationID: operationID,
			Parameters:  params,
			Responses: map[string]Response{
				"200": {Description: "Successful response from upstream"},
				"400": {Description: "Bad request"},
				"401": {Description: "Unauthorized - invalid or missing API key"},
				"403": {Description: "Forbidden - quota exceeded or access denied"},
				"429": {Description: "Rate limit exceeded"},
				"500": {Description: "Internal server error"},
				"502": {Description: "Bad gateway - upstream error"},
				"504": {Description: "Gateway timeout"},
			},
			Security: security,
		}

		// Add request body for methods that typically have one
		if method == "POST" || method == "PUT" || method == "PATCH" {
			op.RequestBody = &RequestBody{
				Description: "Request body to forward to upstream",
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{Type: "object"},
					},
				},
			}
		}

		// Assign to correct method in PathItem
		switch method {
		case "GET":
			pathItem.Get = op
		case "POST":
			pathItem.Post = op
		case "PUT":
			pathItem.Put = op
		case "PATCH":
			pathItem.Patch = op
		case "DELETE":
			pathItem.Delete = op
		}
	}

	spec.Paths[openAPIPath] = pathItem
}

// convertPathPattern converts a route path pattern to OpenAPI path format.
// Returns the OpenAPI path and a list of path parameter names.
func convertPathPattern(pattern string, matchType route.MatchType) (string, []string) {
	var params []string

	switch matchType {
	case route.MatchExact:
		// Exact match - extract any {param} style parameters
		params = extractBraceParams(pattern)
		return pattern, params

	case route.MatchPrefix:
		// Prefix match - /api/* becomes /api/{path}
		if strings.HasSuffix(pattern, "/*") {
			path := strings.TrimSuffix(pattern, "/*")
			return path + "/{path}", []string{"path"}
		}
		if strings.HasSuffix(pattern, "*") {
			path := strings.TrimSuffix(pattern, "*")
			return path + "{path}", []string{"path"}
		}
		// Extract existing params
		params = extractBraceParams(pattern)
		return pattern, params

	case route.MatchRegex:
		// Regex match - convert common patterns
		openAPIPath := convertRegexToOpenAPI(pattern)
		params = extractBraceParams(openAPIPath)
		return openAPIPath, params

	default:
		params = extractBraceParams(pattern)
		return pattern, params
	}
}

// extractBraceParams extracts parameter names from {param} syntax.
func extractBraceParams(path string) []string {
	var params []string
	re := regexp.MustCompile(`\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(path, -1)
	for _, match := range matches {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}
	return params
}

// convertRegexToOpenAPI converts common regex patterns to OpenAPI path params.
func convertRegexToOpenAPI(pattern string) string {
	result := pattern

	// Common patterns:
	// [0-9]+ -> {id}
	// [a-zA-Z0-9_-]+ -> {slug}
	// [a-f0-9-]{36} -> {uuid}
	// .+ -> {path}

	// Replace numeric patterns with {id}
	numericRe := regexp.MustCompile(`\[0-9\]\+`)
	if numericRe.MatchString(result) {
		result = numericRe.ReplaceAllString(result, "{id}")
	}

	// Replace UUID-like patterns with {uuid}
	uuidRe := regexp.MustCompile(`\[a-f0-9-\]\{36\}`)
	if uuidRe.MatchString(result) {
		result = uuidRe.ReplaceAllString(result, "{uuid}")
	}

	// Replace alphanumeric slug patterns with {slug}
	slugRe := regexp.MustCompile(`\[a-zA-Z0-9_-\]\+`)
	if slugRe.MatchString(result) {
		result = slugRe.ReplaceAllString(result, "{slug}")
	}

	// Replace generic .+ patterns with {path}
	genericRe := regexp.MustCompile(`\.\+`)
	if genericRe.MatchString(result) {
		result = genericRe.ReplaceAllString(result, "{path}")
	}

	// Clean up any remaining regex anchors
	result = strings.TrimPrefix(result, "^")
	result = strings.TrimSuffix(result, "$")

	return result
}

// sanitizeSchemeName converts a name to a valid OpenAPI security scheme name.
func sanitizeSchemeName(name string) string {
	// Replace spaces and special chars with underscores
	result := strings.ToLower(name)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	result = re.ReplaceAllString(result, "_")
	result = strings.Trim(result, "_")
	if result == "" {
		return "upstream"
	}
	return result
}

// generateOperationID creates a unique operation ID from route name and method.
func generateOperationID(routeName, method string) string {
	// Clean route name
	name := strings.ToLower(routeName)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	name = re.ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")

	if name == "" {
		name = "route"
	}

	return strings.ToLower(method) + "_" + name
}
