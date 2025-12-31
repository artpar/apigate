package remote

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/artpar/apigate/domain/billing"
	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/domain/usage"
)

// =============================================================================
// Client Tests (remote.go)
// =============================================================================

func TestNewClient(t *testing.T) {
	tests := []struct {
		name     string
		cfg      ClientConfig
		wantBase string
		wantKey  string
	}{
		{
			name: "with all fields",
			cfg: ClientConfig{
				BaseURL: "https://api.example.com",
				APIKey:  "test-key",
				Timeout: 30 * time.Second,
				Headers: map[string]string{"X-Custom": "value"},
			},
			wantBase: "https://api.example.com",
			wantKey:  "test-key",
		},
		{
			name: "with default timeout",
			cfg: ClientConfig{
				BaseURL: "https://api.example.com",
				APIKey:  "test-key",
			},
			wantBase: "https://api.example.com",
			wantKey:  "test-key",
		},
		{
			name:     "empty config",
			cfg:      ClientConfig{},
			wantBase: "",
			wantKey:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.cfg)
			if client == nil {
				t.Fatal("NewClient returned nil")
			}
			if client.baseURL != tt.wantBase {
				t.Errorf("baseURL = %q, want %q", client.baseURL, tt.wantBase)
			}
			if client.apiKey != tt.wantKey {
				t.Errorf("apiKey = %q, want %q", client.apiKey, tt.wantKey)
			}
			if client.httpClient == nil {
				t.Error("httpClient is nil")
			}
		})
	}
}

func TestClientRequest_Success(t *testing.T) {
	expectedBody := map[string]string{"message": "hello"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/test" {
			t.Errorf("Path = %q, want /test", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q, want application/json", r.Header.Get("Accept"))
		}
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("Authorization = %q, want Bearer test-api-key", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-Custom") != "custom-value" {
			t.Errorf("X-Custom = %q, want custom-value", r.Header.Get("X-Custom"))
		}

		// Return response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedBody)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		APIKey:  "test-api-key",
		Headers: map[string]string{"X-Custom": "custom-value"},
	})

	var result map[string]string
	err := client.Request(context.Background(), http.MethodPost, "/test", map[string]string{"key": "value"}, &result)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if result["message"] != "hello" {
		t.Errorf("result[message] = %q, want %q", result["message"], "hello")
	}
}

func TestClientRequest_NoAuthHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("Authorization header should be empty when no API key is set")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL: server.URL,
	})

	err := client.Request(context.Background(), http.MethodGet, "/test", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
}

func TestClientRequest_ErrorResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{"bad request", http.StatusBadRequest, "invalid input"},
		{"unauthorized", http.StatusUnauthorized, "invalid token"},
		{"not found", http.StatusNotFound, "resource not found"},
		{"internal error", http.StatusInternalServerError, "server error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := NewClient(ClientConfig{BaseURL: server.URL})

			err := client.Request(context.Background(), http.MethodGet, "/test", nil, nil)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			remoteErr, ok := err.(*RemoteError)
			if !ok {
				t.Fatalf("Expected *RemoteError, got %T", err)
			}
			if remoteErr.StatusCode != tt.statusCode {
				t.Errorf("StatusCode = %d, want %d", remoteErr.StatusCode, tt.statusCode)
			}
			if remoteErr.Message != tt.body {
				t.Errorf("Message = %q, want %q", remoteErr.Message, tt.body)
			}
		})
	}
}

func TestClientRequest_NilBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Body should be empty for nil body
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})

	err := client.Request(context.Background(), http.MethodGet, "/test", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
}

func TestClientRequest_InvalidBody(t *testing.T) {
	client := NewClient(ClientConfig{BaseURL: "http://localhost"})

	// Using a channel which cannot be marshaled to JSON
	invalidBody := make(chan int)

	err := client.Request(context.Background(), http.MethodPost, "/test", invalidBody, nil)
	if err == nil {
		t.Fatal("Expected error for unmarshalable body")
	}
}

func TestClientRequest_InvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})

	var result map[string]string
	err := client.Request(context.Background(), http.MethodGet, "/test", nil, &result)
	if err == nil {
		t.Fatal("Expected error for invalid JSON response")
	}
}

func TestClientRequest_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		Timeout: 10 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := client.Request(ctx, http.MethodGet, "/test", nil, nil)
	if err == nil {
		t.Fatal("Expected error for cancelled context")
	}
}

