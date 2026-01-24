package app

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/artpar/apigate/domain/billing"
	"github.com/artpar/apigate/ports"
	"github.com/rs/zerolog"
)

// Mock implementations for testing

type mockUserStore struct {
	users  []ports.User
	getErr error
}

func (m *mockUserStore) Get(ctx context.Context, id string) (ports.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return ports.User{}, errors.New("not found")
}

func (m *mockUserStore) GetByEmail(ctx context.Context, email string) (ports.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return ports.User{}, errors.New("not found")
}

func (m *mockUserStore) GetByStripeID(ctx context.Context, stripeID string) (ports.User, error) {
	for _, u := range m.users {
		if u.StripeID == stripeID {
			return u, nil
		}
	}
	return ports.User{}, errors.New("not found")
}

func (m *mockUserStore) Create(ctx context.Context, u ports.User) error { return nil }
func (m *mockUserStore) Update(ctx context.Context, u ports.User) error { return nil }
func (m *mockUserStore) Delete(ctx context.Context, id string) error    { return nil }
func (m *mockUserStore) List(ctx context.Context, limit, offset int) ([]ports.User, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.users, nil
}
func (m *mockUserStore) Count(ctx context.Context) (int, error) { return len(m.users), nil }

type mockSubscriptionStore struct {
	subscriptions []billing.Subscription
	createErr     error
	updateErr     error
}

func (m *mockSubscriptionStore) Get(ctx context.Context, id string) (billing.Subscription, error) {
	for _, s := range m.subscriptions {
		if s.ID == id {
			return s, nil
		}
	}
	return billing.Subscription{}, errors.New("not found")
}

func (m *mockSubscriptionStore) GetByUser(ctx context.Context, userID string) (billing.Subscription, error) {
	for _, s := range m.subscriptions {
		if s.UserID == userID && s.IsActive() {
			return s, nil
		}
	}
	return billing.Subscription{}, errors.New("not found")
}

func (m *mockSubscriptionStore) GetByProviderID(ctx context.Context, providerID string) (billing.Subscription, error) {
	for _, s := range m.subscriptions {
		if s.ProviderID == providerID {
			return s, nil
		}
	}
	return billing.Subscription{}, errors.New("not found")
}

func (m *mockSubscriptionStore) Create(ctx context.Context, sub billing.Subscription) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.subscriptions = append(m.subscriptions, sub)
	return nil
}

func (m *mockSubscriptionStore) Update(ctx context.Context, sub billing.Subscription) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	for i, s := range m.subscriptions {
		if s.ID == sub.ID {
			m.subscriptions[i] = sub
			return nil
		}
	}
	return errors.New("not found")
}

type mockPlanStore struct {
	plans []ports.Plan
}

func (m *mockPlanStore) List(ctx context.Context) ([]ports.Plan, error) { return m.plans, nil }
func (m *mockPlanStore) Get(ctx context.Context, id string) (ports.Plan, error) {
	for _, p := range m.plans {
		if p.ID == id {
			return p, nil
		}
	}
	return ports.Plan{}, errors.New("not found")
}
func (m *mockPlanStore) Create(ctx context.Context, p ports.Plan) error                  { return nil }
func (m *mockPlanStore) Update(ctx context.Context, p ports.Plan) error                  { return nil }
func (m *mockPlanStore) Delete(ctx context.Context, id string) error                     { return nil }
func (m *mockPlanStore) ClearOtherDefaults(ctx context.Context, exceptID string) error { return nil }

type mockIDGenerator struct {
	counter int
}

func (m *mockIDGenerator) New() string {
	m.counter++
	return fmt.Sprintf("generated-id-%d", m.counter)
}

