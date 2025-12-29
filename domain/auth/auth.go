// Package auth provides authentication value types and pure validation functions.
// This package has NO dependencies on I/O or external packages.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// TokenType identifies the purpose of a token.
type TokenType string

const (
	TokenTypeEmailVerification TokenType = "email_verification"
	TokenTypePasswordReset     TokenType = "password_reset"
)

// Token represents a verification or password reset token (immutable value type).
type Token struct {
	ID        string
	UserID    string
	Email     string    // Email at time of token creation
	Type      TokenType
	Hash      []byte    // bcrypt hash of the token value
	ExpiresAt time.Time
	UsedAt    *time.Time // nil = not used
	CreatedAt time.Time
}

// TokenResult represents the outcome of token generation.
type TokenResult struct {
	Token    Token  // Token to store (with hash)
	RawToken string // Raw token to send to user (only available at creation)
}

// GenerateToken creates a new token of the specified type.
// Returns the raw token (to send to user) and the Token struct (to store).
func GenerateToken(userID, email string, tokenType TokenType, expiresIn time.Duration) TokenResult {
	// Generate 32 random bytes = 64 hex chars
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		panic("crypto/rand failed")
	}

	rawToken := hex.EncodeToString(randomBytes)

	// Generate token ID
	idBytes := make([]byte, 8)
	rand.Read(idBytes)
	tokenID := "tok_" + hex.EncodeToString(idBytes)

	now := time.Now().UTC()
	token := Token{
		ID:        tokenID,
		UserID:    userID,
		Email:     email,
		Type:      tokenType,
		ExpiresAt: now.Add(expiresIn),
		CreatedAt: now,
	}

	return TokenResult{
		Token:    token,
		RawToken: rawToken,
	}
}

// IsExpired returns true if the token has expired.
func (t Token) IsExpired() bool {
	return time.Now().UTC().After(t.ExpiresAt)
}

// IsUsed returns true if the token has been used.
func (t Token) IsUsed() bool {
	return t.UsedAt != nil
}

// IsValid returns true if the token is not expired and not used.
func (t Token) IsValid() bool {
	return !t.IsExpired() && !t.IsUsed()
}

// WithHash returns a copy of the token with the hash set.
func (t Token) WithHash(hash []byte) Token {
	t.Hash = hash
	return t
}

// MarkUsed returns a copy of the token marked as used.
func (t Token) MarkUsed(at time.Time) Token {
	t.UsedAt = &at
	return t
}

