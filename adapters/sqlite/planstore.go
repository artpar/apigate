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
			   is_default, enabled,
			   COALESCE(meter_type, 'requests'), COALESCE(estimated_cost_per_req, 1.0)
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
		var meterType string
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.RateLimitPerMinute, &p.RequestsPerMonth,
			&p.PriceMonthly, &p.OveragePrice, &p.StripePriceID,
			&p.PaddlePriceID, &p.LemonVariantID, &p.IsDefault, &p.Enabled,
			&meterType, &p.EstimatedCostPerReq,
		); err != nil {
			continue
		}
		p.MeterType = ports.MeterType(meterType)
		plans = append(plans, p)
	}
	return plans, nil
}

// Get retrieves a plan by ID.
func (s *PlanStore) Get(ctx context.Context, id string) (ports.Plan, error) {
	var p ports.Plan
	var meterType string
	err := s.db.DB.QueryRowContext(ctx, `
		SELECT id, name, COALESCE(description, ''), rate_limit_per_minute, requests_per_month,
			   price_monthly, overage_price, COALESCE(stripe_price_id, ''),
			   COALESCE(paddle_price_id, ''), COALESCE(lemon_variant_id, ''),
			   is_default, enabled,
			   COALESCE(meter_type, 'requests'), COALESCE(estimated_cost_per_req, 1.0)
		FROM plans WHERE id = ?
	`, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.RateLimitPerMinute, &p.RequestsPerMonth,
		&p.PriceMonthly, &p.OveragePrice, &p.StripePriceID,
		&p.PaddlePriceID, &p.LemonVariantID, &p.IsDefault, &p.Enabled,
		&meterType, &p.EstimatedCostPerReq,
	)
	if err == sql.ErrNoRows {
		return p, sql.ErrNoRows
	}
	p.MeterType = ports.MeterType(meterType)
	return p, err
}

// Create stores a new plan.
func (s *PlanStore) Create(ctx context.Context, p ports.Plan) error {
	meterType := string(p.MeterType)
	if meterType == "" {
		meterType = "requests"
	}
	estimatedCost := p.EstimatedCostPerReq
	if estimatedCost <= 0 {
		estimatedCost = 1.0
	}
	_, err := s.db.DB.ExecContext(ctx, `
		INSERT INTO plans (id, name, description, rate_limit_per_minute, requests_per_month,
						   price_monthly, overage_price, stripe_price_id, paddle_price_id,
						   lemon_variant_id, is_default, enabled, meter_type, estimated_cost_per_req)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.ID, p.Name, p.Description, p.RateLimitPerMinute, p.RequestsPerMonth,
		p.PriceMonthly, p.OveragePrice, p.StripePriceID, p.PaddlePriceID,
		p.LemonVariantID, p.IsDefault, p.Enabled, meterType, estimatedCost)
	return err
}

// Update modifies a plan.
func (s *PlanStore) Update(ctx context.Context, p ports.Plan) error {
	meterType := string(p.MeterType)
	if meterType == "" {
		meterType = "requests"
	}
	estimatedCost := p.EstimatedCostPerReq
	if estimatedCost <= 0 {
		estimatedCost = 1.0
	}
	_, err := s.db.DB.ExecContext(ctx, `
		UPDATE plans SET name = ?, description = ?, rate_limit_per_minute = ?,
						 requests_per_month = ?, price_monthly = ?, overage_price = ?,
						 stripe_price_id = ?, paddle_price_id = ?, lemon_variant_id = ?,
						 is_default = ?, enabled = ?, meter_type = ?, estimated_cost_per_req = ?,
						 updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, p.Name, p.Description, p.RateLimitPerMinute, p.RequestsPerMonth,
		p.PriceMonthly, p.OveragePrice, p.StripePriceID, p.PaddlePriceID,
		p.LemonVariantID, p.IsDefault, p.Enabled, meterType, estimatedCost, p.ID)
	return err
}

// Delete removes a plan.
func (s *PlanStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.DB.ExecContext(ctx, "DELETE FROM plans WHERE id = ?", id)
	return err
}

// ClearOtherDefaults clears is_default on all plans except the specified one.
func (s *PlanStore) ClearOtherDefaults(ctx context.Context, exceptID string) error {
	_, err := s.db.DB.ExecContext(ctx, `
		UPDATE plans SET is_default = 0, updated_at = CURRENT_TIMESTAMP
		WHERE id != ? AND is_default = 1
	`, exceptID)
	return err
}
