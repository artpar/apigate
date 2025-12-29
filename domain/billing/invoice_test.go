package billing_test

import (
	"testing"
	"time"

	"github.com/artpar/apigate/domain/billing"
)

func TestSubscription_IsActive(t *testing.T) {
	tests := []struct {
		status billing.SubscriptionStatus
		want   bool
	}{
		{billing.SubscriptionStatusActive, true},
		{billing.SubscriptionStatusTrialing, true},
		{billing.SubscriptionStatusPastDue, false},
		{billing.SubscriptionStatusCancelled, false},
		{billing.SubscriptionStatusPaused, false},
		{billing.SubscriptionStatusUnpaid, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			s := billing.Subscription{Status: tt.status}
			if s.IsActive() != tt.want {
				t.Errorf("IsActive() = %v, want %v", s.IsActive(), tt.want)
			}
		})
	}
}

func TestSubscription_IsCancelling(t *testing.T) {
	s := billing.Subscription{CancelAtPeriodEnd: true}
	if !s.IsCancelling() {
		t.Error("expected IsCancelling true")
	}

	s = billing.Subscription{CancelAtPeriodEnd: false}
	if s.IsCancelling() {
		t.Error("expected IsCancelling false")
	}
}

func TestSubscription_Fields(t *testing.T) {
	now := time.Now()
	cancelled := now.Add(-time.Hour)

	s := billing.Subscription{
		ID:                 "sub_123",
		UserID:             "user_456",
		PlanID:             "pro",
		ProviderID:         "stripe_sub_abc",
		Provider:           "stripe",
		ProviderItemID:     "si_xyz",
		Status:             billing.SubscriptionStatusActive,
		CurrentPeriodStart: now.Add(-30 * 24 * time.Hour),
		CurrentPeriodEnd:   now,
		CancelAtPeriodEnd:  false,
		CancelledAt:        &cancelled,
		CreatedAt:          now.Add(-90 * 24 * time.Hour),
		UpdatedAt:          now,
	}

	if s.ID != "sub_123" {
		t.Error("ID mismatch")
	}
	if s.Provider != "stripe" {
		t.Error("Provider mismatch")
	}
	if s.CancelledAt == nil || !s.CancelledAt.Equal(cancelled) {
		t.Error("CancelledAt mismatch")
	}
}

func TestInvoiceItem_Fields(t *testing.T) {
	item := billing.InvoiceItem{
		Description: "API Usage",
		Quantity:    1000,
		UnitPrice:   1, // 1 cent per request
		Amount:      1000,
	}

	if item.Description != "API Usage" {
		t.Error("Description mismatch")
	}
	if item.Amount != 1000 {
		t.Error("Amount mismatch")
	}
}

func TestInvoice_Fields(t *testing.T) {
	now := time.Now()
	due := now.Add(30 * 24 * time.Hour)
	paid := now.Add(7 * 24 * time.Hour)

	inv := billing.Invoice{
		ID:          "inv_123",
		UserID:      "user_456",
		ProviderID:  "stripe_inv_abc",
		Provider:    "stripe",
		PeriodStart: now.Add(-30 * 24 * time.Hour),
		PeriodEnd:   now,
		Items: []billing.InvoiceItem{
			{Description: "Pro Plan", Amount: 2999},
		},
		Subtotal:   2999,
		Tax:        0,
		Total:      2999,
		Currency:   "usd",
		Status:     billing.InvoiceStatusPaid,
		DueDate:    &due,
		PaidAt:     &paid,
		InvoiceURL: "https://stripe.com/invoice/123",
		CreatedAt:  now,
	}

	if inv.ID != "inv_123" {
		t.Error("ID mismatch")
	}
	if inv.Total != 2999 {
		t.Error("Total mismatch")
	}
	if inv.Status != billing.InvoiceStatusPaid {
		t.Error("Status mismatch")
	}
	if len(inv.Items) != 1 {
		t.Error("Items mismatch")
	}
}

func TestPlan_Fields(t *testing.T) {
	plan := billing.Plan{
		ID:                 "pro",
		Name:               "Pro Plan",
		Description:        "For growing teams",
		RateLimitPerMinute: 1000,
		RequestsPerMonth:   1000000,
		PriceMonthly:       2999,
		OveragePrice:       1,
		Features:           []string{"Unlimited users", "Priority support"},
		StripePriceID:      "price_abc",
		PaddlePriceID:      "123456",
		LemonVariantID:     "var_789",
		IsDefault:          false,
		Enabled:            true,
	}

	if plan.Name != "Pro Plan" {
		t.Error("Name mismatch")
	}
	if plan.PriceMonthly != 2999 {
		t.Error("PriceMonthly mismatch")
	}
	if len(plan.Features) != 2 {
		t.Error("Features mismatch")
	}
}

