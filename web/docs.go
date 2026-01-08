// Package web provides the developer documentation portal.
package web

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/artpar/apigate/core/openapi"
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderQuickstart(baseURL)))
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderExamples(baseURL)))
}

// TryItPage renders the interactive API console.
func (h *DocsHandler) TryItPage(w http.ResponseWriter, r *http.Request) {
	baseURL := h.getBaseURL(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(h.renderTryIt(baseURL)))
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
// Template Rendering
// =============================================================================

func (h *DocsHandler) renderDocsHome() string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Documentation - %s</title>
    <style>%s</style>
</head>
<body>
    %s
    <main class="docs-content">
        <div class="docs-hero">
            <h1>API Documentation</h1>
            <p>Everything you need to integrate with the API</p>
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
</body>
</html>`, h.appName, docsCSS, h.renderDocsNav("home"))
}

func (h *DocsHandler) renderQuickstart(baseURL string) string {
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

            <div class="code-tabs">
                <button class="code-tab active" data-lang="curl">cURL</button>
                <button class="code-tab" data-lang="javascript">JavaScript</button>
                <button class="code-tab" data-lang="python">Python</button>
            </div>

            <pre class="code-block" data-lang="curl"><code>curl -X GET "%s/your-endpoint" \
  -H "X-API-Key: your_api_key_here" \
  -H "Content-Type: application/json"</code></pre>

            <pre class="code-block hidden" data-lang="javascript"><code>const response = await fetch('%s/your-endpoint', {
  method: 'GET',
  headers: {
    'X-API-Key': 'your_api_key_here',
    'Content-Type': 'application/json'
  }
});

const data = await response.json();
console.log(data);</code></pre>

            <pre class="code-block hidden" data-lang="python"><code>import requests

response = requests.get(
    '%s/your-endpoint',
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
</html>`, h.appName, docsCSS, h.renderDocsNav("quickstart"), h.appName, baseURL, baseURL, baseURL, docsJS)
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

	// Collect only concrete endpoints (no wildcards)
	var endpoints []concreteEndpoint
	for path, pathItem := range spec.Paths {
		// Skip wildcard/proxy routes - they can't be meaningfully documented
		if strings.Contains(path, "{path}") || strings.Contains(path, "*") {
			continue
		}

		var methods []string
		var summary string
		if pathItem.Get != nil {
			methods = append(methods, "GET")
			if summary == "" {
				summary = pathItem.Get.Summary
			}
		}
		if pathItem.Post != nil {
			methods = append(methods, "POST")
			if summary == "" {
				summary = pathItem.Post.Summary
			}
		}
		if pathItem.Put != nil {
			methods = append(methods, "PUT")
		}
		if pathItem.Patch != nil {
			methods = append(methods, "PATCH")
		}
		if pathItem.Delete != nil {
			methods = append(methods, "DELETE")
		}

		if len(methods) > 0 {
			endpoints = append(endpoints, concreteEndpoint{
				path:    path,
				methods: methods,
				summary: summary,
			})
		}
	}

	// Sort by path
	sort.Slice(endpoints, func(i, j int) bool {
		return endpoints[i].path < endpoints[j].path
	})

	// Build endpoint list
	endpointsHTML := ""
	if len(endpoints) == 0 {
		endpointsHTML = `<p style="color: #666; font-size: 14px;">No specific endpoints documented. Contact the API provider for endpoint details.</p>`
	} else {
		for _, ep := range endpoints {
			methodBadges := ""
			for _, m := range ep.methods {
				methodBadges += fmt.Sprintf(`<span class="method-badge">%s</span>`, m)
			}
			summaryText := ""
			if ep.summary != "" {
				summaryText = fmt.Sprintf(`<span style="color: #666; font-size: 13px; margin-left: 8px;">%s</span>`, ep.summary)
			}
			endpointsHTML += fmt.Sprintf(`
				<div style="display: flex; align-items: center; gap: 8px; padding: 12px 0; border-bottom: 1px solid #e5e5e5;">
					<div style="display: flex; gap: 4px; min-width: 60px;">%s</div>
					<code style="flex: 1;">%s</code>%s
				</div>`, methodBadges, ep.path, summaryText)
		}
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>API Reference - %s</title>
    <style>
        %s
        .method-badge { display: inline-block; padding: 2px 6px; border-radius: 3px; font-size: 11px; font-weight: 500; background: #111; color: #fff; }
    </style>
</head>
<body>
    %s
    <main class="docs-content">
        <nav class="docs-breadcrumb">
            <a href="/docs">Documentation</a> / <span>API Reference</span>
        </nav>

        <h1>API Reference</h1>
        <p class="docs-lead">All requests require authentication via the <code>X-API-Key</code> header.</p>

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
</html>`, h.appName, docsCSS, h.renderDocsNav("api-reference"), baseURL, baseURL, endpointsHTML)
}

// concreteEndpoint represents a specific endpoint (not a wildcard)
type concreteEndpoint struct {
	path    string
	methods []string
	summary string
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

func (h *DocsHandler) renderExamples(baseURL string) string {
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
        <p class="docs-lead">Ready-to-use code snippets for common operations.</p>

        <div class="docs-section">
            <h2>Making a GET Request</h2>
            <div class="code-tabs">
                <button class="code-tab active" data-lang="curl">cURL</button>
                <button class="code-tab" data-lang="javascript">JavaScript</button>
                <button class="code-tab" data-lang="python">Python</button>
                <button class="code-tab" data-lang="go">Go</button>
            </div>

            <pre class="code-block" data-lang="curl"><code>curl -X GET "%s/api/resource" \
  -H "X-API-Key: your_api_key" \
  -H "Content-Type: application/json"</code></pre>

            <pre class="code-block hidden" data-lang="javascript"><code>const API_KEY = 'your_api_key';
const BASE_URL = '%s';

async function getResource() {
  const response = await fetch(BASE_URL + '/api/resource', {
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
getResource()
  .then(data => console.log(data))
  .catch(err => console.error(err));</code></pre>

            <pre class="code-block hidden" data-lang="python"><code>import requests

API_KEY = 'your_api_key'
BASE_URL = '%s'

def get_resource():
    response = requests.get(
        f'{BASE_URL}/api/resource',
        headers={
            'X-API-Key': API_KEY,
            'Content-Type': 'application/json'
        }
    )
    response.raise_for_status()
    return response.json()

# Usage
data = get_resource()
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

func getResource() (map[string]interface{}, error) {
    req, err := http.NewRequest("GET", baseURL+"/api/resource", nil)
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
    data, err := getResource()
    if err != nil {
        fmt.Println("Error:", err)
        return
    }
    fmt.Println(data)
}</code></pre>
        </div>

        <div class="docs-section">
            <h2>Making a POST Request</h2>
            <div class="code-tabs">
                <button class="code-tab active" data-lang="curl2">cURL</button>
                <button class="code-tab" data-lang="javascript2">JavaScript</button>
                <button class="code-tab" data-lang="python2">Python</button>
            </div>

            <pre class="code-block" data-lang="curl2"><code>curl -X POST "%s/api/resource" \
  -H "X-API-Key: your_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Example",
    "value": 123
  }'</code></pre>

            <pre class="code-block hidden" data-lang="javascript2"><code>async function createResource(data) {
  const response = await fetch(BASE_URL + '/api/resource', {
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
createResource({ name: 'Example', value: 123 })
  .then(data => console.log(data));</code></pre>

            <pre class="code-block hidden" data-lang="python2"><code>def create_resource(data):
    response = requests.post(
        f'{BASE_URL}/api/resource',
        headers={
            'X-API-Key': API_KEY,
            'Content-Type': 'application/json'
        },
        json=data
    )
    return response.json()

# Usage
result = create_resource({'name': 'Example', 'value': 123})
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
</html>`, h.appName, docsCSS, h.renderDocsNav("examples"), baseURL, baseURL, baseURL, baseURL, baseURL, docsJS)
}

