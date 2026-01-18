// Package web provides the developer documentation portal.
package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/artpar/apigate/core/openapi"
	"github.com/artpar/apigate/domain/settings"
	"github.com/artpar/apigate/ports"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// DocsHandler provides the developer documentation portal endpoints.
type DocsHandler struct {
	openAPIService *openapi.Service
	settings       ports.SettingsStore
	logger         zerolog.Logger
	appName        string
}

// DocsDeps contains dependencies for the docs handler.
type DocsDeps struct {
	OpenAPIService *openapi.Service
	Settings       ports.SettingsStore
	Logger         zerolog.Logger
	AppName        string
}

// NewDocsHandler creates a new documentation handler.
func NewDocsHandler(deps DocsDeps) *DocsHandler {
	appName := deps.AppName
	if appName == "" {
		appName = "APIGate"
	}

	return &DocsHandler{
		openAPIService: deps.OpenAPIService,
		settings:       deps.Settings,
		logger:         deps.Logger,
		appName:        appName,
	}
}

// Router returns the docs portal router.
func (h *DocsHandler) Router() chi.Router {
	r := chi.NewRouter()

	// Documentation pages (public)
	r.Get("/", h.DocsHome)
	r.Get("/quickstart", h.QuickstartPage)
	r.Get("/authentication", h.AuthenticationPage)
	r.Get("/api-reference", h.APIReferencePage)
	r.Get("/examples", h.ExamplesPage)
	r.Get("/try-it", h.TryItPage)

	// API endpoints for docs
	r.Get("/openapi.json", h.OpenAPISpec)
	r.Get("/openapi.yaml", h.OpenAPISpecYAML)

	return r
}

// DocsHome renders the documentation homepage.
func (h *DocsHandler) DocsHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderDocsHome()))
}

// QuickstartPage renders the quickstart guide.
func (h *DocsHandler) QuickstartPage(w http.ResponseWriter, r *http.Request) {
	baseURL := h.getBaseURL(r)
	spec := h.generateOpenAPISpec(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderQuickstart(baseURL, spec)))
}

// AuthenticationPage renders the authentication documentation.
func (h *DocsHandler) AuthenticationPage(w http.ResponseWriter, r *http.Request) {
	baseURL := h.getBaseURL(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderAuthentication(baseURL)))
}

// APIReferencePage renders the API reference from OpenAPI spec.
func (h *DocsHandler) APIReferencePage(w http.ResponseWriter, r *http.Request) {
	spec := h.generateOpenAPISpec(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderAPIReference(spec)))
}

// ExamplesPage renders code examples in multiple languages.
func (h *DocsHandler) ExamplesPage(w http.ResponseWriter, r *http.Request) {
	baseURL := h.getBaseURL(r)
	spec := h.generateOpenAPISpec(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderExamples(baseURL, spec)))
}

// TryItPage renders the interactive API console.
func (h *DocsHandler) TryItPage(w http.ResponseWriter, r *http.Request) {
	baseURL := h.getBaseURL(r)
	spec := h.generateOpenAPISpec(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderTryIt(baseURL, spec)))
}

// OpenAPISpec returns the OpenAPI JSON specification.
func (h *DocsHandler) OpenAPISpec(w http.ResponseWriter, r *http.Request) {
	spec := h.generateOpenAPISpec(r)

	w.Header().Set("Content-Type", "application/json")
	data, _ := spec.ToJSON()
	w.Write(data)
}

// OpenAPISpecYAML returns the OpenAPI YAML specification.
func (h *DocsHandler) OpenAPISpecYAML(w http.ResponseWriter, r *http.Request) {
	spec := h.generateOpenAPISpec(r)

	w.Header().Set("Content-Type", "application/x-yaml")
	// Convert to YAML (simplified - just return JSON for now)
	data, _ := spec.ToJSON()
	w.Write(data)
}

func (h *DocsHandler) getBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func (h *DocsHandler) generateOpenAPISpec(r *http.Request) *openapi.Spec {
	baseURL := h.getBaseURL(r)
	if h.openAPIService != nil {
		// Use customer-only spec for public documentation (excludes admin endpoints)
		return h.openAPIService.GetCustomerSpec(r.Context(), baseURL)
	}
	// Fallback to empty spec if service not configured
	return &openapi.Spec{
		OpenAPI: "3.0.3",
		Info: openapi.Info{
			Title:       h.appName + " API",
			Description: "API documentation for " + h.appName,
			Version:     "1.0.0",
		},
		Servers: []openapi.Server{{URL: baseURL, Description: "Current server"}},
		Paths:   make(map[string]openapi.PathItem),
	}
}

// =============================================================================
// Customization Helpers
// =============================================================================

// getCustomSetting returns a custom setting value or empty string if not set.
func (h *DocsHandler) getCustomSetting(key string) string {
	if h.settings == nil {
		return ""
	}
	all, err := h.settings.GetAll(context.Background())
	if err != nil {
		return ""
	}
	return all.Get(key)
}

// getCustomCSS returns custom CSS if configured, wrapped in a style tag.
func (h *DocsHandler) getCustomCSS() string {
	customCSS := h.getCustomSetting(settings.KeyCustomDocsCSS)
	if customCSS == "" {
		return ""
	}
	return fmt.Sprintf("<style>%s</style>", customCSS)
}

// getCustomFooter returns custom footer HTML if configured.
func (h *DocsHandler) getCustomFooter() string {
	return h.getCustomSetting(settings.KeyCustomFooterHTML)
}

// getPrimaryColor returns the custom primary color or default.
func (h *DocsHandler) getPrimaryColor() string {
	color := h.getCustomSetting(settings.KeyCustomPrimaryColor)
	if color == "" {
		return "#111"
	}
	return color
}

// getLogoURL returns custom logo URL if configured.
func (h *DocsHandler) getLogoURL() string {
	return h.getCustomSetting(settings.KeyCustomLogoURL)
}

// =============================================================================
// Template Rendering
// =============================================================================