func TestCalculateInvoice_NoOverage(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	inv := billing.CalculateInvoice(
		"user_123",
		start, end,
		"Pro Plan",
		2999,  // $29.99
		50000, // 50k requests used
		100000, // 100k included
		1,     // 1 cent overage
	)

	if inv.UserID != "user_123" {
		t.Error("UserID mismatch")
	}
	if !inv.PeriodStart.Equal(start) {
		t.Error("PeriodStart mismatch")
	}
	if !inv.PeriodEnd.Equal(end) {
		t.Error("PeriodEnd mismatch")
	}
	if len(inv.Items) != 1 {
		t.Errorf("expected 1 item (no overage), got %d", len(inv.Items))
	}
	if inv.Subtotal != 2999 {
		t.Errorf("Subtotal = %d, want 2999", inv.Subtotal)
	}
	if inv.Total != 2999 {
		t.Errorf("Total = %d, want 2999", inv.Total)
	}
	if inv.Status != billing.InvoiceStatusDraft {
		t.Error("Status should be draft")
	}
}

func TestCalculateInvoice_WithOverage(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	inv := billing.CalculateInvoice(
		"user_123",
		start, end,
		"Pro Plan",
		2999,   // $29.99
		150000, // 150k requests used
		100000, // 100k included
		1,      // 1 cent per overage request
	)

	if len(inv.Items) != 2 {
		t.Fatalf("expected 2 items (base + overage), got %d", len(inv.Items))
	}

	// Base subscription
	if inv.Items[0].Amount != 2999 {
		t.Errorf("base amount = %d, want 2999", inv.Items[0].Amount)
	}

	// Overage: 50k requests * 1 cent = $5.00 = 500 cents
	if inv.Items[1].Quantity != 50000 {
		t.Errorf("overage quantity = %d, want 50000", inv.Items[1].Quantity)
	}
	if inv.Items[1].Amount != 50000 {
		t.Errorf("overage amount = %d, want 50000", inv.Items[1].Amount)
	}

	// Total: $29.99 + $500.00 = $529.99 = 52999 cents
	if inv.Subtotal != 52999 {
		t.Errorf("Subtotal = %d, want 52999", inv.Subtotal)
	}
	if inv.Total != 52999 {
		t.Errorf("Total = %d, want 52999", inv.Total)
	}
}

func TestCalculateInvoice_UnlimitedPlan(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	inv := billing.CalculateInvoice(
		"user_123",
		start, end,
		"Enterprise",
		9999,    // $99.99
		1000000, // 1M requests used
		-1,      // -1 = unlimited
		0,       // no overage price
	)

	if len(inv.Items) != 1 {
		t.Errorf("expected 1 item (no overage for unlimited), got %d", len(inv.Items))
	}
	if inv.Total != 9999 {
		t.Errorf("Total = %d, want 9999", inv.Total)
	}
}

func TestFormatAmount(t *testing.T) {
	tests := []struct {
		cents int64
		want  string
	}{
		{0, "$0"},
		{100, "$1"},
		{2999, "$29.99"},
		{10000, "$100"},
		{100000, "$1,000"},
		{999999, "$9,999.99"},
		{1234567, "$12,345.67"},
		{100000000, "$1,000,000"},
		{50, "$0.50"},
		{5, "$0.05"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := billing.FormatAmount(tt.cents)
			if got != tt.want {
				t.Errorf("FormatAmount(%d) = %q, want %q", tt.cents, got, tt.want)
			}
		})
	}
}

func TestInvoiceStatus_Constants(t *testing.T) {
	statuses := []billing.InvoiceStatus{
		billing.InvoiceStatusDraft,
		billing.InvoiceStatusOpen,
		billing.InvoiceStatusPaid,
		billing.InvoiceStatusVoid,
		billing.InvoiceStatusUncollectible,
	}

	for _, s := range statuses {
		if s == "" {
			t.Error("status should not be empty")
		}
	}
}

func TestSubscriptionStatus_Constants(t *testing.T) {
	statuses := []billing.SubscriptionStatus{
		billing.SubscriptionStatusActive,
		billing.SubscriptionStatusPastDue,
		billing.SubscriptionStatusCancelled,
		billing.SubscriptionStatusPaused,
		billing.SubscriptionStatusTrialing,
		billing.SubscriptionStatusUnpaid,
	}

	for _, s := range statuses {
		if s == "" {
			t.Error("status should not be empty")
		}
	}
}
