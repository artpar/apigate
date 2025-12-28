/**
 * Expr Editor - Autocomplete and validation for Expr expressions
 * Used in route forms for path rewrite, metering, and transform expressions
 */

// Autocomplete data structure
const ExprAutocomplete = {
    // Context variables available in different expression contexts
    contexts: {
        request: {
            description: "Available in request transforms and path rewrite",
            variables: [
                { name: "path", type: "string", desc: "Request URL path (e.g., /v1/chat/completions)" },
                { name: "method", type: "string", desc: "HTTP method (GET, POST, PUT, DELETE, etc.)" },
                { name: "query", type: "map[string]string", desc: "URL query parameters as key-value map" },
                { name: "headers", type: "map[string]string", desc: "Request headers as key-value map" },
                { name: "body", type: "any", desc: "Parsed JSON request body" },
                { name: "rawBody", type: "[]byte", desc: "Raw request body bytes" },
                { name: "userID", type: "string", desc: "Authenticated user's ID" },
                { name: "planID", type: "string", desc: "User's plan/tier ID" },
                { name: "keyID", type: "string", desc: "API key ID used for authentication" }
            ]
        },
        response: {
            description: "Available in response transforms and metering (buffered)",
            variables: [
                { name: "status", type: "int", desc: "HTTP response status code (200, 404, 500, etc.)" },
                { name: "respBody", type: "any", desc: "Parsed JSON response body" },
                { name: "respHeaders", type: "map[string]string", desc: "Response headers as key-value map" },
                { name: "responseBytes", type: "int64", desc: "Response body size in bytes" },
                { name: "requestBytes", type: "int64", desc: "Request body size in bytes" },
                { name: "path", type: "string", desc: "Original request path" },
                { name: "method", type: "string", desc: "Original HTTP method" },
                { name: "userID", type: "string", desc: "Authenticated user's ID" },
                { name: "planID", type: "string", desc: "User's plan/tier ID" },
                { name: "keyID", type: "string", desc: "API key ID" }
            ]
        },
        streaming: {
            description: "Available in SSE/streaming metering expressions",
            variables: [
                { name: "status", type: "int", desc: "HTTP response status code" },
                { name: "allData", type: "[]byte", desc: "All accumulated stream data" },
                { name: "lastChunk", type: "[]byte", desc: "Last received chunk" },
                { name: "responseBytes", type: "int64", desc: "Total bytes streamed" },
                { name: "path", type: "string", desc: "Original request path" },
                { name: "method", type: "string", desc: "Original HTTP method" },
                { name: "userID", type: "string", desc: "Authenticated user's ID" },
                { name: "planID", type: "string", desc: "User's plan/tier ID" },
                { name: "keyID", type: "string", desc: "API key ID" }
            ]
        }
    },

    // Functions available in all expressions
    functions: [
        // String functions
        { name: "lower", signature: "lower(s string) string", desc: "Convert string to lowercase", category: "string" },
        { name: "upper", signature: "upper(s string) string", desc: "Convert string to uppercase", category: "string" },
        { name: "trim", signature: "trim(s string) string", desc: "Remove leading/trailing whitespace", category: "string" },
        { name: "trimPrefix", signature: "trimPrefix(s, prefix string) string", desc: "Remove prefix from string", category: "string", example: 'trimPrefix(path, "/v1")' },
        { name: "trimSuffix", signature: "trimSuffix(s, suffix string) string", desc: "Remove suffix from string", category: "string" },
        { name: "replace", signature: "replace(s, old, new string) string", desc: "Replace all occurrences", category: "string", example: 'replace(path, "old", "new")' },
        { name: "split", signature: "split(s, sep string) []string", desc: "Split string by separator", category: "string" },
        { name: "join", signature: "join(arr []string, sep string) string", desc: "Join array with separator", category: "string" },
        { name: "contains", signature: "contains(s, substr string) bool", desc: "Check if string contains substring", category: "string" },
        { name: "hasPrefix", signature: "hasPrefix(s, prefix string) bool", desc: "Check if string starts with prefix", category: "string" },
        { name: "hasSuffix", signature: "hasSuffix(s, suffix string) bool", desc: "Check if string ends with suffix", category: "string" },

        // Encoding functions
        { name: "base64Encode", signature: "base64Encode(s string) string", desc: "Encode string to base64", category: "encoding" },
        { name: "base64Decode", signature: "base64Decode(s string) string", desc: "Decode base64 string", category: "encoding" },
        { name: "urlEncode", signature: "urlEncode(s string) string", desc: "URL-encode string", category: "encoding" },
        { name: "urlDecode", signature: "urlDecode(s string) string", desc: "URL-decode string", category: "encoding" },
        { name: "jsonEncode", signature: "jsonEncode(obj any) string", desc: "Encode object to JSON string", category: "encoding" },
        { name: "jsonDecode", signature: "jsonDecode(s string) any", desc: "Parse JSON string to object", category: "encoding" },

        // Data parsing functions
        { name: "json", signature: "json(data []byte|string) any", desc: "Parse JSON from bytes or string", category: "data", example: "json(sseLastData(allData))" },
        { name: "get", signature: "get(obj any, path string) any", desc: "Safe nested field access with dot notation", category: "data", example: 'get(respBody, "usage.total_tokens")' },
        { name: "lines", signature: "lines(data []byte|string) []string", desc: "Split data into lines", category: "data" },
        { name: "linesNonEmpty", signature: "linesNonEmpty(data []byte|string) []string", desc: "Split into non-empty lines", category: "data", example: "count(linesNonEmpty(allData))" },

        // SSE parsing functions
        { name: "sseEvents", signature: "sseEvents(data []byte) []SSEEvent", desc: "Parse SSE stream into array of events", category: "sse", example: "sseEvents(allData)[0].data" },
        { name: "sseLastData", signature: "sseLastData(data []byte) string", desc: "Get data field from last SSE event", category: "sse", example: "json(sseLastData(allData)).usage.total_tokens" },
        { name: "sseAllData", signature: "sseAllData(data []byte) string", desc: "Concatenate all SSE data fields", category: "sse" },

        // Array/collection functions
        { name: "len", signature: "len(arr any) int", desc: "Get length of array, string, or map", category: "array", example: "len(respBody)" },
        { name: "count", signature: "count(arr any) int", desc: "Count items in array (alias for len)", category: "array", example: "count(sseEvents(allData))" },
        { name: "first", signature: "first(arr []any) any", desc: "Get first element of array", category: "array" },
        { name: "last", signature: "last(arr []any) any", desc: "Get last element of array", category: "array" },
        { name: "sum", signature: "sum(arr []number) number", desc: "Sum all numbers in array", category: "array" },
        { name: "avg", signature: "avg(arr []number) number", desc: "Average of numbers in array", category: "array" },
        { name: "min", signature: "min(arr []number) number", desc: "Minimum value in array", category: "array" },
        { name: "max", signature: "max(arr []number) number", desc: "Maximum value in array", category: "array" },

        // Utility functions
        { name: "env", signature: "env(name string) string", desc: "Get environment variable value", category: "utility", example: 'env("API_KEY")' },
        { name: "now", signature: "now() int64", desc: "Current Unix timestamp (seconds)", category: "utility" },
        { name: "nowRFC3339", signature: "nowRFC3339() string", desc: "Current time in RFC3339 format", category: "utility" },
        { name: "uuid", signature: "uuid() string", desc: "Generate random UUID", category: "utility" },
        { name: "coalesce", signature: "coalesce(values ...any) any", desc: "Return first non-nil value", category: "utility", example: "coalesce(respBody.tokens, 1)" },
        { name: "default", signature: "default(val, defaultVal any) any", desc: "Return default if val is nil/empty", category: "utility" }
    ],

    // Operators
    operators: [
        { name: "??", desc: "Nil coalescing - return right side if left is nil", example: "respBody.tokens ?? 1" },
        { name: "?:", desc: "Ternary conditional", example: "status < 400 ? respBody.units : 0" },
        { name: "&&", desc: "Logical AND" },
        { name: "||", desc: "Logical OR" },
        { name: "!", desc: "Logical NOT" },
        { name: "==", desc: "Equality" },
        { name: "!=", desc: "Inequality" },
        { name: "<", desc: "Less than" },
        { name: ">", desc: "Greater than" },
        { name: "<=", desc: "Less than or equal" },
        { name: ">=", desc: "Greater than or equal" },
        { name: "+", desc: "Addition / string concatenation" },
        { name: "-", desc: "Subtraction" },
        { name: "*", desc: "Multiplication" },
        { name: "/", desc: "Division" },
        { name: "%", desc: "Modulo" }
    ]
};

