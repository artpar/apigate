package payment

import (
	"context"
	"testing"
	"time"

	"github.com/artpar/apigate/domain/billing"
)

func TestNewDummyProvider(t *testing.T) {
	p := NewDummyProvider("http://localhost:8080")
	if p == nil {
		t.Error("NewDummyProvider should return non-nil provider")
	}
	if p.baseURL != "http://localhost:8080" {
		t.Errorf("baseURL = %q, want %q", p.baseURL, "http://localhost:8080")
	}
}

func TestDummyProvider_Name(t *testing.T) {
	p := NewDummyProvider("")
	if p.Name() != "dummy" {
		t.Errorf("Name() = %q, want %q", p.Name(), "dummy")
	}
}

func TestDummyProvider_CreateCustomer(t *testing.T) {
	p := NewDummyProvider("")
	ctx := context.Background()

	customerID, err := p.CreateCustomer(ctx, "test@example.com", "Test User", "user-12345678")
	if err != nil {
		t.Fatalf("CreateCustomer error: %v", err)
	}
	if customerID != "cus_dummy_user-123" {
		t.Errorf("customerID = %q, want %q", customerID, "cus_dummy_user-123")
	}
}

func TestDummyProvider_CreateCheckoutSession(t *testing.T) {
	p := NewDummyProvider("")
	ctx := context.Background()

	successURL := "http://localhost/success"
	cancelURL := "http://localhost/cancel"

	url, err := p.CreateCheckoutSession(ctx, "cus_123", "price_123", successURL, cancelURL, 14)
	if err != nil {
		t.Fatalf("CreateCheckoutSession error: %v", err)
	}
	if url != successURL {
		t.Errorf("URL = %q, want %q", url, successURL)
	}
}

func TestDummyProvider_CreatePortalSession(t *testing.T) {
	p := NewDummyProvider("")
	ctx := context.Background()

	returnURL := "http://localhost/return"

	url, err := p.CreatePortalSession(ctx, "cus_123", returnURL)
	if err != nil {
		t.Fatalf("CreatePortalSession error: %v", err)
	}
	if url != returnURL {
		t.Errorf("URL = %q, want %q", url, returnURL)
	}
}

func TestDummyProvider_CancelSubscription(t *testing.T) {
	p := NewDummyProvider("")
	ctx := context.Background()

	err := p.CancelSubscription(ctx, "sub_123", false)
	if err != nil {
		t.Fatalf("CancelSubscription error: %v", err)
	}

	// Test immediate cancellation
	err = p.CancelSubscription(ctx, "sub_123", true)
	if err != nil {
		t.Fatalf("CancelSubscription (immediately) error: %v", err)
	}
}

func TestDummyProvider_GetSubscription(t *testing.T) {
	p := NewDummyProvider("")
	ctx := context.Background()

	sub, err := p.GetSubscription(ctx, "sub_123")
	if err != nil {
		t.Fatalf("GetSubscription error: %v", err)
	}
	if sub.ID != "sub_123" {
		t.Errorf("ID = %q, want %q", sub.ID, "sub_123")
	}
	if sub.Status != billing.SubscriptionStatusActive {
		t.Errorf("Status = %q, want %q", sub.Status, billing.SubscriptionStatusActive)
	}
	if sub.CurrentPeriodEnd.Before(sub.CurrentPeriodStart) {
		t.Error("CurrentPeriodEnd should be after CurrentPeriodStart")
	}
}

func TestDummyProvider_ReportUsage(t *testing.T) {
	p := NewDummyProvider("")
	ctx := context.Background()

	err := p.ReportUsage(ctx, "si_123", 100, time.Now())
	if err != nil {
		t.Fatalf("ReportUsage error: %v", err)
	}
}

func TestDummyProvider_ParseWebhook(t *testing.T) {
	p := NewDummyProvider("")

	eventType, data, err := p.ParseWebhook([]byte("{}"), "sig_123")
	if err != nil {
		t.Fatalf("ParseWebhook error: %v", err)
	}
	if eventType != "dummy.event" {
		t.Errorf("eventType = %q, want %q", eventType, "dummy.event")
	}
	if data["id"] == nil {
		t.Error("data should contain 'id' field")
	}
}
