// Package billing provides invoice and pricing value types and pure functions.
package billing

import "time"

// Invoice represents a billing invoice (value type).
type Invoice struct {
	ID              string
	UserID          string
	StripeInvoiceID string
	PeriodStart     time.Time
	PeriodEnd       time.Time
	Items           []InvoiceItem
	Subtotal        int64 // cents
	Tax             int64 // cents
	Total           int64 // cents
	Status          InvoiceStatus
	PaidAt          *time.Time
	CreatedAt       time.Time
}

// InvoiceItem represents a line item on an invoice (value type).
type InvoiceItem struct {
	Description string
	Quantity    int64
	UnitPrice   int64 // cents
	Amount      int64 // cents
}

// InvoiceStatus represents the state of an invoice.
type InvoiceStatus string

const (
	InvoiceStatusDraft     InvoiceStatus = "draft"
	InvoiceStatusOpen      InvoiceStatus = "open"
	InvoiceStatusPaid      InvoiceStatus = "paid"
	InvoiceStatusVoid      InvoiceStatus = "void"
	InvoiceStatusUncollect InvoiceStatus = "uncollectible"
)

// Subscription represents a billing subscription (value type).
type Subscription struct {
	ID                   string
	UserID               string
	PlanID               string
	StripeSubscriptionID string
	StripeItemID         string // For usage reporting
	Status               SubscriptionStatus
	CurrentPeriodStart   time.Time
	CurrentPeriodEnd     time.Time
	CancelledAt          *time.Time
	CreatedAt            time.Time
}

// SubscriptionStatus represents subscription state.
type SubscriptionStatus string

const (
	SubStatusActive   SubscriptionStatus = "active"
	SubStatusPastDue  SubscriptionStatus = "past_due"
	SubStatusCanceled SubscriptionStatus = "canceled"
	SubStatusTrialing SubscriptionStatus = "trialing"
)

// CalculateInvoice creates an invoice from usage and plan.
// This is a PURE function.
func CalculateInvoice(
	userID string,
	periodStart, periodEnd time.Time,
	planName string,
	planPrice int64,
	requestsUsed, requestsIncluded int64,
	overagePrice int64,
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

	// Add overage if applicable
	if requestsIncluded >= 0 && requestsUsed > requestsIncluded {
		overage := requestsUsed - requestsIncluded
		overageAmount := overage * overagePrice

		items = append(items, InvoiceItem{
			Description: "API overage (" + formatNumber(overage) + " requests)",
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