func (h *DocsHandler) renderTryIt(baseURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Try It - %s API</title>
    <style>%s</style>
</head>
<body>
    %s
    <main class="docs-content">
        <nav class="docs-breadcrumb">
            <a href="/docs">Documentation</a> / <span>Try It</span>
        </nav>

        <h1>Try It</h1>
        <p class="docs-lead">Test API endpoints directly from your browser.</p>

        <div class="try-it-console">
            <div class="try-it-form">
                <div class="form-group">
                    <label>API Key</label>
                    <input type="password" id="apiKey" placeholder="Enter your API key" class="form-input">
                </div>

                <div class="form-row">
                    <div class="form-group" style="width: 120px;">
                        <label>Method</label>
                        <select id="method" class="form-input">
                            <option value="GET">GET</option>
                            <option value="POST">POST</option>
                            <option value="PUT">PUT</option>
                            <option value="PATCH">PATCH</option>
                            <option value="DELETE">DELETE</option>
                        </select>
                    </div>

                    <div class="form-group" style="flex: 1;">
                        <label>Endpoint</label>
                        <input type="text" id="endpoint" placeholder="/your-endpoint" value="/" class="form-input">
                    </div>
                </div>

                <div class="form-group" id="bodyGroup">
                    <label>Request Body (JSON)</label>
                    <textarea id="requestBody" rows="6" class="form-input" placeholder='{"key": "value"}'></textarea>
                </div>

                <button id="sendRequest" class="btn btn-primary">Send Request</button>
            </div>

            <div class="try-it-response">
                <h3>Response</h3>
                <div class="response-meta" id="responseMeta"></div>
                <pre class="response-body" id="responseBody">Click "Send Request" to see the response</pre>
            </div>
        </div>
    </main>
    <script>
        const baseURL = '%s';

        document.getElementById('sendRequest').addEventListener('click', async () => {
            const apiKey = document.getElementById('apiKey').value;
            const method = document.getElementById('method').value;
            const endpoint = document.getElementById('endpoint').value;
            const bodyInput = document.getElementById('requestBody').value;

            if (!apiKey) {
                alert('Please enter your API key');
                return;
            }

            const options = {
                method: method,
                headers: {
                    'X-API-Key': apiKey,
                    'Content-Type': 'application/json'
                }
            };

            if (['POST', 'PUT', 'PATCH'].includes(method) && bodyInput) {
                try {
                    options.body = JSON.stringify(JSON.parse(bodyInput));
                } catch (e) {
                    alert('Invalid JSON in request body');
                    return;
                }
            }

            const responseMeta = document.getElementById('responseMeta');
            const responseBody = document.getElementById('responseBody');

            responseMeta.textContent = 'Loading...';
            responseBody.textContent = '';

            try {
                const startTime = performance.now();
                const response = await fetch(baseURL + endpoint, options);
                const endTime = performance.now();
                const duration = Math.round(endTime - startTime);

                const statusClass = response.ok ? 'status-success' : 'status-error';
                responseMeta.innerHTML = '<span class="' + statusClass + '">Status: ' + response.status + ' ' + response.statusText + '</span> | Time: ' + duration + 'ms';

                const text = await response.text();
                try {
                    const data = JSON.parse(text);
                    responseBody.textContent = JSON.stringify(data, null, 2);
                } catch (e) {
                    // Not JSON - show raw response
                    responseBody.textContent = text || '(empty response)';
                }
            } catch (err) {
                responseMeta.innerHTML = '<span class="status-error">Network Error</span>';
                responseBody.textContent = 'Request failed: ' + err.message;
            }
        });

        // Show/hide body based on method
        document.getElementById('method').addEventListener('change', (e) => {
            const bodyGroup = document.getElementById('bodyGroup');
            if (['POST', 'PUT', 'PATCH'].includes(e.target.value)) {
                bodyGroup.style.display = 'block';
            } else {
                bodyGroup.style.display = 'none';
            }
        });
    </script>
</body>
</html>`, h.appName, docsCSS, h.renderDocsNav("try-it"), baseURL)
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

	return fmt.Sprintf(`
    <header class="docs-header">
        <div class="docs-header-content">
            <a href="/docs" class="docs-logo">%s Docs</a>
            <nav class="docs-nav">%s</nav>
            <a href="/portal" class="btn btn-sm">Get API Key</a>
        </div>
    </header>`, h.appName, navItems)
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