// Session represents a user portal session (immutable value type).
type Session struct {
	ID        string
	UserID    string
	Email     string
	IPAddress string
	UserAgent string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// GenerateSession creates a new session.
func GenerateSession(userID, email, ipAddress, userAgent string, expiresIn time.Duration) Session {
	idBytes := make([]byte, 16)
	rand.Read(idBytes)
	sessionID := "sess_" + hex.EncodeToString(idBytes)

	now := time.Now().UTC()
	return Session{
		ID:        sessionID,
		UserID:    userID,
		Email:     email,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		ExpiresAt: now.Add(expiresIn),
		CreatedAt: now,
	}
}

// IsExpired returns true if the session has expired.
func (s Session) IsExpired() bool {
	return time.Now().UTC().After(s.ExpiresAt)
}

// SignupRequest represents a user signup request (value type).
type SignupRequest struct {
	Email    string
	Password string
	Name     string
}

// SignupResult represents the outcome of signup validation.
type SignupResult struct {
	Valid  bool
	Errors map[string]string // field -> error message
}

// ValidateSignup validates a signup request (pure function).
func ValidateSignup(req SignupRequest) SignupResult {
	errors := make(map[string]string)

	// Validate email
	if req.Email == "" {
		errors["email"] = "Email is required"
	} else if !isValidEmail(req.Email) {
		errors["email"] = "Invalid email format"
	}

	// Validate password
	if req.Password == "" {
		errors["password"] = "Password is required"
	} else if len(req.Password) < 8 {
		errors["password"] = "Password must be at least 8 characters"
	} else if !isStrongPassword(req.Password) {
		errors["password"] = "Password must contain uppercase, lowercase, and number"
	}

	// Validate name
	if req.Name == "" {
		errors["name"] = "Name is required"
	} else if len(req.Name) < 2 {
		errors["name"] = "Name must be at least 2 characters"
	} else if len(req.Name) > 100 {
		errors["name"] = "Name must be less than 100 characters"
	}

	return SignupResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// LoginRequest represents a login request (value type).
type LoginRequest struct {
	Email    string
	Password string
}

// LoginResult represents the outcome of login validation.
type LoginResult struct {
	Valid  bool
	Errors map[string]string
}

// ValidateLogin validates a login request (pure function).
func ValidateLogin(req LoginRequest) LoginResult {
	errors := make(map[string]string)

	if req.Email == "" {
		errors["email"] = "Email is required"
	}

	if req.Password == "" {
		errors["password"] = "Password is required"
	}

	return LoginResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// PasswordResetRequest represents a password reset request (value type).
type PasswordResetRequest struct {
	Email string
}

// ValidatePasswordResetRequest validates a password reset request (pure function).
func ValidatePasswordResetRequest(req PasswordResetRequest) (bool, string) {
	if req.Email == "" {
		return false, "Email is required"
	}
	if !isValidEmail(req.Email) {
		return false, "Invalid email format"
	}
	return true, ""
}

// PasswordResetConfirm represents confirming a password reset (value type).
type PasswordResetConfirm struct {
	Token       string
	NewPassword string
}

// ValidatePasswordResetConfirm validates password reset confirmation (pure function).
func ValidatePasswordResetConfirm(req PasswordResetConfirm) SignupResult {
	errors := make(map[string]string)

	if req.Token == "" {
		errors["token"] = "Reset token is required"
	}

	if req.NewPassword == "" {
		errors["password"] = "New password is required"
	} else if len(req.NewPassword) < 8 {
		errors["password"] = "Password must be at least 8 characters"
	} else if !isStrongPassword(req.NewPassword) {
		errors["password"] = "Password must contain uppercase, lowercase, and number"
	}

	return SignupResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// ChangePasswordRequest represents a password change request (value type).
type ChangePasswordRequest struct {
	CurrentPassword string
	NewPassword     string
}

// ValidateChangePassword validates a password change request (pure function).
func ValidateChangePassword(req ChangePasswordRequest) SignupResult {
	errors := make(map[string]string)

	if req.CurrentPassword == "" {
		errors["current_password"] = "Current password is required"
	}

	if req.NewPassword == "" {
		errors["new_password"] = "New password is required"
	} else if len(req.NewPassword) < 8 {
		errors["new_password"] = "Password must be at least 8 characters"
	} else if !isStrongPassword(req.NewPassword) {
		errors["new_password"] = "Password must contain uppercase, lowercase, and number"
	} else if req.NewPassword == req.CurrentPassword {
		errors["new_password"] = "New password must be different from current password"
	}

	return SignupResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// HashToken creates a SHA-256 hash of a raw token for storage/lookup.
// This is used to look up tokens without storing the raw value.
func HashToken(rawToken string) []byte {
	h := sha256.Sum256([]byte(rawToken))
	return h[:]
}

// Helper functions (pure)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func isValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	return emailRegex.MatchString(email)
}

func isStrongPassword(password string) bool {
	var hasUpper, hasLower, hasDigit bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		}
	}
	return hasUpper && hasLower && hasDigit
}

// PasswordStrength returns a score from 0-4 for password strength.
func PasswordStrength(password string) int {
	score := 0

	if len(password) >= 8 {
		score++
	}
	if len(password) >= 12 {
		score++
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			hasSpecial = true
		}
	}

	if hasUpper && hasLower {
		score++
	}
	if hasDigit && hasSpecial {
		score++
	}

	return score
}