/**
 * ExprEditor class - manages autocomplete for an input/textarea
 */
class ExprEditor {
    constructor(element, options = {}) {
        this.element = element;
        this.context = options.context || 'response'; // request, response, streaming
        this.onValidate = options.onValidate || null;
        this.dropdown = null;
        this.suggestions = [];
        this.selectedIndex = -1;
        this.debounceTimer = null;
        this.validateTimer = null;
        this.validationIndicator = null;
        this.lastValidatedValue = null;

        this.init();
    }

    init() {
        // Add expr-input class for styling
        this.element.classList.add('expr-input');

        // Create wrapper for relative positioning if not already
        if (!this.element.parentNode.classList.contains('expr-wrapper')) {
            const wrapper = document.createElement('div');
            wrapper.className = 'expr-wrapper';
            wrapper.style.position = 'relative';
            this.element.parentNode.insertBefore(wrapper, this.element);
            wrapper.appendChild(this.element);
        }
        const wrapper = this.element.parentNode;

        // Create dropdown container
        this.dropdown = document.createElement('div');
        this.dropdown.className = 'expr-dropdown';
        this.dropdown.style.display = 'none';
        wrapper.appendChild(this.dropdown);

        // Create validation indicator
        this.validationIndicator = document.createElement('div');
        this.validationIndicator.className = 'expr-validation';
        this.validationIndicator.style.display = 'none';
        wrapper.appendChild(this.validationIndicator);

        // Event listeners
        this.element.addEventListener('input', (e) => this.onInput(e));
        this.element.addEventListener('keydown', (e) => this.onKeyDown(e));
        this.element.addEventListener('blur', () => {
            this.hideDropdown();
            this.validateDebounced();
        });
        this.element.addEventListener('focus', () => this.onInput());
    }