func (h *DocsHandler) renderDocsHome() string {
	// Check for full custom HTML override
	customHTML := h.getCustomSetting(settings.KeyCustomDocsHomeHTML)
	if customHTML != "" {
		// Replace template variables in custom HTML
		customHTML = strings.ReplaceAll(customHTML, "{{APP_NAME}}", h.appName)
		customHTML = strings.ReplaceAll(customHTML, "{{NAV}}", h.renderDocsNav("home"))
		customHTML = strings.ReplaceAll(customHTML, "{{CUSTOM_CSS}}", h.getCustomCSS())
		customHTML = strings.ReplaceAll(customHTML, "{{FOOTER}}", h.getCustomFooter())
		customHTML = strings.ReplaceAll(customHTML, "{{PRIMARY_COLOR}}", h.getPrimaryColor())
		customHTML = strings.ReplaceAll(customHTML, "{{LOGO_URL}}", h.getLogoURL())
		return customHTML
	}

	// Use custom hero title/subtitle if configured
	heroTitle := h.getCustomSetting(settings.KeyCustomDocsHeroTitle)
	if heroTitle == "" {
		heroTitle = "API Documentation"
	}
	heroSubtitle := h.getCustomSetting(settings.KeyCustomDocsHeroSubtitle)
	if heroSubtitle == "" {
		heroSubtitle = "Everything you need to integrate with the API"
	}

	// Get custom footer
	footer := h.getCustomFooter()
	if footer != "" {
		footer = fmt.Sprintf("<footer class=\"docs-footer\">%s</footer>", footer)
	}

	// Build custom CSS with primary color override
	primaryColor := h.getPrimaryColor()
	colorCSS := ""
	if primaryColor != "#111" {
		colorCSS = fmt.Sprintf(`
<style>
:root { --primary-color: %s; }
.docs-header { background: %s; }
.btn { background: %s; }
.btn:hover { background: %s; filter: brightness(1.2); }
.docs-nav a.active { color: white; }
</style>`, primaryColor, primaryColor, primaryColor, primaryColor)
	}

	// Get logo
	logoHTML := h.appName
	logoURL := h.getLogoURL()
	if logoURL != "" {
		logoHTML = fmt.Sprintf(`<img src="%s" alt="%s" style="height: 24px;">`, logoURL, h.appName)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Documentation - %s</title>
    <style>%s</style>
    %s
    %s
</head>
<body>
    %s
    <main class="docs-content">
        <div class="docs-hero">
            <h1>%s</h1>
            <p>%s</p>
        </div>

        <div class="docs-cards">
            <a href="/docs/quickstart" class="docs-card">
                <h3>Quickstart</h3>
                <p>Get started in a few minutes.</p>
            </a>

            <a href="/docs/authentication" class="docs-card">
                <h3>Authentication</h3>
                <p>How to authenticate your requests.</p>
            </a>

            <a href="/docs/api-reference" class="docs-card">
                <h3>API Reference</h3>
                <p>Available endpoints and response codes.</p>
            </a>

            <a href="/docs/examples" class="docs-card">
                <h3>Code Examples</h3>
                <p>cURL, JavaScript, Python, and Go.</p>
            </a>

            <a href="/docs/try-it" class="docs-card">
                <h3>Try It</h3>
                <p>Test the API in your browser.</p>
            </a>

            <a href="/docs/openapi.json" class="docs-card" target="_blank">
                <h3>OpenAPI Spec</h3>
                <p>Download the OpenAPI 3.0 spec.</p>
            </a>
        </div>
    </main>
    %s
</body>
</html>`, h.appName, docsCSS, colorCSS, h.getCustomCSS(),
		h.renderDocsNavWithLogo("home", logoHTML),
		heroTitle, heroSubtitle,
		footer)
}

func (h *DocsHandler) renderQuickstart(baseURL string, spec *openapi.Spec) string {
	// Find a real documented endpoint to use as an example
	exampleEndpoint := "/your-endpoint"
	exampleMethod := "GET"
	exampleDescription := ""

	// Look for a concrete endpoint (not a wildcard) to use as an example
	for path, pathItem := range spec.Paths {
		// Skip wildcard/proxy routes
		if strings.Contains(path, "{path}") || strings.Contains(path, "*") {
			continue
		}

		// Prefer GET endpoints for examples
		if pathItem.Get != nil {
			exampleEndpoint = path
			exampleMethod = "GET"
			if pathItem.Get.Summary != "" {
				exampleDescription = pathItem.Get.Summary
			} else if pathItem.Get.Description != "" {
				exampleDescription = pathItem.Get.Description
			}
			break
		}
		// Fall back to other methods
		if pathItem.Post != nil {
			exampleEndpoint = path
			exampleMethod = "POST"
			if pathItem.Post.Summary != "" {
				exampleDescription = pathItem.Post.Summary
			}
		}
	}

	// Build endpoint info section
	endpointInfo := ""
	if exampleEndpoint != "/your-endpoint" {
		endpointInfo = fmt.Sprintf(`
            <div class="docs-callout info">
                <strong>Example Endpoint:</strong> <code>%s %s</code>
                <p style="margin-top: 4px; margin-bottom: 0;">%s</p>
                <p style="margin-top: 8px; margin-bottom: 0;"><a href="/docs/api-reference">See all available endpoints →</a></p>
            </div>`, exampleMethod, exampleEndpoint, exampleDescription)
	} else {
		endpointInfo = `
            <div class="docs-callout warning">
                <strong>No endpoints documented yet.</strong>
                <p style="margin-top: 4px; margin-bottom: 0;">Check the <a href="/docs/api-reference">API Reference</a> for available endpoints, or contact the API provider for documentation.</p>
            </div>`
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Quickstart - %s API</title>
    <style>%s</style>
</head>
<body>
    %s
    <main class="docs-content">
        <nav class="docs-breadcrumb">
            <a href="/docs">Documentation</a> / <span>Quickstart</span>
        </nav>

        <h1>Quickstart Guide</h1>
        <p class="docs-lead">Get started with the %s API in just a few steps.</p>

        <div class="docs-section">
            <h2>Step 1: Get Your API Key</h2>
            <p>Sign up for an account and get your API key from the <a href="/portal">customer portal</a>.</p>
            <ol>
                <li>Create an account at <a href="/portal/signup">/portal/signup</a></li>
                <li>Navigate to <strong>API Keys</strong> in your dashboard</li>
                <li>Click <strong>Create New Key</strong></li>
                <li>Copy your API key (it will only be shown once)</li>
            </ol>
        </div>

        <div class="docs-section">
            <h2>Step 2: Make Your First Request</h2>
            <p>Use your API key to authenticate requests. Include it in the <code>X-API-Key</code> header:</p>

            %s

            <div class="code-tabs">
                <button class="code-tab active" data-lang="curl">cURL</button>
                <button class="code-tab" data-lang="javascript">JavaScript</button>
                <button class="code-tab" data-lang="python">Python</button>
            </div>

            <pre class="code-block" data-lang="curl"><code>curl -X %s "%s%s" \
  -H "X-API-Key: your_api_key_here" \
  -H "Content-Type: application/json"</code></pre>

            <pre class="code-block hidden" data-lang="javascript"><code>const response = await fetch('%s%s', {
  method: '%s',
  headers: {
    'X-API-Key': 'your_api_key_here',
    'Content-Type': 'application/json'
  }
});

const data = await response.json();
console.log(data);</code></pre>

            <pre class="code-block hidden" data-lang="python"><code>import requests

response = requests.%s(
    '%s%s',
    headers={
        'X-API-Key': 'your_api_key_here',
        'Content-Type': 'application/json'
    }
)

print(response.json())</code></pre>
        </div>

        <div class="docs-section">
            <h2>Step 3: Handle the Response</h2>
            <p>The API returns JSON responses. A successful response looks like:</p>
            <pre class="code-block"><code>{
  "data": {
    // Your response data here
  }
}</code></pre>

            <p>Error responses include an error object:</p>
            <pre class="code-block"><code>{
  "error": {
    "code": "invalid_api_key",
    "message": "The provided API key is invalid or expired"
  }
}</code></pre>
        </div>

        <div class="docs-section">
            <h2>Next Steps</h2>
            <ul>
                <li><a href="/docs/authentication">Learn more about authentication</a></li>
                <li><a href="/docs/api-reference">Explore the full API reference</a></li>
                <li><a href="/docs/examples">See more code examples</a></li>
                <li><a href="/docs/try-it">Try the API in your browser</a></li>
            </ul>
        </div>
    </main>
    <script>%s</script>
</body>
</html>`, h.appName, docsCSS, h.renderDocsNav("quickstart"), h.appName, endpointInfo,
		exampleMethod, baseURL, exampleEndpoint,
		baseURL, exampleEndpoint, exampleMethod,
		strings.ToLower(exampleMethod), baseURL, exampleEndpoint,
		docsJS)
}

