package auth

import (
	"testing"
	"time"
)

func TestGenerateToken(t *testing.T) {
	result := GenerateToken("user123", "test@example.com", TokenTypeEmailVerification, time.Hour)

	// Token should have correct fields
	if result.Token.UserID != "user123" {
		t.Errorf("UserID = %s, want user123", result.Token.UserID)
	}
	if result.Token.Email != "test@example.com" {
		t.Errorf("Email = %s, want test@example.com", result.Token.Email)
	}
	if result.Token.Type != TokenTypeEmailVerification {
		t.Errorf("Type = %s, want %s", result.Token.Type, TokenTypeEmailVerification)
	}

	// Token ID should be prefixed
	if len(result.Token.ID) < 20 || result.Token.ID[:4] != "tok_" {
		t.Errorf("Invalid token ID format: %s", result.Token.ID)
	}

	// Raw token should be 64 hex chars
	if len(result.RawToken) != 64 {
		t.Errorf("RawToken length = %d, want 64", len(result.RawToken))
	}

	// Expiry should be in the future
	if !result.Token.ExpiresAt.After(time.Now()) {
		t.Error("Token should expire in the future")
	}

	// Token should not be used
	if result.Token.UsedAt != nil {
		t.Error("New token should not be used")
	}
}

func TestToken_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "not expired",
			expiresAt: time.Now().Add(time.Hour),
			want:      false,
		},
		{
			name:      "expired",
			expiresAt: time.Now().Add(-time.Hour),
			want:      true,
		},
		{
			name:      "just expired",
			expiresAt: time.Now().Add(-time.Second),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := Token{ExpiresAt: tt.expiresAt}
			if got := token.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToken_IsUsed(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name   string
		usedAt *time.Time
		want   bool
	}{
		{
			name:   "not used",
			usedAt: nil,
			want:   false,
		},
		{
			name:   "used",
			usedAt: &now,
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := Token{UsedAt: tt.usedAt}
			if got := token.IsUsed(); got != tt.want {
				t.Errorf("IsUsed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToken_IsValid(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		expiresAt time.Time
		usedAt    *time.Time
		want      bool
	}{
		{
			name:      "valid - not expired, not used",
			expiresAt: time.Now().Add(time.Hour),
			usedAt:    nil,
			want:      true,
		},
		{
			name:      "invalid - expired",
			expiresAt: time.Now().Add(-time.Hour),
			usedAt:    nil,
			want:      false,
		},
		{
			name:      "invalid - used",
			expiresAt: time.Now().Add(time.Hour),
			usedAt:    &now,
			want:      false,
		},
		{
			name:      "invalid - expired and used",
			expiresAt: time.Now().Add(-time.Hour),
			usedAt:    &now,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := Token{ExpiresAt: tt.expiresAt, UsedAt: tt.usedAt}
			if got := token.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToken_WithHash(t *testing.T) {
	token := Token{ID: "tok_123", UserID: "user1"}
	hash := []byte("somehash")

	newToken := token.WithHash(hash)

	// Original should be unchanged
	if token.Hash != nil {
		t.Error("Original token should not have hash")
	}

	// New token should have hash
	if string(newToken.Hash) != "somehash" {
		t.Errorf("NewToken hash = %s, want somehash", newToken.Hash)
	}

	// Other fields should be preserved
	if newToken.ID != "tok_123" {
		t.Errorf("ID not preserved: %s", newToken.ID)
	}
}

func TestToken_MarkUsed(t *testing.T) {
	token := Token{ID: "tok_123"}
	usedAt := time.Now()

	newToken := token.MarkUsed(usedAt)

	// Original should be unchanged
	if token.UsedAt != nil {
		t.Error("Original token should not be marked used")
	}

	// New token should be marked used
	if newToken.UsedAt == nil || !newToken.UsedAt.Equal(usedAt) {
		t.Error("NewToken should be marked used with correct time")
	}
}

func TestGenerateSession(t *testing.T) {
	session := GenerateSession("user123", "test@example.com", "192.168.1.1", "Mozilla/5.0", 24*time.Hour)

	if session.UserID != "user123" {
		t.Errorf("UserID = %s, want user123", session.UserID)
	}
	if session.Email != "test@example.com" {
		t.Errorf("Email = %s, want test@example.com", session.Email)
	}
	if session.IPAddress != "192.168.1.1" {
		t.Errorf("IPAddress = %s, want 192.168.1.1", session.IPAddress)
	}
	if session.UserAgent != "Mozilla/5.0" {
		t.Errorf("UserAgent = %s, want Mozilla/5.0", session.UserAgent)
	}

	// Session ID should be prefixed
	if len(session.ID) < 37 || session.ID[:5] != "sess_" {
		t.Errorf("Invalid session ID format: %s", session.ID)
	}

	// Expiry should be ~24 hours from now
	expectedExpiry := time.Now().Add(24 * time.Hour)
	if session.ExpiresAt.Before(expectedExpiry.Add(-time.Minute)) ||
		session.ExpiresAt.After(expectedExpiry.Add(time.Minute)) {
		t.Errorf("ExpiresAt not approximately 24h from now: %v", session.ExpiresAt)
	}
}

func TestSession_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "not expired",
			expiresAt: time.Now().Add(time.Hour),
			want:      false,
		},
		{
			name:      "expired",
			expiresAt: time.Now().Add(-time.Hour),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := Session{ExpiresAt: tt.expiresAt}
			if got := session.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateSignup(t *testing.T) {
	tests := []struct {
		name       string
		req        SignupRequest
		wantValid  bool
		wantErrors []string // fields that should have errors
	}{
		{
			name: "valid signup",
			req: SignupRequest{
				Email:    "test@example.com",
				Password: "SecurePass123",
				Name:     "John Doe",
			},
			wantValid:  true,
			wantErrors: nil,
		},
		{
			name: "missing email",
			req: SignupRequest{
				Email:    "",
				Password: "SecurePass123",
				Name:     "John Doe",
			},
			wantValid:  false,
			wantErrors: []string{"email"},
		},
		{
			name: "invalid email format",
			req: SignupRequest{
				Email:    "notanemail",
				Password: "SecurePass123",
				Name:     "John Doe",
			},
			wantValid:  false,
			wantErrors: []string{"email"},
		},
		{
			name: "password too short",
			req: SignupRequest{
				Email:    "test@example.com",
				Password: "Short1",
				Name:     "John Doe",
			},
			wantValid:  false,
			wantErrors: []string{"password"},
		},
		{
			name: "password missing uppercase",
			req: SignupRequest{
				Email:    "test@example.com",
				Password: "weakpassword123",
				Name:     "John Doe",
			},
			wantValid:  false,
			wantErrors: []string{"password"},
		},
		{
			name: "password missing lowercase",
			req: SignupRequest{
				Email:    "test@example.com",
				Password: "WEAKPASSWORD123",
				Name:     "John Doe",
			},
			wantValid:  false,
			wantErrors: []string{"password"},
		},
		{
			name: "password missing number",
			req: SignupRequest{
				Email:    "test@example.com",
				Password: "WeakPassword",
				Name:     "John Doe",
			},
			wantValid:  false,
			wantErrors: []string{"password"},
		},
		{
			name: "name too short",
			req: SignupRequest{
				Email:    "test@example.com",
				Password: "SecurePass123",
				Name:     "J",
			},
			wantValid:  false,
			wantErrors: []string{"name"},
		},
		{
			name: "multiple errors",
			req: SignupRequest{
				Email:    "",
				Password: "",
				Name:     "",
			},
			wantValid:  false,
			wantErrors: []string{"email", "password", "name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateSignup(tt.req)

			if result.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v", result.Valid, tt.wantValid)
			}

			for _, field := range tt.wantErrors {
				if _, ok := result.Errors[field]; !ok {
					t.Errorf("Expected error for field %s, got none", field)
				}
			}

			if tt.wantValid && len(result.Errors) > 0 {
				t.Errorf("Expected no errors but got: %v", result.Errors)
			}
		})
	}
}

func TestValidateLogin(t *testing.T) {
	tests := []struct {
		name       string
		req        LoginRequest
		wantValid  bool
		wantErrors []string
	}{
		{
			name: "valid login",
			req: LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			wantValid:  true,
			wantErrors: nil,
		},
		{
			name: "missing email",
			req: LoginRequest{
				Email:    "",
				Password: "password123",
			},
			wantValid:  false,
			wantErrors: []string{"email"},
		},
		{
			name: "missing password",
			req: LoginRequest{
				Email:    "test@example.com",
				Password: "",
			},
			wantValid:  false,
			wantErrors: []string{"password"},
		},
		{
			name: "both missing",
			req: LoginRequest{
				Email:    "",
				Password: "",
			},
			wantValid:  false,
			wantErrors: []string{"email", "password"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateLogin(tt.req)

			if result.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v", result.Valid, tt.wantValid)
			}

			for _, field := range tt.wantErrors {
				if _, ok := result.Errors[field]; !ok {
					t.Errorf("Expected error for field %s", field)
				}
			}
		})
	}
}

func TestValidatePasswordResetRequest(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		wantValid bool
	}{
		{
			name:      "valid email",
			email:     "test@example.com",
			wantValid: true,
		},
		{
			name:      "empty email",
			email:     "",
			wantValid: false,
		},
		{
			name:      "invalid email format",
			email:     "notanemail",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := PasswordResetRequest{Email: tt.email}
			valid, _ := ValidatePasswordResetRequest(req)

			if valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v", valid, tt.wantValid)
			}
		})
	}
}

func TestValidatePasswordResetConfirm(t *testing.T) {
	tests := []struct {
		name       string
		req        PasswordResetConfirm
		wantValid  bool
		wantErrors []string
	}{
		{
			name: "valid reset",
			req: PasswordResetConfirm{
				Token:       "abc123",
				NewPassword: "SecurePass123",
			},
			wantValid:  true,
			wantErrors: nil,
		},
		{
			name: "missing token",
			req: PasswordResetConfirm{
				Token:       "",
				NewPassword: "SecurePass123",
			},
			wantValid:  false,
			wantErrors: []string{"token"},
		},
		{
			name: "weak password",
			req: PasswordResetConfirm{
				Token:       "abc123",
				NewPassword: "weak",
			},
			wantValid:  false,
			wantErrors: []string{"password"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidatePasswordResetConfirm(tt.req)

			if result.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v", result.Valid, tt.wantValid)
			}

			for _, field := range tt.wantErrors {
				if _, ok := result.Errors[field]; !ok {
					t.Errorf("Expected error for field %s", field)
				}
			}
		})
	}
}

