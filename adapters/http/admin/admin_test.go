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

// ============================================================================
// Plans API Tests - Structure and Semantics
// ============================================================================

func TestListPlans_ResponseStructure(t *testing.T) {
	h, rawKey := setupHandler(t)

	resp := doRequest(t, h, "GET", "/plans", nil, rawKey)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// Verify response structure
	if result["plans"] == nil {
		t.Fatal("Response missing 'plans' field")
	}
	if result["total"] == nil {
		t.Fatal("Response missing 'total' field")
	}

	// Verify plan structure
	plans := result["plans"].([]interface{})
	if len(plans) == 0 {
		t.Fatal("Expected at least one plan")
	}

	plan := plans[0].(map[string]interface{})
	requiredFields := []string{"id", "name", "rate_limit_per_minute", "requests_per_month",
		"price_monthly", "overage_price", "is_default", "enabled", "created_at", "updated_at"}
	for _, field := range requiredFields {
		if _, ok := plan[field]; !ok {
			t.Errorf("Plan missing required field: %s", field)
		}
	}
}

func TestGetPlan_Success(t *testing.T) {
	h, rawKey := setupHandler(t)

	resp := doRequest(t, h, "GET", "/plans/free", nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var plan map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&plan)

	if plan["id"] != "free" {
		t.Errorf("Expected id='free', got %v", plan["id"])
	}
	if plan["name"] != "Free" {
		t.Errorf("Expected name='Free', got %v", plan["name"])
	}
}

func TestGetPlan_NotFound(t *testing.T) {
	h, rawKey := setupHandler(t)

	resp := doRequest(t, h, "GET", "/plans/nonexistent", nil, rawKey)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected 404, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	errData := result["error"].(map[string]interface{})
	if errData["code"] != "not_found" {
		t.Errorf("Expected error code 'not_found', got %v", errData["code"])
	}
}

func TestCreatePlan_Success(t *testing.T) {
	h, rawKey := setupHandler(t)

	body := map[string]interface{}{
		"id":                   "enterprise",
		"name":                 "Enterprise",
		"description":          "For large organizations",
		"rate_limit_per_minute": 1000,
		"requests_per_month":   1000000,
		"price_monthly":        99.99,
		"overage_price":        0.001,
		"stripe_price_id":      "price_enterprise",
		"enabled":              true,
	}

	resp := doRequest(t, h, "POST", "/plans", body, rawKey)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var plan map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&plan)

	// Verify all fields
	if plan["id"] != "enterprise" {
		t.Errorf("Expected id='enterprise', got %v", plan["id"])
	}
	if plan["name"] != "Enterprise" {
		t.Errorf("Expected name='Enterprise', got %v", plan["name"])
	}
	if plan["description"] != "For large organizations" {
		t.Errorf("Expected description='For large organizations', got %v", plan["description"])
	}
	if int(plan["rate_limit_per_minute"].(float64)) != 1000 {
		t.Errorf("Expected rate_limit_per_minute=1000, got %v", plan["rate_limit_per_minute"])
	}
	// Price should be returned as dollars (converted back from cents)
	if plan["price_monthly"].(float64) != 99.99 {
		t.Errorf("Expected price_monthly=99.99, got %v", plan["price_monthly"])
	}
}

func TestCreatePlan_MissingID(t *testing.T) {
	h, rawKey := setupHandler(t)

	body := map[string]interface{}{
		"name":    "Test Plan",
		"enabled": true,
	}

	resp := doRequest(t, h, "POST", "/plans", body, rawKey)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	errData := result["error"].(map[string]interface{})
	if errData["code"] != "missing_id" {
		t.Errorf("Expected error code 'missing_id', got %v", errData["code"])
	}
}

func TestCreatePlan_MissingName(t *testing.T) {
	h, rawKey := setupHandler(t)

	body := map[string]interface{}{
		"id":      "test_plan",
		"enabled": true,
	}

	resp := doRequest(t, h, "POST", "/plans", body, rawKey)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	errData := result["error"].(map[string]interface{})
	if errData["code"] != "missing_name" {
		t.Errorf("Expected error code 'missing_name', got %v", errData["code"])
	}
}