func TestRemoteError_Error(t *testing.T) {
	err := &RemoteError{
		StatusCode: 404,
		Message:    "not found",
	}

	expected := "remote error 404: not found"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "404 error",
			err:  &RemoteError{StatusCode: 404, Message: "not found"},
			want: true,
		},
		{
			name: "400 error",
			err:  &RemoteError{StatusCode: 400, Message: "bad request"},
			want: false,
		},
		{
			name: "500 error",
			err:  &RemoteError{StatusCode: 500, Message: "internal error"},
			want: false,
		},
		{
			name: "non-remote error",
			err:  context.DeadlineExceeded,
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotFound(tt.err)
			if got != tt.want {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

// =============================================================================
// KeyStore Tests (keystore.go)
// =============================================================================

func TestNewKeyStore(t *testing.T) {
	client := NewClient(ClientConfig{BaseURL: "http://localhost"})
	ks := NewKeyStore(client)

	if ks == nil {
		t.Fatal("NewKeyStore returned nil")
	}
	if ks.client != client {
		t.Error("KeyStore client not set correctly")
	}
}

func TestKeyStore_Get(t *testing.T) {
	now := time.Now().UTC()
	expires := now.Add(24 * time.Hour)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/keys/prefix/ak_abc12345" {
			t.Errorf("Path = %q, want /keys/prefix/ak_abc12345", r.URL.Path)
		}

		resp := struct {
			Keys []RemoteKey `json:"keys"`
		}{
			Keys: []RemoteKey{
				{
					ID:        "key-123",
					UserID:    "user-456",
					Prefix:    "ak_abc12345",
					Name:      "Test Key",
					Scopes:    []string{"read", "write"},
					ExpiresAt: &expires,
					CreatedAt: now,
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})
	ks := NewKeyStore(client)

	keys, err := ks.Get(context.Background(), "ak_abc12345")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("len(keys) = %d, want 1", len(keys))
	}

	k := keys[0]
	if k.ID != "key-123" {
		t.Errorf("ID = %q, want %q", k.ID, "key-123")
	}
	if k.UserID != "user-456" {
		t.Errorf("UserID = %q, want %q", k.UserID, "user-456")
	}
	if k.Prefix != "ak_abc12345" {
		t.Errorf("Prefix = %q, want %q", k.Prefix, "ak_abc12345")
	}
	if k.Name != "Test Key" {
		t.Errorf("Name = %q, want %q", k.Name, "Test Key")
	}
	if len(k.Scopes) != 2 || k.Scopes[0] != "read" || k.Scopes[1] != "write" {
		t.Errorf("Scopes = %v, want [read write]", k.Scopes)
	}
}

func TestKeyStore_Get_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})
	ks := NewKeyStore(client)

	keys, err := ks.Get(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if keys != nil {
		t.Errorf("keys = %v, want nil", keys)
	}
}

func TestKeyStore_Get_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})
	ks := NewKeyStore(client)

	_, err := ks.Get(context.Background(), "test")
	if err == nil {
		t.Fatal("Expected error")
	}
}

func TestKeyStore_ValidateKey(t *testing.T) {
	now := time.Now().UTC()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/keys/validate" {
			t.Errorf("Path = %q, want /keys/validate", r.URL.Path)
		}

		var req KeyValidateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req.APIKey != "ak_abc12345secret" {
			t.Errorf("APIKey = %q, want %q", req.APIKey, "ak_abc12345secret")
		}
		if req.Prefix != "ak_abc12345" {
			t.Errorf("Prefix = %q, want %q", req.Prefix, "ak_abc12345")
		}

		resp := KeyValidateResponse{
			Valid: true,
			Key: &RemoteKey{
				ID:        "key-123",
				UserID:    "user-456",
				Prefix:    "ak_abc12345",
				CreatedAt: now,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})
	ks := NewKeyStore(client)

	k, valid, reason, err := ks.ValidateKey(context.Background(), "ak_abc12345secret", "ak_abc12345")
	if err != nil {
		t.Fatalf("ValidateKey failed: %v", err)
	}

	if !valid {
		t.Error("Expected valid=true")
	}
	if reason != "" {
		t.Errorf("reason = %q, want empty", reason)
	}
	if k.ID != "key-123" {
		t.Errorf("ID = %q, want %q", k.ID, "key-123")
	}
}

func TestKeyStore_ValidateKey_Invalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := KeyValidateResponse{
			Valid:  false,
			Reason: "key_expired",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})
	ks := NewKeyStore(client)

	k, valid, reason, err := ks.ValidateKey(context.Background(), "test", "test")
	if err != nil {
		t.Fatalf("ValidateKey failed: %v", err)
	}

	if valid {
		t.Error("Expected valid=false")
	}
	if reason != "key_expired" {
		t.Errorf("reason = %q, want %q", reason, "key_expired")
	}
	if k.ID != "" {
		t.Errorf("ID = %q, want empty", k.ID)
	}
}