func TestValidateChangePassword(t *testing.T) {
	tests := []struct {
		name       string
		req        ChangePasswordRequest
		wantValid  bool
		wantErrors []string
	}{
		{
			name: "valid change",
			req: ChangePasswordRequest{
				CurrentPassword: "OldPass123",
				NewPassword:     "NewPass456",
			},
			wantValid:  true,
			wantErrors: nil,
		},
		{
			name: "missing current password",
			req: ChangePasswordRequest{
				CurrentPassword: "",
				NewPassword:     "NewPass456",
			},
			wantValid:  false,
			wantErrors: []string{"current_password"},
		},
		{
			name: "same password",
			req: ChangePasswordRequest{
				CurrentPassword: "SamePass123",
				NewPassword:     "SamePass123",
			},
			wantValid:  false,
			wantErrors: []string{"new_password"},
		},
		{
			name: "weak new password",
			req: ChangePasswordRequest{
				CurrentPassword: "OldPass123",
				NewPassword:     "weak",
			},
			wantValid:  false,
			wantErrors: []string{"new_password"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateChangePassword(tt.req)

			if result.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v", result.Valid, tt.wantValid)
			}

			for _, field := range tt.wantErrors {
				if _, ok := result.Errors[field]; !ok {
					t.Errorf("Expected error for field %s", field)
				}
			}
		})
	}
}

