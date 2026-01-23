package route

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// MatchResult contains information about a successful route match.
type MatchResult struct {
	Route      *Route
	PathParams map[string]string // Extracted path parameters (e.g., {id} -> "123")
}

// Matcher provides route matching with compiled patterns.
// Routes are sorted by priority (descending) for deterministic matching.
type Matcher struct {
	routes   []Route
	patterns []compiledPattern
}

type compiledPattern struct {
	routeIdx   int
	regex      *regexp.Regexp // For regex and prefix patterns
	exact      string         // For exact patterns
	paramNames []string       // Names of path parameters

	// Host matching
	hostRegex    *regexp.Regexp // For regex host patterns
	hostExact    string         // For exact host patterns (lowercase)
	hostWildcard string         // For wildcard host patterns (suffix after *)
}

// NewMatcher creates a new Matcher from a list of routes.
// Routes are sorted by priority (highest first) and patterns are compiled.
func NewMatcher(routes []Route) (*Matcher, error) {
	// Sort routes by priority descending, then by specificity
	// Order: priority > host specificity > path match type > pattern length
	sorted := make([]Route, len(routes))
	copy(sorted, routes)
	sort.Slice(sorted, func(i, j int) bool {
		// 1. Priority (higher first)
		if sorted[i].Priority != sorted[j].Priority {
			return sorted[i].Priority > sorted[j].Priority
		}
		// 2. Host specificity (exact > wildcard > regex > none)
		hostPrioI := hostMatchTypePriority(sorted[i].HostMatchType)
		hostPrioJ := hostMatchTypePriority(sorted[j].HostMatchType)
		if hostPrioI != hostPrioJ {
			return hostPrioI > hostPrioJ
		}
		// 3. Path match type specificity (exact > prefix > regex)
		if sorted[i].MatchType != sorted[j].MatchType {
			return matchTypePriority(sorted[i].MatchType) > matchTypePriority(sorted[j].MatchType)
		}
		// 4. Pattern length (longer first)
		return len(sorted[i].PathPattern) > len(sorted[j].PathPattern)
	})

	patterns := make([]compiledPattern, 0, len(sorted))
	for i, r := range sorted {
		if !r.Enabled {
			continue
		}

		cp, err := compilePattern(r, i)
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, cp)
	}

	return &Matcher{
		routes:   sorted,
		patterns: patterns,
	}, nil
}

func matchTypePriority(mt MatchType) int {
	switch mt {
	case MatchExact:
		return 3
	case MatchPrefix:
		return 2
	case MatchRegex:
		return 1
	default:
		return 0
	}
}

func hostMatchTypePriority(hmt HostMatchType) int {
	switch hmt {
	case HostMatchExact:
		return 4
	case HostMatchWildcard:
		return 3
	case HostMatchRegex:
		return 2
	case HostMatchNone:
		return 1
	default:
		return 0
	}
}

func compilePattern(r Route, routeIdx int) (compiledPattern, error) {
	cp := compiledPattern{
		routeIdx: routeIdx,
	}

	// Compile host pattern
	if err := compileHostPattern(&cp, r.HostPattern, r.HostMatchType); err != nil {
		return cp, err
	}

	// Compile path pattern
	pattern := r.PathPattern
	matchType := r.MatchType

	switch matchType {
	case MatchExact:
		cp.exact = pattern

	case MatchPrefix:
		// Convert prefix pattern to regex
		// /api/* -> ^/api/.*$
		// /api/v1 -> ^/api/v1.*$
		regexPattern := "^" + regexp.QuoteMeta(strings.TrimSuffix(pattern, "*"))
		if strings.HasSuffix(pattern, "*") {
			regexPattern += ".*"
		}
		regexPattern += "$"

		regex, err := regexp.Compile(regexPattern)
		if err != nil {
			return cp, err
		}
		cp.regex = regex

	case MatchRegex:
		// Check for path parameters like {id}
		paramPattern := regexp.MustCompile(`\{([^}]+)\}`)
		params := paramPattern.FindAllStringSubmatch(pattern, -1)
		for _, p := range params {
			cp.paramNames = append(cp.paramNames, p[1])
		}

		// Convert {param} to named capture groups
		regexPattern := paramPattern.ReplaceAllString(pattern, `(?P<$1>[^/]+)`)
		if !strings.HasPrefix(regexPattern, "^") {
			regexPattern = "^" + regexPattern
		}
		if !strings.HasSuffix(regexPattern, "$") {
			regexPattern += "$"
		}

		regex, err := regexp.Compile(regexPattern)
		if err != nil {
			return cp, err
		}
		cp.regex = regex
	}

	return cp, nil
}

// compileHostPattern compiles the host pattern into the compiledPattern struct.
func compileHostPattern(cp *compiledPattern, hostPattern string, hostMatchType HostMatchType) error {
	if hostPattern == "" {
		return nil // No host matching
	}

	// Infer match type from pattern if not specified
	// This ensures host patterns are always respected even if match type is empty
	if hostMatchType == HostMatchNone {
		if strings.HasPrefix(hostPattern, "*.") {
			hostMatchType = HostMatchWildcard
		} else {
			hostMatchType = HostMatchExact
		}
	}

	switch hostMatchType {
	case HostMatchExact:
		// Store lowercase for case-insensitive comparison
		cp.hostExact = strings.ToLower(hostPattern)

	case HostMatchWildcard:
		// *.example.com -> store ".example.com" as suffix
		// Must start with *. for wildcard matching
		if !strings.HasPrefix(hostPattern, "*.") {
			return fmt.Errorf("invalid wildcard pattern %q: must start with *.", hostPattern)
		}
		cp.hostWildcard = strings.ToLower(hostPattern[1:]) // Store ".example.com"

	case HostMatchRegex:
		// Compile regex pattern
		regex, err := regexp.Compile("(?i)" + hostPattern) // Case-insensitive
		if err != nil {
			return err
		}
		cp.hostRegex = regex
	}

	return nil
}

