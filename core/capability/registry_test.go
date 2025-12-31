package capability_test

import (
	"context"
	"testing"

	"github.com/artpar/apigate/core/capability"
)

// =============================================================================
// Registry Tests
// =============================================================================

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := capability.NewRegistry()

	// Register a payment provider
	info := capability.ProviderInfo{
		Name:       "stripe_prod",
		Module:     "payment_stripe",
		Capability: capability.Payment,
		Enabled:    true,
		IsDefault:  true,
	}

	err := reg.Register(info)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Get the provider
	got, ok := reg.Get("stripe_prod")
	if !ok {
		t.Fatal("Get() should find registered provider")
	}
	if got.Name != info.Name {
		t.Errorf("Get() name = %v, want %v", got.Name, info.Name)
	}
}

func TestRegistry_RegisterDuplicate(t *testing.T) {
	reg := capability.NewRegistry()

	info := capability.ProviderInfo{
		Name:       "stripe_prod",
		Module:     "payment_stripe",
		Capability: capability.Payment,
		Enabled:    true,
	}

	// First registration should succeed
	if err := reg.Register(info); err != nil {
		t.Fatalf("First Register() error = %v", err)
	}

	// Second registration with same name should fail
	if err := reg.Register(info); err == nil {
		t.Error("Second Register() should fail with duplicate name")
	}
}

func TestRegistry_GetByCapability(t *testing.T) {
	reg := capability.NewRegistry()

	// Register multiple providers for same capability
	providers := []capability.ProviderInfo{
		{Name: "stripe_prod", Module: "payment_stripe", Capability: capability.Payment, Enabled: true},
		{Name: "stripe_test", Module: "payment_stripe", Capability: capability.Payment, Enabled: false},
		{Name: "paddle_prod", Module: "payment_paddle", Capability: capability.Payment, Enabled: false},
		{Name: "smtp_main", Module: "email_smtp", Capability: capability.Email, Enabled: true},
	}

	for _, p := range providers {
		if err := reg.Register(p); err != nil {
			t.Fatalf("Register(%s) error = %v", p.Name, err)
		}
	}

	// Get all payment providers
	paymentProviders := reg.GetByCapability(capability.Payment)
	if len(paymentProviders) != 3 {
		t.Errorf("GetByCapability(Payment) got %d providers, want 3", len(paymentProviders))
	}

	// Get all email providers
	emailProviders := reg.GetByCapability(capability.Email)
	if len(emailProviders) != 1 {
		t.Errorf("GetByCapability(Email) got %d providers, want 1", len(emailProviders))
	}

	// Get non-existent capability
	cacheProviders := reg.GetByCapability(capability.Cache)
	if len(cacheProviders) != 0 {
		t.Errorf("GetByCapability(Cache) got %d providers, want 0", len(cacheProviders))
	}
}

func TestRegistry_GetEnabled(t *testing.T) {
	reg := capability.NewRegistry()

	// Register providers with different enabled states
	providers := []capability.ProviderInfo{
		{Name: "stripe_prod", Module: "payment_stripe", Capability: capability.Payment, Enabled: true, IsDefault: true},
		{Name: "stripe_test", Module: "payment_stripe", Capability: capability.Payment, Enabled: false},
		{Name: "paddle_prod", Module: "payment_paddle", Capability: capability.Payment, Enabled: true},
	}

	for _, p := range providers {
		if err := reg.Register(p); err != nil {
			t.Fatalf("Register(%s) error = %v", p.Name, err)
		}
	}

	// Get enabled payment providers
	enabled := reg.GetEnabled(capability.Payment)
	if len(enabled) != 2 {
		t.Errorf("GetEnabled(Payment) got %d providers, want 2", len(enabled))
	}
}

func TestRegistry_GetDefault(t *testing.T) {
	reg := capability.NewRegistry()

	providers := []capability.ProviderInfo{
		{Name: "stripe_prod", Module: "payment_stripe", Capability: capability.Payment, Enabled: true, IsDefault: true},
		{Name: "paddle_prod", Module: "payment_paddle", Capability: capability.Payment, Enabled: true, IsDefault: false},
	}

	for _, p := range providers {
		if err := reg.Register(p); err != nil {
			t.Fatalf("Register(%s) error = %v", p.Name, err)
		}
	}

	// Get default payment provider
	def, ok := reg.GetDefault(capability.Payment)
	if !ok {
		t.Fatal("GetDefault(Payment) should find default provider")
	}
	if def.Name != "stripe_prod" {
		t.Errorf("GetDefault(Payment) = %s, want stripe_prod", def.Name)
	}

	// No default for email
	_, ok = reg.GetDefault(capability.Email)
	if ok {
		t.Error("GetDefault(Email) should not find default provider")
	}
}

