package admin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/hasher"
	"github.com/artpar/apigate/adapters/http/admin"
	"github.com/artpar/apigate/adapters/memory"
	"github.com/artpar/apigate/config"
	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/ports"
	"github.com/rs/zerolog"
)

func TestLogin_WithAPIKey(t *testing.T) {
	h, rawKey := setupHandler(t)

	body := map[string]string{
		"api_key": rawKey,
		"email":   "admin@test.com",
	}
	resp := doRequest(t, h, "POST", "/login", body, "")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Login failed: status=%d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["session_id"] == nil {
		t.Error("Expected session_id in response")
	}
}

func TestLogin_InvalidAPIKey(t *testing.T) {
	h, _ := setupHandler(t)

	body := map[string]string{
		"api_key": "invalid_key",
	}
	resp := doRequest(t, h, "POST", "/login", body, "")

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestListUsers_Authenticated(t *testing.T) {
	h, rawKey := setupHandler(t)

	resp := doRequest(t, h, "GET", "/users", nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	users := result["users"].([]interface{})
	if len(users) != 1 { // We created one admin user
		t.Errorf("Expected 1 user, got %d", len(users))
	}
}

func TestListUsers_Unauthenticated(t *testing.T) {
	h, _ := setupHandler(t)

	resp := doRequest(t, h, "GET", "/users", nil, "")

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestCreateUser(t *testing.T) {
	h, rawKey := setupHandler(t)

	body := map[string]string{
		"email":   "newuser@test.com",
		"plan_id": "pro",
	}
	resp := doRequest(t, h, "POST", "/users", body, rawKey)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["email"] != "newuser@test.com" {
		t.Errorf("Expected email=newuser@test.com, got %s", result["email"])
	}
	if result["plan_id"] != "pro" {
		t.Errorf("Expected plan_id=pro, got %s", result["plan_id"])
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	h, rawKey := setupHandler(t)

	// First user already exists (admin@test.com)
	body := map[string]string{
		"email": "admin@test.com",
	}
	resp := doRequest(t, h, "POST", "/users", body, rawKey)

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("Expected 409 for duplicate email, got %d", resp.StatusCode)
	}
}

func TestGetUser(t *testing.T) {
	h, rawKey := setupHandler(t)

	// Create a user first
	body := map[string]string{"email": "getme@test.com"}
	createResp := doRequest(t, h, "POST", "/users", body, rawKey)

	var created map[string]interface{}
	json.NewDecoder(createResp.Body).Decode(&created)
	userID := created["id"].(string)

	// Get the user
	resp := doRequest(t, h, "GET", "/users/"+userID, nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["email"] != "getme@test.com" {
		t.Errorf("Expected email=getme@test.com, got %s", result["email"])
	}
}

func TestUpdateUser(t *testing.T) {
	h, rawKey := setupHandler(t)

	// Create a user
	createBody := map[string]string{"email": "update@test.com"}
	createResp := doRequest(t, h, "POST", "/users", createBody, rawKey)

	var created map[string]interface{}
	json.NewDecoder(createResp.Body).Decode(&created)
	userID := created["id"].(string)

	// Update the user
	updateBody := map[string]string{"plan_id": "enterprise", "name": "Updated Name"}
	resp := doRequest(t, h, "PUT", "/users/"+userID, updateBody, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["plan_id"] != "enterprise" {
		t.Errorf("Expected plan_id=enterprise, got %s", result["plan_id"])
	}
	if result["name"] != "Updated Name" {
		t.Errorf("Expected name=Updated Name, got %s", result["name"])
	}
}

func TestDeleteUser(t *testing.T) {
	h, rawKey := setupHandler(t)

	// Create a user
	createBody := map[string]string{"email": "delete@test.com"}
	createResp := doRequest(t, h, "POST", "/users", createBody, rawKey)

	var created map[string]interface{}
	json.NewDecoder(createResp.Body).Decode(&created)
	userID := created["id"].(string)

	// Delete the user
	resp := doRequest(t, h, "DELETE", "/users/"+userID, nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	// Verify user is marked as deleted
	getResp := doRequest(t, h, "GET", "/users/"+userID, nil, rawKey)
	var result map[string]interface{}
	json.NewDecoder(getResp.Body).Decode(&result)

	if result["status"] != "deleted" {
		t.Errorf("Expected status=deleted, got %s", result["status"])
	}
}

func TestCreateKey(t *testing.T) {
	h, rawKey := setupHandler(t)

	// Create a user first
	userBody := map[string]string{"email": "keyuser@test.com"}
	userResp := doRequest(t, h, "POST", "/users", userBody, rawKey)

	var user map[string]interface{}
	json.NewDecoder(userResp.Body).Decode(&user)
	userID := user["id"].(string)

	// Create a key
	keyBody := map[string]string{"user_id": userID, "name": "Test Key"}
	resp := doRequest(t, h, "POST", "/keys", keyBody, rawKey)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["key"] == nil {
		t.Error("Expected key in response")
	}
	if result["user_id"] != userID {
		t.Errorf("Expected user_id=%s, got %s", userID, result["user_id"])
	}
}

func TestListKeys(t *testing.T) {
	h, rawKey := setupHandler(t)

	resp := doRequest(t, h, "GET", "/keys", nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// Should have at least the admin key
	keys := result["keys"].([]interface{})
	if len(keys) < 1 {
		t.Errorf("Expected at least 1 key, got %d", len(keys))
	}
}

func TestRevokeKey(t *testing.T) {
	h, rawKey := setupHandler(t)

	// Create a user and key
	userBody := map[string]string{"email": "revokekey@test.com"}
	userResp := doRequest(t, h, "POST", "/users", userBody, rawKey)
	var user map[string]interface{}
	json.NewDecoder(userResp.Body).Decode(&user)

	keyBody := map[string]string{"user_id": user["id"].(string)}
	keyResp := doRequest(t, h, "POST", "/keys", keyBody, rawKey)
	var keyResult map[string]interface{}
	json.NewDecoder(keyResp.Body).Decode(&keyResult)
	keyID := keyResult["key_id"].(string)

	// Revoke the key
	resp := doRequest(t, h, "DELETE", "/keys/"+keyID, nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestListPlans(t *testing.T) {
	h, rawKey := setupHandler(t)

	resp := doRequest(t, h, "GET", "/plans", nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	plans := result["plans"].([]interface{})
	if len(plans) != 2 { // free and pro from test config
		t.Errorf("Expected 2 plans, got %d", len(plans))
	}
}

func TestGetUsage(t *testing.T) {
	h, rawKey := setupHandler(t)

	resp := doRequest(t, h, "GET", "/usage?period=month", nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["period"] != "month" {
		t.Errorf("Expected period=month, got %s", result["period"])
	}
}

func TestGetSettings(t *testing.T) {
	h, rawKey := setupHandler(t)

	resp := doRequest(t, h, "GET", "/settings", nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	server := result["server"].(map[string]interface{})
	if server["port"].(float64) != 8080 {
		t.Errorf("Expected port=8080, got %v", server["port"])
	}
}

func TestDoctor(t *testing.T) {
	h, rawKey := setupHandler(t)

	resp := doRequest(t, h, "GET", "/doctor", nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["status"] == nil {
		t.Error("Expected status in response")
	}
	if result["checks"] == nil {
		t.Error("Expected checks in response")
	}
	if result["system"] == nil {
		t.Error("Expected system in response")
	}
}

// Test helpers

func setupHandler(t *testing.T) (*admin.Handler, string) {
	t.Helper()

	// Create stores
	userStore := memory.NewUserStore()
	keyStore := memory.NewKeyStore()
	h := hasher.NewBcrypt(4) // low cost for tests

	// Create admin user
	adminUser := ports.User{
		ID:        "user_admin",
		Email:     "admin@test.com",
		PlanID:    "free",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	userStore.Create(context.Background(), adminUser)

	// Create admin API key
	rawKey, keyData := key.Generate("ak_")
	keyData = keyData.WithUserID(adminUser.ID)
	keyStore.Create(context.Background(), keyData)

	// Create config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Upstream: config.UpstreamConfig{
			URL: "http://localhost:3000",
		},
		Plans: []config.PlanConfig{
			{ID: "free", Name: "Free", RateLimitPerMinute: 60},
			{ID: "pro", Name: "Pro", RateLimitPerMinute: 600},
		},
		RateLimit: config.RateLimitConfig{Enabled: true},
	}

	// Create handler
	handler := admin.NewHandler(admin.Deps{
		Users:  userStore,
		Keys:   keyStore,
		Usage:  nil, // Usage store not needed for most tests
		Config: cfg,
		Logger: zerolog.Nop(),
		Hasher: h,
	})

	return handler, rawKey
}

func doRequest(t *testing.T, h *admin.Handler, method, path string, body interface{}, apiKey string) *http.Response {
	t.Helper()

	var bodyReader *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(b)
	} else {
		bodyReader = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	rec := httptest.NewRecorder()
	h.Router().ServeHTTP(rec, req)

	return rec.Result()
}
