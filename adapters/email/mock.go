package email

import (
	"context"
	"fmt"
	"sync"

	"github.com/artpar/apigate/ports"
)

// MockSender is a mock email sender for testing.
// It stores sent emails in memory instead of actually sending them.
type MockSender struct {
	mu     sync.Mutex
	emails []SentEmail

	// Config for generating links
	BaseURL string
	AppName string

	// Optional: fail if set
	ShouldFail bool
	FailError  error
}

// SentEmail represents an email that was "sent" (stored in memory).
type SentEmail struct {
	To       string
	Subject  string
	HTMLBody string
	TextBody string
	Type     string // "verification", "password_reset", "welcome", "custom"
	Token    string // For verification and password reset emails
	Name     string // Recipient name
}

// NewMockSender creates a new mock email sender.
func NewMockSender(baseURL, appName string) *MockSender {
	return &MockSender{
		BaseURL: baseURL,
		AppName: appName,
	}
}

// Send stores the email in memory.
func (m *MockSender) Send(ctx context.Context, msg ports.EmailMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ShouldFail {
		if m.FailError != nil {
			return m.FailError
		}
		return fmt.Errorf("mock email send failure")
	}

	m.emails = append(m.emails, SentEmail{
		To:       msg.To,
		Subject:  msg.Subject,
		HTMLBody: msg.HTMLBody,
		TextBody: msg.TextBody,
		Type:     "custom",
	})

	return nil
}

// SendVerification stores a verification email in memory.
func (m *MockSender) SendVerification(ctx context.Context, to, name, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ShouldFail {
		if m.FailError != nil {
			return m.FailError
		}
		return fmt.Errorf("mock email send failure")
	}

	m.emails = append(m.emails, SentEmail{
		To:      to,
		Subject: fmt.Sprintf("Verify your email for %s", m.AppName),
		Type:    "verification",
		Token:   token,
		Name:    name,
	})

	return nil
}

// SendPasswordReset stores a password reset email in memory.
func (m *MockSender) SendPasswordReset(ctx context.Context, to, name, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ShouldFail {
		if m.FailError != nil {
			return m.FailError
		}
		return fmt.Errorf("mock email send failure")
	}

	m.emails = append(m.emails, SentEmail{
		To:      to,
		Subject: fmt.Sprintf("Reset your password for %s", m.AppName),
		Type:    "password_reset",
		Token:   token,
		Name:    name,
	})

	return nil
}

// SendWelcome stores a welcome email in memory.
func (m *MockSender) SendWelcome(ctx context.Context, to, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ShouldFail {
		if m.FailError != nil {
			return m.FailError
		}
		return fmt.Errorf("mock email send failure")
	}

	m.emails = append(m.emails, SentEmail{
		To:      to,
		Subject: fmt.Sprintf("Welcome to %s!", m.AppName),
		Type:    "welcome",
		Name:    name,
	})

	return nil
}

// GetEmails returns all stored emails.
func (m *MockSender) GetEmails() []SentEmail {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]SentEmail, len(m.emails))
	copy(result, m.emails)
	return result
}

// GetLastEmail returns the most recently stored email.
func (m *MockSender) GetLastEmail() (SentEmail, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.emails) == 0 {
		return SentEmail{}, false
	}
	return m.emails[len(m.emails)-1], true
}

// FindByTo finds all emails sent to a specific address.
func (m *MockSender) FindByTo(to string) []SentEmail {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []SentEmail
	for _, e := range m.emails {
		if e.To == to {
			result = append(result, e)
		}
	}
	return result
}

// FindByType finds all emails of a specific type.
func (m *MockSender) FindByType(emailType string) []SentEmail {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []SentEmail
	for _, e := range m.emails {
		if e.Type == emailType {
			result = append(result, e)
		}
	}
	return result
}

// Count returns the number of emails sent.
func (m *MockSender) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.emails)
}

// Clear removes all stored emails.
func (m *MockSender) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.emails = nil
}

// SetShouldFail configures the mock to fail on all send attempts.
func (m *MockSender) SetShouldFail(fail bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ShouldFail = fail
	m.FailError = err
}

// Ensure interface compliance.
var _ ports.EmailSender = (*MockSender)(nil)