    onInput() {
        clearTimeout(this.debounceTimer);
        this.debounceTimer = setTimeout(() => {
            const word = this.getCurrentWord();
            if (word.length >= 1) {
                this.showSuggestions(word);
            } else {
                this.hideDropdown();
            }
        }, 100);
    }

    getCurrentWord() {
        const cursorPos = this.element.selectionStart;
        const text = this.element.value.substring(0, cursorPos);
        // Match word characters, dots, and opening parens
        const match = text.match(/[\w.]+$/);
        return match ? match[0] : '';
    }

    showSuggestions(query) {
        const lowerQuery = query.toLowerCase();
        this.suggestions = [];

        // Get context variables
        const contextVars = ExprAutocomplete.contexts[this.context]?.variables || [];
        for (const v of contextVars) {
            if (v.name.toLowerCase().includes(lowerQuery)) {
                this.suggestions.push({
                    type: 'variable',
                    name: v.name,
                    detail: v.type,
                    desc: v.desc,
                    insert: v.name
                });
            }
        }

        // Get functions
        for (const f of ExprAutocomplete.functions) {
            if (f.name.toLowerCase().includes(lowerQuery)) {
                this.suggestions.push({
                    type: 'function',
                    name: f.name,
                    detail: f.signature,
                    desc: f.desc,
                    insert: f.name + '(',
                    example: f.example
                });
            }
        }

        // Sort: exact prefix matches first, then by name
        this.suggestions.sort((a, b) => {
            const aPrefix = a.name.toLowerCase().startsWith(lowerQuery);
            const bPrefix = b.name.toLowerCase().startsWith(lowerQuery);
            if (aPrefix && !bPrefix) return -1;
            if (!aPrefix && bPrefix) return 1;
            return a.name.localeCompare(b.name);
        });

        // Limit results
        this.suggestions = this.suggestions.slice(0, 10);

        if (this.suggestions.length > 0) {
            this.renderDropdown();
            this.selectedIndex = 0;
            this.updateSelection();
        } else {
            this.hideDropdown();
        }
    }

    renderDropdown() {
        this.dropdown.innerHTML = this.suggestions.map((s, i) => `
            <div class="expr-dropdown-item" data-index="${i}">
                <div class="expr-item-header">
                    <span class="expr-item-icon">${s.type === 'function' ? 'fn' : 'v'}</span>
                    <span class="expr-item-name">${this.escapeHtml(s.name)}</span>
                    <span class="expr-item-type">${this.escapeHtml(s.detail)}</span>
                </div>
                <div class="expr-item-desc">${this.escapeHtml(s.desc)}</div>
                ${s.example ? `<div class="expr-item-example">${this.escapeHtml(s.example)}</div>` : ''}
            </div>
        `).join('');

        // Add click handlers
        this.dropdown.querySelectorAll('.expr-dropdown-item').forEach(item => {
            item.addEventListener('mousedown', (e) => {
                e.preventDefault();
                this.selectSuggestion(parseInt(item.dataset.index));
            });
        });

        this.dropdown.style.display = 'block';
    }

    updateSelection() {
        this.dropdown.querySelectorAll('.expr-dropdown-item').forEach((item, i) => {
            item.classList.toggle('selected', i === this.selectedIndex);
        });

        // Scroll selected into view
        const selected = this.dropdown.querySelector('.expr-dropdown-item.selected');
        if (selected) {
            selected.scrollIntoView({ block: 'nearest' });
        }
    }