func TestKeyStore_ValidateKey_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})
	ks := NewKeyStore(client)

	_, _, _, err := ks.ValidateKey(context.Background(), "test", "test")
	if err == nil {
		t.Fatal("Expected error")
	}
}

func TestKeyStore_Create(t *testing.T) {
	now := time.Now().UTC()
	expires := now.Add(24 * time.Hour)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/keys" {
			t.Errorf("Path = %q, want /keys", r.URL.Path)
		}

		var req RemoteKey
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req.ID != "key-123" {
			t.Errorf("ID = %q, want %q", req.ID, "key-123")
		}
		if req.UserID != "user-456" {
			t.Errorf("UserID = %q, want %q", req.UserID, "user-456")
		}

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})
	ks := NewKeyStore(client)

	k := key.Key{
		ID:        "key-123",
		UserID:    "user-456",
		Prefix:    "ak_abc12345",
		Name:      "Test Key",
		Scopes:    []string{"read"},
		ExpiresAt: &expires,
		CreatedAt: now,
	}

	err := ks.Create(context.Background(), k)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
}

func TestKeyStore_Revoke(t *testing.T) {
	revokedAt := time.Now().UTC()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/keys/key-123/revoke" {
			t.Errorf("Path = %q, want /keys/key-123/revoke", r.URL.Path)
		}

		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req["revoked_at"] == nil {
			t.Error("revoked_at not set")
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})
	ks := NewKeyStore(client)

	err := ks.Revoke(context.Background(), "key-123", revokedAt)
	if err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}
}

func TestKeyStore_ListByUser(t *testing.T) {
	now := time.Now().UTC()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/keys/user/user-456" {
			t.Errorf("Path = %q, want /keys/user/user-456", r.URL.Path)
		}

		resp := struct {
			Keys []RemoteKey `json:"keys"`
		}{
			Keys: []RemoteKey{
				{ID: "key-1", UserID: "user-456", Prefix: "ak_111", CreatedAt: now},
				{ID: "key-2", UserID: "user-456", Prefix: "ak_222", CreatedAt: now},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})
	ks := NewKeyStore(client)

	keys, err := ks.ListByUser(context.Background(), "user-456")
	if err != nil {
		t.Fatalf("ListByUser failed: %v", err)
	}

	if len(keys) != 2 {
		t.Fatalf("len(keys) = %d, want 2", len(keys))
	}

	if keys[0].ID != "key-1" {
		t.Errorf("keys[0].ID = %q, want %q", keys[0].ID, "key-1")
	}
	if keys[1].ID != "key-2" {
		t.Errorf("keys[1].ID = %q, want %q", keys[1].ID, "key-2")
	}
}

func TestKeyStore_UpdateLastUsed(t *testing.T) {
	lastUsed := time.Now().UTC()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("Method = %q, want PATCH", r.Method)
		}
		if r.URL.Path != "/keys/key-123/last-used" {
			t.Errorf("Path = %q, want /keys/key-123/last-used", r.URL.Path)
		}

		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req["last_used"] == nil {
			t.Error("last_used not set")
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})
	ks := NewKeyStore(client)

	err := ks.UpdateLastUsed(context.Background(), "key-123", lastUsed)
	if err != nil {
		t.Fatalf("UpdateLastUsed failed: %v", err)
	}
}

func TestToKey(t *testing.T) {
	now := time.Now().UTC()
	expires := now.Add(24 * time.Hour)
	revoked := now.Add(-1 * time.Hour)
	lastUsed := now.Add(-10 * time.Minute)

	rk := RemoteKey{
		ID:        "key-123",
		UserID:    "user-456",
		Hash:      []byte("hash"),
		Prefix:    "ak_abc12345",
		Name:      "Test Key",
		Scopes:    []string{"read", "write"},
		ExpiresAt: &expires,
		RevokedAt: &revoked,
		CreatedAt: now,
		LastUsed:  &lastUsed,
	}

	k := toKey(rk)

	if k.ID != rk.ID {
		t.Errorf("ID = %q, want %q", k.ID, rk.ID)
	}
	if k.UserID != rk.UserID {
		t.Errorf("UserID = %q, want %q", k.UserID, rk.UserID)
	}
	if string(k.Hash) != string(rk.Hash) {
		t.Errorf("Hash mismatch")
	}
	if k.Prefix != rk.Prefix {
		t.Errorf("Prefix = %q, want %q", k.Prefix, rk.Prefix)
	}
	if k.Name != rk.Name {
		t.Errorf("Name = %q, want %q", k.Name, rk.Name)
	}
	if len(k.Scopes) != len(rk.Scopes) {
		t.Errorf("Scopes length mismatch")
	}
	if k.ExpiresAt == nil || !k.ExpiresAt.Equal(expires) {
		t.Errorf("ExpiresAt mismatch")
	}
	if k.RevokedAt == nil || !k.RevokedAt.Equal(revoked) {
		t.Errorf("RevokedAt mismatch")
	}
	if !k.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt mismatch")
	}
	if k.LastUsed == nil || !k.LastUsed.Equal(lastUsed) {
		t.Errorf("LastUsed mismatch")
	}
}