func (h *DocsHandler) renderAuthentication(baseURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Authentication - %s API</title>
    <style>%s</style>
</head>
<body>
    %s
    <main class="docs-content">
        <nav class="docs-breadcrumb">
            <a href="/docs">Documentation</a> / <span>Authentication</span>
        </nav>

        <h1>Authentication</h1>
        <p class="docs-lead">Learn how to authenticate your API requests.</p>

        <div class="docs-section">
            <h2>API Key Authentication</h2>
            <p>All API requests require authentication using an API key. Include your key in the <code>X-API-Key</code> header:</p>
            <pre class="code-block"><code>X-API-Key: ak_your_api_key_here</code></pre>

            <div class="docs-callout info">
                <strong>Security Note:</strong> Keep your API key secret. Never expose it in client-side code or public repositories.
            </div>
        </div>

        <div class="docs-section">
            <h2>Getting an API Key</h2>
            <ol>
                <li>Sign up at <a href="/portal/signup">%s/portal/signup</a></li>
                <li>Log in to your <a href="/portal">customer portal</a></li>
                <li>Navigate to <strong>API Keys</strong></li>
                <li>Click <strong>Create New Key</strong></li>
                <li>Copy and securely store your key</li>
            </ol>

            <div class="docs-callout warning">
                <strong>Important:</strong> Your API key is only shown once at creation. If you lose it, you'll need to create a new one.
            </div>
        </div>

        <div class="docs-section">
            <h2>Rate Limits</h2>
            <p>API requests are rate-limited based on your subscription plan:</p>
            <table class="docs-table">
                <thead>
                    <tr>
                        <th>Plan</th>
                        <th>Rate Limit</th>
                        <th>Monthly Quota</th>
                    </tr>
                </thead>
                <tbody>
                    <tr>
                        <td>Free</td>
                        <td>60 requests/min</td>
                        <td>1,000 requests</td>
                    </tr>
                    <tr>
                        <td>Pro</td>
                        <td>300 requests/min</td>
                        <td>50,000 requests</td>
                    </tr>
                    <tr>
                        <td>Enterprise</td>
                        <td>1,000 requests/min</td>
                        <td>Unlimited</td>
                    </tr>
                </tbody>
            </table>

            <p>Rate limit information is included in response headers:</p>
            <pre class="code-block"><code>X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1704067200</code></pre>
        </div>

        <div class="docs-section">
            <h2>Error Responses</h2>
            <p>Authentication errors return appropriate HTTP status codes:</p>
            <table class="docs-table">
                <thead>
                    <tr>
                        <th>Status</th>
                        <th>Code</th>
                        <th>Description</th>
                    </tr>
                </thead>
                <tbody>
                    <tr>
                        <td>401</td>
                        <td>missing_api_key</td>
                        <td>No API key provided</td>
                    </tr>
                    <tr>
                        <td>401</td>
                        <td>invalid_api_key</td>
                        <td>API key is invalid or revoked</td>
                    </tr>
                    <tr>
                        <td>429</td>
                        <td>rate_limit_exceeded</td>
                        <td>Too many requests</td>
                    </tr>
                    <tr>
                        <td>403</td>
                        <td>quota_exceeded</td>
                        <td>Monthly quota exhausted</td>
                    </tr>
                </tbody>
            </table>
        </div>
    </main>
</body>
</html>`, h.appName, docsCSS, h.renderDocsNav("authentication"), baseURL)
}

func (h *DocsHandler) renderAPIReference(spec *openapi.Spec) string {
	baseURL := ""
	if len(spec.Servers) > 0 {
		baseURL = spec.Servers[0].URL
	}

	// Check for wildcard routes and collect concrete endpoints
	hasWildcardRoutes := false
	var wildcardPaths []string
	var endpoints []concreteEndpoint
	for path, pathItem := range spec.Paths {
		// Detect wildcard/proxy routes
		if strings.Contains(path, "{path}") || strings.Contains(path, "*") || strings.HasSuffix(path, "/*") {
			hasWildcardRoutes = true
			// Clean up path for display
			displayPath := strings.TrimSuffix(path, "/{path}")
			displayPath = strings.TrimSuffix(displayPath, "/*")
			if displayPath == "" {
				displayPath = "/*"
			} else {
				displayPath += "/*"
			}
			wildcardPaths = append(wildcardPaths, displayPath)
			continue
		}

		// Collect all operations with their details
		if pathItem.Get != nil {
			endpoints = append(endpoints, concreteEndpoint{
				path:        path,
				methods:     []string{"GET"},
				summary:     pathItem.Get.Summary,
				description: pathItem.Get.Description,
				operation:   pathItem.Get,
			})
		}
		if pathItem.Post != nil {
			endpoints = append(endpoints, concreteEndpoint{
				path:        path,
				methods:     []string{"POST"},
				summary:     pathItem.Post.Summary,
				description: pathItem.Post.Description,
				operation:   pathItem.Post,
			})
		}
		if pathItem.Put != nil {
			endpoints = append(endpoints, concreteEndpoint{
				path:        path,
				methods:     []string{"PUT"},
				summary:     pathItem.Put.Summary,
				description: pathItem.Put.Description,
				operation:   pathItem.Put,
			})
		}
		if pathItem.Patch != nil {
			endpoints = append(endpoints, concreteEndpoint{
				path:        path,
				methods:     []string{"PATCH"},
				summary:     pathItem.Patch.Summary,
				description: pathItem.Patch.Description,
				operation:   pathItem.Patch,
			})
		}
		if pathItem.Delete != nil {
			endpoints = append(endpoints, concreteEndpoint{
				path:        path,
				methods:     []string{"DELETE"},
				summary:     pathItem.Delete.Summary,
				description: pathItem.Delete.Description,
				operation:   pathItem.Delete,
			})
		}
	}

	// Sort by path then method
	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].path != endpoints[j].path {
			return endpoints[i].path < endpoints[j].path
		}
		return endpoints[i].methods[0] < endpoints[j].methods[0]
	})

	// Build endpoint list with detailed documentation
	endpointsHTML := ""
	if len(endpoints) == 0 {
		endpointsHTML = `
		<div class="docs-callout info">
			<strong>No endpoints documented yet.</strong>
			<p style="margin-top: 8px;">The API administrator needs to create specific routes with documentation.</p>
			<p>If you're the admin, go to <strong>Routes</strong> in the admin panel and add endpoint descriptions, example requests, and example responses to your routes.</p>
		</div>`
	} else {
		for _, ep := range endpoints {
			methodBadge := fmt.Sprintf(`<span class="method-badge method-%s">%s</span>`, strings.ToLower(ep.methods[0]), ep.methods[0])

			descText := ""
			if ep.description != "" {
				descText = fmt.Sprintf(`<p class="endpoint-desc">%s</p>`, ep.description)
			}

			// Build example sections
			exampleHTML := ""
			if ep.operation != nil {
				// Request example
				if ep.operation.RequestBody != nil && ep.operation.RequestBody.Content != nil {
					if jsonMedia, ok := ep.operation.RequestBody.Content["application/json"]; ok && jsonMedia.Schema != nil && jsonMedia.Schema.Example != nil {
						exampleJSON := h.formatExample(jsonMedia.Schema.Example)
						exampleHTML += fmt.Sprintf(`
						<div class="example-section">
							<h5>Example Request</h5>
							<pre class="code-block"><code>%s</code></pre>
						</div>`, exampleJSON)
					}
				}

				// Response example
				if resp, ok := ep.operation.Responses["200"]; ok && resp.Content != nil {
					if jsonMedia, ok := resp.Content["application/json"]; ok && jsonMedia.Schema != nil && jsonMedia.Schema.Example != nil {
						exampleJSON := h.formatExample(jsonMedia.Schema.Example)
						exampleHTML += fmt.Sprintf(`
						<div class="example-section">
							<h5>Example Response</h5>
							<pre class="code-block"><code>%s</code></pre>
						</div>`, exampleJSON)
					}
				}
			}

			endpointsHTML += fmt.Sprintf(`
				<div class="endpoint-card">
					<div class="endpoint-header">
						%s
						<code class="endpoint-path">%s</code>
					</div>
					%s
					%s
				</div>`, methodBadge, ep.path, descText, exampleHTML)
		}
	}

	// Build wildcard warning banner if applicable
	wildcardBanner := ""
	if hasWildcardRoutes {
		sort.Strings(wildcardPaths)
		pathList := ""
		for _, p := range wildcardPaths {
			pathList += fmt.Sprintf("<code>%s</code> ", p)
		}
		wildcardBanner = fmt.Sprintf(`
        <div class="docs-callout warning" style="margin-bottom: 24px;">
            <strong>Additional endpoints may be available</strong>
            <p style="margin-top: 8px; margin-bottom: 8px;">This API includes wildcard routes (%s) that proxy requests to the upstream server. These routes may expose additional endpoints not documented here.</p>
            <p style="margin-bottom: 0;">Contact the API provider for complete documentation of available endpoints.</p>
        </div>`, pathList)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>API Reference - %s</title>
    <style>%s</style>
</head>
<body>
    %s
    <main class="docs-content">
        <nav class="docs-breadcrumb">
            <a href="/docs">Documentation</a> / <span>API Reference</span>
        </nav>

        <h1>API Reference</h1>
        <p class="docs-lead">All requests require authentication via the <code>X-API-Key</code> header.</p>

        %s

        <div class="docs-section">
            <h2>Base URL</h2>
            <pre class="code-block"><code>%s</code></pre>
        </div>

        <div class="docs-section">
            <h2>Authentication</h2>
            <p>Include your API key in the <code>X-API-Key</code> header:</p>
            <pre class="code-block"><code>curl "%s/endpoint" \
  -H "X-API-Key: your_api_key"</code></pre>
        </div>

        <div class="docs-section">
            <h2>Endpoints</h2>
            %s
        </div>

        <div class="docs-section">
            <h2>Response Codes</h2>
            <table class="docs-table">
                <thead><tr><th>Code</th><th>Meaning</th></tr></thead>
                <tbody>
                    <tr><td><code>200</code></td><td>Success</td></tr>
                    <tr><td><code>400</code></td><td>Bad request</td></tr>
                    <tr><td><code>401</code></td><td>Invalid or missing API key</td></tr>
                    <tr><td><code>403</code></td><td>Quota exceeded</td></tr>
                    <tr><td><code>429</code></td><td>Rate limit exceeded</td></tr>
                    <tr><td><code>502</code></td><td>Upstream error</td></tr>
                </tbody>
            </table>
        </div>
    </main>
</body>
</html>`, h.appName, docsCSS, h.renderDocsNav("api-reference"), wildcardBanner, baseURL, baseURL, endpointsHTML)
}

