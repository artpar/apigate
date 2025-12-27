package remote

import (
	"context"
	"time"

	"github.com/artpar/apigate/domain/billing"
	"github.com/artpar/apigate/ports"
)

// BillingProvider delegates billing operations to an external HTTP service.
// This enables customers to use their own billing/subscription system.
//
// API Contract:
//
//	POST /billing/customers
//	Request:  {"email": "...", "name": "...", "user_id": "..."}
//	Response: {"customer_id": "cus_123"}
//
//	POST /billing/subscriptions
//	Request:  {"customer_id": "cus_123", "plan_id": "pro"}
//	Response: {"subscription": {...}}
//
//	POST /billing/subscriptions/{id}/cancel
//	Response: {}
//
//	POST /billing/usage
//	Request:  {"subscription_id": "...", "quantity": 100, "timestamp": "..."}
//	Response: {}
//
//	POST /billing/invoices
//	Request:  {"customer_id": "...", "items": [...]}
//	Response: {"invoice": {...}}
type BillingProvider struct {
	client *Client
}

// NewBillingProvider creates a remote billing provider.
func NewBillingProvider(client *Client) *BillingProvider {
	return &BillingProvider{client: client}
}

// RemoteSubscription is the wire format for subscriptions.
type RemoteSubscription struct {
	ID                 string    `json:"id"`
	UserID             string    `json:"user_id"`
	CustomerID         string    `json:"customer_id"`
	PlanID             string    `json:"plan_id"`
	Status             string    `json:"status"`
	CurrentPeriodStart time.Time `json:"current_period_start"`
	CurrentPeriodEnd   time.Time `json:"current_period_end"`
	CanceledAt         *time.Time `json:"canceled_at,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

// RemoteInvoice is the wire format for invoices.
type RemoteInvoice struct {
	ID            string               `json:"id"`
	UserID        string               `json:"user_id"`
	CustomerID    string               `json:"customer_id"`
	Amount        int64                `json:"amount"`
	Currency      string               `json:"currency"`
	Status        string               `json:"status"`
	PeriodStart   time.Time            `json:"period_start"`
	PeriodEnd     time.Time            `json:"period_end"`
	Items         []RemoteInvoiceItem  `json:"items"`
	PaidAt        *time.Time           `json:"paid_at,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
}

// RemoteInvoiceItem is a line item on an invoice.
type RemoteInvoiceItem struct {
	Description string `json:"description"`
	Quantity    int64  `json:"quantity"`
	UnitPrice   int64  `json:"unit_price"`
	Amount      int64  `json:"amount"`
}

// CreateCustomer creates a customer in the billing system.
func (p *BillingProvider) CreateCustomer(ctx context.Context, email, name, userID string) (string, error) {
	req := map[string]string{
		"email":   email,
		"name":    name,
		"user_id": userID,
	}

	var resp struct {
		CustomerID string `json:"customer_id"`
	}

	err := p.client.Request(ctx, "POST", "/billing/customers", req, &resp)
	if err != nil {
		return "", err
	}

	return resp.CustomerID, nil
}

// CreateSubscription creates a subscription for a customer.
func (p *BillingProvider) CreateSubscription(ctx context.Context, customerID, priceID string) (billing.Subscription, error) {
	req := map[string]string{
		"customer_id": customerID,
		"plan_id":     priceID,
	}

	var resp struct {
		Subscription RemoteSubscription `json:"subscription"`
	}

	err := p.client.Request(ctx, "POST", "/billing/subscriptions", req, &resp)
	if err != nil {
		return billing.Subscription{}, err
	}

	return toSubscription(resp.Subscription), nil
}

// CancelSubscription cancels a subscription.
func (p *BillingProvider) CancelSubscription(ctx context.Context, subscriptionID string) error {
	return p.client.Request(ctx, "POST", "/billing/subscriptions/"+subscriptionID+"/cancel", nil, nil)
}

// ReportUsage reports metered usage.
func (p *BillingProvider) ReportUsage(ctx context.Context, subscriptionItemID string, quantity int64, timestamp time.Time) error {
	req := map[string]interface{}{
		"subscription_item_id": subscriptionItemID,
		"quantity":             quantity,
		"timestamp":            timestamp,
	}

	return p.client.Request(ctx, "POST", "/billing/usage", req, nil)
}

// CreateInvoice creates an invoice for a customer.
func (p *BillingProvider) CreateInvoice(ctx context.Context, customerID string, items []billing.InvoiceItem) (billing.Invoice, error) {
	remoteItems := make([]RemoteInvoiceItem, len(items))
	for i, item := range items {
		remoteItems[i] = RemoteInvoiceItem{
			Description: item.Description,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			Amount:      item.Amount,
		}
	}

	req := map[string]interface{}{
		"customer_id": customerID,
		"items":       remoteItems,
	}

	var resp struct {
		Invoice RemoteInvoice `json:"invoice"`
	}

	err := p.client.Request(ctx, "POST", "/billing/invoices", req, &resp)
	if err != nil {
		return billing.Invoice{}, err
	}

	return toInvoice(resp.Invoice), nil
}

func toSubscription(rs RemoteSubscription) billing.Subscription {
	return billing.Subscription{
		ID:                   rs.ID,
		UserID:               rs.UserID,
		PlanID:               rs.PlanID,
		Status:               billing.SubscriptionStatus(rs.Status),
		CurrentPeriodStart:   rs.CurrentPeriodStart,
		CurrentPeriodEnd:     rs.CurrentPeriodEnd,
		CancelledAt:          rs.CanceledAt,
		CreatedAt:            rs.CreatedAt,
	}
}

func toInvoice(ri RemoteInvoice) billing.Invoice {
	items := make([]billing.InvoiceItem, len(ri.Items))
	for i, item := range ri.Items {
		items[i] = billing.InvoiceItem{
			Description: item.Description,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			Amount:      item.Amount,
		}
	}

	return billing.Invoice{
		ID:          ri.ID,
		UserID:      ri.UserID,
		PeriodStart: ri.PeriodStart,
		PeriodEnd:   ri.PeriodEnd,
		Items:       items,
		Total:       ri.Amount,
		Status:      billing.InvoiceStatus(ri.Status),
		PaidAt:      ri.PaidAt,
		CreatedAt:   ri.CreatedAt,
	}
}

// Ensure interface compliance.
var _ ports.BillingProvider = (*BillingProvider)(nil)
