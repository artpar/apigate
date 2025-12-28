// Package app provides application services that orchestrate domain logic.
package app

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/artpar/apigate/domain/proxy"
	"github.com/artpar/apigate/domain/route"
	"github.com/artpar/apigate/domain/streaming"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// TransformService applies Expr-based transformations to requests and responses.
type TransformService struct {
	// Compiled program cache
	cache   map[string]*vm.Program
	cacheMu sync.RWMutex

	// Expr environment options with custom functions
	envOptions []expr.Option
}

// NewTransformService creates a new transform service with custom Expr functions.
func NewTransformService() *TransformService {
	s := &TransformService{
		cache: make(map[string]*vm.Program),
	}

	// Register custom functions available in all expressions
	s.envOptions = []expr.Option{
		// String functions
		expr.Function("lower", func(params ...any) (any, error) {
			if len(params) != 1 {
				return nil, fmt.Errorf("lower requires 1 argument")
			}
			return strings.ToLower(toString(params[0])), nil
		}),
		expr.Function("upper", func(params ...any) (any, error) {
			if len(params) != 1 {
				return nil, fmt.Errorf("upper requires 1 argument")
			}
			return strings.ToUpper(toString(params[0])), nil
		}),
		expr.Function("trim", func(params ...any) (any, error) {
			if len(params) != 1 {
				return nil, fmt.Errorf("trim requires 1 argument")
			}
			return strings.TrimSpace(toString(params[0])), nil
		}),
		expr.Function("trimPrefix", func(params ...any) (any, error) {
			if len(params) != 2 {
				return nil, fmt.Errorf("trimPrefix requires 2 arguments")
			}
			return strings.TrimPrefix(toString(params[0]), toString(params[1])), nil
		}),
		expr.Function("trimSuffix", func(params ...any) (any, error) {
			if len(params) != 2 {
				return nil, fmt.Errorf("trimSuffix requires 2 arguments")
			}
			return strings.TrimSuffix(toString(params[0]), toString(params[1])), nil
		}),
		expr.Function("replace", func(params ...any) (any, error) {
			if len(params) != 3 {
				return nil, fmt.Errorf("replace requires 3 arguments (str, old, new)")
			}
			return strings.ReplaceAll(toString(params[0]), toString(params[1]), toString(params[2])), nil
		}),
		expr.Function("split", func(params ...any) (any, error) {
			if len(params) != 2 {
				return nil, fmt.Errorf("split requires 2 arguments")
			}
			return strings.Split(toString(params[0]), toString(params[1])), nil
		}),
		expr.Function("join", func(params ...any) (any, error) {
			if len(params) != 2 {
				return nil, fmt.Errorf("join requires 2 arguments")
			}
			arr, ok := params[0].([]string)
			if !ok {
				// Try to convert []any to []string
				anyArr, ok := params[0].([]any)
				if !ok {
					return nil, fmt.Errorf("join first argument must be array")
				}
				arr = make([]string, len(anyArr))
				for i, v := range anyArr {
					arr[i] = toString(v)
				}
			}
			return strings.Join(arr, toString(params[1])), nil
		}),

		// Encoding functions
		expr.Function("base64Encode", func(params ...any) (any, error) {
			if len(params) != 1 {
				return nil, fmt.Errorf("base64Encode requires 1 argument")
			}
			return base64.StdEncoding.EncodeToString([]byte(toString(params[0]))), nil
		}),
		expr.Function("base64Decode", func(params ...any) (any, error) {
			if len(params) != 1 {
				return nil, fmt.Errorf("base64Decode requires 1 argument")
			}
			decoded, err := base64.StdEncoding.DecodeString(toString(params[0]))
			if err != nil {
				return nil, err
			}
			return string(decoded), nil
		}),
		expr.Function("urlEncode", func(params ...any) (any, error) {
			if len(params) != 1 {
				return nil, fmt.Errorf("urlEncode requires 1 argument")
			}
			return url.QueryEscape(toString(params[0])), nil
		}),
		expr.Function("urlDecode", func(params ...any) (any, error) {
			if len(params) != 1 {
				return nil, fmt.Errorf("urlDecode requires 1 argument")
			}
			decoded, err := url.QueryUnescape(toString(params[0]))
			if err != nil {
				return nil, err
			}
			return decoded, nil
		}),
		expr.Function("jsonEncode", func(params ...any) (any, error) {
			if len(params) != 1 {
				return nil, fmt.Errorf("jsonEncode requires 1 argument")
			}
			b, err := json.Marshal(params[0])
			if err != nil {
				return nil, err
			}
			return string(b), nil
		}),
		expr.Function("jsonDecode", func(params ...any) (any, error) {
			if len(params) != 1 {
				return nil, fmt.Errorf("jsonDecode requires 1 argument")
			}
			var result any
			if err := json.Unmarshal([]byte(toString(params[0])), &result); err != nil {
				return nil, err
			}
			return result, nil
		}),

		// Crypto functions
		expr.Function("sha256", func(params ...any) (any, error) {
			if len(params) != 1 {
				return nil, fmt.Errorf("sha256 requires 1 argument")
			}
			h := sha256.Sum256([]byte(toString(params[0])))
			return hex.EncodeToString(h[:]), nil
		}),
		expr.Function("hmacSha256", func(params ...any) (any, error) {
			if len(params) != 2 {
				return nil, fmt.Errorf("hmacSha256 requires 2 arguments (data, key)")
			}
			mac := hmac.New(sha256.New, []byte(toString(params[1])))
			mac.Write([]byte(toString(params[0])))
			return hex.EncodeToString(mac.Sum(nil)), nil
		}),

		// Environment and utilities
		expr.Function("env", func(params ...any) (any, error) {
			if len(params) != 1 {
				return nil, fmt.Errorf("env requires 1 argument")
			}
			return os.Getenv(toString(params[0])), nil
		}),
		expr.Function("now", func(params ...any) (any, error) {
			return time.Now().Unix(), nil
		}),
		expr.Function("nowRFC3339", func(params ...any) (any, error) {
			return time.Now().Format(time.RFC3339), nil
		}),
		expr.Function("coalesce", func(params ...any) (any, error) {
			for _, p := range params {
				if p != nil && p != "" {
					return p, nil
				}
			}
			return nil, nil
		}),
		expr.Function("default", func(params ...any) (any, error) {
			if len(params) != 2 {
				return nil, fmt.Errorf("default requires 2 arguments (value, defaultValue)")
			}
			if params[0] == nil || params[0] == "" {
				return params[1], nil
			}
			return params[0], nil
		}),

		// Type conversion
		expr.Function("toString", func(params ...any) (any, error) {
			if len(params) != 1 {
				return nil, fmt.Errorf("toString requires 1 argument")
			}
			return toString(params[0]), nil
		}),
		expr.Function("toInt", func(params ...any) (any, error) {
			if len(params) != 1 {
				return nil, fmt.Errorf("toInt requires 1 argument")
			}
			return toInt(params[0]), nil
		}),
		expr.Function("toFloat", func(params ...any) (any, error) {
			if len(params) != 1 {
				return nil, fmt.Errorf("toFloat requires 1 argument")
			}
			return toFloat(params[0]), nil
		}),

		// Data parsing functions (for metering from streaming responses)

		// json(data) - Parse JSON from bytes or string. Alias for jsonDecode.
		expr.Function("json", func(params ...any) (any, error) {
			if len(params) != 1 {
				return nil, fmt.Errorf("json requires 1 argument")
			}
			data := toBytes(params[0])
			if len(data) == 0 {
				return nil, nil
			}
			var result any
			if err := json.Unmarshal(data, &result); err != nil {
				return nil, err
			}
			return result, nil
		}),

		// lines(data) - Split bytes/string into array of lines
		expr.Function("lines", func(params ...any) (any, error) {
			if len(params) != 1 {
				return nil, fmt.Errorf("lines requires 1 argument")
			}
			data := toBytes(params[0])
			return streaming.SplitLines(data), nil
		}),

		// linesNonEmpty(data) - Split into non-empty lines only
		expr.Function("linesNonEmpty", func(params ...any) (any, error) {
			if len(params) != 1 {
				return nil, fmt.Errorf("linesNonEmpty requires 1 argument")
			}
			data := toBytes(params[0])
			return streaming.SplitLinesNonEmpty(data), nil
		}),

		// sseEvents(data) - Parse SSE format into array of {event, data, id} objects
		expr.Function("sseEvents", func(params ...any) (any, error) {
			if len(params) != 1 {
				return nil, fmt.Errorf("sseEvents requires 1 argument")
			}
			data := toBytes(params[0])
			events := streaming.ParseSSEEvents(data)
			// Convert to []any for Expr compatibility
			result := make([]any, len(events))
			for i, e := range events {
				result[i] = map[string]any{
					"event": e.Event,
					"data":  e.Data,
					"id":    e.ID,
				}
			}
			return result, nil
		}),

		// sseLastData(data) - Get the data field from the last SSE event
		expr.Function("sseLastData", func(params ...any) (any, error) {
			if len(params) != 1 {
				return nil, fmt.Errorf("sseLastData requires 1 argument")
			}
			data := toBytes(params[0])
			return streaming.ExtractSSELastData(data), nil
		}),

		// sseAllData(data) - Get all SSE data fields concatenated
		expr.Function("sseAllData", func(params ...any) (any, error) {
			if len(params) != 1 {
				return nil, fmt.Errorf("sseAllData requires 1 argument")
			}
			data := toBytes(params[0])
			return streaming.ExtractSSEData(data), nil
		}),

		// last(array) - Get the last element of an array
		expr.Function("last", func(params ...any) (any, error) {
			if len(params) != 1 {
				return nil, fmt.Errorf("last requires 1 argument")
			}
			arr, ok := toSlice(params[0])
			if !ok || len(arr) == 0 {
				return nil, nil
			}
			return arr[len(arr)-1], nil
		}),

		// first(array) - Get the first element of an array
		expr.Function("first", func(params ...any) (any, error) {
			if len(params) != 1 {
				return nil, fmt.Errorf("first requires 1 argument")
			}
			arr, ok := toSlice(params[0])
			if !ok || len(arr) == 0 {
				return nil, nil
			}
			return arr[0], nil
		}),

		// sum(array) or sum(array, field) - Sum numbers or sum a field across objects
		expr.Function("sum", func(params ...any) (any, error) {
			if len(params) < 1 || len(params) > 2 {
				return nil, fmt.Errorf("sum requires 1 or 2 arguments")
			}
			arr, ok := toSlice(params[0])
			if !ok {
				return 0.0, nil
			}

			var total float64
			if len(params) == 1 {
				// Sum numbers directly
				for _, v := range arr {
					total += toFloat(v)
				}
			} else {
				// Sum a field from objects
				field := toString(params[1])
				for _, v := range arr {
					if m, ok := v.(map[string]any); ok {
						if val, exists := m[field]; exists {
							total += toFloat(val)
						}
					}
				}
			}
			return total, nil
		}),

		// get(obj, path) - Safely get nested field with dot notation
		expr.Function("get", func(params ...any) (any, error) {
			if len(params) != 2 {
				return nil, fmt.Errorf("get requires 2 arguments (obj, path)")
			}
			obj := params[0]
			path := toString(params[1])

			parts := strings.Split(path, ".")
			current := obj
			for _, part := range parts {
				if m, ok := current.(map[string]any); ok {
					current = m[part]
				} else {
					return nil, nil
				}
			}
			return current, nil
		}),

		// count(array) - Count elements in array
		expr.Function("count", func(params ...any) (any, error) {
			if len(params) != 1 {
				return nil, fmt.Errorf("count requires 1 argument")
			}
			arr, ok := toSlice(params[0])
			if !ok {
				return 0, nil
			}
			return len(arr), nil
		}),

		// avg(array) or avg(array, field) - Average of numbers or field values
		expr.Function("avg", func(params ...any) (any, error) {
			if len(params) < 1 || len(params) > 2 {
				return nil, fmt.Errorf("avg requires 1 or 2 arguments")
			}
			arr, ok := toSlice(params[0])
			if !ok || len(arr) == 0 {
				return 0.0, nil
			}

			var total float64
			if len(params) == 1 {
				for _, v := range arr {
					total += toFloat(v)
				}
			} else {
				field := toString(params[1])
				for _, v := range arr {
					if m, ok := v.(map[string]any); ok {
						if val, exists := m[field]; exists {
							total += toFloat(val)
						}
					}
				}
			}
			return total / float64(len(arr)), nil
		}),

		// max(array) or max(array, field) - Maximum value
		expr.Function("max", func(params ...any) (any, error) {
			if len(params) < 1 || len(params) > 2 {
				return nil, fmt.Errorf("max requires 1 or 2 arguments")
			}
			arr, ok := toSlice(params[0])
			if !ok || len(arr) == 0 {
				return nil, nil
			}

			var maxVal float64
			first := true
			if len(params) == 1 {
				for _, v := range arr {
					val := toFloat(v)
					if first || val > maxVal {
						maxVal = val
						first = false
					}
				}
			} else {
				field := toString(params[1])
				for _, v := range arr {
					if m, ok := v.(map[string]any); ok {
						if fv, exists := m[field]; exists {
							val := toFloat(fv)
							if first || val > maxVal {
								maxVal = val
								first = false
							}
						}
					}
				}
			}
			return maxVal, nil
		}),

		// min(array) or min(array, field) - Minimum value
		expr.Function("min", func(params ...any) (any, error) {
			if len(params) < 1 || len(params) > 2 {
				return nil, fmt.Errorf("min requires 1 or 2 arguments")
			}
			arr, ok := toSlice(params[0])
			if !ok || len(arr) == 0 {
				return nil, nil
			}

			var minVal float64
			first := true
			if len(params) == 1 {
				for _, v := range arr {
					val := toFloat(v)
					if first || val < minVal {
						minVal = val
						first = false
					}
				}
			} else {
				field := toString(params[1])
				for _, v := range arr {
					if m, ok := v.(map[string]any); ok {
						if fv, exists := m[field]; exists {
							val := toFloat(fv)
							if first || val < minVal {
								minVal = val
								first = false
							}
						}
					}
				}
			}
			return minVal, nil
		}),
	}

	return s
}

