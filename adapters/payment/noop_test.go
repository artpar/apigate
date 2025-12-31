package payment

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewNoopProvider(t *testing.T) {
	provider := NewNoopProvider()

	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestNoopProvider_Name(t *testing.T) {
	provider := NewNoopProvider()

	name := provider.Name()

	if name != "none" {
		t.Errorf("Name() = %s, want none", name)
	}
}

func TestNoopProvider_CreateCustomer(t *testing.T) {
	provider := NewNoopProvider()
	ctx := context.Background()

	tests := []struct {
		name   string
		email  string
		uname  string
		userID string
	}{
		{"basic customer", "test@example.com", "Test User", "user_123"},
		{"empty email", "", "Test User", "user_456"},
		{"empty name", "test@example.com", "", "user_789"},
		{"empty user ID", "test@example.com", "Test User", ""},
		{"all empty", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customerID, err := provider.CreateCustomer(ctx, tt.email, tt.uname, tt.userID)

			if !errors.Is(err, ErrPaymentsDisabled) {
				t.Errorf("expected ErrPaymentsDisabled, got %v", err)
			}
			if customerID != "" {
				t.Errorf("expected empty customerID, got %s", customerID)
			}
		})
	}
}

func TestNoopProvider_CreateCheckoutSession(t *testing.T) {
	provider := NewNoopProvider()
	ctx := context.Background()

	tests := []struct {
		name       string
		customerID string
		priceID    string
		successURL string
		cancelURL  string
		trialDays  int
	}{
		{"basic session", "cus_123", "price_abc", "https://success.com", "https://cancel.com", 0},
		{"with trial", "cus_456", "price_def", "https://success.com", "https://cancel.com", 14},
		{"empty URLs", "cus_789", "price_ghi", "", "", 0},
		{"negative trial", "cus_000", "price_jkl", "https://success.com", "https://cancel.com", -1},
		{"large trial", "cus_111", "price_mno", "https://success.com", "https://cancel.com", 365},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := provider.CreateCheckoutSession(ctx, tt.customerID, tt.priceID, tt.successURL, tt.cancelURL, tt.trialDays)

			if !errors.Is(err, ErrPaymentsDisabled) {
				t.Errorf("expected ErrPaymentsDisabled, got %v", err)
			}
			if url != "" {
				t.Errorf("expected empty URL, got %s", url)
			}
		})
	}
}

func TestNoopProvider_CreatePortalSession(t *testing.T) {
	provider := NewNoopProvider()
	ctx := context.Background()

	tests := []struct {
		name       string
		customerID string
		returnURL  string
	}{
		{"basic portal", "cus_123", "https://return.com"},
		{"empty customer", "", "https://return.com"},
		{"empty URL", "cus_456", ""},
		{"all empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := provider.CreatePortalSession(ctx, tt.customerID, tt.returnURL)

			if !errors.Is(err, ErrPaymentsDisabled) {
				t.Errorf("expected ErrPaymentsDisabled, got %v", err)
			}
			if url != "" {
				t.Errorf("expected empty URL, got %s", url)
			}
		})
	}
}

func TestNoopProvider_CancelSubscription(t *testing.T) {
	provider := NewNoopProvider()
	ctx := context.Background()

	tests := []struct {
		name           string
		subscriptionID string
		immediately    bool
	}{
		{"immediate cancellation", "sub_123", true},
		{"end of period cancellation", "sub_456", false},
		{"empty subscription ID", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.CancelSubscription(ctx, tt.subscriptionID, tt.immediately)

			if !errors.Is(err, ErrPaymentsDisabled) {
				t.Errorf("expected ErrPaymentsDisabled, got %v", err)
			}
		})
	}
}

func TestNoopProvider_GetSubscription(t *testing.T) {
	provider := NewNoopProvider()
	ctx := context.Background()

	tests := []struct {
		name           string
		subscriptionID string
	}{
		{"basic subscription", "sub_123"},
		{"empty subscription ID", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub, err := provider.GetSubscription(ctx, tt.subscriptionID)

			if !errors.Is(err, ErrPaymentsDisabled) {
				t.Errorf("expected ErrPaymentsDisabled, got %v", err)
			}
			if sub.ID != "" {
				t.Errorf("expected empty subscription ID, got %s", sub.ID)
			}
		})
	}
}