// concreteEndpoint represents a specific endpoint (not a wildcard)
type concreteEndpoint struct {
	path        string
	methods     []string
	summary     string
	description string
	operation   *openapi.Operation
}

func (h *DocsHandler) renderEndpoint(method, path string, op *openapi.Operation) string {
	_ = path // unused but kept for potential future use
	methodClass := strings.ToLower(method)

	// Build parameters table
	paramsHTML := ""
	if len(op.Parameters) > 0 {
		paramsHTML = `<h4>Parameters</h4><table class="docs-table"><thead><tr><th>Name</th><th>In</th><th>Type</th><th>Required</th><th>Description</th></tr></thead><tbody>`
		for _, p := range op.Parameters {
			required := ""
			if p.Required {
				required = "Yes"
			}
			schemaType := ""
			if p.Schema != nil {
				schemaType = p.Schema.Type
			}
			paramsHTML += fmt.Sprintf(`<tr><td><code>%s</code></td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>`,
				p.Name, p.In, schemaType, required, p.Description)
		}
		paramsHTML += `</tbody></table>`
	}

	// Build request body
	bodyHTML := ""
	if op.RequestBody != nil && op.RequestBody.Content != nil {
		if jsonMedia, ok := op.RequestBody.Content["application/json"]; ok && jsonMedia.Schema != nil {
			bodyHTML = `<h4>Request Body</h4><p>Content-Type: <code>application/json</code></p>`
			if jsonMedia.Schema.Ref != "" {
				bodyHTML += fmt.Sprintf(`<p>Schema: <code>%s</code></p>`, jsonMedia.Schema.Ref)
			}
		}
	}

	tags := ""
	if len(op.Tags) > 0 {
		tags = fmt.Sprintf(`<span class="endpoint-tag">%s</span>`, op.Tags[0])
	}

	return fmt.Sprintf(`
        <div class="endpoint">
            <div class="endpoint-header">
                <span class="method method-%s">%s</span>
                <code class="endpoint-path">%s</code>
                %s
            </div>
            <p class="endpoint-summary">%s</p>
            <div class="endpoint-details">
                %s
                %s
            </div>
        </div>`, methodClass, method, path, tags, op.Summary, paramsHTML, bodyHTML)
}

// formatExample formats an example value as pretty-printed JSON
func (h *DocsHandler) formatExample(v any) string {
	if v == nil {
		return ""
	}
	// If it's already a string, check if it's valid JSON to pretty print
	if str, ok := v.(string); ok {
		var parsed any
		if err := json.Unmarshal([]byte(str), &parsed); err == nil {
			if b, err := json.MarshalIndent(parsed, "", "  "); err == nil {
				return string(b)
			}
		}
		return str
	}
	// Otherwise, marshal and pretty-print
	if b, err := json.MarshalIndent(v, "", "  "); err == nil {
		return string(b)
	}
	return fmt.Sprintf("%v", v)
}

