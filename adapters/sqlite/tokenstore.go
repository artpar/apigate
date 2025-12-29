package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/artpar/apigate/domain/auth"
	"github.com/artpar/apigate/ports"
)

// TokenStore implements ports.TokenStore using SQLite.
type TokenStore struct {
	db *DB
}

// NewTokenStore creates a new SQLite token store.
func NewTokenStore(db *DB) *TokenStore {
	return &TokenStore{db: db}
}

// Create stores a new token.
func (s *TokenStore) Create(ctx context.Context, token auth.Token) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO auth_tokens (id, user_id, email, token_type, token_hash, expires_at, used_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, token.ID, token.UserID, token.Email, string(token.Type), token.Hash,
		token.ExpiresAt, token.UsedAt, token.CreatedAt)

	return err
}

// GetByHash retrieves a token by its hash.
func (s *TokenStore) GetByHash(ctx context.Context, hash []byte) (auth.Token, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, email, token_type, token_hash, expires_at, used_at, created_at
		FROM auth_tokens
		WHERE token_hash = ?
	`, hash)

	return scanToken(row)
}

// GetByUserAndType retrieves the latest token for a user of a specific type.
func (s *TokenStore) GetByUserAndType(ctx context.Context, userID string, tokenType auth.TokenType) (auth.Token, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, email, token_type, token_hash, expires_at, used_at, created_at
		FROM auth_tokens
		WHERE user_id = ? AND token_type = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, userID, string(tokenType))

	return scanToken(row)
}

// MarkUsed marks a token as used.
func (s *TokenStore) MarkUsed(ctx context.Context, id string, usedAt time.Time) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE auth_tokens SET used_at = ? WHERE id = ?
	`, usedAt, id)
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

// DeleteExpired removes all expired tokens.
func (s *TokenStore) DeleteExpired(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM auth_tokens WHERE expires_at < ?
	`, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// DeleteByUser removes all tokens for a user.
func (s *TokenStore) DeleteByUser(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM auth_tokens WHERE user_id = ?
	`, userID)
	return err
}

func scanToken(row *sql.Row) (auth.Token, error) {
	var t auth.Token
	var tokenType string
	var usedAt sql.NullTime

	err := row.Scan(
		&t.ID, &t.UserID, &t.Email, &tokenType, &t.Hash,
		&t.ExpiresAt, &usedAt, &t.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return auth.Token{}, ErrNotFound
	}
	if err != nil {
		return auth.Token{}, err
	}

	t.Type = auth.TokenType(tokenType)
	if usedAt.Valid {
		t.UsedAt = &usedAt.Time
	}

	return t, nil
}

// Ensure interface compliance.
var _ ports.TokenStore = (*TokenStore)(nil)