func TestNoopProvider_ReportUsage(t *testing.T) {
	provider := NewNoopProvider()
	ctx := context.Background()

	tests := []struct {
		name               string
		subscriptionItemID string
		quantity           int64
		timestamp          time.Time
	}{
		{"basic usage", "si_123", 1000, time.Now()},
		{"zero quantity", "si_456", 0, time.Now()},
		{"negative quantity", "si_789", -100, time.Now()},
		{"empty item ID", "", 100, time.Now()},
		{"past timestamp", "si_000", 50, time.Now().Add(-24 * time.Hour)},
		{"future timestamp", "si_111", 200, time.Now().Add(24 * time.Hour)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.ReportUsage(ctx, tt.subscriptionItemID, tt.quantity, tt.timestamp)

			if !errors.Is(err, ErrPaymentsDisabled) {
				t.Errorf("expected ErrPaymentsDisabled, got %v", err)
			}
		})
	}
}

func TestNoopProvider_ParseWebhook(t *testing.T) {
	provider := NewNoopProvider()

	tests := []struct {
		name      string
		payload   []byte
		signature string
	}{
		{"basic webhook", []byte(`{"event":"test"}`), "signature123"},
		{"empty payload", []byte{}, "signature456"},
		{"nil payload", nil, "signature789"},
		{"empty signature", []byte(`{"event":"test"}`), ""},
		{"both empty", []byte{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventType, data, err := provider.ParseWebhook(tt.payload, tt.signature)

			if !errors.Is(err, ErrPaymentsDisabled) {
				t.Errorf("expected ErrPaymentsDisabled, got %v", err)
			}
			if eventType != "" {
				t.Errorf("expected empty eventType, got %s", eventType)
			}
			if data != nil {
				t.Errorf("expected nil data, got %v", data)
			}
		})
	}
}

func TestErrPaymentsDisabled_ErrorMessage(t *testing.T) {
	expected := "payments are not configured"

	if ErrPaymentsDisabled.Error() != expected {
		t.Errorf("ErrPaymentsDisabled.Error() = %s, want %s", ErrPaymentsDisabled.Error(), expected)
	}
}

func TestErrPaymentsDisabled_ErrorIs(t *testing.T) {
	// Test that errors.Is works with ErrPaymentsDisabled
	err := ErrPaymentsDisabled

	if !errors.Is(err, ErrPaymentsDisabled) {
		t.Error("errors.Is should return true for same error")
	}

	otherErr := errors.New("some other error")
	if errors.Is(otherErr, ErrPaymentsDisabled) {
		t.Error("errors.Is should return false for different error")
	}
}

func TestNoopProvider_ContextCancellation(t *testing.T) {
	provider := NewNoopProvider()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// All operations should still return ErrPaymentsDisabled, not context error
	_, err := provider.CreateCustomer(ctx, "test@example.com", "Test", "user_123")
	if !errors.Is(err, ErrPaymentsDisabled) {
		t.Errorf("CreateCustomer: expected ErrPaymentsDisabled, got %v", err)
	}

	_, err = provider.CreateCheckoutSession(ctx, "cus_123", "price_abc", "https://success.com", "https://cancel.com", 0)
	if !errors.Is(err, ErrPaymentsDisabled) {
		t.Errorf("CreateCheckoutSession: expected ErrPaymentsDisabled, got %v", err)
	}

	_, err = provider.CreatePortalSession(ctx, "cus_123", "https://return.com")
	if !errors.Is(err, ErrPaymentsDisabled) {
		t.Errorf("CreatePortalSession: expected ErrPaymentsDisabled, got %v", err)
	}

	err = provider.CancelSubscription(ctx, "sub_123", true)
	if !errors.Is(err, ErrPaymentsDisabled) {
		t.Errorf("CancelSubscription: expected ErrPaymentsDisabled, got %v", err)
	}

	_, err = provider.GetSubscription(ctx, "sub_123")
	if !errors.Is(err, ErrPaymentsDisabled) {
		t.Errorf("GetSubscription: expected ErrPaymentsDisabled, got %v", err)
	}

	err = provider.ReportUsage(ctx, "si_123", 100, time.Now())
	if !errors.Is(err, ErrPaymentsDisabled) {
		t.Errorf("ReportUsage: expected ErrPaymentsDisabled, got %v", err)
	}
}

func TestNoopProvider_ConcurrentCalls(t *testing.T) {
	provider := NewNoopProvider()
	ctx := context.Background()

	// Test concurrent access
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			_, _ = provider.CreateCustomer(ctx, "test@example.com", "Test", "user_123")
			_, _ = provider.CreateCheckoutSession(ctx, "cus_123", "price_abc", "https://success.com", "https://cancel.com", 0)
			_ = provider.Name()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
