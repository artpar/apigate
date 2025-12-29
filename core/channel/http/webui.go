// Package http provides Web UI serving from embedded assets.
package http

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
)

//go:embed all:webui/dist
var webUIAssets embed.FS

// WebUIHandler returns an HTTP handler that serves the embedded Web UI.
// It serves static files from the embedded assets and falls back to index.html
// for client-side routing support.
func WebUIHandler() http.Handler {
	// Try to get the dist subfolder
	distFS, err := fs.Sub(webUIAssets, "webui/dist")
	if err != nil {
		// Return a handler that shows build instructions
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Web UI Not Built</title></head>
<body style="font-family: system-ui; padding: 2rem; max-width: 600px; margin: 0 auto;">
<h1>Web UI Not Built</h1>
<p>The web UI assets have not been built yet. To build them:</p>
<pre style="background: #f4f4f4; padding: 1rem; border-radius: 4px;">
cd webui
npm install
npm run build
</pre>
<p>Then rebuild the Go binary to embed the assets.</p>
</body>
</html>`))
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get path and strip prefixes
		// The path may be /mod/ui/... or /ui/... depending on how routing works
		urlPath := r.URL.Path
		urlPath = strings.TrimPrefix(urlPath, "/mod")
		urlPath = strings.TrimPrefix(urlPath, "/ui")
		urlPath = path.Clean(urlPath)
		if urlPath == "/" || urlPath == "" || urlPath == "." {
			urlPath = "index.html"
		} else {
			urlPath = strings.TrimPrefix(urlPath, "/")
		}

		// Try to serve the requested file
		content, err := fs.ReadFile(distFS, urlPath)
		if err == nil {
			// Set content type based on extension
			contentType := "application/octet-stream"
			switch {
			case strings.HasSuffix(urlPath, ".html"):
				contentType = "text/html; charset=utf-8"
			case strings.HasSuffix(urlPath, ".js"):
				contentType = "application/javascript"
			case strings.HasSuffix(urlPath, ".css"):
				contentType = "text/css"
			case strings.HasSuffix(urlPath, ".svg"):
				contentType = "image/svg+xml"
			case strings.HasSuffix(urlPath, ".json"):
				contentType = "application/json"
			case strings.HasSuffix(urlPath, ".png"):
				contentType = "image/png"
			case strings.HasSuffix(urlPath, ".ico"):
				contentType = "image/x-icon"
			}
			w.Header().Set("Content-Type", contentType)
			w.Write(content)
			return
		}

		// Check if it's an asset request (has file extension) - return 404
		if strings.Contains(path.Base(urlPath), ".") {
			http.NotFound(w, r)
			return
		}

		// SPA fallback: serve index.html for client-side routing
		content, err = fs.ReadFile(distFS, "index.html")
		if err != nil {
			http.Error(w, "index.html not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(content)
	})
}

// DevWebUIHandler returns a handler that proxies to the Vite dev server.
// Use this during development for hot module replacement.
func DevWebUIHandler(viteURL string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This is a placeholder - in production you'd use httputil.ReverseProxy
		http.Redirect(w, r, viteURL+r.URL.Path, http.StatusTemporaryRedirect)
	})
}

// IsWebUIBuilt checks if the web UI has been built.
func IsWebUIBuilt() bool {
	_, err := fs.Stat(webUIAssets, "webui/dist/index.html")
	return err == nil
}

// WebUIDevMode checks if we should use dev mode (Vite server).
func WebUIDevMode() bool {
	return os.Getenv("APIGATE_WEBUI_DEV") == "1"
}
