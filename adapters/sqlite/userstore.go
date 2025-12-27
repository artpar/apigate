package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/artpar/apigate/ports"
)

// ErrNotFound is returned when an entity is not found.
var ErrNotFound = errors.New("not found")

// ErrDuplicate is returned when a unique constraint is violated.
var ErrDuplicate = errors.New("already exists")

// UserStore implements ports.UserStore using SQLite.
type UserStore struct {
	db *DB
}

// NewUserStore creates a new SQLite user store.
func NewUserStore(db *DB) *UserStore {
	return &UserStore{db: db}
}

// Get retrieves a user by ID.
func (s *UserStore) Get(ctx context.Context, id string) (ports.User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, email, name, stripe_id, plan_id, status, created_at, updated_at
		FROM users
		WHERE id = ?
	`, id)
	return scanUser(row)
}

// GetByEmail retrieves a user by email.
func (s *UserStore) GetByEmail(ctx context.Context, email string) (ports.User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, email, name, stripe_id, plan_id, status, created_at, updated_at
		FROM users
		WHERE email = ?
	`, email)
	return scanUser(row)
}

// Create stores a new user.
func (s *UserStore) Create(ctx context.Context, u ports.User) error {
	now := time.Now().UTC()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = now
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (id, email, name, stripe_id, plan_id, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, u.ID, u.Email, u.Name, nullString(u.StripeID), u.PlanID, u.Status, u.CreatedAt, u.UpdatedAt)

	if err != nil && isUniqueConstraintError(err) {
		return ErrDuplicate
	}
	return err
}

// Update modifies an existing user.
func (s *UserStore) Update(ctx context.Context, u ports.User) error {
	u.UpdatedAt = time.Now().UTC()

	result, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET email = ?, name = ?, stripe_id = ?, plan_id = ?, status = ?, updated_at = ?
		WHERE id = ?
	`, u.Email, u.Name, nullString(u.StripeID), u.PlanID, u.Status, u.UpdatedAt, u.ID)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrDuplicate
		}
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

// List returns users with pagination.
func (s *UserStore) List(ctx context.Context, limit, offset int) ([]ports.User, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, email, name, stripe_id, plan_id, status, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []ports.User
	for rows.Next() {
		u, err := scanUserRows(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// Count returns total user count.
func (s *UserStore) Count(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

// Delete permanently removes a user.
func (s *UserStore) Delete(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
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

func scanUser(row *sql.Row) (ports.User, error) {
	var u ports.User
	var stripeID sql.NullString

	err := row.Scan(
		&u.ID, &u.Email, &u.Name, &stripeID, &u.PlanID, &u.Status, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ports.User{}, ErrNotFound
	}
	if err != nil {
		return ports.User{}, err
	}

	if stripeID.Valid {
		u.StripeID = stripeID.String
	}
	return u, nil
}

func scanUserRows(rows *sql.Rows) (ports.User, error) {
	var u ports.User
	var stripeID sql.NullString

	err := rows.Scan(
		&u.ID, &u.Email, &u.Name, &stripeID, &u.PlanID, &u.Status, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return ports.User{}, err
	}

	if stripeID.Valid {
		u.StripeID = stripeID.String
	}
	return u, nil
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func isUniqueConstraintError(err error) bool {
	return err != nil && (errors.Is(err, sql.ErrNoRows) == false) &&
		(containsString(err.Error(), "UNIQUE constraint failed") ||
			containsString(err.Error(), "unique constraint"))
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Ensure interface compliance.
var _ ports.UserStore = (*UserStore)(nil)