func TestCreatePlan_DuplicateID(t *testing.T) {
	h, rawKey := setupHandler(t)

	// Try to create a plan with existing ID
	body := map[string]interface{}{
		"id":      "free", // Already exists
		"name":    "Another Free",
		"enabled": true,
	}

	resp := doRequest(t, h, "POST", "/plans", body, rawKey)

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("Expected 409, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	errData := result["error"].(map[string]interface{})
	if errData["code"] != "plan_exists" {
		t.Errorf("Expected error code 'plan_exists', got %v", errData["code"])
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

	body := map[string]interface{}{
		"name":                 "Free Updated",
		"rate_limit_per_minute": 120,
		"price_monthly":        9.99,
	}

	resp := doRequest(t, h, "PUT", "/plans/free", body, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var plan map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&plan)

	if plan["name"] != "Free Updated" {
		t.Errorf("Expected name='Free Updated', got %v", plan["name"])
	}
	if int(plan["rate_limit_per_minute"].(float64)) != 120 {
		t.Errorf("Expected rate_limit_per_minute=120, got %v", plan["rate_limit_per_minute"])
	}
	if plan["price_monthly"].(float64) != 9.99 {
		t.Errorf("Expected price_monthly=9.99, got %v", plan["price_monthly"])
	}
}

func TestUpdatePlan_PartialUpdate(t *testing.T) {
	h, rawKey := setupHandler(t)

	// Only update name, other fields should remain unchanged
	body := map[string]interface{}{
		"name": "Pro Premium",
	}

	resp := doRequest(t, h, "PUT", "/plans/pro", body, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var plan map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&plan)

	if plan["name"] != "Pro Premium" {
		t.Errorf("Expected name='Pro Premium', got %v", plan["name"])
	}
	// Rate limit should remain unchanged (600 from test setup)
	if int(plan["rate_limit_per_minute"].(float64)) != 600 {
		t.Errorf("Expected rate_limit_per_minute=600, got %v", plan["rate_limit_per_minute"])
	}
}

func TestUpdatePlan_NotFound(t *testing.T) {
	h, rawKey := setupHandler(t)

	body := map[string]interface{}{
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
	body := map[string]interface{}{
		"enabled":    false,
		"is_default": true,
	}

	resp := doRequest(t, h, "PUT", "/plans/pro", body, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var plan map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&plan)

	if plan["enabled"] != false {
		t.Errorf("Expected enabled=false, got %v", plan["enabled"])
	}
	if plan["is_default"] != true {
		t.Errorf("Expected is_default=true, got %v", plan["is_default"])
	}
}

func TestDeletePlan_Success(t *testing.T) {
	h, rawKey := setupHandler(t)

	// First create a plan to delete
	createBody := map[string]interface{}{
		"id":      "to_delete",
		"name":    "To Delete",
		"enabled": true,
	}
	doRequest(t, h, "POST", "/plans", createBody, rawKey)

	// Now delete it
	resp := doRequest(t, h, "DELETE", "/plans/to_delete", nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["status"] != "deleted" {
		t.Errorf("Expected status='deleted', got %v", result["status"])
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

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	errData := result["error"].(map[string]interface{})
	if errData["code"] != "plan_in_use" {
		t.Errorf("Expected error code 'plan_in_use', got %v", errData["code"])
	}
}

func TestPlan_PriceConversion(t *testing.T) {
	h, rawKey := setupHandler(t)

	// Create plan with specific price values
	body := map[string]interface{}{
		"id":            "price_test",
		"name":          "Price Test",
		"price_monthly": 29.99,    // $29.99
		"overage_price": 0.005,    // $0.005 (half a cent)
		"enabled":       true,
	}

	resp := doRequest(t, h, "POST", "/plans", body, rawKey)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var plan map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&plan)

	// Prices should be stored as cents internally but returned as dollars
	if plan["price_monthly"].(float64) != 29.99 {
		t.Errorf("Expected price_monthly=29.99, got %v", plan["price_monthly"])
	}
	if plan["overage_price"].(float64) != 0.005 { // Now stored as hundredths of cents: 0.005 * 10000 = 50
		t.Errorf("Expected overage_price=0.005, got %v", plan["overage_price"])
	}
}

func TestPlan_PaymentProviderFields(t *testing.T) {
	h, rawKey := setupHandler(t)

	body := map[string]interface{}{
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

	var plan map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&plan)

	if plan["stripe_price_id"] != "price_abc123" {
		t.Errorf("Expected stripe_price_id='price_abc123', got %v", plan["stripe_price_id"])
	}
	if plan["paddle_price_id"] != "pri_xyz789" {
		t.Errorf("Expected paddle_price_id='pri_xyz789', got %v", plan["paddle_price_id"])
	}
	if plan["lemon_variant_id"] != "var_456" {
		t.Errorf("Expected lemon_variant_id='var_456', got %v", plan["lemon_variant_id"])
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

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["session_id"] == nil {
		t.Error("Expected session_id in response")
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

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestLogin_MissingPassword(t *testing.T) {
	h, _ := setupHandler(t)

	body := map[string]string{
		"email": "admin@test.com",
	}
	resp := doRequest(t, h, "POST", "/login", body, "")

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d", resp.StatusCode)
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

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	user := result["user"].(map[string]interface{})
	if user["email"] != "admin@apigate" {
		t.Errorf("Expected default admin email, got %s", user["email"])
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

	var loginResult map[string]interface{}
	json.NewDecoder(loginResp.Body).Decode(&loginResult)
	sessionID := loginResult["session_id"].(string)

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

	var loginResult map[string]interface{}
	json.NewDecoder(loginResp.Body).Decode(&loginResult)
	sessionID := loginResult["session_id"].(string)

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

	var loginResult map[string]interface{}
	json.NewDecoder(loginResp.Body).Decode(&loginResult)
	sessionID := loginResult["session_id"].(string)

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

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d", resp.StatusCode)
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

	var created map[string]interface{}
	json.NewDecoder(createResp.Body).Decode(&created)
	userID := created["id"].(string)

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
	var created map[string]interface{}
	json.NewDecoder(createResp.Body).Decode(&created)
	userID := created["id"].(string)

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

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d", resp.StatusCode)
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
	var user map[string]interface{}
	json.NewDecoder(userResp.Body).Decode(&user)
	userID := user["id"].(string)

	// Create key with expiry
	expiryTime := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	body := map[string]interface{}{
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
	var user map[string]interface{}
	json.NewDecoder(userResp.Body).Decode(&user)
	userID := user["id"].(string)

	keyBody := map[string]string{"user_id": userID, "name": "Filtered Key"}
	doRequest(t, h, "POST", "/keys", keyBody, rawKey)

	// List keys filtered by user
	resp := doRequest(t, h, "GET", "/keys?user_id="+userID, nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	keys := result["keys"].([]interface{})
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

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["period"] != "day" {
		t.Errorf("Expected period=day, got %s", result["period"])
	}
}

func TestGetUsage_WeekPeriod(t *testing.T) {
	h, rawKey := setupHandlerWithUsage(t)

	resp := doRequest(t, h, "GET", "/usage?period=week", nil, rawKey)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["period"] != "week" {
		t.Errorf("Expected period=week, got %s", result["period"])
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

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestGetUsage_InvalidEndDate(t *testing.T) {
	h, rawKey := setupHandlerWithUsage(t)

	startDate := time.Now().AddDate(0, 0, -7).Format(time.RFC3339)

	resp := doRequest(t, h, "GET", "/usage?start_date="+startDate+"&end_date=invalid", nil, rawKey)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d", resp.StatusCode)
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

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	checks := result["checks"].([]interface{})
	foundUpstream := false
	for _, check := range checks {
		c := check.(map[string]interface{})
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

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// Verify all required fields are present
	requiredFields := []string{"status", "timestamp", "version", "checks", "system", "statistics"}
	for _, field := range requiredFields {
		if result[field] == nil {
			t.Errorf("Missing required field: %s", field)
		}
	}

	// Verify system info structure
	system := result["system"].(map[string]interface{})
	systemFields := []string{"go_version", "num_cpu", "num_goroutine", "mem_alloc", "mem_sys", "uptime"}
	for _, field := range systemFields {
		if system[field] == nil {
			t.Errorf("Missing system field: %s", field)
		}
	}

	// Verify statistics structure
	stats := result["statistics"].(map[string]interface{})
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
	for i := 0; i < 5; i++ {
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

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	users := result["users"].([]interface{})
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