func TestPaymentWebhookService_HandleCheckoutCompleted(t *testing.T) {
	logger := zerolog.Nop()

	users := &mockUserStore{
		users: []ports.User{
			{ID: "user-1", Email: "test@example.com", StripeID: "cus_123"},
		},
	}

	subscriptions := &mockSubscriptionStore{}

	plans := &mockPlanStore{
		plans: []ports.Plan{
			{ID: "plan-1", Name: "Pro", Enabled: true},
		},
	}

	idGen := &mockIDGenerator{}

	service := NewPaymentWebhookService(users, subscriptions, plans, idGen, logger)

	t.Run("creates subscription and updates user plan", func(t *testing.T) {
		ctx := context.Background()

		err := service.HandleCheckoutCompleted(ctx, "cus_123", "sub_abc123", "plan-1")
		if err != nil {
			t.Fatalf("HandleCheckoutCompleted failed: %v", err)
		}

		// Verify subscription was created
		if len(subscriptions.subscriptions) != 1 {
			t.Fatalf("expected 1 subscription, got %d", len(subscriptions.subscriptions))
		}

		sub := subscriptions.subscriptions[0]
		if sub.ProviderID != "sub_abc123" {
			t.Errorf("ProviderID = %s, want sub_abc123", sub.ProviderID)
		}
		if sub.UserID != "user-1" {
			t.Errorf("UserID = %s, want user-1", sub.UserID)
		}
		if sub.PlanID != "plan-1" {
			t.Errorf("PlanID = %s, want plan-1", sub.PlanID)
		}
		if sub.Status != billing.SubscriptionStatusActive {
			t.Errorf("Status = %s, want active", sub.Status)
		}
	})

	t.Run("returns error for unknown customer", func(t *testing.T) {
		ctx := context.Background()

		err := service.HandleCheckoutCompleted(ctx, "unknown_customer", "sub_xyz", "plan-1")
		if err == nil {
			t.Error("expected error for unknown customer")
		}
	})

	t.Run("returns error for unknown plan", func(t *testing.T) {
		ctx := context.Background()

		err := service.HandleCheckoutCompleted(ctx, "cus_123", "sub_xyz", "unknown-plan")
		if err == nil {
			t.Error("expected error for unknown plan")
		}
	})
}

func TestPaymentWebhookService_HandleSubscriptionUpdated(t *testing.T) {
	logger := zerolog.Nop()

	users := &mockUserStore{
		users: []ports.User{
			{ID: "user-1", Email: "test@example.com", StripeID: "cus_123"},
		},
	}

	subscriptions := &mockSubscriptionStore{
		subscriptions: []billing.Subscription{
			{
				ID:         "sub-1",
				UserID:     "user-1",
				PlanID:     "plan-1",
				ProviderID: "sub_abc123",
				Status:     billing.SubscriptionStatusActive,
			},
		},
	}

	plans := &mockPlanStore{
		plans: []ports.Plan{
			{ID: "plan-1", Name: "Pro", Enabled: true, IsDefault: true},
		},
	}

	idGen := &mockIDGenerator{}

	service := NewPaymentWebhookService(users, subscriptions, plans, idGen, logger)

	t.Run("updates subscription status", func(t *testing.T) {
		ctx := context.Background()

		err := service.HandleSubscriptionUpdated(ctx, "sub_abc123", billing.SubscriptionStatusPastDue)
		if err != nil {
			t.Fatalf("HandleSubscriptionUpdated failed: %v", err)
		}

		// Verify status was updated
		sub := subscriptions.subscriptions[0]
		if sub.Status != billing.SubscriptionStatusPastDue {
			t.Errorf("Status = %s, want past_due", sub.Status)
		}
	})

	t.Run("returns error for unknown subscription", func(t *testing.T) {
		ctx := context.Background()

		err := service.HandleSubscriptionUpdated(ctx, "unknown_sub", billing.SubscriptionStatusActive)
		if err == nil {
			t.Error("expected error for unknown subscription")
		}
	})
}