// TransformContext contains data available to Expr expressions.
type TransformContext struct {
	// Request fields
	Method   string            `expr:"method"`
	Path     string            `expr:"path"`
	Query    map[string]string `expr:"query"`
	Headers  map[string]string `expr:"headers"`
	Body     any               `expr:"body"`    // Parsed JSON body
	RawBody  []byte            `expr:"rawBody"` // Raw body bytes

	// Auth context
	UserID string `expr:"userID"`
	PlanID string `expr:"planID"`
	KeyID  string `expr:"keyID"`

	// Response fields (for response transforms)
	Status      int               `expr:"status"`
	RespHeaders map[string]string `expr:"respHeaders"`
	RespBody    any               `expr:"respBody"` // Parsed JSON response

	// Metering
	ResponseBytes int64 `expr:"responseBytes"`
}

// TransformRequest applies request transformations.
func (s *TransformService) TransformRequest(
	ctx context.Context,
	req proxy.Request,
	transform *route.Transform,
	auth *proxy.AuthContext,
) (proxy.Request, error) {
	if transform == nil {
		return req, nil
	}

	// Build transform context
	tctx := s.buildRequestContext(req, auth)

	// Apply header deletions
	for _, h := range transform.DeleteHeaders {
		delete(req.Headers, h)
	}

	// Apply header sets/computations
	for name, valueExpr := range transform.SetHeaders {
		value, err := s.EvalString(ctx, valueExpr, tctx)
		if err != nil {
			return req, fmt.Errorf("eval header %s: %w", name, err)
		}
		if req.Headers == nil {
			req.Headers = make(map[string]string)
		}
		req.Headers[name] = value
	}

	// Apply query deletions
	if len(transform.DeleteQuery) > 0 {
		q, _ := url.ParseQuery(req.Query)
		for _, key := range transform.DeleteQuery {
			q.Del(key)
		}
		req.Query = q.Encode()
	}

	// Apply query sets
	if len(transform.SetQuery) > 0 {
		q, _ := url.ParseQuery(req.Query)
		for name, valueExpr := range transform.SetQuery {
			value, err := s.EvalString(ctx, valueExpr, tctx)
			if err != nil {
				return req, fmt.Errorf("eval query %s: %w", name, err)
			}
			q.Set(name, value)
		}
		req.Query = q.Encode()
	}

	// Apply body transformation
	if transform.BodyExpr != "" {
		result, err := s.Eval(ctx, transform.BodyExpr, tctx)
		if err != nil {
			return req, fmt.Errorf("eval body: %w", err)
		}

		// Convert result to JSON
		newBody, err := json.Marshal(result)
		if err != nil {
			return req, fmt.Errorf("marshal body: %w", err)
		}
		req.Body = newBody

		// Update Content-Type if body was transformed
		if req.Headers == nil {
			req.Headers = make(map[string]string)
		}
		req.Headers["Content-Type"] = "application/json"
	}

	return req, nil
}