func TestPasswordStrength(t *testing.T) {
	tests := []struct {
		password string
		minScore int
		maxScore int
	}{
		{"short", 0, 0},
		{"eightchr", 1, 1},
		{"Eightchr", 1, 2},
		{"Eightchr1", 1, 2},
		{"TwelveChars1", 2, 3},
		{"TwelveChars1!", 3, 4},
		{"VerySecure123!", 4, 4},
	}

	for _, tt := range tests {
		t.Run(tt.password, func(t *testing.T) {
			score := PasswordStrength(tt.password)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("PasswordStrength(%s) = %d, want between %d and %d",
					tt.password, score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email string
		valid bool
	}{
		{"test@example.com", true},
		{"user.name@domain.co.uk", true},
		{"user+tag@example.com", true},
		{"", false},
		{"notanemail", false},
		{"@nodomain.com", false},
		{"noat.com", false},
		{"spaces in@email.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			if got := isValidEmail(tt.email); got != tt.valid {
				t.Errorf("isValidEmail(%s) = %v, want %v", tt.email, got, tt.valid)
			}
		})
	}
}

func TestIsStrongPassword(t *testing.T) {
	tests := []struct {
		password string
		strong   bool
	}{
		{"SecurePass123", true},
		{"AbC123", true},
		{"alllowercase123", false},
		{"ALLUPPERCASE123", false},
		{"NoNumbersHere", false},
		{"12345678", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.password, func(t *testing.T) {
			if got := isStrongPassword(tt.password); got != tt.strong {
				t.Errorf("isStrongPassword(%s) = %v, want %v", tt.password, got, tt.strong)
			}
		})
	}
}
