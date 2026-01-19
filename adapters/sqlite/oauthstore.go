package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/artpar/apigate/domain/oauth"
	"github.com/artpar/apigate/ports"
)

// OAuthIdentityStore implements ports.OAuthIdentityStore using SQLite.
type OAuthIdentityStore struct {
	db *DB
}

// NewOAuthIdentityStore creates a new SQLite OAuth identity store.
func NewOAuthIdentityStore(db *DB) *OAuthIdentityStore {
	return &OAuthIdentityStore{db: db}
}

// Get retrieves an identity by ID.
func (s *OAuthIdentityStore) Get(ctx context.Context, id string) (oauth.Identity, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, provider, provider_user_id, email, name, avatar_url,
		       access_token, refresh_token, token_expires_at, raw_data, created_at, updated_at
		FROM oauth_identities
		WHERE id = ?
	`, id)
	return scanOAuthIdentity(row)
}

// GetByProviderUser retrieves an identity by provider and provider user ID.
func (s *OAuthIdentityStore) GetByProviderUser(ctx context.Context, provider oauth.Provider, providerUserID string) (oauth.Identity, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, provider, provider_user_id, email, name, avatar_url,
		       access_token, refresh_token, token_expires_at, raw_data, created_at, updated_at
		FROM oauth_identities
		WHERE provider = ? AND provider_user_id = ?
	`, string(provider), providerUserID)
	return scanOAuthIdentity(row)
}

