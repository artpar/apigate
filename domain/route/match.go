package route

import (
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
	routeIdx int
	regex    *regexp.Regexp   // For regex and prefix patterns
	exact    string           // For exact patterns
	paramNames []string       // Names of path parameters
}

// NewMatcher creates a new Matcher from a list of routes.
// Routes are sorted by priority (highest first) and patterns are compiled.
func NewMatcher(routes []Route) (*Matcher, error) {
	// Sort routes by priority descending, then by specificity
	sorted := make([]Route, len(routes))
	copy(sorted, routes)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Priority != sorted[j].Priority {
			return sorted[i].Priority > sorted[j].Priority
		}
		// More specific patterns first (longer path, exact before prefix)
		if sorted[i].MatchType != sorted[j].MatchType {
			return matchTypePriority(sorted[i].MatchType) > matchTypePriority(sorted[j].MatchType)
		}
		return len(sorted[i].PathPattern) > len(sorted[j].PathPattern)
	})

	patterns := make([]compiledPattern, 0, len(sorted))
	for i, r := range sorted {
		if !r.Enabled {
			continue
		}

		cp, err := compilePattern(r.PathPattern, r.MatchType, i)
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

func compilePattern(pattern string, matchType MatchType, routeIdx int) (compiledPattern, error) {
	cp := compiledPattern{
		routeIdx: routeIdx,
	}

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

// Match finds the first matching route for the given request.
// Returns nil if no route matches.
func (m *Matcher) Match(method, path string, headers map[string]string) *MatchResult {
	for _, cp := range m.patterns {
		route := &m.routes[cp.routeIdx]

		// Check method match
		if !matchMethod(route.Methods, method) {
			continue
		}

		// Check path match
		pathParams := matchPath(cp, path)
		if pathParams == nil {
			continue
		}

		// Check header conditions
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
