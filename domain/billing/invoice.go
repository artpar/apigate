// Package billing provides invoice and pricing value types and pure functions.
package billing

import "time"

// SubscriptionStatus represents subscription state.
type SubscriptionStatus string

const (
	SubscriptionStatusActive    SubscriptionStatus = "active"
	SubscriptionStatusPastDue   SubscriptionStatus = "past_due"
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled"
	SubscriptionStatusPaused    SubscriptionStatus = "paused"
	SubscriptionStatusTrialing  SubscriptionStatus = "trialing"
	SubscriptionStatusUnpaid    SubscriptionStatus = "unpaid"
)

// Subscription represents a billing subscription (value type).
type Subscription struct {
	ID                 string
	UserID             string
	PlanID             string
	ProviderID         string             // External ID (Stripe, Paddle, LemonSqueezy)
	Provider           string             // "stripe", "paddle", "lemonsqueezy"
	ProviderItemID     string             // For usage reporting (Stripe subscription item ID)
	Status             SubscriptionStatus
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
	CancelAtPeriodEnd  bool
	CancelledAt        *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// IsActive returns true if the subscription is in an active state.
func (s Subscription) IsActive() bool {
	return s.Status == SubscriptionStatusActive || s.Status == SubscriptionStatusTrialing
}

// IsCancelling returns true if subscription will cancel at period end.
func (s Subscription) IsCancelling() bool {
	return s.CancelAtPeriodEnd
}

// InvoiceStatus represents the state of an invoice.
type InvoiceStatus string

const (
	InvoiceStatusDraft        InvoiceStatus = "draft"
	InvoiceStatusOpen         InvoiceStatus = "open"
	InvoiceStatusPaid         InvoiceStatus = "paid"
	InvoiceStatusVoid         InvoiceStatus = "void"
	InvoiceStatusUncollectible InvoiceStatus = "uncollectible"
)

// Invoice represents a billing invoice (value type).
type Invoice struct {
	ID          string
	UserID      string
	ProviderID  string // External ID (Stripe, Paddle, LemonSqueezy)
	Provider    string // "stripe", "paddle", "lemonsqueezy"
	PeriodStart time.Time
	PeriodEnd   time.Time
	Items       []InvoiceItem
	Subtotal    int64 // cents
	Tax         int64 // cents
	Total       int64 // cents
	Currency    string
	Status      InvoiceStatus
	DueDate     *time.Time
	PaidAt      *time.Time
	InvoiceURL  string // URL to view/download invoice
	CreatedAt   time.Time
}

// InvoiceItem represents a line item on an invoice (value type).
type InvoiceItem struct {
	Description string
	Quantity    int64
	UnitPrice   int64 // cents
	Amount      int64 // cents
}

// Plan represents a subscription plan (value type).
type Plan struct {
	ID                 string
	Name               string
	Description        string
	RateLimitPerMinute int
	RequestsPerMonth   int64
	PriceMonthly       int64    // In cents
	OveragePrice       int64    // In cents per request over limit
	Features           []string // Feature list for display
	StripePriceID      string
	PaddlePriceID      string
	LemonVariantID     string
	IsDefault          bool
	Enabled            bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// MeterType determines which metric is used for billing.
type MeterType string

const (
	MeterTypeRequests     MeterType = "requests"
	MeterTypeComputeUnits MeterType = "compute_units"
)

// CalculateInvoice creates an invoice from usage and plan.
// This is a PURE function. Backward compatible - uses request count.
func CalculateInvoice(
	userID string,
	periodStart, periodEnd time.Time,
	planName string,
	planPrice int64,
	requestsUsed, requestsIncluded int64,
	overagePrice int64,
) Invoice {
	return CalculateInvoiceWithMeterType(
		userID, periodStart, periodEnd,
		planName, planPrice,
		requestsUsed, requestsIncluded,
		overagePrice, MeterTypeRequests,
	)
}

// CalculateInvoiceWithMeterType creates an invoice from usage and plan.
// Uses meterType to determine labels: "requests" or "compute units".
// This is a PURE function.
func CalculateInvoiceWithMeterType(
	userID string,
	periodStart, periodEnd time.Time,
	planName string,
	planPrice int64,
	unitsUsed, unitsIncluded int64,
	overagePrice int64,
	meterType MeterType,
) Invoice {
	items := []InvoiceItem{
		{
			Description: planName + " - Monthly subscription",
			Quantity:    1,
			UnitPrice:   planPrice,
			Amount:      planPrice,
		},
	}

	subtotal := planPrice

	// Determine unit label based on meter type
	unitLabel := "requests"
	if meterType == MeterTypeComputeUnits {
		unitLabel = "compute units"
	}

	// Add overage if applicable
	if unitsIncluded >= 0 && unitsUsed > unitsIncluded {
		overage := unitsUsed - unitsIncluded
		overageAmount := overage * overagePrice

		items = append(items, InvoiceItem{
			Description: "API overage (" + formatNumber(overage) + " " + unitLabel + ")",
			Quantity:    overage,
			UnitPrice:   overagePrice,
			Amount:      overageAmount,
		})

		subtotal += overageAmount
	}

	return Invoice{
		UserID:      userID,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		Items:       items,
		Subtotal:    subtotal,
		Total:       subtotal, // No tax calculation for simplicity
		Status:      InvoiceStatusDraft,
		CreatedAt:   periodEnd,
	}
}

// FormatAmount formats cents as dollars string.
// This is a PURE function.
func FormatAmount(cents int64) string {
	dollars := cents / 100
	remainder := cents % 100
	if remainder == 0 {
		return "$" + formatNumber(dollars)
	}
	return "$" + formatNumber(dollars) + "." + padZero(remainder)
}

// formatNumber adds comma separators.
func formatNumber(n int64) string {
	if n < 0 {
		return "-" + formatNumber(-n)
	}
	if n < 1000 {
		return itoa(n)
	}
	return formatNumber(n/1000) + "," + padThree(n%1000)
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func padZero(n int64) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

func padThree(n int64) string {
	s := itoa(n)
	for len(s) < 3 {
		s = "0" + s
	}
	return s
}
