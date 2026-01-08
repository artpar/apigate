package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/artpar/apigate/domain/entitlement"
)

// PlanEntitlementStore implements ports.PlanEntitlementStore with SQLite.
type PlanEntitlementStore struct {
	db *DB
}

// NewPlanEntitlementStore creates a new SQLite plan entitlement store.
func NewPlanEntitlementStore(db *DB) *PlanEntitlementStore {
	return &PlanEntitlementStore{db: db}
}

// List returns all plan-entitlement mappings.
func (s *PlanEntitlementStore) List(ctx context.Context) ([]entitlement.PlanEntitlement, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, plan_id, entitlement_id, COALESCE(value, ''),
		       COALESCE(notes, ''), enabled, created_at, updated_at
		FROM plan_entitlements
		ORDER BY plan_id, entitlement_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanPlanEntitlements(rows)
}

// ListByPlan returns all entitlements for a specific plan.
func (s *PlanEntitlementStore) ListByPlan(ctx context.Context, planID string) ([]entitlement.PlanEntitlement, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, plan_id, entitlement_id, COALESCE(value, ''),
		       COALESCE(notes, ''), enabled, created_at, updated_at
		FROM plan_entitlements
		WHERE plan_id = ? AND enabled = 1
		ORDER BY entitlement_id
	`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanPlanEntitlements(rows)
}

// ListByEntitlement returns all plans that have a specific entitlement.
func (s *PlanEntitlementStore) ListByEntitlement(ctx context.Context, entitlementID string) ([]entitlement.PlanEntitlement, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, plan_id, entitlement_id, COALESCE(value, ''),
		       COALESCE(notes, ''), enabled, created_at, updated_at
		FROM plan_entitlements
		WHERE entitlement_id = ? AND enabled = 1
		ORDER BY plan_id
	`, entitlementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanPlanEntitlements(rows)
}

// Get retrieves a plan-entitlement mapping by ID.
func (s *PlanEntitlementStore) Get(ctx context.Context, id string) (entitlement.PlanEntitlement, error) {
	var pe entitlement.PlanEntitlement
	err := s.db.QueryRowContext(ctx, `
		SELECT id, plan_id, entitlement_id, COALESCE(value, ''),
		       COALESCE(notes, ''), enabled, created_at, updated_at
		FROM plan_entitlements WHERE id = ?
	`, id).Scan(
		&pe.ID, &pe.PlanID, &pe.EntitlementID, &pe.Value,
		&pe.Notes, &pe.Enabled, &pe.CreatedAt, &pe.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return pe, sql.ErrNoRows
	}
	return pe, err
}

// GetByPlanAndEntitlement retrieves a specific mapping.
func (s *PlanEntitlementStore) GetByPlanAndEntitlement(ctx context.Context, planID, entitlementID string) (entitlement.PlanEntitlement, error) {
	var pe entitlement.PlanEntitlement
	err := s.db.QueryRowContext(ctx, `
		SELECT id, plan_id, entitlement_id, COALESCE(value, ''),
		       COALESCE(notes, ''), enabled, created_at, updated_at
		FROM plan_entitlements
		WHERE plan_id = ? AND entitlement_id = ?
	`, planID, entitlementID).Scan(
		&pe.ID, &pe.PlanID, &pe.EntitlementID, &pe.Value,
		&pe.Notes, &pe.Enabled, &pe.CreatedAt, &pe.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return pe, sql.ErrNoRows
	}
	return pe, err
}

// Create stores a new plan-entitlement mapping.
func (s *PlanEntitlementStore) Create(ctx context.Context, pe entitlement.PlanEntitlement) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO plan_entitlements (id, plan_id, entitlement_id, value, notes, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, pe.ID, pe.PlanID, pe.EntitlementID, pe.Value, pe.Notes, pe.Enabled, now, now)
	return err
}

// Update modifies a plan-entitlement mapping.
func (s *PlanEntitlementStore) Update(ctx context.Context, pe entitlement.PlanEntitlement) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE plan_entitlements SET
			plan_id = ?, entitlement_id = ?, value = ?, notes = ?, enabled = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, pe.PlanID, pe.EntitlementID, pe.Value, pe.Notes, pe.Enabled, pe.ID)
	return err
}

// Delete removes a plan-entitlement mapping.
func (s *PlanEntitlementStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM plan_entitlements WHERE id = ?", id)
	return err
}

// scanPlanEntitlements scans rows into plan entitlement slice.
func scanPlanEntitlements(rows *sql.Rows) ([]entitlement.PlanEntitlement, error) {
	var pes []entitlement.PlanEntitlement
	for rows.Next() {
		var pe entitlement.PlanEntitlement
		if err := rows.Scan(
			&pe.ID, &pe.PlanID, &pe.EntitlementID, &pe.Value,
			&pe.Notes, &pe.Enabled, &pe.CreatedAt, &pe.UpdatedAt,
		); err != nil {
			continue
		}
		pes = append(pes, pe)
	}
	return pes, rows.Err()
}
