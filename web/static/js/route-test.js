/**
 * Route Test Panel - Test route matching and transformations
 */

class RouteTestPanel {
    constructor(options = {}) {
        this.routeId = options.routeId || null;
        this.panel = null;
        this.resultDiv = null;
        this.isOpen = false;

        this.init();
    }

    init() {
        this.panel = document.getElementById('route-test-panel');
        if (!this.panel) return;

        this.resultDiv = document.getElementById('test-result');

        // Toggle panel
        const header = this.panel.querySelector('.test-panel-header');
        if (header) {
            header.addEventListener('click', () => this.toggle());
        }

        // Test button
        const testBtn = document.getElementById('test-route-btn');
        if (testBtn) {
            testBtn.addEventListener('click', () => this.runTest());
        }

        // Auto-fill path from form if available
        const pathPattern = document.getElementById('path_pattern');
        const testPath = document.getElementById('test-path');
        if (pathPattern && testPath && !testPath.value) {
            // Convert pattern to sample path
            let sample = pathPattern.value || '/api/v1/test';
            sample = sample.replace(/\*/g, 'test');
            sample = sample.replace(/\{[^}]+\}/g, '123');
            testPath.value = sample;
        }
    }

    toggle() {
        this.isOpen = !this.isOpen;
        this.panel.classList.toggle('open', this.isOpen);
    }

    async runTest() {
        const method = document.getElementById('test-method')?.value || 'GET';
        const path = document.getElementById('test-path')?.value || '/';
        const headersText = document.getElementById('test-headers')?.value || '';
        const body = document.getElementById('test-body')?.value || '';

        // Parse headers
        const headers = {};
        headersText.split('\n').forEach(line => {
            line = line.trim();
            if (!line) return;
            const colonIdx = line.indexOf(':');
            if (colonIdx > 0) {
                const name = line.substring(0, colonIdx).trim();
                const value = line.substring(colonIdx + 1).trim();
                headers[name] = value;
            }
        });

        // Build request
        const request = {
            method,
            path,
            headers,
            body
        };

        // If we have a route ID, test that specific route
        if (this.routeId) {
            request.route_id = this.routeId;
        }

        // Show loading
        this.showLoading();

        try {
            const response = await fetch('/api/routes/test', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(request)
            });

            const result = await response.json();
            this.showResult(result);
        } catch (err) {
            this.showError('Network error: ' + err.message);
        }
    }

    showLoading() {
        if (!this.resultDiv) return;
        this.resultDiv.innerHTML = `
            <div class="test-result-status">
                <span class="test-result-badge" style="background: #1e40af; color: #bfdbfe;">Testing...</span>
            </div>
        `;
        this.resultDiv.style.display = 'block';
    }

    showError(message) {
        if (!this.resultDiv) return;
        this.resultDiv.innerHTML = `
            <div class="test-result-status">
                <span class="test-result-badge error">Error</span>
                <span class="test-result-reason">${this.escapeHtml(message)}</span>
            </div>
        `;
        this.resultDiv.style.display = 'block';
    }

    showResult(result) {
        if (!this.resultDiv) return;

        let html = '<div class="test-result-status">';

        if (result.error) {
            html += `
                <span class="test-result-badge error">Error</span>
                <span class="test-result-reason">${this.escapeHtml(result.error)}</span>
            `;
        } else if (result.matched) {
            html += `
                <span class="test-result-badge matched">Matched</span>
                <span class="test-result-reason">${this.escapeHtml(result.match_reason || '')}</span>
            `;
        } else {
            html += `
                <span class="test-result-badge not-matched">No Match</span>
                <span class="test-result-reason">${this.escapeHtml(result.match_reason || 'No route matched this request')}</span>
            `;
        }
        html += '</div>';

        if (result.matched) {
            // Route info
            html += `
                <div class="test-result-section">
                    <div class="test-result-section-title">Route Information</div>
                    <div class="test-result-grid">
                        <div class="test-result-item">
                            <div class="test-result-label">Route Name</div>
                            <div class="test-result-value">${this.escapeHtml(result.route_name || '-')}</div>
                        </div>
                        <div class="test-result-item">
                            <div class="test-result-label">Upstream</div>
                            <div class="test-result-value">${this.escapeHtml(result.upstream_name || '-')}</div>
                        </div>
                    </div>
                </div>
            `;

            // Path params if any
            if (result.path_params && Object.keys(result.path_params).length > 0) {
                html += `
                    <div class="test-result-section">
                        <div class="test-result-section-title">Path Parameters</div>
                        <div class="test-result-grid">
                `;
                for (const [key, value] of Object.entries(result.path_params)) {
                    html += `
                        <div class="test-result-item">
                            <div class="test-result-label">{${this.escapeHtml(key)}}</div>
                            <div class="test-result-value">${this.escapeHtml(value)}</div>
                        </div>
                    `;
                }
                html += '</div></div>';
            }

            // Transformed request
            html += `
                <div class="test-result-section">
                    <div class="test-result-section-title">Transformed Request</div>
                    <div class="test-result-grid">
                        <div class="test-result-item">
                            <div class="test-result-label">Method</div>
                            <div class="test-result-value">${this.escapeHtml(result.transformed_method || '-')}</div>
                        </div>
                        <div class="test-result-item">
                            <div class="test-result-label">Path</div>
                            <div class="test-result-value">${this.formatExprValue(result.transformed_path || '-')}</div>
                        </div>
                    </div>
                </div>
            `;

            // Upstream URL
            if (result.upstream_url) {
                html += `
                    <div class="test-result-section">
                        <div class="test-result-section-title">Upstream URL</div>
                        <div class="test-result-item">
                            <div class="test-result-value">${this.escapeHtml(result.upstream_url)}</div>
                        </div>
                    </div>
                `;
            }

            // Headers
            if (result.transformed_headers && Object.keys(result.transformed_headers).length > 0) {
                html += `
                    <div class="test-result-section">
                        <div class="test-result-section-title">Request Headers</div>
                        <div class="test-result-headers">
                `;
                for (const [name, value] of Object.entries(result.transformed_headers)) {
                    html += `
                        <div class="test-result-header-row">
                            <div class="test-result-header-name">${this.escapeHtml(name)}</div>
                            <div class="test-result-header-value">${this.formatExprValue(value)}</div>
                        </div>
                    `;
                }
                html += '</div></div>';
            }

            // Body transform
            if (result.transformed_body && result.transformed_body !== document.getElementById('test-body')?.value) {
                html += `
                    <div class="test-result-section">
                        <div class="test-result-section-title">Transformed Body</div>
                        <div class="test-result-item">
                            <div class="test-result-value">${this.formatExprValue(result.transformed_body)}</div>
                        </div>
                    </div>
                `;
            }

            // Metering
            if (result.metering_expr) {
                html += `
                    <div class="test-result-section">
                        <div class="test-result-section-title">Metering</div>
                        <div class="test-result-grid">
                            <div class="test-result-item">
                                <div class="test-result-label">Expression</div>
                                <div class="test-result-value" style="color: #22d3ee;">${this.escapeHtml(result.metering_expr)}</div>
                            </div>
                            <div class="test-result-item">
                                <div class="test-result-label">Sample Value</div>
                                <div class="test-result-value">${result.metering_sample || 1}</div>
                            </div>
                        </div>
                    </div>
                `;
            }
        }

        this.resultDiv.innerHTML = html;
        this.resultDiv.style.display = 'block';
    }

    formatExprValue(value) {
        if (value && value.startsWith('[expr:')) {
            return `<span class="test-result-header-expr">${this.escapeHtml(value)}</span>`;
        }
        return this.escapeHtml(value);
    }

    escapeHtml(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

// Auto-initialize on page load
function initRouteTestPanel() {
    // Get route ID from URL if editing
    const match = window.location.pathname.match(/\/routes\/([^\/]+)/);
    const routeId = match ? match[1] : null;

    // Only init if routeId exists (editing) or if panel exists anyway
    if (document.getElementById('route-test-panel')) {
        new RouteTestPanel({ routeId });
    }
}

if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initRouteTestPanel);
} else {
    initRouteTestPanel();
}

window.RouteTestPanel = RouteTestPanel;
