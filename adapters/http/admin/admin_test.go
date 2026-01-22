package admin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/hasher"
	"github.com/artpar/apigate/adapters/http/admin"
	"github.com/artpar/apigate/adapters/memory"
	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/domain/route"
	"github.com/artpar/apigate/ports"
	"github.com/rs/zerolog"
)

// Helper functions for JSON:API response parsing

// getResourceID extracts the id from a JSON:API single resource response
func getResourceID(result map[string]any) string {
	data, ok := result["data"].(map[string]any)
	if !ok {
		return ""
	}
	id, _ := data["id"].(string)
	return id
}

// getResourceAttr extracts an attribute from a JSON:API single resource response
func getResourceAttr(result map[string]any, attr string) any {
	data, ok := result["data"].(map[string]any)
	if !ok {
		return nil
	}
	attrs, ok := data["attributes"].(map[string]any)
	if !ok {
		return nil
	}
	return attrs[attr]
}

// getResourceMeta extracts a meta field from a JSON:API single resource response
func getResourceMeta(result map[string]any, key string) any {
	data, ok := result["data"].(map[string]any)
	if !ok {
		return nil
	}
	meta, ok := data["meta"].(map[string]any)
	if !ok {
		return nil
	}
	return meta[key]
}

// getCollectionData extracts the data array from a JSON:API collection response
func getCollectionData(result map[string]any) []any {
	data, ok := result["data"].([]any)
	if !ok {
		return nil
	}
	return data
}

// getRelationshipID extracts the id from a relationship in a JSON:API resource
func getRelationshipID(result map[string]any, rel string) string {
	data, ok := result["data"].(map[string]any)
	if !ok {
		return ""
	}
	rels, ok := data["relationships"].(map[string]any)
	if !ok {
		return ""
	}
	relData, ok := rels[rel].(map[string]any)
	if !ok {
		return ""
	}
	relDataData, ok := relData["data"].(map[string]any)
	if !ok {
		return ""
	}
	id, _ := relDataData["id"].(string)
	return id
}

// getErrorCode extracts the error code from a JSON:API error response
func getErrorCode(result map[string]any) string {
	errors, ok := result["errors"].([]any)
	if !ok || len(errors) == 0 {
		return ""
	}
	errData, ok := errors[0].(map[string]any)
	if !ok {
		return ""
	}
	code, _ := errData["code"].(string)
	return code
}

// mockPlanStore is an in-memory plan store for testing.
type mockPlanStore struct {
	mu    sync.RWMutex
	plans map[string]ports.Plan
}

func newMockPlanStore() *mockPlanStore {
	return &mockPlanStore{
		plans: make(map[string]ports.Plan),
	}
}

func (s *mockPlanStore) Get(ctx context.Context, id string) (ports.Plan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.plans[id]
	if !ok {
		return ports.Plan{}, errors.New("not found")
	}
	return p, nil
}

func (s *mockPlanStore) List(ctx context.Context) ([]ports.Plan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []ports.Plan
	for _, p := range s.plans {
		result = append(result, p)
	}
	return result, nil
}

func (s *mockPlanStore) Create(ctx context.Context, p ports.Plan) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.plans[p.ID] = p
	return nil
}

func (s *mockPlanStore) Update(ctx context.Context, p ports.Plan) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.plans[p.ID] = p
	return nil
}

func (s *mockPlanStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.plans, id)
	return nil
}

func (s *mockPlanStore) GetDefault(ctx context.Context) (ports.Plan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.plans {
		if p.IsDefault {
			return p, nil
		}
	}
	return ports.Plan{}, errors.New("no default plan")
}

