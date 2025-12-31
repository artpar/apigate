package app_test

import (
	"context"
	"os"
	"testing"

	"github.com/artpar/apigate/app"
	"github.com/artpar/apigate/domain/proxy"
	"github.com/artpar/apigate/domain/route"
)

func TestTransformService_EvalString(t *testing.T) {
	svc := app.NewTransformService()
	ctx := context.Background()

	tests := []struct {
		name    string
		expr    string
		data    map[string]any
		want    string
		wantErr bool
	}{
		{
			"simple string",
			`"hello"`,
			nil,
			"hello",
			false,
		},
		{
			"string concatenation",
			`"Hello, " + name`,
			map[string]any{"name": "World"},
			"Hello, World",
			false,
		},
		{
			"path manipulation",
			`"/v2" + trimPrefix(path, "/v1")`,
			map[string]any{"path": "/v1/users"},
			"/v2/users",
			false,
		},
		{
			"lower function",
			`lower(text)`,
			map[string]any{"text": "HELLO"},
			"hello",
			false,
		},
		{
			"upper function",
			`upper(text)`,
			map[string]any{"text": "hello"},
			"HELLO",
			false,
		},
		{
			"trim function",
			`trim(text)`,
			map[string]any{"text": "  hello  "},
			"hello",
			false,
		},
		{
			"replace function",
			`replace(text, "old", "new")`,
			map[string]any{"text": "old value old"},
			"new value new",
			false,
		},
		{
			"base64 encode",
			`base64Encode("hello")`,
			nil,
			"aGVsbG8=",
			false,
		},
		{
			"base64 decode",
			`base64Decode("aGVsbG8=")`,
			nil,
			"hello",
			false,
		},
		{
			"url encode",
			`urlEncode("hello world")`,
			nil,
			"hello+world",
			false,
		},
		{
			"sha256",
			`sha256("test")`,
			nil,
			"9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
			false,
		},
		{
			"coalesce",
			`coalesce(null, "", "default")`,
			map[string]any{"null": nil},
			"default",
			false,
		},
		{
			"default function",
			`default(value, "fallback")`,
			map[string]any{"value": ""},
			"fallback",
			false,
		},
		{
			"toString",
			`toString(123)`,
			nil,
			"123",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.EvalString(ctx, tt.expr, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("got = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestTransformService_EvalFloat(t *testing.T) {
	svc := app.NewTransformService()
	ctx := context.Background()

	tests := []struct {
		name    string
		expr    string
		data    map[string]any
		want    float64
		wantErr bool
	}{
		{
			"simple number",
			`42`,
			nil,
			42.0,
			false,
		},
		{
			"arithmetic",
			`10 + 5 * 2`,
			nil,
			20.0,
			false,
		},
		{
			"division for metering",
			`responseBytes / 1000`,
			map[string]any{"responseBytes": int64(5000)},
			5.0,
			false,
		},
		{
			"conditional metering",
			`status < 400 ? units : 0`,
			map[string]any{"status": 200, "units": 10},
			10.0,
			false,
		},
		{
			"conditional metering error",
			`status < 400 ? units : 0`,
			map[string]any{"status": 500, "units": 10},
			0.0,
			false,
		},
		{
			"nested field access",
			`respBody.usage.tokens`,
			map[string]any{
				"respBody": map[string]any{
					"usage": map[string]any{
						"tokens": 100,
					},
				},
			},
			100.0,
			false,
		},
		{
			"toFloat",
			`toFloat("3.14")`,
			nil,
			3.14,
			false,
		},
		{
			"len for metering",
			`len(items)`,
			map[string]any{"items": []any{1, 2, 3, 4, 5}},
			5.0,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.EvalFloat(ctx, tt.expr, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("got = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestTransformService_EvalEnv(t *testing.T) {
	svc := app.NewTransformService()
	ctx := context.Background()

	// Set test env var
	os.Setenv("TEST_API_KEY", "secret123")
	defer os.Unsetenv("TEST_API_KEY")

	result, err := svc.EvalString(ctx, `env("TEST_API_KEY")`, nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "secret123" {
		t.Errorf("got = %s, want secret123", result)
	}
}

func TestTransformService_TransformRequest(t *testing.T) {
	svc := app.NewTransformService()
	ctx := context.Background()

	tests := []struct {
		name      string
		req       proxy.Request
		transform *route.Transform
		auth      *proxy.AuthContext
		check     func(t *testing.T, req proxy.Request)
	}{
		{
			"set headers",
			proxy.Request{
				Method:  "GET",
				Path:    "/api/data",
				Headers: map[string]string{"Existing": "value"},
			},
			&route.Transform{
				SetHeaders: map[string]string{
					"X-Custom":    `"custom-value"`,
					"X-User":      `"user_" + userID`,
					"X-Timestamp": `nowRFC3339()`,
				},
			},
			&proxy.AuthContext{UserID: "123"},
			func(t *testing.T, req proxy.Request) {
				if req.Headers["X-Custom"] != "custom-value" {
					t.Errorf("X-Custom = %s, want custom-value", req.Headers["X-Custom"])
				}
				if req.Headers["X-User"] != "user_123" {
					t.Errorf("X-User = %s, want user_123", req.Headers["X-User"])
				}
				if req.Headers["Existing"] != "value" {
					t.Errorf("Existing header should be preserved")
				}
			},
		},
		{
			"delete headers",
			proxy.Request{
				Headers: map[string]string{
					"Keep":   "value1",
					"Remove": "value2",
				},
			},
			&route.Transform{
				DeleteHeaders: []string{"Remove"},
			},
			nil,
			func(t *testing.T, req proxy.Request) {
				if _, ok := req.Headers["Remove"]; ok {
					t.Error("Remove header should be deleted")
				}
				if req.Headers["Keep"] != "value1" {
					t.Error("Keep header should be preserved")
				}
			},
		},
		{
			"set query params",
			proxy.Request{
				Query: "existing=value",
			},
			&route.Transform{
				SetQuery: map[string]string{
					"added": `"new-value"`,
				},
			},
			nil,
			func(t *testing.T, req proxy.Request) {
				if req.Query != "added=new-value&existing=value" && req.Query != "existing=value&added=new-value" {
					t.Errorf("unexpected query: %s", req.Query)
				}
			},
		},
		{
			"delete query params",
			proxy.Request{
				Query: "keep=value1&remove=value2",
			},
			&route.Transform{
				DeleteQuery: []string{"remove"},
			},
			nil,
			func(t *testing.T, req proxy.Request) {
				if req.Query != "keep=value1" {
					t.Errorf("query = %s, want keep=value1", req.Query)
				}
			},
		},
		{
			"transform body",
			proxy.Request{
				Body: []byte(`{"original": "data"}`),
			},
			&route.Transform{
				BodyExpr: `{"wrapped": body, "extra": "added"}`,
			},
			nil,
			func(t *testing.T, req proxy.Request) {
				// Body should be transformed
				if req.Headers["Content-Type"] != "application/json" {
					t.Errorf("Content-Type should be set to application/json")
				}
			},
		},
		{
			"nil transform",
			proxy.Request{
				Method:  "GET",
				Path:    "/api/data",
				Headers: map[string]string{"X-Test": "value"},
			},
			nil,
			nil,
			func(t *testing.T, req proxy.Request) {
				if req.Headers["X-Test"] != "value" {
					t.Error("request should be unchanged")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.TransformRequest(ctx, tt.req, tt.transform, tt.auth)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			tt.check(t, result)
		})
	}
}

func TestTransformService_TransformResponse(t *testing.T) {
	svc := app.NewTransformService()
	ctx := context.Background()

	tests := []struct {
		name      string
		resp      proxy.Response
		transform *route.Transform
		auth      *proxy.AuthContext
		check     func(t *testing.T, resp proxy.Response)
	}{
		{
			"set response headers",
			proxy.Response{
				Status:  200,
				Headers: map[string]string{},
			},
			&route.Transform{
				SetHeaders: map[string]string{
					"X-Response-ID": `"resp-123"`,
				},
			},
			nil,
			func(t *testing.T, resp proxy.Response) {
				if resp.Headers["X-Response-ID"] != "resp-123" {
					t.Errorf("X-Response-ID = %s, want resp-123", resp.Headers["X-Response-ID"])
				}
			},
		},
		{
			"delete response headers",
			proxy.Response{
				Status: 200,
				Headers: map[string]string{
					"Keep":           "value",
					"X-Internal":     "secret",
					"X-Debug":        "info",
				},
			},
			&route.Transform{
				DeleteHeaders: []string{"X-Internal", "X-Debug"},
			},
			nil,
			func(t *testing.T, resp proxy.Response) {
				if _, ok := resp.Headers["X-Internal"]; ok {
					t.Error("X-Internal should be deleted")
				}
				if _, ok := resp.Headers["X-Debug"]; ok {
					t.Error("X-Debug should be deleted")
				}
				if resp.Headers["Keep"] != "value" {
					t.Error("Keep should be preserved")
				}
			},
		},
		{
			"wrap response body",
			proxy.Response{
				Status: 200,
				Body:   []byte(`{"data": "test"}`),
			},
			&route.Transform{
				BodyExpr: `{"success": true, "result": respBody}`,
			},
			nil,
			func(t *testing.T, resp proxy.Response) {
				if resp.Headers["Content-Type"] != "application/json" {
					t.Errorf("Content-Type should be application/json")
				}
			},
		},
		{
			"nil transform",
			proxy.Response{
				Status: 200,
				Body:   []byte(`original`),
			},
			nil,
			nil,
			func(t *testing.T, resp proxy.Response) {
				if string(resp.Body) != "original" {
					t.Error("response should be unchanged")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.TransformResponse(ctx, tt.resp, tt.transform, tt.auth)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			tt.check(t, result)
		})
	}
}

func TestTransformService_CacheCompilation(t *testing.T) {
	svc := app.NewTransformService()
	ctx := context.Background()

	// Same expression should use cached compilation
	expr := `"Hello, " + name`
	data := map[string]any{"name": "World"}

	// First call compiles
	result1, err := svc.EvalString(ctx, expr, data)
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}

	// Second call uses cache
	result2, err := svc.EvalString(ctx, expr, data)
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}

	if result1 != result2 {
		t.Errorf("results differ: %s vs %s", result1, result2)
	}

	// Clear cache and verify still works
	svc.ClearCache()

	result3, err := svc.EvalString(ctx, expr, data)
	if err != nil {
		t.Fatalf("after cache clear error: %v", err)
	}

	if result3 != "Hello, World" {
		t.Errorf("got %s, want Hello, World", result3)
	}
}

func TestTransformService_ExprFunctions(t *testing.T) {
	svc := app.NewTransformService()
	ctx := context.Background()

	tests := []struct {
		name string
		expr string
		data map[string]any
		want any
	}{
		{
			"split and join",
			`join(split("a,b,c", ","), "-")`,
			nil,
			"a-b-c",
		},
		{
			"trimSuffix",
			`trimSuffix("/api/v1/", "/")`,
			nil,
			"/api/v1",
		},
		{
			"hmacSha256",
			`hmacSha256("data", "secret")`,
			nil,
			"1b2c16b75bd2a870c114153ccda5bcfca63314bc722fa160d690de133ccbb9db",
		},
		{
			"jsonEncode",
			`jsonEncode(data)`,
			map[string]any{"data": map[string]any{"key": "value"}},
			`{"key":"value"}`,
		},
		{
			"toInt",
			`toInt("42")`,
			nil,
			42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.Eval(ctx, tt.expr, tt.data)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			// Compare as strings for simplicity
			if result != tt.want {
				t.Errorf("got %v, want %v", result, tt.want)
			}
		})
	}
}

func TestTransformService_InvalidExpr(t *testing.T) {
	svc := app.NewTransformService()
	ctx := context.Background()

	_, err := svc.EvalString(ctx, `invalid syntax !!!`, nil)
	if err == nil {
		t.Error("expected error for invalid expression")
	}
}

func TestTransformService_ValidateExpr(t *testing.T) {
	svc := app.NewTransformService()

	tests := []struct {
		name       string
		expr       string
		context    string
		wantValid  bool
	}{
		{"empty expression", "", "request", true},
		{"valid request expr", `method + " " + path`, "request", true},
		{"valid response expr", `status > 200`, "response", true},
		{"valid streaming expr", `responseBytes / 1000`, "streaming", true},
		{"invalid syntax", `method + `, "request", false},
		{"unknown variable in request", `nonexistent`, "request", false},
		{"default context", `method`, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.ValidateExpr(tt.expr, tt.context)
			if result.Valid != tt.wantValid {
				t.Errorf("ValidateExpr() valid = %v, want %v, error = %s", result.Valid, tt.wantValid, result.Error)
			}
		})
	}
}

func TestTransformService_AdvancedExprFunctions(t *testing.T) {
	svc := app.NewTransformService()
	ctx := context.Background()

	tests := []struct {
		name string
		expr string
		data map[string]any
		want any
	}{
		{
			"json parse",
			`json(data).key`,
			map[string]any{"data": []byte(`{"key":"value"}`)},
			"value",
		},
		{
			"now function",
			`now() > 0`,
			nil,
			true,
		},
		{
			"nowRFC3339",
			`len(nowRFC3339()) > 0`,
			nil,
			true,
		},
		{
			"urlDecode",
			`urlDecode("hello%20world")`,
			nil,
			"hello world",
		},
		{
			"jsonDecode",
			`jsonDecode("{\"a\":1}").a`,
			nil,
			1.0,
		},
		{
			"first function",
			`first(arr)`,
			map[string]any{"arr": []any{1, 2, 3}},
			1,
		},
		{
			"last function",
			`last(arr)`,
			map[string]any{"arr": []any{1, 2, 3}},
			3,
		},
		{
			"count function",
			`count(arr)`,
			map[string]any{"arr": []any{1, 2, 3}},
			3,
		},
		{
			"sum function",
			`sum(arr)`,
			map[string]any{"arr": []any{1.0, 2.0, 3.0}},
			6.0,
		},
		{
			"sum with field",
			`sum(arr, "val")`,
			map[string]any{"arr": []any{
				map[string]any{"val": 1.0},
				map[string]any{"val": 2.0},
				map[string]any{"val": 3.0},
			}},
			6.0,
		},
		{
			"avg function",
			`avg(arr)`,
			map[string]any{"arr": []any{1.0, 2.0, 3.0}},
			2.0,
		},
		{
			"avg with field",
			`avg(arr, "val")`,
			map[string]any{"arr": []any{
				map[string]any{"val": 2.0},
				map[string]any{"val": 4.0},
			}},
			3.0,
		},
		{
			"max function",
			`max(arr)`,
			map[string]any{"arr": []any{1.0, 5.0, 3.0}},
			5.0,
		},
		{
			"max with field",
			`max(arr, "val")`,
			map[string]any{"arr": []any{
				map[string]any{"val": 1.0},
				map[string]any{"val": 5.0},
			}},
			5.0,
		},
		{
			"min function",
			`min(arr)`,
			map[string]any{"arr": []any{3.0, 1.0, 5.0}},
			1.0,
		},
		{
			"min with field",
			`min(arr, "val")`,
			map[string]any{"arr": []any{
				map[string]any{"val": 3.0},
				map[string]any{"val": 1.0},
			}},
			1.0,
		},
		{
			"get nested field",
			`get(obj, "a.b.c")`,
			map[string]any{"obj": map[string]any{
				"a": map[string]any{
					"b": map[string]any{
						"c": "value",
					},
				},
			}},
			"value",
		},
		{
			"lines function",
			`len(lines(data))`,
			map[string]any{"data": []byte("line1\nline2\nline3")},
			3,
		},
		{
			"linesNonEmpty function",
			`len(linesNonEmpty(data))`,
			map[string]any{"data": []byte("line1\n\nline2\n\n")},
			2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.Eval(ctx, tt.expr, tt.data)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.want {
				t.Errorf("got %v (%T), want %v (%T)", result, result, tt.want, tt.want)
			}
		})
	}
}

func TestTransformService_SSEFunctions(t *testing.T) {
	svc := app.NewTransformService()
	ctx := context.Background()

	sseData := []byte("event: message\ndata: {\"msg\":\"hello\"}\nid: 1\n\nevent: done\ndata: {\"msg\":\"bye\"}\n\n")

	tests := []struct {
		name string
		expr string
		data map[string]any
	}{
		{
			"sseEvents",
			`len(sseEvents(data)) > 0`,
			map[string]any{"data": sseData},
		},
		{
			"sseLastData",
			`len(sseLastData(data)) > 0`,
			map[string]any{"data": sseData},
		},
		{
			"sseAllData",
			`len(sseAllData(data)) > 0`,
			map[string]any{"data": sseData},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.Eval(ctx, tt.expr, tt.data)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != true {
				t.Errorf("expected true, got %v", result)
			}
		})
	}
}

func TestTransformService_EdgeCases(t *testing.T) {
	svc := app.NewTransformService()
	ctx := context.Background()

	// Empty arrays
	result, _ := svc.Eval(ctx, `first(arr)`, map[string]any{"arr": []any{}})
	if result != nil {
		t.Errorf("first of empty array should be nil")
	}

	result, _ = svc.Eval(ctx, `last(arr)`, map[string]any{"arr": []any{}})
	if result != nil {
		t.Errorf("last of empty array should be nil")
	}

	result, _ = svc.Eval(ctx, `count(nil)`, map[string]any{})
	if result != 0 {
		t.Errorf("count of nil should be 0")
	}

	result, _ = svc.Eval(ctx, `sum(nil)`, map[string]any{})
	if result != 0.0 {
		t.Errorf("sum of nil should be 0")
	}

	// Empty avg returns 0
	result, _ = svc.Eval(ctx, `avg(arr)`, map[string]any{"arr": []any{}})
	if result != 0.0 {
		t.Errorf("avg of empty array should be 0")
	}

	// max/min of empty array
	result, _ = svc.Eval(ctx, `max(arr)`, map[string]any{"arr": []any{}})
	if result != nil {
		t.Errorf("max of empty array should be nil")
	}

	result, _ = svc.Eval(ctx, `min(arr)`, map[string]any{"arr": []any{}})
	if result != nil {
		t.Errorf("min of empty array should be nil")
	}

	// get with non-map returns nil
	result, _ = svc.Eval(ctx, `get(obj, "a.b")`, map[string]any{"obj": "not a map"})
	if result != nil {
		t.Errorf("get on non-map should be nil")
	}

	// json with empty bytes
	result, _ = svc.Eval(ctx, `json(data)`, map[string]any{"data": []byte{}})
	if result != nil {
		t.Errorf("json of empty bytes should be nil")
	}
}

func TestTransformService_TypeConversions(t *testing.T) {
	svc := app.NewTransformService()
	ctx := context.Background()

	tests := []struct {
		name string
		expr string
		data map[string]any
		want any
	}{
		{
			"toString nil",
			`toString(nil)`,
			map[string]any{"nil": nil},
			"",
		},
		{
			"toString bytes",
			`toString(data)`,
			map[string]any{"data": []byte("hello")},
			"hello",
		},
		{
			"toInt nil",
			`toInt(nil)`,
			map[string]any{"nil": nil},
			0,
		},
		{
			"toInt int64",
			`toInt(val)`,
			map[string]any{"val": int64(42)},
			42,
		},
		{
			"toInt float",
			`toInt(val)`,
			map[string]any{"val": 42.9},
			42,
		},
		{
			"toInt string",
			`toInt(val)`,
			map[string]any{"val": "42"},
			42,
		},
		{
			"toFloat nil",
			`toFloat(nil)`,
			map[string]any{"nil": nil},
			0.0,
		},
		{
			"toFloat float32",
			`toFloat(val)`,
			map[string]any{"val": float32(3.14)},
			float64(float32(3.14)),
		},
		{
			"toFloat int",
			`toFloat(val)`,
			map[string]any{"val": 42},
			42.0,
		},
		{
			"toFloat int64",
			`toFloat(val)`,
			map[string]any{"val": int64(42)},
			42.0,
		},
		{
			"toFloat string",
			`toFloat(val)`,
			map[string]any{"val": "3.14"},
			3.14,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.Eval(ctx, tt.expr, tt.data)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.want {
				t.Errorf("got %v (%T), want %v (%T)", result, result, tt.want, tt.want)
			}
		})
	}
}

func TestTransformService_TransformWithNilHeaders(t *testing.T) {
	svc := app.NewTransformService()
	ctx := context.Background()

	// Request with nil headers
	req := proxy.Request{
		Method: "GET",
		Path:   "/api/data",
	}
	transform := &route.Transform{
		SetHeaders: map[string]string{
			"X-Added": `"value"`,
		},
	}

	result, err := svc.TransformRequest(ctx, req, transform, nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.Headers["X-Added"] != "value" {
		t.Errorf("header not set correctly")
	}
}

func TestTransformService_ResponseTransformWithNilHeaders(t *testing.T) {
	svc := app.NewTransformService()
	ctx := context.Background()

	// Response with nil headers
	resp := proxy.Response{
		Status: 200,
		Body:   []byte(`{"data":"test"}`),
	}
	transform := &route.Transform{
		SetHeaders: map[string]string{
			"X-Response": `"value"`,
		},
	}

	result, err := svc.TransformResponse(ctx, resp, transform, nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.Headers["X-Response"] != "value" {
		t.Errorf("header not set correctly")
	}
}

func TestTransformService_TransformErrors(t *testing.T) {
	svc := app.NewTransformService()
	ctx := context.Background()

	// Invalid header expression
	req := proxy.Request{
		Method:  "GET",
		Path:    "/api/data",
		Headers: map[string]string{},
	}
	transform := &route.Transform{
		SetHeaders: map[string]string{
			"X-Bad": `nonexistent_var`,
		},
	}

	_, err := svc.TransformRequest(ctx, req, transform, nil)
	if err == nil {
		t.Error("expected error for invalid header expression")
	}

	// Invalid query expression
	transform2 := &route.Transform{
		SetQuery: map[string]string{
			"bad": `nonexistent_var`,
		},
	}

	_, err = svc.TransformRequest(ctx, req, transform2, nil)
	if err == nil {
		t.Error("expected error for invalid query expression")
	}

	// Invalid body expression
	transform3 := &route.Transform{
		BodyExpr: `nonexistent_var`,
	}

	_, err = svc.TransformRequest(ctx, req, transform3, nil)
	if err == nil {
		t.Error("expected error for invalid body expression")
	}
}

func TestTransformService_ResponseTransformErrors(t *testing.T) {
	svc := app.NewTransformService()
	ctx := context.Background()

	resp := proxy.Response{
		Status:  200,
		Headers: map[string]string{},
	}

	// Invalid header expression
	transform := &route.Transform{
		SetHeaders: map[string]string{
			"X-Bad": `nonexistent_var`,
		},
	}

	_, err := svc.TransformResponse(ctx, resp, transform, nil)
	if err == nil {
		t.Error("expected error for invalid header expression")
	}

	// Invalid body expression
	transform2 := &route.Transform{
		BodyExpr: `nonexistent_var`,
	}

	_, err = svc.TransformResponse(ctx, resp, transform2, nil)
	if err == nil {
		t.Error("expected error for invalid body expression")
	}
}

func TestTransformService_JoinWithAnyArray(t *testing.T) {
	svc := app.NewTransformService()
	ctx := context.Background()

	// join with []any
	result, err := svc.Eval(ctx, `join(arr, "-")`, map[string]any{
		"arr": []any{"a", "b", "c"},
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "a-b-c" {
		t.Errorf("got %v, want a-b-c", result)
	}
}

func TestTransformService_ToSliceConversions(t *testing.T) {
	svc := app.NewTransformService()
	ctx := context.Background()

	// Test with string slice
	result, err := svc.Eval(ctx, `count(arr)`, map[string]any{
		"arr": []string{"a", "b", "c"},
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 3 {
		t.Errorf("got %v, want 3", result)
	}

	// Test with map slice
	result, err = svc.Eval(ctx, `count(arr)`, map[string]any{
		"arr": []map[string]any{{"k": "v"}, {"k": "v2"}},
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 2 {
		t.Errorf("got %v, want 2", result)
	}
}
