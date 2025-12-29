package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/artpar/apigate/domain/auth"
	"github.com/artpar/apigate/ports"
)

// SessionStore implements ports.SessionStore using SQLite.
type SessionStore struct {
	db *DB
}

// NewSessionStore creates a new SQLite session store.
func NewSessionStore(db *DB) *SessionStore {
	return &SessionStore{db: db}
}

// Create stores a new session.
func (s *SessionStore) Create(ctx context.Context, session auth.Session) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_sessions (id, user_id, email, ip_address, user_agent, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, session.ID, session.UserID, session.Email, session.IPAddress,
		session.UserAgent, session.ExpiresAt, session.CreatedAt)

	return err
}

// Get retrieves a session by ID.
func (s *SessionStore) Get(ctx context.Context, id string) (auth.Session, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, email, ip_address, user_agent, expires_at, created_at
		FROM user_sessions
		WHERE id = ?
	`, id)

	return scanSession(row)
}

// Delete removes a session.
func (s *SessionStore) Delete(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM user_sessions WHERE id = ?
	`, id)
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

// DeleteByUser removes all sessions for a user.
func (s *SessionStore) DeleteByUser(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM user_sessions WHERE user_id = ?
	`, userID)
	return err
}

// DeleteExpired removes all expired sessions.
func (s *SessionStore) DeleteExpired(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM user_sessions WHERE expires_at < ?
	`, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func scanSession(row *sql.Row) (auth.Session, error) {
	var sess auth.Session
	var ipAddress, userAgent sql.NullString

	err := row.Scan(
		&sess.ID, &sess.UserID, &sess.Email, &ipAddress, &userAgent,
		&sess.ExpiresAt, &sess.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return auth.Session{}, ErrNotFound
	}
	if err != nil {
		return auth.Session{}, err
	}

	if ipAddress.Valid {
		sess.IPAddress = ipAddress.String
	}
	if userAgent.Valid {
		sess.UserAgent = userAgent.String
	}

	return sess, nil
}

// Ensure interface compliance.
var _ ports.SessionStore = (*SessionStore)(nil)
