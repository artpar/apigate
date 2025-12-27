package remote

import (
	"context"
	"time"

	"github.com/artpar/apigate/domain/key"
	"github.com/artpar/apigate/ports"
)

// KeyStore delegates key operations to an external HTTP service.
// The remote service must implement the Key Validation API.
//
// API Contract:
//
//	POST /keys/validate
//	Request:  {"api_key": "ak_...", "prefix": "ak_abc12345"}
//	Response: {"valid": true, "key": {...}, "reason": ""}
//
//	POST /keys/create
//	Request:  {"key": {...}}
//	Response: {"id": "key-123"}
//
//	POST /keys/{id}/revoke
//	Request:  {"revoked_at": "2024-01-15T12:00:00Z"}
//	Response: {}
//
//	GET /keys/user/{user_id}
//	Response: {"keys": [...]}
type KeyStore struct {
	client *Client
}

// NewKeyStore creates a remote key store.
func NewKeyStore(client *Client) *KeyStore {
	return &KeyStore{client: client}
}

// KeyValidateRequest is the request for key validation.
type KeyValidateRequest struct {
	APIKey string `json:"api_key"`
	Prefix string `json:"prefix"`
}

// KeyValidateResponse is the response for key validation.
type KeyValidateResponse struct {
	Valid  bool       `json:"valid"`
	Key    *RemoteKey `json:"key,omitempty"`
	Reason string     `json:"reason,omitempty"`
}

// RemoteKey represents a key from the remote service.
type RemoteKey struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Hash      []byte     `json:"hash,omitempty"` // Only if using server-side comparison
	Prefix    string     `json:"prefix"`
	Name      string     `json:"name,omitempty"`
	Scopes    []string   `json:"scopes,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	LastUsed  *time.Time `json:"last_used,omitempty"`
}

// Get retrieves keys matching a prefix.
// For remote validation, we call the validation endpoint.
func (s *KeyStore) Get(ctx context.Context, prefix string) ([]key.Key, error) {
	var resp struct {
		Keys []RemoteKey `json:"keys"`
	}

	err := s.client.Request(ctx, "GET", "/keys/prefix/"+prefix, nil, &resp)
	if err != nil {
		if IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	keys := make([]key.Key, len(resp.Keys))
	for i, rk := range resp.Keys {
		keys[i] = toKey(rk)
	}
	return keys, nil
}

// ValidateKey validates a full API key with the remote service.
// This is the preferred method when using remote auth.
func (s *KeyStore) ValidateKey(ctx context.Context, apiKey, prefix string) (key.Key, bool, string, error) {
	req := KeyValidateRequest{
		APIKey: apiKey,
		Prefix: prefix,
	}

	var resp KeyValidateResponse
	err := s.client.Request(ctx, "POST", "/keys/validate", req, &resp)
	if err != nil {
		return key.Key{}, false, "", err
	}

	if !resp.Valid {
		return key.Key{}, false, resp.Reason, nil
	}

	return toKey(*resp.Key), true, "", nil
}

// Create stores a new key.
func (s *KeyStore) Create(ctx context.Context, k key.Key) error {
	req := fromKey(k)
	return s.client.Request(ctx, "POST", "/keys", req, nil)
}

// Revoke marks a key as revoked.
func (s *KeyStore) Revoke(ctx context.Context, id string, at time.Time) error {
	req := map[string]interface{}{
		"revoked_at": at,
	}
	return s.client.Request(ctx, "POST", "/keys/"+id+"/revoke", req, nil)
}

// ListByUser returns all keys for a user.
func (s *KeyStore) ListByUser(ctx context.Context, userID string) ([]key.Key, error) {
	var resp struct {
		Keys []RemoteKey `json:"keys"`
	}

	err := s.client.Request(ctx, "GET", "/keys/user/"+userID, nil, &resp)
	if err != nil {
		return nil, err
	}

	keys := make([]key.Key, len(resp.Keys))
	for i, rk := range resp.Keys {
		keys[i] = toKey(rk)
	}
	return keys, nil
}

// UpdateLastUsed updates the last used timestamp.
func (s *KeyStore) UpdateLastUsed(ctx context.Context, id string, at time.Time) error {
	req := map[string]interface{}{
		"last_used": at,
	}
	return s.client.Request(ctx, "PATCH", "/keys/"+id+"/last-used", req, nil)
}

func toKey(rk RemoteKey) key.Key {
	return key.Key{
		ID:        rk.ID,
		UserID:    rk.UserID,
		Hash:      rk.Hash,
		Prefix:    rk.Prefix,
		Name:      rk.Name,
		Scopes:    rk.Scopes,
		ExpiresAt: rk.ExpiresAt,
		RevokedAt: rk.RevokedAt,
		CreatedAt: rk.CreatedAt,
		LastUsed:  rk.LastUsed,
	}
}

func fromKey(k key.Key) RemoteKey {
	return RemoteKey{
		ID:        k.ID,
		UserID:    k.UserID,
		Hash:      k.Hash,
		Prefix:    k.Prefix,
		Name:      k.Name,
		Scopes:    k.Scopes,
		ExpiresAt: k.ExpiresAt,
		RevokedAt: k.RevokedAt,
		CreatedAt: k.CreatedAt,
		LastUsed:  k.LastUsed,
	}
}

// Ensure interface compliance.
var _ ports.KeyStore = (*KeyStore)(nil)