// Create stores a new identity.
func (s *OAuthIdentityStore) Create(ctx context.Context, identity oauth.Identity) error {
	var rawData []byte
	if identity.RawData != nil {
		var err error
		rawData, err = json.Marshal(identity.RawData)
		if err != nil {
			return err
		}
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO oauth_identities (id, user_id, provider, provider_user_id, email, name, avatar_url,
		                              access_token, refresh_token, token_expires_at, raw_data, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, identity.ID, identity.UserID, string(identity.Provider), identity.ProviderUserID,
		nullStringOAuth(identity.Email), nullStringOAuth(identity.Name), nullStringOAuth(identity.AvatarURL),
		nullStringOAuth(identity.AccessToken), nullStringOAuth(identity.RefreshToken),
		nullTimeVal(identity.TokenExpiresAt), rawData, identity.CreatedAt, identity.UpdatedAt)
	return err
}

// Update modifies an identity.
func (s *OAuthIdentityStore) Update(ctx context.Context, identity oauth.Identity) error {
	var rawData []byte
	if identity.RawData != nil {
		var err error
		rawData, err = json.Marshal(identity.RawData)
		if err != nil {
			return err
		}
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE oauth_identities
		SET email = ?, name = ?, avatar_url = ?, access_token = ?, refresh_token = ?,
		    token_expires_at = ?, raw_data = ?, updated_at = ?
		WHERE id = ?
	`, nullStringOAuth(identity.Email), nullStringOAuth(identity.Name), nullStringOAuth(identity.AvatarURL),
		nullStringOAuth(identity.AccessToken), nullStringOAuth(identity.RefreshToken),
		nullTimeVal(identity.TokenExpiresAt), rawData, time.Now().UTC(), identity.ID)
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

// Delete removes an identity.
func (s *OAuthIdentityStore) Delete(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM oauth_identities WHERE id = ?`, id)
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

// ListByUser returns all identities for a user.
func (s *OAuthIdentityStore) ListByUser(ctx context.Context, userID string) ([]oauth.Identity, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, provider, provider_user_id, email, name, avatar_url,
		       access_token, refresh_token, token_expires_at, raw_data, created_at, updated_at
		FROM oauth_identities
		WHERE user_id = ?
		ORDER BY provider ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanOAuthIdentities(rows)
}

// GetByUserAndProvider retrieves identity for a user from a specific provider.
func (s *OAuthIdentityStore) GetByUserAndProvider(ctx context.Context, userID string, provider oauth.Provider) (oauth.Identity, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, provider, provider_user_id, email, name, avatar_url,
		       access_token, refresh_token, token_expires_at, raw_data, created_at, updated_at
		FROM oauth_identities
		WHERE user_id = ? AND provider = ?
	`, userID, string(provider))
	return scanOAuthIdentity(row)
}

func scanOAuthIdentity(row *sql.Row) (oauth.Identity, error) {
	var identity oauth.Identity
	var provider string
	var email, name, avatarURL, accessToken, refreshToken sql.NullString
	var tokenExpiresAt sql.NullTime
	var rawData []byte

	err := row.Scan(
		&identity.ID, &identity.UserID, &provider, &identity.ProviderUserID,
		&email, &name, &avatarURL, &accessToken, &refreshToken, &tokenExpiresAt,
		&rawData, &identity.CreatedAt, &identity.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return oauth.Identity{}, ErrNotFound
	}
	if err != nil {
		return oauth.Identity{}, err
	}

	identity.Provider = oauth.Provider(provider)
	identity.Email = email.String
	identity.Name = name.String
	identity.AvatarURL = avatarURL.String
	identity.AccessToken = accessToken.String
	identity.RefreshToken = refreshToken.String
	if tokenExpiresAt.Valid {
		identity.TokenExpiresAt = &tokenExpiresAt.Time
	}
	if len(rawData) > 0 {
		json.Unmarshal(rawData, &identity.RawData)
	}

	return identity, nil
}

func scanOAuthIdentities(rows *sql.Rows) ([]oauth.Identity, error) {
	var identities []oauth.Identity
	for rows.Next() {
		var identity oauth.Identity
		var provider string
		var email, name, avatarURL, accessToken, refreshToken sql.NullString
		var tokenExpiresAt sql.NullTime
		var rawData []byte

		err := rows.Scan(
			&identity.ID, &identity.UserID, &provider, &identity.ProviderUserID,
			&email, &name, &avatarURL, &accessToken, &refreshToken, &tokenExpiresAt,
			&rawData, &identity.CreatedAt, &identity.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		identity.Provider = oauth.Provider(provider)
		identity.Email = email.String
		identity.Name = name.String
		identity.AvatarURL = avatarURL.String
		identity.AccessToken = accessToken.String
		identity.RefreshToken = refreshToken.String
		if tokenExpiresAt.Valid {
			identity.TokenExpiresAt = &tokenExpiresAt.Time
		}
		if len(rawData) > 0 {
			json.Unmarshal(rawData, &identity.RawData)
		}

		identities = append(identities, identity)
	}
	return identities, rows.Err()
}

func nullTimeVal(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return *t
}

func nullStringOAuth(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

// Ensure interface compliance.
var _ ports.OAuthIdentityStore = (*OAuthIdentityStore)(nil)

// OAuthStateStore implements ports.OAuthStateStore using SQLite.
// Database-backed for horizontal scaling (stateless servers).
type OAuthStateStore struct {
	db *DB
}

// NewOAuthStateStore creates a new SQLite OAuth state store.
func NewOAuthStateStore(db *DB) *OAuthStateStore {
	return &OAuthStateStore{db: db}
}

// Create stores a new state.
func (s *OAuthStateStore) Create(ctx context.Context, state oauth.State) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO oauth_states (state, provider, redirect_uri, code_verifier, nonce, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, state.State, string(state.Provider), state.RedirectURI, state.CodeVerifier,
		state.Nonce, state.ExpiresAt, state.CreatedAt)
	return err
}

// Get retrieves a state by state string.
func (s *OAuthStateStore) Get(ctx context.Context, state string) (oauth.State, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT state, provider, redirect_uri, code_verifier, nonce, expires_at, created_at
		FROM oauth_states
		WHERE state = ?
	`, state)
	return scanOAuthState(row)
}

// Delete removes a state (after use).
func (s *OAuthStateStore) Delete(ctx context.Context, state string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM oauth_states WHERE state = ?`, state)
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

// DeleteExpired removes all expired states.
func (s *OAuthStateStore) DeleteExpired(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM oauth_states WHERE expires_at < ?
	`, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func scanOAuthState(row *sql.Row) (oauth.State, error) {
	var state oauth.State
	var provider string

	err := row.Scan(
		&state.State, &provider, &state.RedirectURI, &state.CodeVerifier,
		&state.Nonce, &state.ExpiresAt, &state.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return oauth.State{}, ErrNotFound
	}
	if err != nil {
		return oauth.State{}, err
	}

	state.Provider = oauth.Provider(provider)
	return state, nil
}

// Ensure interface compliance.
var _ ports.OAuthStateStore = (*OAuthStateStore)(nil)