    onKeyDown(e) {
        if (this.dropdown.style.display === 'none') return;

        switch (e.key) {
            case 'ArrowDown':
                e.preventDefault();
                this.selectedIndex = Math.min(this.selectedIndex + 1, this.suggestions.length - 1);
                this.updateSelection();
                break;
            case 'ArrowUp':
                e.preventDefault();
                this.selectedIndex = Math.max(this.selectedIndex - 1, 0);
                this.updateSelection();
                break;
            case 'Enter':
            case 'Tab':
                if (this.selectedIndex >= 0) {
                    e.preventDefault();
                    this.selectSuggestion(this.selectedIndex);
                }
                break;
            case 'Escape':
                this.hideDropdown();
                break;
        }
    }

    selectSuggestion(index) {
        const suggestion = this.suggestions[index];
        if (!suggestion) return;

        const cursorPos = this.element.selectionStart;
        const text = this.element.value;
        const word = this.getCurrentWord();
        const wordStart = cursorPos - word.length;

        // Replace current word with suggestion
        this.element.value = text.substring(0, wordStart) + suggestion.insert + text.substring(cursorPos);

        // Position cursor
        const newPos = wordStart + suggestion.insert.length;
        this.element.setSelectionRange(newPos, newPos);
        this.element.focus();

        this.hideDropdown();

        // Trigger input event for any listeners
        this.element.dispatchEvent(new Event('input', { bubbles: true }));
    }

    hideDropdown() {
        this.dropdown.style.display = 'none';
        this.selectedIndex = -1;
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // Update context (e.g., when protocol changes)
    setContext(context) {
        this.context = context;
    }

    // Validate expression with debounce
    validateDebounced() {
        clearTimeout(this.validateTimer);
        this.validateTimer = setTimeout(() => this.validate(), 300);
    }

    // Validate expression against server
    async validate() {
        const expression = this.element.value.trim();

        // Skip if empty or unchanged
        if (!expression) {
            this.hideValidation();
            return;
        }
        if (expression === this.lastValidatedValue) {
            return;
        }
        this.lastValidatedValue = expression;

        // Show loading state
        this.showValidation('loading', 'Validating...');

        try {
            const response = await fetch('/api/expr/validate', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    expression: expression,
                    context: this.context
                })
            });

            const result = await response.json();

            if (result.valid) {
                this.showValidation('success', 'Valid expression');
                this.element.classList.remove('expr-invalid');
                this.element.classList.add('expr-valid');
            } else {
                this.showValidation('error', result.error || 'Invalid expression');
                this.element.classList.remove('expr-valid');
                this.element.classList.add('expr-invalid');
            }

            // Call optional callback
            if (this.onValidate) {
                this.onValidate(result);
            }
        } catch (err) {
            // Network error - don't show as validation failure
            this.hideValidation();
            console.warn('Expr validation failed:', err);
        }
    }

    showValidation(type, message) {
        if (!this.validationIndicator) return;

        this.validationIndicator.className = 'expr-validation expr-validation-' + type;
        this.validationIndicator.textContent = message;
        this.validationIndicator.style.display = 'block';

        // Auto-hide success after 2 seconds
        if (type === 'success') {
            setTimeout(() => {
                if (this.validationIndicator.classList.contains('expr-validation-success')) {
                    this.hideValidation();
                }
            }, 2000);
        }
    }

    hideValidation() {
        if (this.validationIndicator) {
            this.validationIndicator.style.display = 'none';
        }
        this.element.classList.remove('expr-valid', 'expr-invalid');
    }
}

/**
 * Initialize Expr editors on all matching elements
 */
function initExprEditors() {
    // Metering expressions - use streaming context for SSE, response for others
    const meteringExpr = document.getElementById('metering_expr');
    if (meteringExpr) {
        const protocol = document.getElementById('protocol');
        const context = (protocol && (protocol.value === 'sse' || protocol.value === 'http_stream')) ? 'streaming' : 'response';
        const editor = new ExprEditor(meteringExpr, { context });

        // Update context when protocol changes
        if (protocol) {
            protocol.addEventListener('change', () => {
                const newContext = (protocol.value === 'sse' || protocol.value === 'http_stream') ? 'streaming' : 'response';
                editor.setContext(newContext);
            });
        }
    }

    // Path rewrite - request context
    const pathRewrite = document.getElementById('path_rewrite');
    if (pathRewrite) {
        new ExprEditor(pathRewrite, { context: 'request' });
    }

    // Request body transform - request context
    const requestBodyExpr = document.getElementById('request_body_expr');
    if (requestBodyExpr) {
        new ExprEditor(requestBodyExpr, { context: 'request' });
    }

    // Response body transform - response context
    const responseBodyExpr = document.getElementById('response_body_expr');
    if (responseBodyExpr) {
        new ExprEditor(responseBodyExpr, { context: 'response' });
    }
}

// Auto-initialize when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initExprEditors);
} else {
    initExprEditors();
}

// Export for manual use
window.ExprEditor = ExprEditor;
window.ExprAutocomplete = ExprAutocomplete;
