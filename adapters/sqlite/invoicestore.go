package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/artpar/apigate/domain/billing"
	"github.com/artpar/apigate/ports"
)

// InvoiceStore implements ports.InvoiceStore using SQLite.
type InvoiceStore struct {
	db *DB
}

// NewInvoiceStore creates a new SQLite invoice store.
func NewInvoiceStore(db *DB) *InvoiceStore {
	return &InvoiceStore{db: db}
}

// Create stores a new invoice.
func (s *InvoiceStore) Create(ctx context.Context, inv billing.Invoice) error {
	now := time.Now().UTC()
	if inv.CreatedAt.IsZero() {
		inv.CreatedAt = now
	}

	itemsJSON, err := json.Marshal(inv.Items)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO invoices (
			id, user_id, provider, provider_id,
			period_start, period_end, items,
			subtotal, tax, total, currency,
			status, due_date, paid_at, invoice_url, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		inv.ID, inv.UserID, inv.Provider, nullString(inv.ProviderID),
		inv.PeriodStart, inv.PeriodEnd, string(itemsJSON),
		inv.Subtotal, inv.Tax, inv.Total, inv.Currency,
		string(inv.Status), nullTime(inv.DueDate), nullTime(inv.PaidAt),
		nullString(inv.InvoiceURL), inv.CreatedAt,
	)

	if err != nil && isUniqueConstraintError(err) {
		return ErrDuplicate
	}
	return err
}

// ListByUser returns invoices for a user.
func (s *InvoiceStore) ListByUser(ctx context.Context, userID string, limit int) ([]billing.Invoice, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, provider, provider_id,
		       period_start, period_end, items,
		       subtotal, tax, total, currency,
		       status, due_date, paid_at, invoice_url, created_at
		FROM invoices
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []billing.Invoice
	for rows.Next() {
		inv, err := scanInvoiceRow(rows)
		if err != nil {
			return nil, err
		}
		invoices = append(invoices, inv)
	}
	return invoices, rows.Err()
}

// UpdateStatus updates invoice status.
func (s *InvoiceStore) UpdateStatus(ctx context.Context, id string, status billing.InvoiceStatus, paidAt *time.Time) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE invoices
		SET status = ?, paid_at = ?
		WHERE id = ?
	`, string(status), nullTime(paidAt), id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func scanInvoiceRow(rows *sql.Rows) (billing.Invoice, error) {
	var inv billing.Invoice
	var status string
	var providerID, invoiceURL sql.NullString
	var itemsJSON string
	var dueDate, paidAt sql.NullTime

	err := rows.Scan(
		&inv.ID, &inv.UserID, &inv.Provider, &providerID,
		&inv.PeriodStart, &inv.PeriodEnd, &itemsJSON,
		&inv.Subtotal, &inv.Tax, &inv.Total, &inv.Currency,
		&status, &dueDate, &paidAt, &invoiceURL, &inv.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return billing.Invoice{}, ErrNotFound
	}
	if err != nil {
		return billing.Invoice{}, err
	}

	inv.Status = billing.InvoiceStatus(status)
	if providerID.Valid {
		inv.ProviderID = providerID.String
	}
	if invoiceURL.Valid {
		inv.InvoiceURL = invoiceURL.String
	}
	if dueDate.Valid {
		inv.DueDate = &dueDate.Time
	}
	if paidAt.Valid {
		inv.PaidAt = &paidAt.Time
	}

	if itemsJSON != "" {
		if err := json.Unmarshal([]byte(itemsJSON), &inv.Items); err != nil {
			return billing.Invoice{}, err
		}
	}

	return inv, nil
}

// Ensure interface compliance.
var _ ports.InvoiceStore = (*InvoiceStore)(nil)
