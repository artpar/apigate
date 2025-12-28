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
