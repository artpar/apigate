package email

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/artpar/apigate/ports"
)

func TestMockSender_Send(t *testing.T) {
	sender := NewMockSender("https://example.com", "TestApp")
	ctx := context.Background()

	msg := ports.EmailMessage{
		To:       "test@example.com",
		Subject:  "Test Subject",
		HTMLBody: "<p>HTML Body</p>",
		TextBody: "Text Body",
	}

	err := sender.Send(ctx, msg)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if sender.Count() != 1 {
		t.Errorf("Count = %d, want 1", sender.Count())
	}

	email, ok := sender.GetLastEmail()
	if !ok {
		t.Fatal("GetLastEmail returned false")
	}

	if email.To != msg.To {
		t.Errorf("To = %s, want %s", email.To, msg.To)
	}
	if email.Subject != msg.Subject {
		t.Errorf("Subject = %s, want %s", email.Subject, msg.Subject)
	}
	if email.HTMLBody != msg.HTMLBody {
		t.Errorf("HTMLBody = %s, want %s", email.HTMLBody, msg.HTMLBody)
	}
	if email.Type != "custom" {
		t.Errorf("Type = %s, want custom", email.Type)
	}
}

func TestMockSender_SendVerification(t *testing.T) {
	sender := NewMockSender("https://example.com", "TestApp")
	ctx := context.Background()

	err := sender.SendVerification(ctx, "test@example.com", "Test User", "abc123")
	if err != nil {
		t.Fatalf("SendVerification failed: %v", err)
	}

	emails := sender.FindByType("verification")
	if len(emails) != 1 {
		t.Fatalf("len = %d, want 1", len(emails))
	}

	email := emails[0]
	if email.To != "test@example.com" {
		t.Errorf("To = %s, want test@example.com", email.To)
	}
	if email.Token != "abc123" {
		t.Errorf("Token = %s, want abc123", email.Token)
	}
	if email.Name != "Test User" {
		t.Errorf("Name = %s, want Test User", email.Name)
	}
	if !strings.Contains(email.Subject, "Verify") {
		t.Errorf("Subject = %s, should contain 'Verify'", email.Subject)
	}
}

func TestMockSender_SendPasswordReset(t *testing.T) {
	sender := NewMockSender("https://example.com", "TestApp")
	ctx := context.Background()

	err := sender.SendPasswordReset(ctx, "reset@example.com", "Reset User", "reset123")
	if err != nil {
		t.Fatalf("SendPasswordReset failed: %v", err)
	}

	emails := sender.FindByType("password_reset")
	if len(emails) != 1 {
		t.Fatalf("len = %d, want 1", len(emails))
	}

	email := emails[0]
	if email.Token != "reset123" {
		t.Errorf("Token = %s, want reset123", email.Token)
	}
	if !strings.Contains(email.Subject, "Reset") {
		t.Errorf("Subject = %s, should contain 'Reset'", email.Subject)
	}
}

func TestMockSender_SendWelcome(t *testing.T) {
	sender := NewMockSender("https://example.com", "TestApp")
	ctx := context.Background()

	err := sender.SendWelcome(ctx, "welcome@example.com", "Welcome User")
	if err != nil {
		t.Fatalf("SendWelcome failed: %v", err)
	}

	emails := sender.FindByType("welcome")
	if len(emails) != 1 {
		t.Fatalf("len = %d, want 1", len(emails))
	}

	email := emails[0]
	if email.Name != "Welcome User" {
		t.Errorf("Name = %s, want Welcome User", email.Name)
	}
	if !strings.Contains(email.Subject, "Welcome") {
		t.Errorf("Subject = %s, should contain 'Welcome'", email.Subject)
	}
}

func TestMockSender_FindByTo(t *testing.T) {
	sender := NewMockSender("https://example.com", "TestApp")
	ctx := context.Background()

	// Send to different addresses
	sender.SendVerification(ctx, "user1@example.com", "User 1", "token1")
	sender.SendPasswordReset(ctx, "user2@example.com", "User 2", "token2")
	sender.SendWelcome(ctx, "user1@example.com", "User 1")

	user1Emails := sender.FindByTo("user1@example.com")
	if len(user1Emails) != 2 {
		t.Errorf("user1 emails = %d, want 2", len(user1Emails))
	}

	user2Emails := sender.FindByTo("user2@example.com")
	if len(user2Emails) != 1 {
		t.Errorf("user2 emails = %d, want 1", len(user2Emails))
	}
}