// TransformResponse applies response transformations.
func (s *TransformService) TransformResponse(
	ctx context.Context,
	resp proxy.Response,
	transform *route.Transform,
	auth *proxy.AuthContext,
) (proxy.Response, error) {
	if transform == nil {
		return resp, nil
	}

	// Build transform context with response data
	tctx := s.buildResponseContext(resp, auth)

	// Apply header deletions
	for _, h := range transform.DeleteHeaders {
		delete(resp.Headers, h)
	}

	// Apply header sets/computations
	for name, valueExpr := range transform.SetHeaders {
		value, err := s.EvalString(ctx, valueExpr, tctx)
		if err != nil {
			return resp, fmt.Errorf("eval header %s: %w", name, err)
		}
		if resp.Headers == nil {
			resp.Headers = make(map[string]string)
		}
		resp.Headers[name] = value
	}

	// Apply body transformation
	if transform.BodyExpr != "" {
		result, err := s.Eval(ctx, transform.BodyExpr, tctx)
		if err != nil {
			return resp, fmt.Errorf("eval body: %w", err)
		}

		// Convert result to JSON
		newBody, err := json.Marshal(result)
		if err != nil {
			return resp, fmt.Errorf("marshal body: %w", err)
		}
		resp.Body = newBody

		// Update Content-Type
		if resp.Headers == nil {
			resp.Headers = make(map[string]string)
		}
		resp.Headers["Content-Type"] = "application/json"
	}

	return resp, nil
}