func TestRegistry_SetEnabled(t *testing.T) {
	reg := capability.NewRegistry()

	info := capability.ProviderInfo{
		Name:       "stripe_prod",
		Module:     "payment_stripe",
		Capability: capability.Payment,
		Enabled:    false,
	}

	if err := reg.Register(info); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Enable the provider
	if err := reg.SetEnabled("stripe_prod", true); err != nil {
		t.Fatalf("SetEnabled(true) error = %v", err)
	}

	got, _ := reg.Get("stripe_prod")
	if !got.Enabled {
		t.Error("Provider should be enabled")
	}

	// Disable the provider
	if err := reg.SetEnabled("stripe_prod", false); err != nil {
		t.Fatalf("SetEnabled(false) error = %v", err)
	}

	got, _ = reg.Get("stripe_prod")
	if got.Enabled {
		t.Error("Provider should be disabled")
	}
}

func TestRegistry_SetDefault(t *testing.T) {
	reg := capability.NewRegistry()

	providers := []capability.ProviderInfo{
		{Name: "stripe_prod", Module: "payment_stripe", Capability: capability.Payment, Enabled: true, IsDefault: true},
		{Name: "paddle_prod", Module: "payment_paddle", Capability: capability.Payment, Enabled: true, IsDefault: false},
	}

	for _, p := range providers {
		if err := reg.Register(p); err != nil {
			t.Fatalf("Register(%s) error = %v", p.Name, err)
		}
	}

	// Change default to paddle
	if err := reg.SetDefault("paddle_prod"); err != nil {
		t.Fatalf("SetDefault() error = %v", err)
	}

	// Verify paddle is now default
	def, _ := reg.GetDefault(capability.Payment)
	if def.Name != "paddle_prod" {
		t.Errorf("GetDefault() = %s, want paddle_prod", def.Name)
	}

	// Verify stripe is no longer default
	stripe, _ := reg.Get("stripe_prod")
	if stripe.IsDefault {
		t.Error("stripe_prod should no longer be default")
	}
}

func TestRegistry_Unregister(t *testing.T) {
	reg := capability.NewRegistry()

	info := capability.ProviderInfo{
		Name:       "stripe_prod",
		Module:     "payment_stripe",
		Capability: capability.Payment,
		Enabled:    true,
	}

	if err := reg.Register(info); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Unregister
	if err := reg.Unregister("stripe_prod"); err != nil {
		t.Fatalf("Unregister() error = %v", err)
	}

	// Should not be found
	_, ok := reg.Get("stripe_prod")
	if ok {
		t.Error("Provider should not be found after unregister")
	}
}

func TestRegistry_CustomCapability(t *testing.T) {
	reg := capability.NewRegistry()

	// Register a custom capability provider (e.g., reconciliation)
	info := capability.ProviderInfo{
		Name:             "recon_main",
		Module:           "reconciliation_default",
		Capability:       capability.Custom,
		CustomCapability: "reconciliation",
		Enabled:          true,
		IsDefault:        true,
	}

	if err := reg.Register(info); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Get by custom capability name
	providers := reg.GetByCustomCapability("reconciliation")
	if len(providers) != 1 {
		t.Errorf("GetByCustomCapability() got %d providers, want 1", len(providers))
	}
}

func TestRegistry_ListCapabilities(t *testing.T) {
	reg := capability.NewRegistry()

	providers := []capability.ProviderInfo{
		{Name: "stripe_prod", Module: "payment_stripe", Capability: capability.Payment, Enabled: true},
		{Name: "smtp_main", Module: "email_smtp", Capability: capability.Email, Enabled: true},
		{Name: "redis_main", Module: "cache_redis", Capability: capability.Cache, Enabled: true},
		{Name: "recon_main", Module: "reconciliation_default", Capability: capability.Custom, CustomCapability: "reconciliation", Enabled: true},
	}

	for _, p := range providers {
		if err := reg.Register(p); err != nil {
			t.Fatalf("Register(%s) error = %v", p.Name, err)
		}
	}

	// List all capabilities
	caps := reg.ListCapabilities()
	if len(caps) != 4 {
		t.Errorf("ListCapabilities() got %d, want 4", len(caps))
	}

	// Should include both built-in and custom
	found := make(map[string]bool)
	for _, c := range caps {
		found[c] = true
	}

	for _, want := range []string{"payment", "email", "cache", "reconciliation"} {
		if !found[want] {
			t.Errorf("ListCapabilities() missing %s", want)
		}
	}
}