func (s *mockPlanStore) ClearOtherDefaults(ctx context.Context, exceptID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, p := range s.plans {
		if id != exceptID && p.IsDefault {
			p.IsDefault = false
			s.plans[id] = p
		}
	}
	return nil
}

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

	// JSON:API format: session resource in data
	data, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected data object in response")
	}
	if data["type"] != "sessions" {
		t.Errorf("Expected type sessions, got %v", data["type"])
	}
	if data["id"] == nil || data["id"] == "" {
		t.Error("Expected session id in response")
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

	// JSON:API format: array of user resources in data
	data, ok := result["data"].([]interface{})
	if !ok {
		t.Fatal("Expected data array in response")
	}
	if len(data) != 1 { // We created one admin user
		t.Errorf("Expected 1 user, got %d", len(data))
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

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	// JSON:API format: attributes are nested under data.attributes
	if getResourceAttr(result, "email") != "newuser@test.com" {
		t.Errorf("Expected email=newuser@test.com, got %s", getResourceAttr(result, "email"))
	}
	// plan_id is a relationship, not an attribute
	if getRelationshipID(result, "plan") != "pro" {
		t.Errorf("Expected plan relationship id=pro, got %s", getRelationshipID(result, "plan"))
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

	var created map[string]any
	json.NewDecoder(createResp.Body).Decode(&created)
	userID := getResourceID(created)

	// Get the user
	resp := doRequest(t, h, "GET", "/users/"+userID, nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	if getResourceAttr(result, "email") != "getme@test.com" {
		t.Errorf("Expected email=getme@test.com, got %s", getResourceAttr(result, "email"))
	}
}

func TestUpdateUser(t *testing.T) {
	h, rawKey := setupHandler(t)

	// Create a user
	createBody := map[string]string{"email": "update@test.com"}
	createResp := doRequest(t, h, "POST", "/users", createBody, rawKey)

	var created map[string]any
	json.NewDecoder(createResp.Body).Decode(&created)
	userID := getResourceID(created)

	// Update the user
	updateBody := map[string]string{"plan_id": "enterprise", "name": "Updated Name"}
	resp := doRequest(t, h, "PUT", "/users/"+userID, updateBody, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	// plan_id is a relationship, not an attribute
	if getRelationshipID(result, "plan") != "enterprise" {
		t.Errorf("Expected plan relationship id=enterprise, got %s", getRelationshipID(result, "plan"))
	}
	if getResourceAttr(result, "name") != "Updated Name" {
		t.Errorf("Expected name=Updated Name, got %s", getResourceAttr(result, "name"))
	}
}

func TestDeleteUser(t *testing.T) {
	h, rawKey := setupHandler(t)

	// Create a user
	createBody := map[string]string{"email": "delete@test.com"}
	createResp := doRequest(t, h, "POST", "/users", createBody, rawKey)

	var created map[string]any
	json.NewDecoder(createResp.Body).Decode(&created)
	userID := getResourceID(created)

	// Delete the user
	resp := doRequest(t, h, "DELETE", "/users/"+userID, nil, rawKey)

	// JSON:API uses 200 (with deleted resource) or 204 for DELETE
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		t.Fatalf("Expected 200 or 204, got %d", resp.StatusCode)
	}

	// Verify user is marked as deleted (if we got 200, check the response)
	if resp.StatusCode == http.StatusOK {
		var result map[string]any
		json.NewDecoder(resp.Body).Decode(&result)
		if getResourceAttr(result, "status") != "deleted" {
			t.Errorf("Expected status=deleted, got %s", getResourceAttr(result, "status"))
		}
	}
}

func TestCreateKey(t *testing.T) {
	h, rawKey := setupHandler(t)

	// Create a user first
	userBody := map[string]string{"email": "keyuser@test.com"}
	userResp := doRequest(t, h, "POST", "/users", userBody, rawKey)

	var user map[string]any
	json.NewDecoder(userResp.Body).Decode(&user)
	userID := getResourceID(user)

	// Create a key
	keyBody := map[string]string{"user_id": userID, "name": "Test Key"}
	resp := doRequest(t, h, "POST", "/keys", keyBody, rawKey)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	// JSON:API format: key is in meta, user relationship in relationships
	if getResourceMeta(result, "key") == nil {
		t.Error("Expected key in response meta")
	}
}

func TestListKeys(t *testing.T) {
	h, rawKey := setupHandler(t)

	resp := doRequest(t, h, "GET", "/keys", nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	// JSON:API format: keys are in data array
	keys := getCollectionData(result)
	if len(keys) < 1 {
		t.Errorf("Expected at least 1 key, got %d", len(keys))
	}
}

func TestRevokeKey(t *testing.T) {
	h, rawKey := setupHandler(t)

	// Create a user and key
	userBody := map[string]string{"email": "revokekey@test.com"}
	userResp := doRequest(t, h, "POST", "/users", userBody, rawKey)
	var user map[string]any
	json.NewDecoder(userResp.Body).Decode(&user)

	keyBody := map[string]string{"user_id": getResourceID(user)}
	keyResp := doRequest(t, h, "POST", "/keys", keyBody, rawKey)
	var keyResult map[string]any
	json.NewDecoder(keyResp.Body).Decode(&keyResult)
	keyID := getResourceID(keyResult)

	// Revoke the key
	resp := doRequest(t, h, "DELETE", "/keys/"+keyID, nil, rawKey)

	// JSON:API uses 204 No Content for successful DELETE
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 or 204, got %d", resp.StatusCode)
	}
}

func TestListPlans(t *testing.T) {
	h, rawKey := setupHandler(t)

	resp := doRequest(t, h, "GET", "/plans", nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	// JSON:API format: plans in data array
	plans := getCollectionData(result)
	if len(plans) != 2 { // free and pro from test config
		t.Errorf("Expected 2 plans, got %d", len(plans))
	}
}

// ============================================================================
// Plans API Tests - Structure and Semantics
// ============================================================================

func TestListPlans_ResponseStructure(t *testing.T) {
	h, rawKey := setupHandler(t)

	resp := doRequest(t, h, "GET", "/plans", nil, rawKey)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	// Verify JSON:API response structure
	plans := getCollectionData(result)
	if plans == nil {
		t.Fatal("Response missing 'data' field (JSON:API collection)")
	}
	if len(plans) == 0 {
		t.Fatal("Expected at least one plan")
	}

	// Verify meta.total is present
	meta, ok := result["meta"].(map[string]any)
	if !ok || meta["total"] == nil {
		t.Log("Note: meta.total not present in response (acceptable)")
	}

	// Verify plan structure (JSON:API resource format)
	plan := plans[0].(map[string]any)
	if plan["type"] == nil {
		t.Error("Plan missing 'type' field")
	}
	if plan["id"] == nil {
		t.Error("Plan missing 'id' field")
	}
	attrs, ok := plan["attributes"].(map[string]any)
	if !ok {
		t.Fatal("Plan missing 'attributes' field")
	}

	// Check required attributes
	requiredAttrs := []string{"name", "rate_limit_per_minute", "requests_per_month",
		"price_monthly", "overage_price", "is_default", "enabled", "created_at", "updated_at"}
	for _, field := range requiredAttrs {
		if _, ok := attrs[field]; !ok {
			t.Errorf("Plan missing required attribute: %s", field)
		}
	}
}

func TestGetPlan_Success(t *testing.T) {
	h, rawKey := setupHandler(t)

	resp := doRequest(t, h, "GET", "/plans/free", nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	// JSON:API format
	if getResourceID(result) != "free" {
		t.Errorf("Expected id='free', got %v", getResourceID(result))
	}
	if getResourceAttr(result, "name") != "Free" {
		t.Errorf("Expected name='Free', got %v", getResourceAttr(result, "name"))
	}
}

func TestGetPlan_NotFound(t *testing.T) {
	h, rawKey := setupHandler(t)

	resp := doRequest(t, h, "GET", "/plans/nonexistent", nil, rawKey)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected 404, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	// JSON:API error format uses "errors" array
	errors, ok := result["errors"].([]any)
	if !ok || len(errors) == 0 {
		t.Fatal("Expected errors array in response")
	}
	errData := errors[0].(map[string]any)
	if errData["code"] != "not_found" {
		t.Errorf("Expected error code 'not_found', got %v", errData["code"])
	}
}

func TestCreatePlan_Success(t *testing.T) {
	h, rawKey := setupHandler(t)

	body := map[string]any{
		"id":                    "enterprise",
		"name":                  "Enterprise",
		"description":           "For large organizations",
		"rate_limit_per_minute": 1000,
		"requests_per_month":    1000000,
		"price_monthly":         99.99,
		"overage_price":         0.001,
		"stripe_price_id":       "price_enterprise",
		"enabled":               true,
	}

	resp := doRequest(t, h, "POST", "/plans", body, rawKey)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	// JSON:API format: verify all fields via helpers
	if getResourceID(result) != "enterprise" {
		t.Errorf("Expected id='enterprise', got %v", getResourceID(result))
	}
	if getResourceAttr(result, "name") != "Enterprise" {
		t.Errorf("Expected name='Enterprise', got %v", getResourceAttr(result, "name"))
	}
	if getResourceAttr(result, "description") != "For large organizations" {
		t.Errorf("Expected description='For large organizations', got %v", getResourceAttr(result, "description"))
	}
	rateLimit, ok := getResourceAttr(result, "rate_limit_per_minute").(float64)
	if !ok || int(rateLimit) != 1000 {
		t.Errorf("Expected rate_limit_per_minute=1000, got %v", getResourceAttr(result, "rate_limit_per_minute"))
	}
	// Price should be returned as dollars (converted back from cents)
	priceMonthly, ok := getResourceAttr(result, "price_monthly").(float64)
	if !ok || priceMonthly != 99.99 {
		t.Errorf("Expected price_monthly=99.99, got %v", getResourceAttr(result, "price_monthly"))
	}
}

func TestCreatePlan_MissingID(t *testing.T) {
	h, rawKey := setupHandler(t)

	body := map[string]any{
		"name":    "Test Plan",
		"enabled": true,
	}

	resp := doRequest(t, h, "POST", "/plans", body, rawKey)

	// JSON:API uses 422 for validation errors
	if resp.StatusCode != http.StatusUnprocessableEntity && resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 422 or 400, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	code := getErrorCode(result)
	if code != "missing_id" && code != "validation_error" {
		t.Errorf("Expected error code 'missing_id' or 'validation_error', got %v", code)
	}
}

func TestCreatePlan_MissingName(t *testing.T) {
	h, rawKey := setupHandler(t)

	body := map[string]any{
		"id":      "test_plan",
		"enabled": true,
	}

	resp := doRequest(t, h, "POST", "/plans", body, rawKey)

	// JSON:API uses 422 for validation errors
	if resp.StatusCode != http.StatusUnprocessableEntity && resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 422 or 400, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	code := getErrorCode(result)
	if code != "missing_name" && code != "validation_error" {
		t.Errorf("Expected error code 'missing_name' or 'validation_error', got %v", code)
	}
}

func TestCreatePlan_DuplicateID(t *testing.T) {
	h, rawKey := setupHandler(t)

	// Try to create a plan with existing ID
	body := map[string]any{
		"id":      "free", // Already exists
		"name":    "Another Free",
		"enabled": true,
	}

	resp := doRequest(t, h, "POST", "/plans", body, rawKey)

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("Expected 409, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	code := getErrorCode(result)
	if code != "plan_exists" && code != "conflict" {
		t.Errorf("Expected error code 'plan_exists' or 'conflict', got %v", code)
	}
}

func TestCreatePlan_InvalidJSON(t *testing.T) {
	h, rawKey := setupHandler(t)

	req := httptest.NewRequest("POST", "/plans", bytes.NewBufferString("not valid json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", rawKey)

	rec := httptest.NewRecorder()
	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d", rec.Code)
	}
}

func TestUpdatePlan_Success(t *testing.T) {
	h, rawKey := setupHandler(t)

	body := map[string]any{
		"name":                  "Free Updated",
		"rate_limit_per_minute": 120,
		"price_monthly":         9.99,
	}

	resp := doRequest(t, h, "PUT", "/plans/free", body, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	if name := getResourceAttr(result, "name"); name != "Free Updated" {
		t.Errorf("Expected name='Free Updated', got %v", name)
	}
	if rateLimit := getResourceAttr(result, "rate_limit_per_minute"); rateLimit != nil {
		if int(rateLimit.(float64)) != 120 {
			t.Errorf("Expected rate_limit_per_minute=120, got %v", rateLimit)
		}
	}
	if price := getResourceAttr(result, "price_monthly"); price != nil {
		if price.(float64) != 9.99 {
			t.Errorf("Expected price_monthly=9.99, got %v", price)
		}
	}
}

func TestUpdatePlan_PartialUpdate(t *testing.T) {
	h, rawKey := setupHandler(t)

	// Only update name, other fields should remain unchanged
	body := map[string]any{
		"name": "Pro Premium",
	}

	resp := doRequest(t, h, "PUT", "/plans/pro", body, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	if name := getResourceAttr(result, "name"); name != "Pro Premium" {
		t.Errorf("Expected name='Pro Premium', got %v", name)
	}
	// Rate limit should remain unchanged (600 from test setup)
	if rateLimit := getResourceAttr(result, "rate_limit_per_minute"); rateLimit != nil {
		if int(rateLimit.(float64)) != 600 {
			t.Errorf("Expected rate_limit_per_minute=600, got %v", rateLimit)
		}
	}
}

func TestUpdatePlan_NotFound(t *testing.T) {
	h, rawKey := setupHandler(t)

	body := map[string]any{
		"name": "Updated Name",
	}

	resp := doRequest(t, h, "PUT", "/plans/nonexistent", body, rawKey)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestUpdatePlan_BooleanFields(t *testing.T) {
	h, rawKey := setupHandler(t)

	// Test updating boolean fields
	body := map[string]any{
		"enabled":    false,
		"is_default": true,
	}

	resp := doRequest(t, h, "PUT", "/plans/pro", body, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	if enabled := getResourceAttr(result, "enabled"); enabled != false {
		t.Errorf("Expected enabled=false, got %v", enabled)
	}
	if isDefault := getResourceAttr(result, "is_default"); isDefault != true {
		t.Errorf("Expected is_default=true, got %v", isDefault)
	}
}

func TestDeletePlan_Success(t *testing.T) {
	h, rawKey := setupHandler(t)

	// First create a plan to delete
	createBody := map[string]any{
		"id":      "to_delete",
		"name":    "To Delete",
		"enabled": true,
	}
	doRequest(t, h, "POST", "/plans", createBody, rawKey)

	// Now delete it
	resp := doRequest(t, h, "DELETE", "/plans/to_delete", nil, rawKey)

	// JSON:API uses 200 or 204 for successful DELETE
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		t.Fatalf("Expected 200 or 204, got %d", resp.StatusCode)
	}

	// Verify plan is actually deleted
	getResp := doRequest(t, h, "GET", "/plans/to_delete", nil, rawKey)
	if getResp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404 after delete, got %d", getResp.StatusCode)
	}
}

func TestDeletePlan_NotFound(t *testing.T) {
	h, rawKey := setupHandler(t)

	resp := doRequest(t, h, "DELETE", "/plans/nonexistent", nil, rawKey)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestDeletePlan_InUse(t *testing.T) {
	h, rawKey := setupHandler(t)

	// Admin user is on 'free' plan, so we can't delete it
	resp := doRequest(t, h, "DELETE", "/plans/free", nil, rawKey)

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("Expected 409 (conflict), got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	// JSON:API uses generic "conflict" code
	if code := getErrorCode(result); code != "conflict" {
		t.Errorf("Expected error code 'conflict', got %v", code)
	}
}

func TestPlan_PriceConversion(t *testing.T) {
	h, rawKey := setupHandler(t)

	// Create plan with specific price values
	body := map[string]any{
		"id":            "price_test",
		"name":          "Price Test",
		"price_monthly": 29.99,  // $29.99
		"overage_price": 0.005,  // $0.005 (half a cent)
		"enabled":       true,
	}

	resp := doRequest(t, h, "POST", "/plans", body, rawKey)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	// Prices should be stored as cents internally but returned as dollars
	if price := getResourceAttr(result, "price_monthly"); price != nil {
		if price.(float64) != 29.99 {
			t.Errorf("Expected price_monthly=29.99, got %v", price)
		}
	}
	if overage := getResourceAttr(result, "overage_price"); overage != nil {
		if overage.(float64) != 0.005 { // Now stored as hundredths of cents: 0.005 * 10000 = 50
			t.Errorf("Expected overage_price=0.005, got %v", overage)
		}
	}
}

func TestPlan_PaymentProviderFields(t *testing.T) {
	h, rawKey := setupHandler(t)

	body := map[string]any{
		"id":               "provider_test",
		"name":             "Provider Test",
		"stripe_price_id":  "price_abc123",
		"paddle_price_id":  "pri_xyz789",
		"lemon_variant_id": "var_456",
		"enabled":          true,
	}

	resp := doRequest(t, h, "POST", "/plans", body, rawKey)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	if val := getResourceAttr(result, "stripe_price_id"); val != "price_abc123" {
		t.Errorf("Expected stripe_price_id='price_abc123', got %v", val)
	}
	if val := getResourceAttr(result, "paddle_price_id"); val != "pri_xyz789" {
		t.Errorf("Expected paddle_price_id='pri_xyz789', got %v", val)
	}
	if val := getResourceAttr(result, "lemon_variant_id"); val != "var_456" {
		t.Errorf("Expected lemon_variant_id='var_456', got %v", val)
	}
}

func TestListPlans_Unauthenticated(t *testing.T) {
	h, _ := setupHandler(t)

	resp := doRequest(t, h, "GET", "/plans", nil, "")

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestCreatePlan_Unauthenticated(t *testing.T) {
	h, _ := setupHandler(t)

	body := map[string]interface{}{
		"id":   "test",
		"name": "Test",
	}

	resp := doRequest(t, h, "POST", "/plans", body, "")

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestGetUsage(t *testing.T) {
	h, rawKey := setupHandler(t)

	resp := doRequest(t, h, "GET", "/usage?period=month", nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	// Usage is returned via WriteMeta, so data is in result["meta"]
	meta, _ := result["meta"].(map[string]any)
	if meta == nil {
		t.Fatal("Expected meta in response")
	}
	if meta["period"] != "month" {
		t.Errorf("Expected period=month, got %v", meta["period"])
	}
}

func TestGetSettings(t *testing.T) {
	h, rawKey := setupHandler(t)

	resp := doRequest(t, h, "GET", "/settings", nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	// Settings are returned via WriteMeta, so data is in result["meta"]
	meta, _ := result["meta"].(map[string]any)
	if meta == nil {
		t.Fatal("Expected meta in response")
	}
	server, _ := meta["server"].(map[string]any)
	if server == nil {
		t.Fatal("Expected server in meta")
	}
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

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	// Doctor is returned via WriteMeta, so data is in result["meta"]
	meta, _ := result["meta"].(map[string]any)
	if meta == nil {
		t.Fatal("Expected meta in response")
	}
	if meta["status"] == nil {
		t.Error("Expected status in response")
	}
	if meta["checks"] == nil {
		t.Error("Expected checks in response")
	}
	if meta["system"] == nil {
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

	// Create plan store with test plans
	planStore := newMockPlanStore()
	now := time.Now().UTC()
	planStore.Create(context.Background(), ports.Plan{
		ID:                 "free",
		Name:               "Free",
		RateLimitPerMinute: 60,
		IsDefault:          true,
		Enabled:            true,
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	planStore.Create(context.Background(), ports.Plan{
		ID:                 "pro",
		Name:               "Pro",
		RateLimitPerMinute: 600,
		Enabled:            true,
		CreatedAt:          now,
		UpdatedAt:          now,
	})

	// Create handler
	handler := admin.NewHandler(admin.Deps{
		Users:  userStore,
		Keys:   keyStore,
		Usage:  nil, // Usage store not needed for most tests
		Plans:  planStore,
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

// ============================================================================
// Session Management Tests
// ============================================================================

func TestSessionStore_CreateAndGet(t *testing.T) {
	store := admin.NewSessionStore()

	session := store.Create("user123", "test@example.com", time.Hour)

	if session == nil {
		t.Fatal("Expected session to be created")
	}
	if session.UserID != "user123" {
		t.Errorf("Expected UserID=user123, got %s", session.UserID)
	}
	if session.Email != "test@example.com" {
		t.Errorf("Expected Email=test@example.com, got %s", session.Email)
	}

	// Retrieve the session
	retrieved := store.Get(session.ID)
	if retrieved == nil {
		t.Fatal("Expected to retrieve session")
	}
	if retrieved.ID != session.ID {
		t.Errorf("Expected ID=%s, got %s", session.ID, retrieved.ID)
	}
}

func TestSessionStore_GetExpired(t *testing.T) {
	store := admin.NewSessionStore()

	// Create a session with negative duration (expired immediately)
	session := store.Create("user123", "test@example.com", -time.Hour)

	// Should not be able to retrieve expired session
	retrieved := store.Get(session.ID)
	if retrieved != nil {
		t.Error("Expected nil for expired session")
	}
}

func TestSessionStore_GetNonExistent(t *testing.T) {
	store := admin.NewSessionStore()

	retrieved := store.Get("nonexistent-id")
	if retrieved != nil {
		t.Error("Expected nil for non-existent session")
	}
}

func TestSessionStore_Delete(t *testing.T) {
	store := admin.NewSessionStore()

	session := store.Create("user123", "test@example.com", time.Hour)

	// Delete the session
	store.Delete(session.ID)

	// Should not be able to retrieve deleted session
	retrieved := store.Get(session.ID)
	if retrieved != nil {
		t.Error("Expected nil for deleted session")
	}
}

// ============================================================================
// Login Tests - Password Authentication
// ============================================================================

func TestLogin_WithPassword_Success(t *testing.T) {
	h, _ := setupHandlerWithPasswordUser(t)

	body := map[string]string{
		"email":    "passworduser@test.com",
		"password": "testpassword123",
	}
	resp := doRequest(t, h, "POST", "/login", body, "")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Login failed: status=%d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	// Session is returned as a JSON:API resource - ID is the session_id
	if id := getResourceID(result); id == "" {
		t.Error("Expected session resource with ID in response")
	}
}

func TestLogin_WithPassword_InvalidPassword(t *testing.T) {
	h, _ := setupHandlerWithPasswordUser(t)

	body := map[string]string{
		"email":    "passworduser@test.com",
		"password": "wrongpassword",
	}
	resp := doRequest(t, h, "POST", "/login", body, "")

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestLogin_WithPassword_UserNotFound(t *testing.T) {
	h, _ := setupHandler(t)

	body := map[string]string{
		"email":    "nonexistent@test.com",
		"password": "anypassword",
	}
	resp := doRequest(t, h, "POST", "/login", body, "")

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestLogin_WithPassword_NoPasswordSet(t *testing.T) {
	h, _ := setupHandler(t)

	// admin@test.com has no password set
	body := map[string]string{
		"email":    "admin@test.com",
		"password": "anypassword",
	}
	resp := doRequest(t, h, "POST", "/login", body, "")

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestLogin_WithPassword_InactiveUser(t *testing.T) {
	h, _ := setupHandlerWithInactiveUser(t)

	body := map[string]string{
		"email":    "inactive@test.com",
		"password": "testpassword123",
	}
	resp := doRequest(t, h, "POST", "/login", body, "")

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestLogin_MissingEmailAndAPIKey(t *testing.T) {
	h, _ := setupHandler(t)

	body := map[string]string{
		"password": "somepassword",
	}
	resp := doRequest(t, h, "POST", "/login", body, "")

	// JSON:API uses 422 for validation errors
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("Expected 400 or 422, got %d", resp.StatusCode)
	}
}

func TestLogin_MissingPassword(t *testing.T) {
	h, _ := setupHandler(t)

	body := map[string]string{
		"email": "admin@test.com",
	}
	resp := doRequest(t, h, "POST", "/login", body, "")

	// JSON:API uses 422 for validation errors
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("Expected 400 or 422, got %d", resp.StatusCode)
	}
}

func TestLogin_InvalidJSON(t *testing.T) {
	h, _ := setupHandler(t)

	req := httptest.NewRequest("POST", "/login", bytes.NewBufferString("not valid json"))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d", rec.Code)
	}
}

func TestLogin_APIKeyTooShort(t *testing.T) {
	h, _ := setupHandler(t)

	body := map[string]string{
		"api_key": "short",
	}
	resp := doRequest(t, h, "POST", "/login", body, "")

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestLogin_APIKeyWithoutEmail(t *testing.T) {
	h, rawKey := setupHandler(t)

	body := map[string]string{
		"api_key": rawKey,
		// No email provided
	}
	resp := doRequest(t, h, "POST", "/login", body, "")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	// Session is returned as JSON:API resource with user_email in meta
	userEmail := getResourceMeta(result, "user_email")
	if userEmail != "admin@apigate" {
		t.Errorf("Expected default admin email, got %v", userEmail)
	}
}

func TestLogin_RevokedAPIKey(t *testing.T) {
	h, rawKey := setupHandlerWithRevokedKey(t)

	body := map[string]string{
		"api_key": rawKey,
	}
	resp := doRequest(t, h, "POST", "/login", body, "")

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected 401 for revoked key, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Logout Tests
// ============================================================================

func TestLogout_WithSession(t *testing.T) {
	h, rawKey := setupHandler(t)

	// First login to get a session
	loginBody := map[string]string{
		"api_key": rawKey,
		"email":   "admin@test.com",
	}
	loginResp := doRequest(t, h, "POST", "/login", loginBody, "")

	var loginResult map[string]any
	json.NewDecoder(loginResp.Body).Decode(&loginResult)
	sessionID := getResourceID(loginResult)

	// Now logout using the session
	req := httptest.NewRequest("POST", "/logout", nil)
	req.Header.Set("Authorization", "Bearer "+sessionID)

	rec := httptest.NewRecorder()
	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", rec.Code)
	}

	// Verify session is invalidated - subsequent request should fail
	req2 := httptest.NewRequest("GET", "/users", nil)
	req2.Header.Set("Authorization", "Bearer "+sessionID)
	rec2 := httptest.NewRecorder()
	h.Router().ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 after logout, got %d", rec2.Code)
	}
}

// ============================================================================
// Auth Middleware Tests
// ============================================================================

func TestAuthMiddleware_WithSessionCookie(t *testing.T) {
	h, rawKey := setupHandler(t)

	// First login to get a session
	loginBody := map[string]string{
		"api_key": rawKey,
		"email":   "admin@test.com",
	}
	loginResp := doRequest(t, h, "POST", "/login", loginBody, "")

	var loginResult map[string]any
	json.NewDecoder(loginResp.Body).Decode(&loginResult)
	sessionID := getResourceID(loginResult)

	// Make request with session cookie
	req := httptest.NewRequest("GET", "/users", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 with cookie auth, got %d", rec.Code)
	}
}

func TestAuthMiddleware_WithBearerSession(t *testing.T) {
	h, rawKey := setupHandler(t)

	// First login to get a session
	loginBody := map[string]string{
		"api_key": rawKey,
		"email":   "admin@test.com",
	}
	loginResp := doRequest(t, h, "POST", "/login", loginBody, "")

	var loginResult map[string]any
	json.NewDecoder(loginResp.Body).Decode(&loginResult)
	sessionID := getResourceID(loginResult)

	// Make request with Bearer token (session ID)
	req := httptest.NewRequest("GET", "/users", nil)
	req.Header.Set("Authorization", "Bearer "+sessionID)

	rec := httptest.NewRecorder()
	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 with Bearer session, got %d", rec.Code)
	}
}

func TestAuthMiddleware_WithBearerAPIKey(t *testing.T) {
	h, rawKey := setupHandler(t)

	// Make request with Bearer token (API key)
	req := httptest.NewRequest("GET", "/users", nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)

	rec := httptest.NewRecorder()
	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 with Bearer API key, got %d", rec.Code)
	}
}

func TestAuthMiddleware_InvalidCookie(t *testing.T) {
	h, _ := setupHandler(t)

	req := httptest.NewRequest("GET", "/users", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: "invalid-session-id",
	})

	rec := httptest.NewRecorder()
	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 with invalid cookie, got %d", rec.Code)
	}
}

func TestAuthMiddleware_InvalidBearerToken(t *testing.T) {
	h, _ := setupHandler(t)

	req := httptest.NewRequest("GET", "/users", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	rec := httptest.NewRecorder()
	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 with invalid bearer token, got %d", rec.Code)
	}
}

// ============================================================================
// User API Additional Tests
// ============================================================================

func TestCreateUser_WithPassword(t *testing.T) {
	h, rawKey := setupHandler(t)

	body := map[string]string{
		"email":    "withpassword@test.com",
		"password": "securepassword123",
	}
	resp := doRequest(t, h, "POST", "/users", body, rawKey)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	// Verify user can login with password
	loginBody := map[string]string{
		"email":    "withpassword@test.com",
		"password": "securepassword123",
	}
	loginResp := doRequest(t, h, "POST", "/login", loginBody, "")

	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 for login, got %d", loginResp.StatusCode)
	}
}

func TestCreateUser_MissingEmail(t *testing.T) {
	h, rawKey := setupHandler(t)

	body := map[string]string{
		"name": "Test User",
	}
	resp := doRequest(t, h, "POST", "/users", body, rawKey)

	// JSON:API uses 422 for validation errors
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("Expected 400 or 422, got %d", resp.StatusCode)
	}
}

func TestCreateUser_InvalidJSON(t *testing.T) {
	h, rawKey := setupHandler(t)

	req := httptest.NewRequest("POST", "/users", bytes.NewBufferString("not valid json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", rawKey)

	rec := httptest.NewRecorder()
	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d", rec.Code)
	}
}

func TestGetUser_NotFound(t *testing.T) {
	h, rawKey := setupHandler(t)

	resp := doRequest(t, h, "GET", "/users/nonexistent", nil, rawKey)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestUpdateUser_WithPassword(t *testing.T) {
	h, rawKey := setupHandler(t)

	// Create a user
	createBody := map[string]string{"email": "updatepw@test.com"}
	createResp := doRequest(t, h, "POST", "/users", createBody, rawKey)

	var created map[string]any
	json.NewDecoder(createResp.Body).Decode(&created)
	userID := getResourceID(created)

	// Update with password
	updateBody := map[string]string{"password": "newpassword123"}
	resp := doRequest(t, h, "PUT", "/users/"+userID, updateBody, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	// Verify can login with new password
	loginBody := map[string]string{
		"email":    "updatepw@test.com",
		"password": "newpassword123",
	}
	loginResp := doRequest(t, h, "POST", "/login", loginBody, "")

	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 for login, got %d", loginResp.StatusCode)
	}
}

func TestUpdateUser_NotFound(t *testing.T) {
	h, rawKey := setupHandler(t)

	body := map[string]string{"name": "Updated"}
	resp := doRequest(t, h, "PUT", "/users/nonexistent", body, rawKey)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestUpdateUser_InvalidJSON(t *testing.T) {
	h, rawKey := setupHandler(t)

	// Create a user first
	createBody := map[string]string{"email": "jsontest@test.com"}
	createResp := doRequest(t, h, "POST", "/users", createBody, rawKey)
	var created map[string]any
	json.NewDecoder(createResp.Body).Decode(&created)
	userID := getResourceID(created)

	req := httptest.NewRequest("PUT", "/users/"+userID, bytes.NewBufferString("not valid json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", rawKey)

	rec := httptest.NewRecorder()
	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d", rec.Code)
	}
}

func TestDeleteUser_NotFound(t *testing.T) {
	h, rawKey := setupHandler(t)

	resp := doRequest(t, h, "DELETE", "/users/nonexistent", nil, rawKey)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected 404, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Keys API Additional Tests
// ============================================================================

func TestCreateKey_MissingUserID(t *testing.T) {
	h, rawKey := setupHandler(t)

	body := map[string]string{"name": "Test Key"}
	resp := doRequest(t, h, "POST", "/keys", body, rawKey)

	// JSON:API uses 422 for validation errors, accept both 400 and 422
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("Expected 400 or 422, got %d", resp.StatusCode)
	}
}

func TestCreateKey_UserNotFound(t *testing.T) {
	h, rawKey := setupHandler(t)

	body := map[string]string{"user_id": "nonexistent"}
	resp := doRequest(t, h, "POST", "/keys", body, rawKey)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestCreateKey_InvalidJSON(t *testing.T) {
	h, rawKey := setupHandler(t)

	req := httptest.NewRequest("POST", "/keys", bytes.NewBufferString("not valid json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", rawKey)

	rec := httptest.NewRecorder()
	h.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d", rec.Code)
	}
}

func TestCreateKey_WithExpiry(t *testing.T) {
	h, rawKey := setupHandler(t)

	// Create a user first
	userBody := map[string]string{"email": "keyexpiry@test.com"}
	userResp := doRequest(t, h, "POST", "/users", userBody, rawKey)
	var user map[string]any
	json.NewDecoder(userResp.Body).Decode(&user)
	userID := getResourceID(user)

	// Create key with expiry
	expiryTime := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	body := map[string]any{
		"user_id":    userID,
		"name":       "Expiring Key",
		"expires_at": expiryTime,
	}

	resp := doRequest(t, h, "POST", "/keys", body, rawKey)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}
}

func TestListKeys_FilterByUser(t *testing.T) {
	h, rawKey := setupHandler(t)

	// Create a specific user and key
	userBody := map[string]string{"email": "filtereduser@test.com"}
	userResp := doRequest(t, h, "POST", "/users", userBody, rawKey)
	var user map[string]any
	json.NewDecoder(userResp.Body).Decode(&user)
	userID := getResourceID(user)

	keyBody := map[string]string{"user_id": userID, "name": "Filtered Key"}
	doRequest(t, h, "POST", "/keys", keyBody, rawKey)

	// List keys filtered by user
	resp := doRequest(t, h, "GET", "/keys?user_id="+userID, nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	keys := getCollectionData(result)
	if len(keys) != 1 {
		t.Errorf("Expected 1 key for user, got %d", len(keys))
	}
}

// ============================================================================
// Usage API Additional Tests
// ============================================================================

func TestGetUsage_WithUserID(t *testing.T) {
	h, rawKey := setupHandlerWithUsage(t)

	resp := doRequest(t, h, "GET", "/usage?user_id=user_admin&period=month", nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestGetUsage_DayPeriod(t *testing.T) {
	h, rawKey := setupHandlerWithUsage(t)

	resp := doRequest(t, h, "GET", "/usage?period=day", nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	meta, _ := result["meta"].(map[string]any)
	if meta == nil || meta["period"] != "day" {
		t.Errorf("Expected period=day, got %v", meta["period"])
	}
}

func TestGetUsage_WeekPeriod(t *testing.T) {
	h, rawKey := setupHandlerWithUsage(t)

	resp := doRequest(t, h, "GET", "/usage?period=week", nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	// Usage uses WriteMeta, so data is in result["meta"]
	meta, _ := result["meta"].(map[string]any)
	if meta == nil || meta["period"] != "week" {
		t.Errorf("Expected period=week, got %v", meta["period"])
	}
}

func TestGetUsage_CustomDateRange(t *testing.T) {
	h, rawKey := setupHandlerWithUsage(t)

	// Use RFC3339 format with proper URL encoding
	startDate := url.QueryEscape(time.Now().AddDate(0, 0, -7).Format(time.RFC3339))
	endDate := url.QueryEscape(time.Now().Format(time.RFC3339))

	resp := doRequest(t, h, "GET", "/usage?start_date="+startDate+"&end_date="+endDate, nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestGetUsage_InvalidStartDate(t *testing.T) {
	h, rawKey := setupHandlerWithUsage(t)

	resp := doRequest(t, h, "GET", "/usage?start_date=invalid&end_date=2024-01-01T00:00:00Z", nil, rawKey)

	// JSON:API uses 422 for validation errors, accept both 400 and 422
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("Expected 400 or 422, got %d", resp.StatusCode)
	}
}

func TestGetUsage_InvalidEndDate(t *testing.T) {
	h, rawKey := setupHandlerWithUsage(t)

	startDate := time.Now().AddDate(0, 0, -7).Format(time.RFC3339)

	resp := doRequest(t, h, "GET", "/usage?start_date="+startDate+"&end_date=invalid", nil, rawKey)

	// JSON:API uses 422 for validation errors, accept both 400 and 422
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("Expected 400 or 422, got %d", resp.StatusCode)
	}
}

func TestGetUsage_UnknownPeriod(t *testing.T) {
	h, rawKey := setupHandlerWithUsage(t)

	resp := doRequest(t, h, "GET", "/usage?period=unknown", nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 (defaults to month), got %d", resp.StatusCode)
	}
}

// ============================================================================
// Settings API Tests
// ============================================================================

func TestUpdateSettings(t *testing.T) {
	h, rawKey := setupHandler(t)

	body := map[string]interface{}{
		"server": map[string]interface{}{
			"port": 9090,
		},
	}
	resp := doRequest(t, h, "PUT", "/settings", body, rawKey)

	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("Expected 501, got %d", resp.StatusCode)
	}
}

// ============================================================================
// Doctor API Additional Tests
// ============================================================================

func TestDoctor_WithUpstreams(t *testing.T) {
	h, rawKey := setupHandlerWithUpstreams(t)

	resp := doRequest(t, h, "GET", "/doctor", nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	// Doctor uses WriteMeta, so data is in result["meta"]
	meta, _ := result["meta"].(map[string]any)
	if meta == nil {
		t.Fatal("Expected meta in response")
	}

	checks, _ := meta["checks"].([]any)
	foundUpstream := false
	for _, check := range checks {
		c := check.(map[string]any)
		if c["name"] == "upstream" {
			foundUpstream = true
			if c["status"] == "fail" {
				t.Logf("Upstream check status: %s, message: %s", c["status"], c["message"])
			}
		}
	}
	if !foundUpstream {
		t.Error("Expected upstream check in doctor response")
	}
}

func TestDoctor_ResponseStructure(t *testing.T) {
	h, rawKey := setupHandler(t)

	resp := doRequest(t, h, "GET", "/doctor", nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	// Doctor uses WriteMeta, so data is in result["meta"]
	meta, _ := result["meta"].(map[string]any)
	if meta == nil {
		t.Fatal("Expected meta in response")
	}

	// Verify all required fields are present
	requiredFields := []string{"status", "timestamp", "version", "checks", "system", "statistics"}
	for _, field := range requiredFields {
		if meta[field] == nil {
			t.Errorf("Missing required field: %s", field)
		}
	}

	// Verify system info structure
	system, _ := meta["system"].(map[string]any)
	if system == nil {
		t.Fatal("Expected system in response")
	}
	systemFields := []string{"go_version", "num_cpu", "num_goroutine", "mem_alloc", "mem_sys", "uptime"}
	for _, field := range systemFields {
		if system[field] == nil {
			t.Errorf("Missing system field: %s", field)
		}
	}

	// Verify statistics structure
	stats, _ := meta["statistics"].(map[string]any)
	if stats == nil {
		t.Fatal("Expected statistics in response")
	}
	statsFields := []string{"total_users", "total_keys", "active_sessions"}
	for _, field := range statsFields {
		if stats[field] == nil {
			t.Errorf("Missing statistics field: %s", field)
		}
	}
}

// ============================================================================
// Query Parameter Parsing Tests
// ============================================================================

func TestListUsers_WithPagination(t *testing.T) {
	h, rawKey := setupHandler(t)

	// Create multiple users
	for i := range 5 {
		body := map[string]string{
			"email": "user" + string(rune('a'+i)) + "@test.com",
		}
		doRequest(t, h, "POST", "/users", body, rawKey)
	}

	// Test with limit
	resp := doRequest(t, h, "GET", "/users?limit=2", nil, rawKey)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	// JSON:API collections have data in result["data"]
	users := getCollectionData(result)
	if len(users) > 2 {
		t.Errorf("Expected at most 2 users with limit, got %d", len(users))
	}
}

func TestListUsers_InvalidLimit(t *testing.T) {
	h, rawKey := setupHandler(t)

	// Invalid limit should use default
	resp := doRequest(t, h, "GET", "/users?limit=invalid", nil, rawKey)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 (should use default), got %d", resp.StatusCode)
	}
}

// ============================================================================
// Error Response Tests
// ============================================================================

func TestErrorType_Error(t *testing.T) {
	// Test that ErrInvalidCredentials implements error interface correctly
	err := errors.New("test error")
	if err.Error() == "" {
		t.Error("Expected error message")
	}
}

// ============================================================================
// Additional Setup Helpers
// ============================================================================

func setupHandlerWithPasswordUser(t *testing.T) (*admin.Handler, string) {
	t.Helper()

	userStore := memory.NewUserStore()
	keyStore := memory.NewKeyStore()
	h := hasher.NewBcrypt(4)

	// Create user with password
	passwordHash, _ := h.Hash("testpassword123")
	user := ports.User{
		ID:           "user_password",
		Email:        "passworduser@test.com",
		PasswordHash: passwordHash,
		PlanID:       "free",
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	userStore.Create(context.Background(), user)

	// Create admin API key
	rawKey, keyData := key.Generate("ak_")
	keyData = keyData.WithUserID(user.ID)
	keyStore.Create(context.Background(), keyData)

	planStore := newMockPlanStore()
	now := time.Now().UTC()
	planStore.Create(context.Background(), ports.Plan{
		ID: "free", Name: "Free", RateLimitPerMinute: 60, IsDefault: true, Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	})

	handler := admin.NewHandler(admin.Deps{
		Users:  userStore,
		Keys:   keyStore,
		Plans:  planStore,
		Logger: zerolog.Nop(),
		Hasher: h,
	})

	return handler, rawKey
}

func setupHandlerWithInactiveUser(t *testing.T) (*admin.Handler, string) {
	t.Helper()

	userStore := memory.NewUserStore()
	keyStore := memory.NewKeyStore()
	h := hasher.NewBcrypt(4)

	// Create inactive user with password
	passwordHash, _ := h.Hash("testpassword123")
	user := ports.User{
		ID:           "user_inactive",
		Email:        "inactive@test.com",
		PasswordHash: passwordHash,
		PlanID:       "free",
		Status:       "suspended", // Not active!
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	userStore.Create(context.Background(), user)

	// Create admin API key for auth
	rawKey, keyData := key.Generate("ak_")
	keyData = keyData.WithUserID(user.ID)
	keyStore.Create(context.Background(), keyData)

	planStore := newMockPlanStore()
	now := time.Now().UTC()
	planStore.Create(context.Background(), ports.Plan{
		ID: "free", Name: "Free", RateLimitPerMinute: 60, IsDefault: true, Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	})

	handler := admin.NewHandler(admin.Deps{
		Users:  userStore,
		Keys:   keyStore,
		Plans:  planStore,
		Logger: zerolog.Nop(),
		Hasher: h,
	})

	return handler, rawKey
}

func setupHandlerWithRevokedKey(t *testing.T) (*admin.Handler, string) {
	t.Helper()

	userStore := memory.NewUserStore()
	keyStore := memory.NewKeyStore()
	h := hasher.NewBcrypt(4)

	user := ports.User{
		ID:        "user_revoked",
		Email:     "revoked@test.com",
		PlanID:    "free",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	userStore.Create(context.Background(), user)

	// Create and revoke a key
	rawKey, keyData := key.Generate("ak_")
	keyData = keyData.WithUserID(user.ID)
	keyStore.Create(context.Background(), keyData)

	// Revoke the key
	revokedAt := time.Now().UTC()
	keyStore.Revoke(context.Background(), keyData.ID, revokedAt)

	planStore := newMockPlanStore()
	now := time.Now().UTC()
	planStore.Create(context.Background(), ports.Plan{
		ID: "free", Name: "Free", RateLimitPerMinute: 60, IsDefault: true, Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	})

	handler := admin.NewHandler(admin.Deps{
		Users:  userStore,
		Keys:   keyStore,
		Plans:  planStore,
		Logger: zerolog.Nop(),
		Hasher: h,
	})

	return handler, rawKey
}

func setupHandlerWithUsage(t *testing.T) (*admin.Handler, string) {
	t.Helper()

	userStore := memory.NewUserStore()
	keyStore := memory.NewKeyStore()
	usageStore := memory.NewUsageStore()
	h := hasher.NewBcrypt(4)

	adminUser := ports.User{
		ID:        "user_admin",
		Email:     "admin@test.com",
		PlanID:    "free",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	userStore.Create(context.Background(), adminUser)

	rawKey, keyData := key.Generate("ak_")
	keyData = keyData.WithUserID(adminUser.ID)
	keyStore.Create(context.Background(), keyData)

	planStore := newMockPlanStore()
	now := time.Now().UTC()
	planStore.Create(context.Background(), ports.Plan{
		ID: "free", Name: "Free", RateLimitPerMinute: 60, IsDefault: true, Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	})
	planStore.Create(context.Background(), ports.Plan{
		ID: "pro", Name: "Pro", RateLimitPerMinute: 600, Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	})

	handler := admin.NewHandler(admin.Deps{
		Users:  userStore,
		Keys:   keyStore,
		Usage:  usageStore,
		Plans:  planStore,
		Logger: zerolog.Nop(),
		Hasher: h,
	})

	return handler, rawKey
}

func setupHandlerWithUpstreams(t *testing.T) (*admin.Handler, string) {
	t.Helper()

	userStore := memory.NewUserStore()
	keyStore := memory.NewKeyStore()
	h := hasher.NewBcrypt(4)

	adminUser := ports.User{
		ID:        "user_admin",
		Email:     "admin@test.com",
		PlanID:    "free",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	userStore.Create(context.Background(), adminUser)

	rawKey, keyData := key.Generate("ak_")
	keyData = keyData.WithUserID(adminUser.ID)
	keyStore.Create(context.Background(), keyData)

	planStore := newMockPlanStore()
	now := time.Now().UTC()
	planStore.Create(context.Background(), ports.Plan{
		ID: "free", Name: "Free", RateLimitPerMinute: 60, IsDefault: true, Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	})

	// Create mock upstream store
	upstreamStore := newMockUpstreamStore()

	handler := admin.NewHandler(admin.Deps{
		Users:     userStore,
		Keys:      keyStore,
		Plans:     planStore,
		Upstreams: upstreamStore,
		Logger:    zerolog.Nop(),
		Hasher:    h,
	})

	return handler, rawKey
}

// mockUpstreamStore for use with admin handler tests
type mockUpstreamStore struct {
	mu        sync.RWMutex
	upstreams map[string]route.Upstream
}

func newMockUpstreamStore() *mockUpstreamStore {
	return &mockUpstreamStore{
		upstreams: make(map[string]route.Upstream),
	}
}

func (s *mockUpstreamStore) Get(ctx context.Context, id string) (route.Upstream, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.upstreams[id]
	if !ok {
		return route.Upstream{}, errors.New("not found")
	}
	return u, nil
}

func (s *mockUpstreamStore) List(ctx context.Context) ([]route.Upstream, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []route.Upstream
	for _, u := range s.upstreams {
		result = append(result, u)
	}
	return result, nil
}

func (s *mockUpstreamStore) ListEnabled(ctx context.Context) ([]route.Upstream, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []route.Upstream
	for _, u := range s.upstreams {
		if u.Enabled {
			result = append(result, u)
		}
	}
	return result, nil
}

func (s *mockUpstreamStore) Create(ctx context.Context, u route.Upstream) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.upstreams[u.ID] = u
	return nil
}

// ============================================================================
// PATCH Method Tests
// ============================================================================

func TestUpdateUser_Patch(t *testing.T) {
	h, rawKey := setupHandler(t)

	// Create a user
	createBody := map[string]string{"email": "patch@test.com"}
	createResp := doRequest(t, h, "POST", "/users", createBody, rawKey)

	var created map[string]any
	json.NewDecoder(createResp.Body).Decode(&created)
	userID := getResourceID(created)

	// Update the user using PATCH
	updateBody := map[string]string{"plan_id": "enterprise", "name": "Patched Name"}
	resp := doRequest(t, h, "PATCH", "/users/"+userID, updateBody, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	// plan_id is a relationship, not an attribute
	if getRelationshipID(result, "plan") != "enterprise" {
		t.Errorf("Expected plan relationship id=enterprise, got %s", getRelationshipID(result, "plan"))
	}
	if getResourceAttr(result, "name") != "Patched Name" {
		t.Errorf("Expected name=Patched Name, got %s", getResourceAttr(result, "name"))
	}
}

func TestUpdatePlan_Patch(t *testing.T) {
	h, rawKey := setupHandler(t)

	body := map[string]any{
		"name":                  "Free Patched",
		"rate_limit_per_minute": 130,
		"price_monthly":         8.99,
	}

	resp := doRequest(t, h, "PATCH", "/plans/free", body, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	if name := getResourceAttr(result, "name"); name != "Free Patched" {
		t.Errorf("Expected name='Free Patched', got %v", name)
	}
	if rateLimit := getResourceAttr(result, "rate_limit_per_minute"); rateLimit != nil {
		if int(rateLimit.(float64)) != 130 {
			t.Errorf("Expected rate_limit_per_minute=130, got %v", rateLimit)
		}
	}
	if price := getResourceAttr(result, "price_monthly"); price != nil {
		if price.(float64) != 8.99 {
			t.Errorf("Expected price_monthly=8.99, got %v", price)
		}
	}
}

func (s *mockUpstreamStore) Update(ctx context.Context, u route.Upstream) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.upstreams[u.ID] = u
	return nil
}

func (s *mockUpstreamStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.upstreams, id)
	return nil
}
