package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/ports"
)

// KeyStore implements ports.KeyStore using SQLite.
type KeyStore struct {
	db *DB
}

// NewKeyStore creates a new SQLite key store.
func NewKeyStore(db *DB) *KeyStore {
	return &KeyStore{db: db}
}

// Get retrieves keys matching a prefix.
func (s *KeyStore) Get(ctx context.Context, prefix string) ([]key.Key, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, hash, prefix, name, scopes, expires_at, revoked_at, created_at, last_used
		FROM api_keys
		WHERE prefix = ?
	`, prefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []key.Key
	for rows.Next() {
		k, err := scanKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// Create stores a new key.
func (s *KeyStore) Create(ctx context.Context, k key.Key) error {
	scopes, err := json.Marshal(k.Scopes)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO api_keys (id, user_id, hash, prefix, name, scopes, expires_at, revoked_at, created_at, last_used)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, k.ID, k.UserID, k.Hash, k.Prefix, k.Name, string(scopes),
		nullTime(k.ExpiresAt), nullTime(k.RevokedAt), k.CreatedAt, nullTime(k.LastUsed))
	return err
}

// Revoke marks a key as revoked.
func (s *KeyStore) Revoke(ctx context.Context, id string, at time.Time) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE api_keys SET revoked_at = ? WHERE id = ?
	`, at, id)
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

// ListByUser returns all keys for a user.
func (s *KeyStore) ListByUser(ctx context.Context, userID string) ([]key.Key, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, hash, prefix, name, scopes, expires_at, revoked_at, created_at, last_used
		FROM api_keys
		WHERE user_id = ?
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []key.Key
	for rows.Next() {
		k, err := scanKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// UpdateLastUsed updates the last used timestamp.
func (s *KeyStore) UpdateLastUsed(ctx context.Context, id string, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE api_keys SET last_used = ? WHERE id = ?
	`, at, id)
	return err
}

// GetByID retrieves a key by ID.
func (s *KeyStore) GetByID(ctx context.Context, id string) (key.Key, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, hash, prefix, name, scopes, expires_at, revoked_at, created_at, last_used
		FROM api_keys
		WHERE id = ?
	`, id)
	return scanKeyRow(row)
}

// Delete permanently removes a key.
func (s *KeyStore) Delete(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM api_keys WHERE id = ?`, id)
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

func scanKey(rows *sql.Rows) (key.Key, error) {
	var k key.Key
	var scopes string
	var expiresAt, revokedAt, lastUsed sql.NullTime

	err := rows.Scan(
		&k.ID, &k.UserID, &k.Hash, &k.Prefix, &k.Name, &scopes,
		&expiresAt, &revokedAt, &k.CreatedAt, &lastUsed,
	)
	if err != nil {
		return key.Key{}, err
	}

	if scopes != "" && scopes != "null" {
		if err := json.Unmarshal([]byte(scopes), &k.Scopes); err != nil {
			return key.Key{}, err
		}
	}

	if expiresAt.Valid {
		k.ExpiresAt = &expiresAt.Time
	}
	if revokedAt.Valid {
		k.RevokedAt = &revokedAt.Time
	}
	if lastUsed.Valid {
		k.LastUsed = &lastUsed.Time
	}

	return k, nil
}

func scanKeyRow(row *sql.Row) (key.Key, error) {
	var k key.Key
	var scopes string
	var expiresAt, revokedAt, lastUsed sql.NullTime

	err := row.Scan(
		&k.ID, &k.UserID, &k.Hash, &k.Prefix, &k.Name, &scopes,
		&expiresAt, &revokedAt, &k.CreatedAt, &lastUsed,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return key.Key{}, ErrNotFound
	}
	if err != nil {
		return key.Key{}, err
	}

	if scopes != "" && scopes != "null" {
		if err := json.Unmarshal([]byte(scopes), &k.Scopes); err != nil {
			return key.Key{}, err
		}
	}

	if expiresAt.Valid {
		k.ExpiresAt = &expiresAt.Time
	}
	if revokedAt.Valid {
		k.RevokedAt = &revokedAt.Time
	}
	if lastUsed.Valid {
		k.LastUsed = &lastUsed.Time
	}

	return k, nil
}

// nullTime converts a *time.Time to sql.NullTime.
func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// Ensure interface compliance.
var _ ports.KeyStore = (*KeyStore)(nil)
