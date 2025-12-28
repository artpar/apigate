// Package auth provides stateless authentication using JWT.
// Designed for horizontal scaling - no shared state between instances.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the JWT claims for admin authentication.
type Claims struct {
	UserID string `json:"uid"`
	Email  string `json:"email"`
	Role   string `json:"role"` // "admin" or "user"
	jwt.RegisteredClaims
}

// TokenService provides stateless JWT token operations.
// Thread-safe and suitable for concurrent use.
type TokenService struct {
	secret     []byte
	issuer     string
	expiration time.Duration
}

// NewTokenService creates a new JWT token service.
// If secret is empty, a random 32-byte secret is generated.
func NewTokenService(secret string, expiration time.Duration) *TokenService {
	var secretBytes []byte
	if secret == "" {
		secretBytes = make([]byte, 32)
		rand.Read(secretBytes)
	} else {
		secretBytes = []byte(secret)
	}

	if expiration == 0 {
		expiration = 24 * time.Hour
	}

	return &TokenService{
		secret:     secretBytes,
		issuer:     "apigate",
		expiration: expiration,
	}
}

// GenerateToken creates a new JWT token for the given user.
func (s *TokenService) GenerateToken(userID, email, role string) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(s.expiration)

	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, expiresAt, nil
}

// ValidateToken validates a JWT token and returns the claims.
func (s *TokenService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// RefreshToken creates a new token with extended expiration.
func (s *TokenService) RefreshToken(tokenString string) (string, time.Time, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return "", time.Time{}, err
	}

	return s.GenerateToken(claims.UserID, claims.Email, claims.Role)
}

// GenerateSecret generates a random secret suitable for JWT signing.
func GenerateSecret() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