func TestPaymentWebhookService_HandleSubscriptionCancelled(t *testing.T) {
	logger := zerolog.Nop()

	users := &mockUserStore{
		users: []ports.User{
			{ID: "user-1", Email: "test@example.com", StripeID: "cus_123", PlanID: "plan-pro"},
		},
	}

	subscriptions := &mockSubscriptionStore{
		subscriptions: []billing.Subscription{
			{
				ID:         "sub-1",
				UserID:     "user-1",
				PlanID:     "plan-pro",
				ProviderID: "sub_abc123",
				Status:     billing.SubscriptionStatusActive,
			},
		},
	}

	plans := &mockPlanStore{
		plans: []ports.Plan{
			{ID: "plan-free", Name: "Free", Enabled: true, IsDefault: true},
			{ID: "plan-pro", Name: "Pro", Enabled: true},
		},
	}

	idGen := &mockIDGenerator{}

	service := NewPaymentWebhookService(users, subscriptions, plans, idGen, logger)

	t.Run("cancels subscription and reverts to default plan", func(t *testing.T) {
		ctx := context.Background()

		err := service.HandleSubscriptionCancelled(ctx, "sub_abc123")
		if err != nil {
			t.Fatalf("HandleSubscriptionCancelled failed: %v", err)
		}

		// Verify subscription was cancelled
		sub := subscriptions.subscriptions[0]
		if sub.Status != billing.SubscriptionStatusCancelled {
			t.Errorf("Status = %s, want cancelled", sub.Status)
		}
		if sub.CancelledAt == nil {
			t.Error("CancelledAt should be set")
		}
	})

	t.Run("returns error for unknown subscription", func(t *testing.T) {
		ctx := context.Background()

		err := service.HandleSubscriptionCancelled(ctx, "unknown_sub")
		if err == nil {
			t.Error("expected error for unknown subscription")
		}
	})
}

func TestPaymentWebhookService_HandleInvoicePaid(t *testing.T) {
	logger := zerolog.Nop()

	users := &mockUserStore{
		users: []ports.User{
			{ID: "user-1", Email: "test@example.com", StripeID: "cus_123"},
		},
	}

	subscriptions := &mockSubscriptionStore{}
	plans := &mockPlanStore{}
	idGen := &mockIDGenerator{}

	service := NewPaymentWebhookService(users, subscriptions, plans, idGen, logger)

	t.Run("logs invoice payment", func(t *testing.T) {
		ctx := context.Background()

		// Should not error even for known customer
		err := service.HandleInvoicePaid(ctx, "inv_123", "cus_123", 1999)
		if err != nil {
			t.Fatalf("HandleInvoicePaid failed: %v", err)
		}
	})

	t.Run("does not error for unknown customer", func(t *testing.T) {
		ctx := context.Background()

		// Should not error for unknown customer (just logs warning)
		err := service.HandleInvoicePaid(ctx, "inv_456", "unknown_customer", 1999)
		if err != nil {
			t.Fatalf("HandleInvoicePaid should not fail for unknown customer: %v", err)
		}
	})
}

func TestPaymentWebhookService_HandleInvoiceFailed(t *testing.T) {
	logger := zerolog.Nop()

	users := &mockUserStore{
		users: []ports.User{
			{ID: "user-1", Email: "test@example.com", StripeID: "cus_123"},
		},
	}

	subscriptions := &mockSubscriptionStore{
		subscriptions: []billing.Subscription{
			{
				ID:         "sub-1",
				UserID:     "user-1",
				PlanID:     "plan-1",
				ProviderID: "sub_abc123",
				Status:     billing.SubscriptionStatusActive,
				CreatedAt:  time.Now(),
			},
		},
	}

	plans := &mockPlanStore{}
	idGen := &mockIDGenerator{}

	service := NewPaymentWebhookService(users, subscriptions, plans, idGen, logger)

	t.Run("marks subscription as past due", func(t *testing.T) {
		ctx := context.Background()

		err := service.HandleInvoiceFailed(ctx, "inv_123", "cus_123")
		if err != nil {
			t.Fatalf("HandleInvoiceFailed failed: %v", err)
		}

		// Verify subscription status updated
		sub := subscriptions.subscriptions[0]
		if sub.Status != billing.SubscriptionStatusPastDue {
			t.Errorf("Status = %s, want past_due", sub.Status)
		}
	})

	t.Run("does not error for unknown customer", func(t *testing.T) {
		ctx := context.Background()

		err := service.HandleInvoiceFailed(ctx, "inv_456", "unknown_customer")
		if err != nil {
			t.Fatalf("HandleInvoiceFailed should not fail for unknown customer: %v", err)
		}
	})
}