// =============================================================================
// Resolver Tests - Getting actual provider implementations
// =============================================================================

func TestResolver_GetProvider(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)

	// Register a payment provider
	info := capability.ProviderInfo{
		Name:       "stripe_prod",
		Module:     "payment_stripe",
		Capability: capability.Payment,
		Enabled:    true,
		IsDefault:  true,
	}
	reg.Register(info)

	// Create and register a mock payment provider
	mockPayment := &MockPaymentProvider{name: "stripe_prod"}
	resolver.RegisterImplementation("stripe_prod", mockPayment)

	ctx := context.Background()

	// Resolve the default payment provider
	provider, err := resolver.Payment(ctx)
	if err != nil {
		t.Fatalf("Payment() error = %v", err)
	}
	if provider == nil {
		t.Fatal("Payment() returned nil")
	}
	if provider.Name() != "stripe_prod" {
		t.Errorf("Payment().Name() = %s, want stripe_prod", provider.Name())
	}
}

func TestResolver_GetProviderByName(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)

	// Register multiple providers
	providers := []capability.ProviderInfo{
		{Name: "stripe_prod", Module: "payment_stripe", Capability: capability.Payment, Enabled: true, IsDefault: true},
		{Name: "stripe_test", Module: "payment_stripe", Capability: capability.Payment, Enabled: true, IsDefault: false},
	}
	for _, p := range providers {
		reg.Register(p)
	}

	// Register implementations
	resolver.RegisterImplementation("stripe_prod", &MockPaymentProvider{name: "stripe_prod"})
	resolver.RegisterImplementation("stripe_test", &MockPaymentProvider{name: "stripe_test"})

	ctx := context.Background()

	// Get specific provider by name
	provider, err := resolver.PaymentByName(ctx, "stripe_test")
	if err != nil {
		t.Fatalf("PaymentByName() error = %v", err)
	}
	if provider.Name() != "stripe_test" {
		t.Errorf("PaymentByName().Name() = %s, want stripe_test", provider.Name())
	}
}

func TestResolver_NoEnabledProvider(t *testing.T) {
	reg := capability.NewRegistry()
	resolver := capability.NewResolver(reg)

	// Register disabled provider
	info := capability.ProviderInfo{
		Name:       "stripe_prod",
		Module:     "payment_stripe",
		Capability: capability.Payment,
		Enabled:    false,
	}
	reg.Register(info)
	resolver.RegisterImplementation("stripe_prod", &MockPaymentProvider{name: "stripe_prod"})

	ctx := context.Background()

	// Should return error when no enabled provider
	_, err := resolver.Payment(ctx)
	if err == nil {
		t.Error("Payment() should return error when no enabled provider")
	}
}

// =============================================================================
// Mock Implementations for Testing
// =============================================================================

type MockPaymentProvider struct {
	name string
}

func (m *MockPaymentProvider) Name() string { return m.name }
func (m *MockPaymentProvider) CreateCustomer(ctx context.Context, email, name, userID string) (string, error) {
	return "cus_mock_123", nil
}
func (m *MockPaymentProvider) CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string) (string, error) {
	return "https://checkout.mock.com/session", nil
}
func (m *MockPaymentProvider) CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error) {
	return "https://portal.mock.com/session", nil
}
func (m *MockPaymentProvider) CancelSubscription(ctx context.Context, subscriptionID string, immediately bool) error {
	return nil
}
func (m *MockPaymentProvider) GetSubscription(ctx context.Context, subscriptionID string) (capability.Subscription, error) {
	return capability.Subscription{}, nil
}
func (m *MockPaymentProvider) ReportUsage(ctx context.Context, subscriptionItemID string, quantity int64, timestamp int64) error {
	return nil
}
func (m *MockPaymentProvider) ParseWebhook(payload []byte, signature string) (string, map[string]any, error) {
	return "test.event", nil, nil
}
func (m *MockPaymentProvider) CreatePrice(ctx context.Context, name string, amountCents int64, interval string) (string, error) {
	return "price_mock_123", nil
}
