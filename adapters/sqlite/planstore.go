package sqlite

import (
	"context"
	"database/sql"

	"github.com/artpar/apigate/ports"
)

// PlanStore implements ports.PlanStore with SQLite.
type PlanStore struct {
	db *DB
}

// NewPlanStore creates a new SQLite plan store.
func NewPlanStore(db *DB) *PlanStore {
	return &PlanStore{db: db}
}

// List returns all enabled plans.
func (s *PlanStore) List(ctx context.Context) ([]ports.Plan, error) {
	rows, err := s.db.DB.QueryContext(ctx, `
		SELECT id, name, COALESCE(description, ''), rate_limit_per_minute, requests_per_month,
			   price_monthly, overage_price, COALESCE(stripe_price_id, ''),
			   COALESCE(paddle_price_id, ''), COALESCE(lemon_variant_id, ''),
			   is_default, enabled
		FROM plans WHERE enabled = 1
		ORDER BY price_monthly ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []ports.Plan
	for rows.Next() {
		var p ports.Plan
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.RateLimitPerMinute, &p.RequestsPerMonth,
			&p.PriceMonthly, &p.OveragePrice, &p.StripePriceID,
			&p.PaddlePriceID, &p.LemonVariantID, &p.IsDefault, &p.Enabled,
		); err != nil {
			continue
		}
		plans = append(plans, p)
	}
	return plans, nil
}

// Get retrieves a plan by ID.
func (s *PlanStore) Get(ctx context.Context, id string) (ports.Plan, error) {
	var p ports.Plan
	err := s.db.DB.QueryRowContext(ctx, `
		SELECT id, name, COALESCE(description, ''), rate_limit_per_minute, requests_per_month,
			   price_monthly, overage_price, COALESCE(stripe_price_id, ''),
			   COALESCE(paddle_price_id, ''), COALESCE(lemon_variant_id, ''),
			   is_default, enabled
		FROM plans WHERE id = ?
	`, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.RateLimitPerMinute, &p.RequestsPerMonth,
		&p.PriceMonthly, &p.OveragePrice, &p.StripePriceID,
		&p.PaddlePriceID, &p.LemonVariantID, &p.IsDefault, &p.Enabled,
	)
	if err == sql.ErrNoRows {
		return p, sql.ErrNoRows
	}
	return p, err
}

// Create stores a new plan.
func (s *PlanStore) Create(ctx context.Context, p ports.Plan) error {
	_, err := s.db.DB.ExecContext(ctx, `
		INSERT INTO plans (id, name, description, rate_limit_per_minute, requests_per_month,
						   price_monthly, overage_price, stripe_price_id, paddle_price_id,
						   lemon_variant_id, is_default, enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.ID, p.Name, p.Description, p.RateLimitPerMinute, p.RequestsPerMonth,
		p.PriceMonthly, p.OveragePrice, p.StripePriceID, p.PaddlePriceID,
		p.LemonVariantID, p.IsDefault, p.Enabled)
	return err
}

// Update modifies a plan.
func (s *PlanStore) Update(ctx context.Context, p ports.Plan) error {
	_, err := s.db.DB.ExecContext(ctx, `
		UPDATE plans SET name = ?, description = ?, rate_limit_per_minute = ?,
						 requests_per_month = ?, price_monthly = ?, overage_price = ?,
						 stripe_price_id = ?, paddle_price_id = ?, lemon_variant_id = ?,
						 is_default = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, p.Name, p.Description, p.RateLimitPerMinute, p.RequestsPerMonth,
		p.PriceMonthly, p.OveragePrice, p.StripePriceID, p.PaddlePriceID,
		p.LemonVariantID, p.IsDefault, p.Enabled, p.ID)
	return err
}

// Delete removes a plan.
func (s *PlanStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.DB.ExecContext(ctx, "DELETE FROM plans WHERE id = ?", id)
	return err
}