func (h *DocsHandler) renderExamples(baseURL string, spec *openapi.Spec) string {
	// Find actual GET and POST endpoints from the spec
	getEndpoint := "/your-endpoint"
	getDescription := ""
	postEndpoint := "/your-endpoint"
	postDescription := ""
	postExampleBody := `{
    "name": "Example",
    "value": 123
  }`

	// Collect concrete endpoints (skip wildcards)
	for path, pathItem := range spec.Paths {
		if strings.Contains(path, "{path}") || strings.Contains(path, "*") {
			continue
		}

		// Find first GET endpoint
		if getEndpoint == "/your-endpoint" && pathItem.Get != nil {
			getEndpoint = path
			if pathItem.Get.Description != "" {
				getDescription = pathItem.Get.Description
			} else if pathItem.Get.Summary != "" {
				getDescription = pathItem.Get.Summary
			}
		}

		// Find first POST endpoint with example body
		if postEndpoint == "/your-endpoint" && pathItem.Post != nil {
			postEndpoint = path
			if pathItem.Post.Description != "" {
				postDescription = pathItem.Post.Description
			} else if pathItem.Post.Summary != "" {
				postDescription = pathItem.Post.Summary
			}
			// Try to get example request body
			if pathItem.Post.RequestBody != nil && pathItem.Post.RequestBody.Content != nil {
				if jsonMedia, ok := pathItem.Post.RequestBody.Content["application/json"]; ok && jsonMedia.Schema != nil && jsonMedia.Schema.Example != nil {
					postExampleBody = h.formatExample(jsonMedia.Schema.Example)
				}
			}
		}

		// Stop if we found both
		if getEndpoint != "/your-endpoint" && postEndpoint != "/your-endpoint" {
			break
		}
	}

	// Build endpoint info callouts
	getCallout := ""
	if getEndpoint != "/your-endpoint" && getDescription != "" {
		getCallout = fmt.Sprintf(`<div class="docs-callout info" style="margin-bottom: 16px;"><strong>%s</strong><p style="margin: 4px 0 0 0;">%s</p></div>`, getEndpoint, getDescription)
	}

	postCallout := ""
	if postEndpoint != "/your-endpoint" && postDescription != "" {
		postCallout = fmt.Sprintf(`<div class="docs-callout info" style="margin-bottom: 16px;"><strong>%s</strong><p style="margin: 4px 0 0 0;">%s</p></div>`, postEndpoint, postDescription)
	}

	// If no endpoints found, show a warning
	noEndpointsWarning := ""
	if getEndpoint == "/your-endpoint" && postEndpoint == "/your-endpoint" {
		noEndpointsWarning = `
        <div class="docs-callout warning" style="margin-bottom: 24px;">
            <strong>No documented endpoints yet</strong>
            <p style="margin-top: 8px;">The API administrator needs to create routes with documentation. Check the <a href="/docs/api-reference">API Reference</a> for available endpoints.</p>
        </div>`
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Code Examples - %s API</title>
    <style>%s</style>
</head>
<body>
    %s
    <main class="docs-content">
        <nav class="docs-breadcrumb">
            <a href="/docs">Documentation</a> / <span>Code Examples</span>
        </nav>

        <h1>Code Examples</h1>
        <p class="docs-lead">Ready-to-use code snippets for your API endpoints.</p>

        %s

        <div class="docs-section">
            <h2>Making a GET Request</h2>
            %s
            <div class="code-tabs">
                <button class="code-tab active" data-lang="curl">cURL</button>
                <button class="code-tab" data-lang="javascript">JavaScript</button>
                <button class="code-tab" data-lang="python">Python</button>
                <button class="code-tab" data-lang="go">Go</button>
            </div>

            <pre class="code-block" data-lang="curl"><code>curl -X GET "%s%s" \
  -H "X-API-Key: your_api_key" \
  -H "Content-Type: application/json"</code></pre>

            <pre class="code-block hidden" data-lang="javascript"><code>const API_KEY = 'your_api_key';
const BASE_URL = '%s';

async function getData() {
  const response = await fetch(BASE_URL + '%s', {
    method: 'GET',
    headers: {
      'X-API-Key': API_KEY,
      'Content-Type': 'application/json'
    }
  });

  if (!response.ok) {
    throw new Error('HTTP error! status: ' + response.status);
  }

  return response.json();
}

// Usage
getData()
  .then(data => console.log(data))
  .catch(err => console.error(err));</code></pre>

            <pre class="code-block hidden" data-lang="python"><code>import requests

API_KEY = 'your_api_key'
BASE_URL = '%s'

def get_data():
    response = requests.get(
        f'{BASE_URL}%s',
        headers={
            'X-API-Key': API_KEY,
            'Content-Type': 'application/json'
        }
    )
    response.raise_for_status()
    return response.json()

# Usage
data = get_data()
print(data)</code></pre>

            <pre class="code-block hidden" data-lang="go"><code>package main

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

const (
    apiKey  = "your_api_key"
    baseURL = "%s"
)

func getData() (map[string]interface{}, error) {
    req, err := http.NewRequest("GET", baseURL+"%s", nil)
    if err != nil {
        return nil, err
    }

    req.Header.Set("X-API-Key", apiKey)
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)

    var result map[string]interface{}
    json.Unmarshal(body, &result)
    return result, nil
}

func main() {
    data, err := getData()
    if err != nil {
        fmt.Println("Error:", err)
        return
    }
    fmt.Println(data)
}</code></pre>
        </div>

        <div class="docs-section">
            <h2>Making a POST Request</h2>
            %s
            <div class="code-tabs">
                <button class="code-tab active" data-lang="curl2">cURL</button>
                <button class="code-tab" data-lang="javascript2">JavaScript</button>
                <button class="code-tab" data-lang="python2">Python</button>
            </div>

            <pre class="code-block" data-lang="curl2"><code>curl -X POST "%s%s" \
  -H "X-API-Key: your_api_key" \
  -H "Content-Type: application/json" \
  -d '%s'</code></pre>

            <pre class="code-block hidden" data-lang="javascript2"><code>async function createData(data) {
  const response = await fetch(BASE_URL + '%s', {
    method: 'POST',
    headers: {
      'X-API-Key': API_KEY,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(data)
  });

  return response.json();
}

// Usage
createData(%s)
  .then(data => console.log(data));</code></pre>

            <pre class="code-block hidden" data-lang="python2"><code>def create_data(data):
    response = requests.post(
        f'{BASE_URL}%s',
        headers={
            'X-API-Key': API_KEY,
            'Content-Type': 'application/json'
        },
        json=data
    )
    return response.json()

# Usage
result = create_data(%s)
print(result)</code></pre>
        </div>

        <div class="docs-section">
            <h2>Error Handling</h2>
            <pre class="code-block" data-lang="javascript"><code>async function safeApiCall(endpoint) {
  try {
    const response = await fetch(BASE_URL + endpoint, {
      headers: { 'X-API-Key': API_KEY }
    });

    const data = await response.json();

    if (!response.ok) {
      // Handle API error
      console.error('API Error:', data.error.code, data.error.message);
      return null;
    }

    return data;
  } catch (err) {
    // Handle network error
    console.error('Network Error:', err.message);
    return null;
  }
}</code></pre>
        </div>
    </main>
    <script>%s</script>
</body>
</html>`, h.appName, docsCSS, h.renderDocsNav("examples"), noEndpointsWarning,
		getCallout, baseURL, getEndpoint,
		baseURL, getEndpoint,
		baseURL, getEndpoint,
		baseURL, getEndpoint,
		postCallout, baseURL, postEndpoint, postExampleBody,
		postEndpoint, postExampleBody,
		postEndpoint, postExampleBody,
		docsJS)
}

func (h *DocsHandler) renderTryIt(baseURL string, spec *openapi.Spec) string {
	// Collect available endpoints from spec
	type endpointInfo struct {
		method      string
		path        string
		description string
		exampleBody string
	}
	var endpoints []endpointInfo
	availableMethods := make(map[string]bool)

	for path, pathItem := range spec.Paths {
		// Skip wildcards
		if strings.Contains(path, "{path}") || strings.Contains(path, "*") {
			continue
		}

		if pathItem.Get != nil {
			desc := pathItem.Get.Summary
			if desc == "" {
				desc = pathItem.Get.Description
			}
			endpoints = append(endpoints, endpointInfo{method: "GET", path: path, description: desc})
			availableMethods["GET"] = true
		}
		if pathItem.Post != nil {
			desc := pathItem.Post.Summary
			if desc == "" {
				desc = pathItem.Post.Description
			}
			exBody := ""
			if pathItem.Post.RequestBody != nil && pathItem.Post.RequestBody.Content != nil {
				if jsonMedia, ok := pathItem.Post.RequestBody.Content["application/json"]; ok && jsonMedia.Schema != nil && jsonMedia.Schema.Example != nil {
					exBody = h.formatExample(jsonMedia.Schema.Example)
				}
			}
			endpoints = append(endpoints, endpointInfo{method: "POST", path: path, description: desc, exampleBody: exBody})
			availableMethods["POST"] = true
		}
		if pathItem.Put != nil {
			desc := pathItem.Put.Summary
			if desc == "" {
				desc = pathItem.Put.Description
			}
			exBody := ""
			if pathItem.Put.RequestBody != nil && pathItem.Put.RequestBody.Content != nil {
				if jsonMedia, ok := pathItem.Put.RequestBody.Content["application/json"]; ok && jsonMedia.Schema != nil && jsonMedia.Schema.Example != nil {
					exBody = h.formatExample(jsonMedia.Schema.Example)
				}
			}
			endpoints = append(endpoints, endpointInfo{method: "PUT", path: path, description: desc, exampleBody: exBody})
			availableMethods["PUT"] = true
		}
		if pathItem.Patch != nil {
			desc := pathItem.Patch.Summary
			if desc == "" {
				desc = pathItem.Patch.Description
			}
			exBody := ""
			if pathItem.Patch.RequestBody != nil && pathItem.Patch.RequestBody.Content != nil {
				if jsonMedia, ok := pathItem.Patch.RequestBody.Content["application/json"]; ok && jsonMedia.Schema != nil && jsonMedia.Schema.Example != nil {
					exBody = h.formatExample(jsonMedia.Schema.Example)
				}
			}
			endpoints = append(endpoints, endpointInfo{method: "PATCH", path: path, description: desc, exampleBody: exBody})
			availableMethods["PATCH"] = true
		}
		if pathItem.Delete != nil {
			desc := pathItem.Delete.Summary
			if desc == "" {
				desc = pathItem.Delete.Description
			}
			endpoints = append(endpoints, endpointInfo{method: "DELETE", path: path, description: desc})
			availableMethods["DELETE"] = true
		}
	}

	// Sort endpoints by path, then method
	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].path != endpoints[j].path {
			return endpoints[i].path < endpoints[j].path
		}
		return endpoints[i].method < endpoints[j].method
	})

	// Build endpoint buttons HTML
	endpointButtonsHTML := ""
	firstEndpointIdx := 0
	if len(endpoints) > 0 {
		endpointButtonsHTML = `<div class="form-group"><label>Select an Endpoint <span class="label-hint">(click to populate)</span></label><div class="endpoint-buttons">`
		for idx, ep := range endpoints {
			methodClass := strings.ToLower(ep.method)
			title := ep.description
			if title == "" {
				title = ep.method + " " + ep.path
			}
			exampleAttr := ""
			hasExample := ""
			if ep.exampleBody != "" {
				// Escape for HTML attribute
				escaped := strings.ReplaceAll(ep.exampleBody, `"`, `&quot;`)
				escaped = strings.ReplaceAll(escaped, "\n", "&#10;")
				exampleAttr = fmt.Sprintf(` data-example="%s"`, escaped)
				hasExample = " has-example"
			}
			// Escape description for data attribute
			escapedDesc := strings.ReplaceAll(ep.description, `"`, `&quot;`)
			activeClass := ""
			if idx == firstEndpointIdx {
				activeClass = " active"
			}
			endpointButtonsHTML += fmt.Sprintf(`<button type="button" class="endpoint-btn method-%s%s%s" data-method="%s" data-path="%s" data-desc="%s" title="%s"%s><span class="method-badge method-%s">%s</span> %s</button>`,
				methodClass, hasExample, activeClass, ep.method, ep.path, escapedDesc, title, exampleAttr, methodClass, ep.method, ep.path)
		}
		endpointButtonsHTML += `</div></div>`
	} else {
		endpointButtonsHTML = `<div class="docs-callout warning" style="margin-bottom: 16px;"><strong>No documented endpoints available</strong><p style="margin-top: 8px;">The API administrator needs to configure routes with documentation before you can test them here.</p><p style="margin-top: 8px;"><a href="/docs/api-reference">View API Reference</a> | <a href="/portal">Get API Key</a></p></div>`
	}

	// Get default endpoint and example body
	defaultEndpoint := ""
	defaultMethod := "GET"
	defaultExampleBody := ""
	defaultDescription := ""
	if len(endpoints) > 0 {
		defaultEndpoint = endpoints[0].path
		defaultMethod = endpoints[0].method
		defaultExampleBody = endpoints[0].exampleBody
		defaultDescription = endpoints[0].description
	}

	// Build method options - only show methods that have endpoints
	methodOptionsHTML := ""
	allMethods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
	for _, m := range allMethods {
		if availableMethods[m] || len(endpoints) == 0 {
			selected := ""
			if m == defaultMethod {
				selected = " selected"
			}
			methodOptionsHTML += fmt.Sprintf(`<option value="%s"%s>%s</option>`, m, selected, m)
		}
	}

	// Escape default example body for textarea
	escapedExampleBody := defaultExampleBody

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Try It - %s API</title>
    <style>%s