// Match finds the first matching route for the given request.
// Returns nil if no route matches.
// Matching order: host -> method -> path -> headers
func (m *Matcher) Match(method, path string, headers map[string]string) *MatchResult {
	// Extract and normalize host from headers
	host := normalizeHost(headers["Host"])

	for _, cp := range m.patterns {
		route := &m.routes[cp.routeIdx]

		// 1. Check host match (first for multi-tenant routing)
		if !matchHost(cp, host) {
			continue
		}

		// 2. Check method match
		if !matchMethod(route.Methods, method) {
			continue
		}

		// 3. Check path match
		pathParams := matchPath(cp, path)
		if pathParams == nil {
			continue
		}

		// 4. Check header conditions
		if !matchHeaders(route.Headers, headers) {
			continue
		}

		return &MatchResult{
			Route:      route,
			PathParams: pathParams,
		}
	}

	return nil
}

// normalizeHost normalizes the host header value.
// Removes port, trailing dots, and converts to lowercase.
func normalizeHost(host string) string {
	if host == "" {
		return ""
	}

	// Remove port if present (e.g., "api.example.com:8080" -> "api.example.com")
	if colonIdx := strings.LastIndex(host, ":"); colonIdx != -1 {
		// Check if this is an IPv6 address (has brackets)
		if !strings.Contains(host, "]") || colonIdx > strings.Index(host, "]") {
			host = host[:colonIdx]
		}
	}

	// Remove trailing dot (e.g., "api.example.com." -> "api.example.com")
	host = strings.TrimSuffix(host, ".")

	// Convert to lowercase for case-insensitive matching
	return strings.ToLower(host)
}

// matchHost checks if the host matches the compiled host pattern.
// Returns true if: no host pattern is configured (matches any host),
// or the host matches the pattern.
func matchHost(cp compiledPattern, host string) bool {
	// No host matching configured - matches any host (backward compatible)
	if cp.hostExact == "" && cp.hostWildcard == "" && cp.hostRegex == nil {
		return true
	}

	// Exact match
	if cp.hostExact != "" {
		return host == cp.hostExact
	}

	// Wildcard match (*.example.com)
	if cp.hostWildcard != "" {
		// Host must have exactly one more segment than the wildcard suffix
		// e.g., "api.example.com" matches "*.example.com" but "a.b.example.com" does not
		if !strings.HasSuffix(host, cp.hostWildcard) {
			return false
		}
		// Check there's exactly one segment before the suffix
		prefix := host[:len(host)-len(cp.hostWildcard)]
		return prefix != "" && !strings.Contains(prefix, ".")
	}

	// Regex match
	if cp.hostRegex != nil {
		return cp.hostRegex.MatchString(host)
	}

	return false
}

// matchMethod checks if the request method matches the route's allowed methods.
// Empty methods slice means all methods are allowed.
func matchMethod(allowed []string, method string) bool {
	if len(allowed) == 0 {
		return true
	}
	method = strings.ToUpper(method)
	for _, m := range allowed {
		if strings.ToUpper(m) == method {
			return true
		}
	}
	return false
}

// matchPath checks if the path matches the pattern.
// Returns path parameters if matched, nil if no match.
func matchPath(cp compiledPattern, path string) map[string]string {
	params := make(map[string]string)

	if cp.exact != "" {
		if path == cp.exact {
			return params
		}
		return nil
	}

	if cp.regex != nil {
		matches := cp.regex.FindStringSubmatch(path)
		if matches == nil {
			return nil
		}

		// Extract named groups
		for i, name := range cp.regex.SubexpNames() {
			if i > 0 && name != "" && i < len(matches) {
				params[name] = matches[i]
			}
		}
		return params
	}

	return nil
}

// matchHeaders checks if all header conditions are satisfied.
func matchHeaders(conditions []HeaderMatch, headers map[string]string) bool {
	for _, cond := range conditions {
		value, exists := headers[cond.Name]

		if cond.Required && !exists {
			return false
		}

		if !exists {
			continue
		}

		if cond.IsRegex {
			regex, err := regexp.Compile(cond.Value)
			if err != nil || !regex.MatchString(value) {
				return false
			}
		} else if value != cond.Value {
			return false
		}
	}
	return true
}

// FindByID returns a route by ID, or nil if not found.
func FindByID(routes []Route, id string) *Route {
	for i := range routes {
		if routes[i].ID == id {
			return &routes[i]
		}
	}
	return nil
}

// FindUpstreamByID returns an upstream by ID, or nil if not found.
func FindUpstreamByID(upstreams []Upstream, id string) *Upstream {
	for i := range upstreams {
		if upstreams[i].ID == id {
			return &upstreams[i]
		}
	}
	return nil
}

// FilterEnabled returns only enabled routes.
func FilterEnabled(routes []Route) []Route {
	result := make([]Route, 0, len(routes))
	for _, r := range routes {
		if r.Enabled {
			result = append(result, r)
		}
	}
	return result
}

// SortByPriority sorts routes by priority (highest first).
func SortByPriority(routes []Route) {
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Priority > routes[j].Priority
	})
}