func TestFromKey(t *testing.T) {
	now := time.Now().UTC()
	expires := now.Add(24 * time.Hour)

	k := key.Key{
		ID:        "key-123",
		UserID:    "user-456",
		Hash:      []byte("hash"),
		Prefix:    "ak_abc12345",
		Name:      "Test Key",
		Scopes:    []string{"read"},
		ExpiresAt: &expires,
		CreatedAt: now,
	}

	rk := fromKey(k)

	if rk.ID != k.ID {
		t.Errorf("ID = %q, want %q", rk.ID, k.ID)
	}
	if rk.UserID != k.UserID {
		t.Errorf("UserID = %q, want %q", rk.UserID, k.UserID)
	}
	if rk.Prefix != k.Prefix {
		t.Errorf("Prefix = %q, want %q", rk.Prefix, k.Prefix)
	}
}

// =============================================================================
// UsageRecorder Tests (usage.go)
// =============================================================================

func TestNewUsageRecorder(t *testing.T) {
	tests := []struct {
		name          string
		cfg           UsageRecorderConfig
		wantBatchSize int
		wantInterval  time.Duration
	}{
		{
			name:          "default config",
			cfg:           UsageRecorderConfig{},
			wantBatchSize: 100,
			wantInterval:  10 * time.Second,
		},
		{
			name: "custom config",
			cfg: UsageRecorderConfig{
				BatchSize:     50,
				FlushInterval: 5 * time.Second,
			},
			wantBatchSize: 50,
			wantInterval:  5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(ClientConfig{BaseURL: "http://localhost"})
			recorder := NewUsageRecorder(client, tt.cfg)
			defer recorder.Close()

			if recorder == nil {
				t.Fatal("NewUsageRecorder returned nil")
			}
			if recorder.batchSize != tt.wantBatchSize {
				t.Errorf("batchSize = %d, want %d", recorder.batchSize, tt.wantBatchSize)
			}
			if recorder.flushInterval != tt.wantInterval {
				t.Errorf("flushInterval = %v, want %v", recorder.flushInterval, tt.wantInterval)
			}
		})
	}
}