// EvalString evaluates an Expr expression and returns a string.
func (s *TransformService) EvalString(ctx context.Context, expression string, data any) (string, error) {
	result, err := s.Eval(ctx, expression, data)
	if err != nil {
		return "", err
	}
	return toString(result), nil
}

// EvalFloat evaluates an Expr expression and returns a float64.
func (s *TransformService) EvalFloat(ctx context.Context, expression string, data any) (float64, error) {
	result, err := s.Eval(ctx, expression, data)
	if err != nil {
		return 0, err
	}
	return toFloat(result), nil
}

// Eval evaluates an Expr expression with the given data context.
func (s *TransformService) Eval(ctx context.Context, expression string, data any) (any, error) {
	program, err := s.getOrCompile(expression, data)
	if err != nil {
		return nil, fmt.Errorf("compile expression: %w", err)
	}

	result, err := expr.Run(program, data)
	if err != nil {
		return nil, fmt.Errorf("run expression: %w", err)
	}

	return result, nil
}

// getOrCompile returns a cached compiled program or compiles a new one.
func (s *TransformService) getOrCompile(expression string, env any) (*vm.Program, error) {
	// Check cache first
	s.cacheMu.RLock()
	program, ok := s.cache[expression]
	s.cacheMu.RUnlock()

	if ok {
		return program, nil
	}

	// Compile with environment
	opts := append([]expr.Option{expr.Env(env)}, s.envOptions...)
	program, err := expr.Compile(expression, opts...)
	if err != nil {
		return nil, err
	}

	// Cache the compiled program
	s.cacheMu.Lock()
	s.cache[expression] = program
	s.cacheMu.Unlock()

	return program, nil
}

