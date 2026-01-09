package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/artpar/apigate/ports"
)

// inviteStore implements ports.InviteStore using SQLite.
type inviteStore struct {
	db *sql.DB
}

// NewInviteStore creates a new SQLite invite store.
func NewInviteStore(db *sql.DB) ports.InviteStore {
	return &inviteStore{db: db}
}

func (s *inviteStore) Create(ctx context.Context, invite ports.AdminInvite) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO admin_invites (id, email, token_hash, created_by, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, invite.ID, invite.Email, invite.TokenHash, invite.CreatedBy, invite.CreatedAt, invite.ExpiresAt)
	return err
}

func (s *inviteStore) GetByTokenHash(ctx context.Context, hash []byte) (ports.AdminInvite, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, email, token_hash, created_by, created_at, expires_at, used_at
		FROM admin_invites
		WHERE token_hash = ?
	`, hash)

	return s.scanRow(row)
}

func (s *inviteStore) List(ctx context.Context, limit, offset int) ([]ports.AdminInvite, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, email, token_hash, created_by, created_at, expires_at, used_at
		FROM admin_invites
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invites []ports.AdminInvite
	for rows.Next() {
		var invite ports.AdminInvite
		var usedAt sql.NullTime
		if err := rows.Scan(
			&invite.ID,
			&invite.Email,
			&invite.TokenHash,
			&invite.CreatedBy,
			&invite.CreatedAt,
			&invite.ExpiresAt,
			&usedAt,
		); err != nil {
			return nil, err
		}
		if usedAt.Valid {
			invite.UsedAt = &usedAt.Time
		}
		invites = append(invites, invite)
	}
	return invites, rows.Err()
}

func (s *inviteStore) MarkUsed(ctx context.Context, id string, usedAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE admin_invites SET used_at = ? WHERE id = ?
	`, usedAt, id)
	return err
}

func (s *inviteStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM admin_invites WHERE id = ?
	`, id)
	return err
}

func (s *inviteStore) DeleteExpired(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM admin_invites
		WHERE expires_at < ? AND used_at IS NULL
	`, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *inviteStore) Count(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM admin_invites`).Scan(&count)
	return count, err
}

func (s *inviteStore) scanRow(row *sql.Row) (ports.AdminInvite, error) {
	var invite ports.AdminInvite
	var usedAt sql.NullTime
	err := row.Scan(
		&invite.ID,
		&invite.Email,
		&invite.TokenHash,
		&invite.CreatedBy,
		&invite.CreatedAt,
		&invite.ExpiresAt,
		&usedAt,
	)
	if usedAt.Valid {
		invite.UsedAt = &usedAt.Time
	}
	return invite, err
}