func TestUsageRecorder_Record(t *testing.T) {
	var receivedEvents []RemoteUsageEvent
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/usage/events" {
			t.Errorf("Path = %q, want /usage/events", r.URL.Path)
		}

		var req struct {
			Events []RemoteUsageEvent `json:"events"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		mu.Lock()
		receivedEvents = append(receivedEvents, req.Events...)
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})
	recorder := NewUsageRecorder(client, UsageRecorderConfig{
		BatchSize:     3,
		FlushInterval: 1 * time.Hour, // Long interval to prevent auto-flush
	})

	// Record events
	for i := 0; i < 3; i++ {
		recorder.Record(usage.Event{
			ID:            "event-" + string(rune('0'+i)),
			KeyID:         "key-123",
			UserID:        "user-456",
			Method:        "POST",
			Path:          "/api/test",
			StatusCode:    200,
			LatencyMs:     50,
			RequestBytes:  100,
			ResponseBytes: 200,
			CostMultiplier: 1.0,
			IPAddress:     "127.0.0.1",
			UserAgent:     "test-agent",
			Timestamp:     time.Now().UTC(),
		})
	}

	// Wait for batch flush
	time.Sleep(100 * time.Millisecond)

	recorder.Close()

	mu.Lock()
	defer mu.Unlock()

	if len(receivedEvents) != 3 {
		t.Fatalf("len(receivedEvents) = %d, want 3", len(receivedEvents))
	}
}

func TestUsageRecorder_Flush(t *testing.T) {
	flushed := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flushed = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})
	recorder := NewUsageRecorder(client, UsageRecorderConfig{
		BatchSize:     100, // Large batch size
		FlushInterval: 1 * time.Hour,
	})

	recorder.Record(usage.Event{
		ID:        "event-1",
		KeyID:     "key-123",
		Timestamp: time.Now().UTC(),
	})

	err := recorder.Flush(context.Background())
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	recorder.Close()

	if !flushed {
		t.Error("Expected flush to send events")
	}
}

func TestUsageRecorder_Flush_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Should not send request for empty buffer")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})
	recorder := NewUsageRecorder(client, UsageRecorderConfig{
		BatchSize:     100,
		FlushInterval: 1 * time.Hour,
	})

	// Flush without any events
	err := recorder.Flush(context.Background())
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	recorder.Close()
}

func TestUsageRecorder_Flush_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})
	recorder := NewUsageRecorder(client, UsageRecorderConfig{
		BatchSize:     100,
		FlushInterval: 1 * time.Hour,
	})

	recorder.Record(usage.Event{
		ID:        "event-1",
		KeyID:     "key-123",
		Timestamp: time.Now().UTC(),
	})

	err := recorder.Flush(context.Background())
	if err == nil {
		t.Fatal("Expected error")
	}

	recorder.Close()
}

func TestUsageRecorder_Close(t *testing.T) {
	flushed := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flushed = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})
	recorder := NewUsageRecorder(client, UsageRecorderConfig{
		BatchSize:     100,
		FlushInterval: 1 * time.Hour,
	})

	recorder.Record(usage.Event{
		ID:        "event-1",
		KeyID:     "key-123",
		Timestamp: time.Now().UTC(),
	})

	err := recorder.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if !flushed {
		t.Error("Close should flush remaining events")
	}
}

func TestUsageRecorder_FlushLoop(t *testing.T) {
	var flushCount int
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		flushCount++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})
	recorder := NewUsageRecorder(client, UsageRecorderConfig{
		BatchSize:     100,
		FlushInterval: 50 * time.Millisecond,
	})

	recorder.Record(usage.Event{
		ID:        "event-1",
		KeyID:     "key-123",
		Timestamp: time.Now().UTC(),
	})

	// Wait for automatic flush
	time.Sleep(100 * time.Millisecond)

	recorder.Close()

	mu.Lock()
	defer mu.Unlock()

	if flushCount == 0 {
		t.Error("Expected at least one automatic flush")
	}
}

// =============================================================================
// BillingProvider Tests (billing.go)
// =============================================================================

func TestNewBillingProvider(t *testing.T) {
	client := NewClient(ClientConfig{BaseURL: "http://localhost"})
	bp := NewBillingProvider(client)

	if bp == nil {
		t.Fatal("NewBillingProvider returned nil")
	}
	if bp.client != client {
		t.Error("BillingProvider client not set correctly")
	}
}

func TestBillingProvider_CreateCustomer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/billing/customers" {
			t.Errorf("Path = %q, want /billing/customers", r.URL.Path)
		}

		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req["email"] != "test@example.com" {
			t.Errorf("email = %q, want %q", req["email"], "test@example.com")
		}
		if req["name"] != "Test User" {
			t.Errorf("name = %q, want %q", req["name"], "Test User")
		}
		if req["user_id"] != "user-123" {
			t.Errorf("user_id = %q, want %q", req["user_id"], "user-123")
		}

		resp := struct {
			CustomerID string `json:"customer_id"`
		}{
			CustomerID: "cus_123",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})
	bp := NewBillingProvider(client)

	customerID, err := bp.CreateCustomer(context.Background(), "test@example.com", "Test User", "user-123")
	if err != nil {
		t.Fatalf("CreateCustomer failed: %v", err)
	}

	if customerID != "cus_123" {
		t.Errorf("customerID = %q, want %q", customerID, "cus_123")
	}
}

func TestBillingProvider_CreateCustomer_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid email"))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})
	bp := NewBillingProvider(client)

	_, err := bp.CreateCustomer(context.Background(), "invalid", "Test", "user-123")
	if err == nil {
		t.Fatal("Expected error")
	}
}

func TestBillingProvider_CreateSubscription(t *testing.T) {
	now := time.Now().UTC()
	periodEnd := now.Add(30 * 24 * time.Hour)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/billing/subscriptions" {
			t.Errorf("Path = %q, want /billing/subscriptions", r.URL.Path)
		}

		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req["customer_id"] != "cus_123" {
			t.Errorf("customer_id = %q, want %q", req["customer_id"], "cus_123")
		}
		if req["plan_id"] != "pro" {
			t.Errorf("plan_id = %q, want %q", req["plan_id"], "pro")
		}

		resp := struct {
			Subscription RemoteSubscription `json:"subscription"`
		}{
			Subscription: RemoteSubscription{
				ID:                 "sub_123",
				UserID:             "user-456",
				CustomerID:         "cus_123",
				PlanID:             "pro",
				Status:             "active",
				CurrentPeriodStart: now,
				CurrentPeriodEnd:   periodEnd,
				CreatedAt:          now,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})
	bp := NewBillingProvider(client)

	sub, err := bp.CreateSubscription(context.Background(), "cus_123", "pro")
	if err != nil {
		t.Fatalf("CreateSubscription failed: %v", err)
	}

	if sub.ID != "sub_123" {
		t.Errorf("ID = %q, want %q", sub.ID, "sub_123")
	}
	if sub.UserID != "user-456" {
		t.Errorf("UserID = %q, want %q", sub.UserID, "user-456")
	}
	if sub.PlanID != "pro" {
		t.Errorf("PlanID = %q, want %q", sub.PlanID, "pro")
	}
	if sub.Status != billing.SubscriptionStatusActive {
		t.Errorf("Status = %q, want %q", sub.Status, billing.SubscriptionStatusActive)
	}
}

func TestBillingProvider_CreateSubscription_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("customer not found"))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})
	bp := NewBillingProvider(client)

	_, err := bp.CreateSubscription(context.Background(), "invalid", "pro")
	if err == nil {
		t.Fatal("Expected error")
	}
}

func TestBillingProvider_CancelSubscription(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/billing/subscriptions/sub_123/cancel" {
			t.Errorf("Path = %q, want /billing/subscriptions/sub_123/cancel", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})
	bp := NewBillingProvider(client)

	err := bp.CancelSubscription(context.Background(), "sub_123")
	if err != nil {
		t.Fatalf("CancelSubscription failed: %v", err)
	}
}

func TestBillingProvider_ReportUsage(t *testing.T) {
	timestamp := time.Now().UTC()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/billing/usage" {
			t.Errorf("Path = %q, want /billing/usage", r.URL.Path)
		}

		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req["subscription_item_id"] != "si_123" {
			t.Errorf("subscription_item_id = %v, want %q", req["subscription_item_id"], "si_123")
		}
		if int64(req["quantity"].(float64)) != 100 {
			t.Errorf("quantity = %v, want 100", req["quantity"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})
	bp := NewBillingProvider(client)

	err := bp.ReportUsage(context.Background(), "si_123", 100, timestamp)
	if err != nil {
		t.Fatalf("ReportUsage failed: %v", err)
	}
}

func TestBillingProvider_CreateInvoice(t *testing.T) {
	now := time.Now().UTC()
	periodEnd := now.Add(30 * 24 * time.Hour)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/billing/invoices" {
			t.Errorf("Path = %q, want /billing/invoices", r.URL.Path)
		}

		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req["customer_id"] != "cus_123" {
			t.Errorf("customer_id = %v, want %q", req["customer_id"], "cus_123")
		}

		items := req["items"].([]interface{})
		if len(items) != 2 {
			t.Errorf("len(items) = %d, want 2", len(items))
		}

		resp := struct {
			Invoice RemoteInvoice `json:"invoice"`
		}{
			Invoice: RemoteInvoice{
				ID:          "inv_123",
				UserID:      "user-456",
				CustomerID:  "cus_123",
				Amount:      5000,
				Currency:    "usd",
				Status:      "open",
				PeriodStart: now,
				PeriodEnd:   periodEnd,
				Items: []RemoteInvoiceItem{
					{Description: "Pro Plan", Quantity: 1, UnitPrice: 3000, Amount: 3000},
					{Description: "Overage", Quantity: 100, UnitPrice: 20, Amount: 2000},
				},
				CreatedAt: now,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})
	bp := NewBillingProvider(client)

	items := []billing.InvoiceItem{
		{Description: "Pro Plan", Quantity: 1, UnitPrice: 3000, Amount: 3000},
		{Description: "Overage", Quantity: 100, UnitPrice: 20, Amount: 2000},
	}

	inv, err := bp.CreateInvoice(context.Background(), "cus_123", items)
	if err != nil {
		t.Fatalf("CreateInvoice failed: %v", err)
	}

	if inv.ID != "inv_123" {
		t.Errorf("ID = %q, want %q", inv.ID, "inv_123")
	}
	if inv.UserID != "user-456" {
		t.Errorf("UserID = %q, want %q", inv.UserID, "user-456")
	}
	if inv.Total != 5000 {
		t.Errorf("Total = %d, want 5000", inv.Total)
	}
	if len(inv.Items) != 2 {
		t.Errorf("len(Items) = %d, want 2", len(inv.Items))
	}
	if inv.Status != billing.InvoiceStatusOpen {
		t.Errorf("Status = %q, want %q", inv.Status, billing.InvoiceStatusOpen)
	}
}

func TestBillingProvider_CreateInvoice_EmptyItems(t *testing.T) {
	now := time.Now().UTC()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := struct {
			Invoice RemoteInvoice `json:"invoice"`
		}{
			Invoice: RemoteInvoice{
				ID:        "inv_123",
				Items:     []RemoteInvoiceItem{},
				CreatedAt: now,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})
	bp := NewBillingProvider(client)

	inv, err := bp.CreateInvoice(context.Background(), "cus_123", []billing.InvoiceItem{})
	if err != nil {
		t.Fatalf("CreateInvoice failed: %v", err)
	}

	if len(inv.Items) != 0 {
		t.Errorf("len(Items) = %d, want 0", len(inv.Items))
	}
}

func TestToSubscription(t *testing.T) {
	now := time.Now().UTC()
	periodEnd := now.Add(30 * 24 * time.Hour)
	canceledAt := now.Add(-1 * time.Hour)

	rs := RemoteSubscription{
		ID:                 "sub_123",
		UserID:             "user-456",
		CustomerID:         "cus_789",
		PlanID:             "pro",
		Status:             "active",
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   periodEnd,
		CanceledAt:         &canceledAt,
		CreatedAt:          now,
	}

	sub := toSubscription(rs)

	if sub.ID != rs.ID {
		t.Errorf("ID = %q, want %q", sub.ID, rs.ID)
	}
	if sub.UserID != rs.UserID {
		t.Errorf("UserID = %q, want %q", sub.UserID, rs.UserID)
	}
	if sub.PlanID != rs.PlanID {
		t.Errorf("PlanID = %q, want %q", sub.PlanID, rs.PlanID)
	}
	if sub.Status != billing.SubscriptionStatusActive {
		t.Errorf("Status = %q, want %q", sub.Status, billing.SubscriptionStatusActive)
	}
	if !sub.CurrentPeriodStart.Equal(now) {
		t.Errorf("CurrentPeriodStart mismatch")
	}
	if !sub.CurrentPeriodEnd.Equal(periodEnd) {
		t.Errorf("CurrentPeriodEnd mismatch")
	}
	if sub.CancelledAt == nil || !sub.CancelledAt.Equal(canceledAt) {
		t.Errorf("CancelledAt mismatch")
	}
	if !sub.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt mismatch")
	}
}

func TestToInvoice(t *testing.T) {
	now := time.Now().UTC()
	periodEnd := now.Add(30 * 24 * time.Hour)
	paidAt := now.Add(-1 * time.Hour)

	ri := RemoteInvoice{
		ID:          "inv_123",
		UserID:      "user-456",
		CustomerID:  "cus_789",
		Amount:      5000,
		Currency:    "usd",
		Status:      "paid",
		PeriodStart: now,
		PeriodEnd:   periodEnd,
		Items: []RemoteInvoiceItem{
			{Description: "Plan", Quantity: 1, UnitPrice: 3000, Amount: 3000},
			{Description: "Overage", Quantity: 100, UnitPrice: 20, Amount: 2000},
		},
		PaidAt:    &paidAt,
		CreatedAt: now,
	}

	inv := toInvoice(ri)

	if inv.ID != ri.ID {
		t.Errorf("ID = %q, want %q", inv.ID, ri.ID)
	}
	if inv.UserID != ri.UserID {
		t.Errorf("UserID = %q, want %q", inv.UserID, ri.UserID)
	}
	if inv.Total != ri.Amount {
		t.Errorf("Total = %d, want %d", inv.Total, ri.Amount)
	}
	if inv.Status != billing.InvoiceStatusPaid {
		t.Errorf("Status = %q, want %q", inv.Status, billing.InvoiceStatusPaid)
	}
	if len(inv.Items) != 2 {
		t.Errorf("len(Items) = %d, want 2", len(inv.Items))
	}
	if inv.Items[0].Description != "Plan" {
		t.Errorf("Items[0].Description = %q, want %q", inv.Items[0].Description, "Plan")
	}
	if inv.Items[1].Quantity != 100 {
		t.Errorf("Items[1].Quantity = %d, want 100", inv.Items[1].Quantity)
	}
	if inv.PaidAt == nil || !inv.PaidAt.Equal(paidAt) {
		t.Errorf("PaidAt mismatch")
	}
}

func TestToInvoice_EmptyItems(t *testing.T) {
	now := time.Now().UTC()

	ri := RemoteInvoice{
		ID:        "inv_123",
		Items:     []RemoteInvoiceItem{},
		CreatedAt: now,
	}

	inv := toInvoice(ri)

	if len(inv.Items) != 0 {
		t.Errorf("len(Items) = %d, want 0", len(inv.Items))
	}
}

// =============================================================================
// Integration-style Tests
// =============================================================================

func TestKeyStore_RoundTrip(t *testing.T) {
	now := time.Now().UTC()
	expires := now.Add(24 * time.Hour)

	storedKeys := make(map[string]RemoteKey)
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/keys":
			var rk RemoteKey
			json.NewDecoder(r.Body).Decode(&rk)
			storedKeys[rk.ID] = rk
			w.WriteHeader(http.StatusCreated)

		case r.Method == http.MethodGet && len(r.URL.Path) > 13 && r.URL.Path[:13] == "/keys/prefix/":
			prefix := r.URL.Path[13:]
			var matching []RemoteKey
			for _, rk := range storedKeys {
				if rk.Prefix == prefix {
					matching = append(matching, rk)
				}
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"keys": matching})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})
	ks := NewKeyStore(client)

	// Create a key
	k := key.Key{
		ID:        "key-roundtrip",
		UserID:    "user-123",
		Prefix:    "ak_roundtrip",
		Name:      "Round Trip Key",
		ExpiresAt: &expires,
		CreatedAt: now,
	}

	err := ks.Create(context.Background(), k)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Retrieve the key
	keys, err := ks.Get(context.Background(), "ak_roundtrip")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("len(keys) = %d, want 1", len(keys))
	}

	retrieved := keys[0]
	if retrieved.ID != k.ID {
		t.Errorf("ID = %q, want %q", retrieved.ID, k.ID)
	}
	if retrieved.UserID != k.UserID {
		t.Errorf("UserID = %q, want %q", retrieved.UserID, k.UserID)
	}
	if retrieved.Prefix != k.Prefix {
		t.Errorf("Prefix = %q, want %q", retrieved.Prefix, k.Prefix)
	}
}

func TestBillingProvider_FullFlow(t *testing.T) {
	now := time.Now().UTC()
	periodEnd := now.Add(30 * 24 * time.Hour)

	customerID := ""
	subscriptionID := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/billing/customers":
			customerID = "cus_test_123"
			json.NewEncoder(w).Encode(map[string]string{"customer_id": customerID})

		case r.Method == http.MethodPost && r.URL.Path == "/billing/subscriptions":
			subscriptionID = "sub_test_123"
			json.NewEncoder(w).Encode(map[string]interface{}{
				"subscription": RemoteSubscription{
					ID:                 subscriptionID,
					CustomerID:         customerID,
					Status:             "active",
					CurrentPeriodStart: now,
					CurrentPeriodEnd:   periodEnd,
					CreatedAt:          now,
				},
			})

		case r.Method == http.MethodPost && r.URL.Path == "/billing/usage":
			w.WriteHeader(http.StatusOK)

		case r.Method == http.MethodPost && r.URL.Path == "/billing/subscriptions/"+subscriptionID+"/cancel":
			w.WriteHeader(http.StatusOK)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BaseURL: server.URL})
	bp := NewBillingProvider(client)

	// Create customer
	custID, err := bp.CreateCustomer(context.Background(), "flow@example.com", "Flow Test", "user-flow")
	if err != nil {
		t.Fatalf("CreateCustomer failed: %v", err)
	}
	if custID != "cus_test_123" {
		t.Errorf("customerID = %q, want %q", custID, "cus_test_123")
	}

	// Create subscription
	sub, err := bp.CreateSubscription(context.Background(), custID, "pro")
	if err != nil {
		t.Fatalf("CreateSubscription failed: %v", err)
	}
	if sub.ID != "sub_test_123" {
		t.Errorf("subscriptionID = %q, want %q", sub.ID, "sub_test_123")
	}

	// Report usage
	err = bp.ReportUsage(context.Background(), "si_test", 100, now)
	if err != nil {
		t.Fatalf("ReportUsage failed: %v", err)
	}

	// Cancel subscription
	err = bp.CancelSubscription(context.Background(), sub.ID)
	if err != nil {
		t.Fatalf("CancelSubscription failed: %v", err)
	}
}