// buildRequestContext creates a TransformContext from a request.
func (s *TransformService) buildRequestContext(req proxy.Request, auth *proxy.AuthContext) map[string]any {
	ctx := map[string]any{
		"method":   req.Method,
		"path":     req.Path,
		"query":    parseQuery(req.Query),
		"headers":  req.Headers,
		"rawBody":  req.Body,
		"userID":   "",
		"planID":   "",
		"keyID":    "",
	}

	// Parse body as JSON if possible
	if len(req.Body) > 0 {
		var body any
		if err := json.Unmarshal(req.Body, &body); err == nil {
			ctx["body"] = body
		}
	}

	if auth != nil {
		ctx["userID"] = auth.UserID
		ctx["planID"] = auth.PlanID
		ctx["keyID"] = auth.KeyID
	}

	return ctx
}

// buildResponseContext creates a TransformContext from a response.
func (s *TransformService) buildResponseContext(resp proxy.Response, auth *proxy.AuthContext) map[string]any {
	ctx := map[string]any{
		"status":        resp.Status,
		"respHeaders":   resp.Headers,
		"responseBytes": int64(len(resp.Body)),
		"userID":        "",
		"planID":        "",
		"keyID":         "",
	}

	// Parse body as JSON if possible
	if len(resp.Body) > 0 {
		var body any
		if err := json.Unmarshal(resp.Body, &body); err == nil {
			ctx["respBody"] = body
		}
	}

	if auth != nil {
		ctx["userID"] = auth.UserID
		ctx["planID"] = auth.PlanID
		ctx["keyID"] = auth.KeyID
	}

	return ctx
}