.endpoint-buttons { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 8px; }
.endpoint-btn { display: inline-flex; align-items: center; gap: 6px; padding: 8px 12px; border: 1px solid #e5e5e5; border-radius: 6px; background: #fff; cursor: pointer; font-size: 13px; font-family: ui-monospace, monospace; transition: all 0.15s ease; }
.endpoint-btn:hover { border-color: #111; background: #f9f9f9; transform: translateY(-1px); }
.endpoint-btn.active { border-color: #111; background: #111; color: #fff; }
.endpoint-btn.active .method-badge { background: rgba(255,255,255,0.2); color: #fff; }
.endpoint-btn.has-example::after { content: ''; width: 6px; height: 6px; background: #10b981; border-radius: 50%%; margin-left: 4px; }
.label-hint { font-weight: 400; color: #888; font-size: 12px; }
.endpoint-desc { font-size: 13px; color: #666; padding: 8px 12px; background: #f9f9f9; border-radius: 4px; margin-bottom: 16px; min-height: 20px; }
.endpoint-desc:empty::before { content: 'Select an endpoint above to see its description'; color: #999; font-style: italic; }
.validation-error { color: #dc2626; font-size: 12px; margin-top: 4px; display: none; }
.validation-error.show { display: block; }
.form-input.error { border-color: #dc2626; }
.btn-row { display: flex; gap: 8px; flex-wrap: wrap; }
.btn { transition: all 0.15s ease; }
.btn:disabled { opacity: 0.6; cursor: not-allowed; }
.btn-secondary { background: #f3f4f6; color: #374151; border: 1px solid #e5e5e5; }
.btn-secondary:hover:not(:disabled) { background: #e5e7eb; border-color: #d1d5db; }
.btn-icon { display: inline-flex; align-items: center; gap: 6px; }
.spinner { width: 14px; height: 14px; border: 2px solid transparent; border-top-color: currentColor; border-radius: 50%%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.response-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.response-header h3 { margin: 0; }
.response-actions { display: flex; gap: 8px; }
.response-tabs { display: flex; gap: 4px; margin-bottom: 12px; }
.response-tab { padding: 6px 12px; border: none; background: #e5e5e5; cursor: pointer; font-size: 13px; border-radius: 4px; }
.response-tab.active { background: #111; color: white; }
.response-content { display: none; }
.response-content.active { display: block; }
.curl-preview { background: #1e1e1e; color: #d4d4d4; padding: 12px; border-radius: 6px; font-family: ui-monospace, monospace; font-size: 12px; white-space: pre-wrap; word-break: break-all; margin-bottom: 16px; }
.copy-feedback { position: fixed; bottom: 20px; right: 20px; background: #111; color: #fff; padding: 10px 16px; border-radius: 6px; font-size: 13px; opacity: 0; transition: opacity 0.2s ease; z-index: 1000; }
.copy-feedback.show { opacity: 1; }
</style>
</head>
<body>
    %s
    <main class="docs-content">
        <nav class="docs-breadcrumb">
            <a href="/docs">Documentation</a> / <span>Try It</span>
        </nav>

        <h1>Try It</h1>
        <p class="docs-lead">Test API endpoints directly in your browser. Select an endpoint below and send a request.</p>

        <div class="try-it-console">
            <div class="try-it-form">
                <div class="form-group">
                    <label>API Key <span class="label-hint">(<a href="/portal" target="_blank">Get one here</a>)</span></label>
                    <input type="password" id="apiKey" placeholder="Paste your API key here" class="form-input" autocomplete="off">
                    <div class="validation-error" id="apiKeyError">API key is required to make requests</div>
                </div>

                %s

                <div class="endpoint-desc" id="endpointDesc">%s</div>

                <div class="form-row">
                    <div class="form-group" style="width: 120px;">
                        <label>Method</label>
                        <select id="method" class="form-input">%s</select>
                    </div>

                    <div class="form-group" style="flex: 1;">
                        <label>Endpoint</label>
                        <input type="text" id="endpoint" placeholder="/endpoint" value="%s" class="form-input">
                        <div class="validation-error" id="endpointError">Please enter an endpoint path</div>
                    </div>
                </div>

                <div class="form-group" id="bodyGroup"%s>
                    <label>Request Body <span class="label-hint">(JSON format)</span></label>
                    <textarea id="requestBody" rows="6" class="form-input" placeholder='{"key": "value"}'>%s</textarea>
                    <div class="validation-error" id="bodyError">Invalid JSON format</div>
                </div>

                <div class="curl-preview" id="curlPreview"></div>

                <div class="btn-row">
                    <button id="sendRequest" class="btn btn-primary btn-icon">
                        <span class="btn-text">Send Request</span>
                    </button>
                    <button id="copyCurl" class="btn btn-secondary btn-icon" title="Copy as cURL command">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg>
                        Copy cURL
                    </button>
                </div>
            </div>

            <div class="try-it-response">
                <div class="response-header">
                    <h3>Response</h3>
                    <div class="response-actions">
                        <button id="copyResponse" class="btn btn-sm btn-secondary btn-icon" style="display: none;">
                            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg>
                            Copy
                        </button>
                    </div>
                </div>
                <div class="response-meta" id="responseMeta"></div>
                <div class="response-tabs" id="responseTabs" style="display: none;">
                    <button class="response-tab active" data-tab="body">Body</button>
                    <button class="response-tab" data-tab="headers">Headers</button>
                </div>
                <div class="response-content active" id="responseBodyTab">
                    <pre class="response-body" id="responseBody">Select an endpoint and click "Send Request" to see the response here.</pre>
                </div>
                <div class="response-content" id="responseHeadersTab">
                    <pre class="response-body" id="responseHeaders"></pre>
                </div>
            </div>
        </div>
    </main>
    <div class="copy-feedback" id="copyFeedback">Copied to clipboard!</div>
    <script>
        const baseURL = '%s';
        let lastResponseBody = '';
        let lastResponseHeaders = '';

        // Show copy feedback
        function showCopyFeedback(message) {
            const feedback = document.getElementById('copyFeedback');
            feedback.textContent = message || 'Copied to clipboard!';
            feedback.classList.add('show');
            setTimeout(() => feedback.classList.remove('show'), 2000);
        }

        // Generate cURL command
        function generateCurl() {
            const apiKey = document.getElementById('apiKey').value;
            const method = document.getElementById('method').value;
            const endpoint = document.getElementById('endpoint').value;
            const bodyInput = document.getElementById('requestBody').value;

            let curl = 'curl -X ' + method + ' "' + baseURL + endpoint + '"';
            curl += ' \\\n  -H "Content-Type: application/json"';
            if (apiKey) {
                curl += ' \\\n  -H "X-API-Key: ' + apiKey + '"';
            } else {
                curl += ' \\\n  -H "X-API-Key: YOUR_API_KEY"';
            }

            if (['POST', 'PUT', 'PATCH'].includes(method) && bodyInput.trim()) {
                try {
                    const formatted = JSON.stringify(JSON.parse(bodyInput));
                    curl += " \\\n  -d '" + formatted + "'";
                } catch (e) {
                    curl += " \\\n  -d '" + bodyInput.replace(/'/g, "'\\''") + "'";
                }
            }

            document.getElementById('curlPreview').textContent = curl;
            return curl;
        }

        // Update cURL on any input change
        ['apiKey', 'method', 'endpoint', 'requestBody'].forEach(id => {
            document.getElementById(id).addEventListener('input', generateCurl);
            document.getElementById(id).addEventListener('change', generateCurl);
        });

        // Handle endpoint button clicks
        document.querySelectorAll('.endpoint-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                // Update active state
                document.querySelectorAll('.endpoint-btn').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');

                // Update form fields
                const method = btn.dataset.method;
                const path = btn.dataset.path;
                const example = btn.dataset.example;
                const desc = btn.dataset.desc;

                document.getElementById('method').value = method;
                document.getElementById('endpoint').value = path;
                document.getElementById('endpointDesc').textContent = desc || '';

                // Clear validation errors
                clearErrors();

                // Update body field
                const bodyGroup = document.getElementById('bodyGroup');
                const bodyInput = document.getElementById('requestBody');
                if (['POST', 'PUT', 'PATCH'].includes(method)) {
                    bodyGroup.style.display = 'block';
                    if (example) {
                        bodyInput.value = example;
                    } else {
                        bodyInput.value = '';
                    }
                } else {
                    bodyGroup.style.display = 'none';
                    bodyInput.value = '';
                }

                generateCurl();
            });
        });

        // Clear validation errors
        function clearErrors() {
            document.querySelectorAll('.validation-error').forEach(e => e.classList.remove('show'));
            document.querySelectorAll('.form-input').forEach(e => e.classList.remove('error'));
        }

        // Show validation error
        function showError(inputId, errorId) {
            document.getElementById(inputId).classList.add('error');
            document.getElementById(errorId).classList.add('show');
        }

        // Validate form
        function validateForm() {
            clearErrors();
            let valid = true;

            const apiKey = document.getElementById('apiKey').value.trim();
            const endpoint = document.getElementById('endpoint').value.trim();
            const method = document.getElementById('method').value;
            const bodyInput = document.getElementById('requestBody').value.trim();

            if (!apiKey) {
                showError('apiKey', 'apiKeyError');
                valid = false;
            }

            if (!endpoint) {
                showError('endpoint', 'endpointError');
                valid = false;
            }

            if (['POST', 'PUT', 'PATCH'].includes(method) && bodyInput) {
                try {
                    JSON.parse(bodyInput);
                } catch (e) {
                    showError('requestBody', 'bodyError');
                    valid = false;
                }
            }

            return valid;
        }

        // Send request
        document.getElementById('sendRequest').addEventListener('click', async () => {
            if (!validateForm()) return;

            const apiKey = document.getElementById('apiKey').value;
            const method = document.getElementById('method').value;
            const endpoint = document.getElementById('endpoint').value;
            const bodyInput = document.getElementById('requestBody').value;

            const sendBtn = document.getElementById('sendRequest');
            const btnText = sendBtn.querySelector('.btn-text');
            const originalText = btnText.textContent;

            // Show loading state
            sendBtn.disabled = true;
            btnText.innerHTML = '<span class="spinner"></span> Sending...';

            const options = {
                method: method,
                headers: {
                    'X-API-Key': apiKey,
                    'Content-Type': 'application/json'
                }
            };

            if (['POST', 'PUT', 'PATCH'].includes(method) && bodyInput) {
                options.body = JSON.stringify(JSON.parse(bodyInput));
            }

            const responseMeta = document.getElementById('responseMeta');
            const responseBody = document.getElementById('responseBody');
            const responseHeaders = document.getElementById('responseHeaders');
            const responseTabs = document.getElementById('responseTabs');
            const copyResponseBtn = document.getElementById('copyResponse');

            try {
                const startTime = performance.now();
                const response = await fetch(baseURL + endpoint, options);
                const endTime = performance.now();
                const duration = Math.round(endTime - startTime);

                const statusClass = response.ok ? 'status-success' : 'status-error';
                responseMeta.innerHTML = '<span class="' + statusClass + '">Status: ' + response.status + ' ' + response.statusText + '</span> | Time: ' + duration + 'ms';

                // Show headers
                let headersText = '';
                response.headers.forEach((value, key) => {
                    headersText += key + ': ' + value + '\n';
                });
                responseHeaders.textContent = headersText || '(no headers)';
                lastResponseHeaders = headersText;

                const text = await response.text();
                try {
                    const data = JSON.parse(text);
                    lastResponseBody = JSON.stringify(data, null, 2);
                    responseBody.textContent = lastResponseBody;
                } catch (e) {
                    lastResponseBody = text || '(empty response)';
                    responseBody.textContent = lastResponseBody;
                }

                // Show tabs and copy button
                responseTabs.style.display = 'flex';
                copyResponseBtn.style.display = 'inline-flex';
            } catch (err) {
                responseMeta.innerHTML = '<span class="status-error">Network Error</span>';
                responseBody.textContent = 'Request failed: ' + err.message + '\n\nThis could be due to:\n• CORS restrictions\n• Network connectivity issues\n• Invalid endpoint';
                lastResponseBody = '';
                responseTabs.style.display = 'none';
                copyResponseBtn.style.display = 'none';
            } finally {
                sendBtn.disabled = false;
                btnText.textContent = originalText;
            }
        });

        // Response tab switching
        document.querySelectorAll('.response-tab').forEach(tab => {
            tab.addEventListener('click', () => {
                document.querySelectorAll('.response-tab').forEach(t => t.classList.remove('active'));
                document.querySelectorAll('.response-content').forEach(c => c.classList.remove('active'));
                tab.classList.add('active');
                document.getElementById('response' + tab.dataset.tab.charAt(0).toUpperCase() + tab.dataset.tab.slice(1) + 'Tab').classList.add('active');
            });
        });

        // Copy cURL
        document.getElementById('copyCurl').addEventListener('click', () => {
            const curl = generateCurl();
            navigator.clipboard.writeText(curl).then(() => showCopyFeedback('cURL command copied!'));
        });

        // Copy response
        document.getElementById('copyResponse').addEventListener('click', () => {
            const activeTab = document.querySelector('.response-tab.active');
            const content = activeTab.dataset.tab === 'headers' ? lastResponseHeaders : lastResponseBody;
            navigator.clipboard.writeText(content).then(() => showCopyFeedback('Response copied!'));
        });

        // Show/hide body based on method
        document.getElementById('method').addEventListener('change', (e) => {
            const bodyGroup = document.getElementById('bodyGroup');
            if (['POST', 'PUT', 'PATCH'].includes(e.target.value)) {
                bodyGroup.style.display = 'block';
            } else {
                bodyGroup.style.display = 'none';
            }
            generateCurl();
        });

        // Clear error on input
        document.querySelectorAll('.form-input').forEach(input => {
            input.addEventListener('input', () => {
                input.classList.remove('error');
                const errorEl = input.parentElement.querySelector('.validation-error');
                if (errorEl) errorEl.classList.remove('show');
            });
        });

        // Initialize
        generateCurl();
    </script>
</body>
</html>`, h.appName, docsCSS, h.renderDocsNav("try-it"),
		endpointButtonsHTML,
		defaultDescription,
		methodOptionsHTML,
		defaultEndpoint,
		h.hiddenStyle(defaultMethod),
		escapedExampleBody,
		baseURL)
}

// selectedAttr returns " selected" if the values match, empty string otherwise
func (h *DocsHandler) selectedAttr(current, option string) string {
	if current == option {
		return " selected"
	}
	return ""
}

// hiddenStyle returns style to hide body group for GET/DELETE methods
func (h *DocsHandler) hiddenStyle(method string) string {
	if method == "GET" || method == "DELETE" {
		return ` style="display: none;"`
	}
	return ""
}

func (h *DocsHandler) renderDocsNav(active string) string {
	links := []struct {
		path  string
		label string
		key   string
	}{
		{"/docs", "Home", "home"},
		{"/docs/quickstart", "Quickstart", "quickstart"},
		{"/docs/authentication", "Authentication", "authentication"},
		{"/docs/api-reference", "API Reference", "api-reference"},
		{"/docs/examples", "Examples", "examples"},
		{"/docs/try-it", "Try It", "try-it"},
	}

	navItems := ""
	for _, link := range links {
		activeClass := ""
		if link.key == active {
			activeClass = "active"
		}
		navItems += fmt.Sprintf(`<a href="%s" class="%s">%s</a>`, link.path, activeClass, link.label)
	}

	return h.renderDocsNavWithLogo(active, h.appName)
}

// renderDocsNavWithLogo renders the docs navigation with a custom logo/text.
func (h *DocsHandler) renderDocsNavWithLogo(active string, logoHTML string) string {
	links := []struct {
		path  string
		label string
		key   string
	}{
		{"/docs", "Home", "home"},
		{"/docs/quickstart", "Quickstart", "quickstart"},
		{"/docs/authentication", "Authentication", "authentication"},
		{"/docs/api-reference", "API Reference", "api-reference"},
		{"/docs/examples", "Examples", "examples"},
		{"/docs/try-it", "Try It", "try-it"},
	}

	navItems := ""
	for _, link := range links {
		activeClass := ""
		if link.key == active {
			activeClass = "active"
		}
		navItems += fmt.Sprintf(`<a href="%s" class="%s">%s</a>`, link.path, activeClass, link.label)
	}

	return fmt.Sprintf(`
    <header class="docs-header">
        <div class="docs-header-content">
            <a href="/docs" class="docs-logo">%s Docs</a>
            <nav class="docs-nav">%s</nav>
            <a href="/portal" class="btn btn-sm">Get API Key</a>
        </div>
    </header>`, logoHTML, navItems)
}

// =============================================================================
// CSS & JS
// =============================================================================

const docsCSS = `
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #fff; color: #111; line-height: 1.5; }

.docs-header { background: #111; color: white; padding: 0 24px; position: sticky; top: 0; z-index: 100; }
.docs-header-content { max-width: 960px; margin: 0 auto; display: flex; align-items: center; gap: 24px; height: 52px; }
.docs-logo { color: white; text-decoration: none; font-weight: 600; font-size: 16px; }
.docs-nav { display: flex; gap: 4px; flex: 1; }
.docs-nav a { color: #999; text-decoration: none; padding: 6px 12px; border-radius: 4px; font-size: 14px; }
.docs-nav a:hover, .docs-nav a.active { color: white; }

.docs-content { max-width: 720px; margin: 0 auto; padding: 48px 24px; }
.docs-breadcrumb { font-size: 13px; color: #666; margin-bottom: 24px; }
.docs-breadcrumb a { color: #111; text-decoration: underline; }

.docs-hero { text-align: center; padding: 40px 0; }
.docs-hero h1 { font-size: 28px; font-weight: 600; margin-bottom: 8px; letter-spacing: -0.02em; }
.docs-hero p { font-size: 16px; color: #666; }

.docs-cards { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 16px; margin-top: 32px; }
.docs-card { background: #fff; padding: 20px; border-radius: 6px; text-decoration: none; color: inherit; border: 1px solid #e5e5e5; }
.docs-card:hover { border-color: #111; }
.docs-card h3 { font-size: 15px; font-weight: 500; margin-bottom: 4px; }
.docs-card p { font-size: 13px; color: #666; }

.docs-lead { font-size: 16px; color: #666; margin-bottom: 24px; }
.docs-section { margin-bottom: 40px; }
.docs-section h2 { font-size: 18px; font-weight: 500; margin-bottom: 16px; padding-bottom: 8px; border-bottom: 1px solid #e5e5e5; }

.code-tabs { display: flex; gap: 4px; margin-bottom: 0; }
.code-tab { padding: 6px 12px; border: none; background: #e5e5e5; cursor: pointer; font-size: 13px; border-radius: 4px 4px 0 0; }
.code-tab.active { background: #111; color: white; }

.code-block { background: #111; color: #e5e5e5; padding: 16px; border-radius: 0 4px 4px 4px; overflow-x: auto; font-family: ui-monospace, monospace; font-size: 13px; margin-bottom: 16px; }
.code-block.hidden { display: none; }
.code-block code { color: inherit; background: transparent; padding: 0; }

code { background: #f5f5f5; padding: 2px 6px; border-radius: 3px; font-family: ui-monospace, monospace; font-size: 13px; color: #111; }

.docs-callout { padding: 12px 16px; border-radius: 4px; margin: 16px 0; font-size: 14px; }
.docs-callout.info { background: #f5f5f5; border-left: 3px solid #111; }
.docs-callout.warning { background: #fffbeb; border-left: 3px solid #92400e; }

.docs-table { width: 100%; border-collapse: collapse; margin: 16px 0; font-size: 14px; }
.docs-table th, .docs-table td { padding: 10px 12px; text-align: left; border-bottom: 1px solid #e5e5e5; }
.docs-table th { font-weight: 500; color: #666; font-size: 13px; }

.btn { display: inline-block; padding: 8px 16px; background: #111; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; text-decoration: none; }
.btn:hover { background: #333; }
.btn-sm { padding: 6px 12px; font-size: 13px; }

.try-it-console { display: grid; grid-template-columns: 1fr 1fr; gap: 24px; }
.try-it-form, .try-it-response { background: #fff; padding: 20px; border-radius: 6px; border: 1px solid #e5e5e5; }
.form-group { margin-bottom: 16px; }
.form-group label { display: block; font-size: 14px; font-weight: 500; margin-bottom: 6px; }
.form-input { width: 100%; padding: 10px 12px; border: 1px solid #e5e5e5; border-radius: 4px; font-size: 14px; }
.form-input:focus { outline: none; border-color: #111; }
.form-row { display: flex; gap: 16px; }

.response-meta { font-size: 13px; margin-bottom: 12px; }
.status-success { color: #166534; }
.status-error { color: #991b1b; }
.response-body { background: #111; color: #e5e5e5; padding: 16px; border-radius: 4px; overflow-x: auto; font-family: ui-monospace, monospace; font-size: 13px; min-height: 200px; white-space: pre-wrap; }

.endpoint-card { background: #fff; border: 1px solid #e5e5e5; border-radius: 6px; padding: 16px; margin-bottom: 12px; }
.endpoint-card:hover { border-color: #ccc; }
.endpoint-header { display: flex; align-items: center; gap: 10px; margin-bottom: 8px; }
.endpoint-path { font-family: ui-monospace, monospace; font-size: 14px; color: #111; background: #f5f5f5; padding: 4px 8px; border-radius: 4px; }
.endpoint-desc { color: #666; font-size: 14px; margin-bottom: 12px; }

.method-badge { display: inline-block; padding: 3px 8px; border-radius: 3px; font-size: 11px; font-weight: 600; text-transform: uppercase; font-family: ui-monospace, monospace; }
.method-get { background: #dcfce7; color: #166534; }
.method-post { background: #dbeafe; color: #1e40af; }
.method-put { background: #fef3c7; color: #92400e; }
.method-patch { background: #fef3c7; color: #92400e; }
.method-delete { background: #fee2e2; color: #991b1b; }

.example-section { margin-top: 12px; padding-top: 12px; border-top: 1px solid #f0f0f0; }
.example-section h5 { font-size: 12px; font-weight: 500; color: #666; margin-bottom: 8px; text-transform: uppercase; letter-spacing: 0.05em; }
.example-section .code-block { margin-bottom: 0; border-radius: 4px; }

@media (max-width: 768px) {
    .docs-nav { display: none; }
    .try-it-console { grid-template-columns: 1fr; }
}
`

const docsJS = `
document.querySelectorAll('.code-tab').forEach(tab => {
    tab.addEventListener('click', () => {
        const lang = tab.dataset.lang;
        const parent = tab.closest('.docs-section') || document;

        // Update tabs
        parent.querySelectorAll('.code-tab').forEach(t => t.classList.remove('active'));
        tab.classList.add('active');

        // Update code blocks
        parent.querySelectorAll('.code-block').forEach(block => {
            if (block.dataset.lang === lang) {
                block.classList.remove('hidden');
            } else {
                block.classList.add('hidden');
            }
        });
    });
});
`