func TestMockSender_Clear(t *testing.T) {
	sender := NewMockSender("https://example.com", "TestApp")
	ctx := context.Background()

	sender.SendWelcome(ctx, "test@example.com", "Test")
	if sender.Count() != 1 {
		t.Fatalf("Count = %d, want 1", sender.Count())
	}

	sender.Clear()
	if sender.Count() != 0 {
		t.Errorf("Count after Clear = %d, want 0", sender.Count())
	}
}

func TestMockSender_ShouldFail(t *testing.T) {
	sender := NewMockSender("https://example.com", "TestApp")
	ctx := context.Background()

	// Configure to fail
	testErr := fmt.Errorf("network error")
	sender.SetShouldFail(true, testErr)

	err := sender.SendWelcome(ctx, "test@example.com", "Test")
	if err != testErr {
		t.Errorf("err = %v, want %v", err, testErr)
	}

	// Should not have stored the email
	if sender.Count() != 0 {
		t.Errorf("Count = %d, want 0", sender.Count())
	}
}

func TestMockSender_ShouldFail_DefaultError(t *testing.T) {
	sender := NewMockSender("https://example.com", "TestApp")
	ctx := context.Background()

	sender.SetShouldFail(true, nil)

	err := sender.Send(ctx, ports.EmailMessage{To: "test@example.com"})
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "mock email send failure") {
		t.Errorf("err = %v, should contain 'mock email send failure'", err)
	}
}

func TestMockSender_GetEmails(t *testing.T) {
	sender := NewMockSender("https://example.com", "TestApp")
	ctx := context.Background()

	sender.SendVerification(ctx, "a@example.com", "A", "t1")
	sender.SendPasswordReset(ctx, "b@example.com", "B", "t2")
	sender.SendWelcome(ctx, "c@example.com", "C")

	emails := sender.GetEmails()
	if len(emails) != 3 {
		t.Errorf("len = %d, want 3", len(emails))
	}

	// Verify order
	if emails[0].Type != "verification" {
		t.Errorf("emails[0].Type = %s, want verification", emails[0].Type)
	}
	if emails[1].Type != "password_reset" {
		t.Errorf("emails[1].Type = %s, want password_reset", emails[1].Type)
	}
	if emails[2].Type != "welcome" {
		t.Errorf("emails[2].Type = %s, want welcome", emails[2].Type)
	}
}

func TestMockSender_GetLastEmail_Empty(t *testing.T) {
	sender := NewMockSender("https://example.com", "TestApp")

	_, ok := sender.GetLastEmail()
	if ok {
		t.Error("GetLastEmail should return false when no emails sent")
	}
}

// -----------------------------------------------------------------------------
// SMTPSender Template Tests
// -----------------------------------------------------------------------------

func TestSMTPSender_New(t *testing.T) {
	config := DefaultConfig()
	config.BaseURL = "https://example.com"

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	if sender.config.AppName != "APIGate" {
		t.Errorf("AppName = %s, want APIGate", sender.config.AppName)
	}
}

func TestSMTPSender_Templates(t *testing.T) {
	config := DefaultConfig()
	config.BaseURL = "https://example.com"
	config.AppName = "TestApp"

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	// Test that templates are valid
	if sender.verificationTmpl == nil {
		t.Error("verificationTmpl is nil")
	}
	if sender.passwordResetTmpl == nil {
		t.Error("passwordResetTmpl is nil")
	}
	if sender.welcomeTmpl == nil {
		t.Error("welcomeTmpl is nil")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Host != "localhost" {
		t.Errorf("Host = %s, want localhost", config.Host)
	}
	if config.Port != 25 {
		t.Errorf("Port = %d, want 25", config.Port)
	}
	if !config.UseTLS {
		t.Error("UseTLS should be true by default")
	}
	if config.AppName != "APIGate" {
		t.Errorf("AppName = %s, want APIGate", config.AppName)
	}
}

// Note: Actual SMTP sending tests would require a test SMTP server.
// Integration tests with real email sending should be in a separate file
// and skipped in CI unless SMTP_TEST_HOST is set.
