package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/artpar/apigate/domain/settings"
)

// SettingsStore implements ports.SettingsStore using SQLite.
type SettingsStore struct {
	db *DB
}

// NewSettingsStore creates a new settings store.
func NewSettingsStore(db *DB) *SettingsStore {
	return &SettingsStore{db: db}
}

// Get retrieves a single setting by key.
func (s *SettingsStore) Get(ctx context.Context, key string) (settings.Setting, error) {
	var setting settings.Setting
	var updatedAt string

	err := s.db.DB.QueryRowContext(ctx,
		`SELECT key, value, encrypted, updated_at FROM settings WHERE key = ?`,
		key,
	).Scan(&setting.Key, &setting.Value, &setting.Encrypted, &updatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return settings.Setting{}, errors.New("setting not found")
		}
		return settings.Setting{}, err
	}

	setting.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return setting, nil
}

// GetAll retrieves all settings as a map.
func (s *SettingsStore) GetAll(ctx context.Context) (settings.Settings, error) {
	rows, err := s.db.DB.QueryContext(ctx,
		`SELECT key, value FROM settings`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(settings.Settings)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		result[key] = value
	}

	return result, rows.Err()
}

// GetByPrefix retrieves all settings with a given prefix.
func (s *SettingsStore) GetByPrefix(ctx context.Context, prefix string) (settings.Settings, error) {
	rows, err := s.db.DB.QueryContext(ctx,
		`SELECT key, value FROM settings WHERE key LIKE ?`,
		prefix+"%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(settings.Settings)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		result[key] = value
	}

	return result, rows.Err()
}

// Set stores or updates a setting.
func (s *SettingsStore) Set(ctx context.Context, key, value string, encrypted bool) error {
	encryptedInt := 0
	if encrypted {
		encryptedInt = 1
	}

	_, err := s.db.DB.ExecContext(ctx,
		`INSERT INTO settings (key, value, encrypted, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			encrypted = excluded.encrypted,
			updated_at = CURRENT_TIMESTAMP`,
		key, value, encryptedInt,
	)
	return err
}

// SetBatch stores or updates multiple settings.
func (s *SettingsStore) SetBatch(ctx context.Context, batch settings.Settings) error {
	tx, err := s.db.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO settings (key, value, encrypted, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			updated_at = CURRENT_TIMESTAMP`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for key, value := range batch {
		encrypted := 0
		if settings.IsSensitive(key) {
			encrypted = 1
		}
		if _, err := stmt.ExecContext(ctx, key, value, encrypted); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Delete removes a setting.
func (s *SettingsStore) Delete(ctx context.Context, key string) error {
	_, err := s.db.DB.ExecContext(ctx,
		`DELETE FROM settings WHERE key = ?`,
		key,
	)
	return err
}