// Helper functions

func toString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	case fmt.Stringer:
		return val.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func toInt(v any) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case string:
		var i int
		fmt.Sscanf(val, "%d", &i)
		return i
	default:
		return 0
	}
}

func toFloat(v any) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		var f float64
		fmt.Sscanf(val, "%f", &f)
		return f
	default:
		return 0
	}
}

func parseQuery(query string) map[string]string {
	result := make(map[string]string)
	values, _ := url.ParseQuery(query)
	for k, v := range values {
		if len(v) > 0 {
			result[k] = v[0]
		}
	}
	return result
}

func toBytes(v any) []byte {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []byte:
		return val
	case string:
		return []byte(val)
	default:
		return []byte(fmt.Sprintf("%v", v))
	}
}

func toSlice(v any) ([]any, bool) {
	if v == nil {
		return nil, false
	}
	switch val := v.(type) {
	case []any:
		return val, true
	case []string:
		result := make([]any, len(val))
		for i, s := range val {
			result[i] = s
		}
		return result, true
	case []map[string]any:
		result := make([]any, len(val))
		for i, m := range val {
			result[i] = m
		}
		return result, true
	default:
		return nil, false
	}
}

// ClearCache clears the compiled expression cache.
// Useful after configuration changes.
func (s *TransformService) ClearCache() {
	s.cacheMu.Lock()
	s.cache = make(map[string]*vm.Program)
	s.cacheMu.Unlock()
}
