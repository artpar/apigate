package capability_test

import (
	"context"
	"testing"

	"github.com/artpar/apigate/core/capability"
	captest "github.com/artpar/apigate/core/capability/testing"
)

func TestContainer_RegisterAndResolve(t *testing.T) {
	ctx := context.Background()
	container := capability.NewContainer()

	// Register mock providers
	mockPayment := captest.NewMockPayment("stripe_test")
	mockCache := captest.NewMockCache("redis_test")

	if err := container.RegisterPayment("stripe_test", mockPayment, true); err != nil {
		t.Fatalf("RegisterPayment() error = %v", err)
	}

	if err := container.RegisterCache("redis_test", mockCache, true); err != nil {
		t.Fatalf("RegisterCache() error = %v", err)
	}

	// Resolve providers
	payment, err := container.Payment(ctx)
	if err != nil {
		t.Fatalf("Payment() error = %v", err)
	}
	if payment.Name() != "stripe_test" {
		t.Errorf("Payment().Name() = %v, want stripe_test", payment.Name())
	}

	cache, err := container.Cache(ctx)
	if err != nil {
		t.Fatalf("Cache() error = %v", err)
	}
	if cache.Name() != "redis_test" {
		t.Errorf("Cache().Name() = %v, want redis_test", cache.Name())
	}
}

func TestContainer_MultipleProviders(t *testing.T) {
	ctx := context.Background()
	container := capability.NewContainer()

	// Register multiple payment providers
	stripeProd := captest.NewMockPayment("stripe_prod")
	stripeTest := captest.NewMockPayment("stripe_test")
	paddle := captest.NewMockPayment("paddle")

	container.RegisterPayment("stripe_prod", stripeProd, true)  // default
	container.RegisterPayment("stripe_test", stripeTest, false) // not default
	container.RegisterPayment("paddle_prod", paddle, false)     // not default

	// Default should return stripe_prod
	payment, _ := container.Payment(ctx)
	if payment.Name() != "stripe_prod" {
		t.Errorf("Payment() should return default, got %v", payment.Name())
	}

	// Get specific provider by name
	test, _ := container.PaymentByName(ctx, "stripe_test")
	if test.Name() != "stripe_test" {
		t.Errorf("PaymentByName() = %v, want stripe_test", test.Name())
	}

	paddleProvider, _ := container.PaymentByName(ctx, "paddle_prod")
	if paddleProvider.Name() != "paddle" {
		t.Errorf("PaymentByName() = %v, want paddle", paddleProvider.Name())
	}
}

func TestContainer_CustomCapability(t *testing.T) {
	ctx := context.Background()
	container := capability.NewContainer()

	// Register a custom capability (e.g., reconciliation)
	type ReconciliationProvider interface {
		Name() string
		Reconcile() error
	}

	mockRecon := &mockReconciliation{name: "recon_main"}
	err := container.RegisterCustom("reconciliation", "recon_main", mockRecon, true)
	if err != nil {
		t.Fatalf("RegisterCustom() error = %v", err)
	}

	// Retrieve custom provider
	impl, err := container.Custom(ctx, "reconciliation", "recon_main")
	if err != nil {
		t.Fatalf("Custom() error = %v", err)
	}

	recon, ok := impl.(*mockReconciliation)
	if !ok {
		t.Fatal("Custom() returned wrong type")
	}
	if recon.Name() != "recon_main" {
		t.Errorf("Custom provider name = %v, want recon_main", recon.Name())
	}
}

func TestContainer_ListCapabilities(t *testing.T) {
	container := capability.NewContainer()

	container.RegisterPayment("stripe", captest.NewMockPayment("stripe"), true)
	container.RegisterCache("redis", captest.NewMockCache("redis"), true)
	container.RegisterCustom("reconciliation", "recon", &mockReconciliation{name: "recon"}, true)

	caps := container.ListCapabilities()
	if len(caps) != 3 {
		t.Errorf("ListCapabilities() = %d, want 3", len(caps))
	}

	// Verify specific capabilities exist
	found := make(map[string]bool)
	for _, c := range caps {
		found[c] = true
	}

	for _, want := range []string{"payment", "cache", "reconciliation"} {
		if !found[want] {
			t.Errorf("ListCapabilities() missing %s", want)
		}
	}
}

func TestContainer_Close(t *testing.T) {
	container := capability.NewContainer()

	cache := captest.NewMockCache("test")
	container.RegisterCache("test", cache, true)

	// Close should not error
	if err := container.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

// Mock reconciliation provider for testing custom capabilities
type mockReconciliation struct {
	name string
}

func (m *mockReconciliation) Name() string {
	return m.name
}

func (m *mockReconciliation) Reconcile() error {
	return nil
}
