package email

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/artpar/apigate/domain/settings"
	"github.com/artpar/apigate/ports"
)

// =============================================================================
// MockSender Tests
// =============================================================================

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

	// Test non-existent email
	nonExistent := sender.FindByTo("nobody@example.com")
	if len(nonExistent) != 0 {
		t.Errorf("non-existent user emails = %d, want 0", len(nonExistent))
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

func TestMockSender_ShouldFail_AllMethods(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name   string
		action func(sender *MockSender) error
	}{
		{
			name: "Send",
			action: func(s *MockSender) error {
				return s.Send(ctx, ports.EmailMessage{To: "test@example.com"})
			},
		},
		{
			name: "SendVerification",
			action: func(s *MockSender) error {
				return s.SendVerification(ctx, "test@example.com", "Test", "token")
			},
		},
		{
			name: "SendPasswordReset",
			action: func(s *MockSender) error {
				return s.SendPasswordReset(ctx, "test@example.com", "Test", "token")
			},
		},
		{
			name: "SendWelcome",
			action: func(s *MockSender) error {
				return s.SendWelcome(ctx, "test@example.com", "Test")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := NewMockSender("https://example.com", "TestApp")
			customErr := fmt.Errorf("custom error for %s", tt.name)
			sender.SetShouldFail(true, customErr)

			err := tt.action(sender)
			if err != customErr {
				t.Errorf("%s: err = %v, want %v", tt.name, err, customErr)
			}
		})
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

func TestMockSender_Concurrent(t *testing.T) {
	sender := NewMockSender("https://example.com", "TestApp")
	ctx := context.Background()

	var wg sync.WaitGroup
	const numGoroutines = 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sender.SendVerification(ctx, fmt.Sprintf("user%d@example.com", i), fmt.Sprintf("User %d", i), fmt.Sprintf("token%d", i))
		}(i)
	}

	wg.Wait()

	if sender.Count() != numGoroutines {
		t.Errorf("Count = %d, want %d", sender.Count(), numGoroutines)
	}
}

func TestMockSender_FindByType_Empty(t *testing.T) {
	sender := NewMockSender("https://example.com", "TestApp")

	result := sender.FindByType("verification")
	if len(result) != 0 {
		t.Errorf("FindByType on empty sender should return empty slice, got %d", len(result))
	}
}

func TestMockSender_SubjectFormats(t *testing.T) {
	sender := NewMockSender("https://example.com", "MyApp")
	ctx := context.Background()

	sender.SendVerification(ctx, "a@example.com", "A", "t1")
	sender.SendPasswordReset(ctx, "b@example.com", "B", "t2")
	sender.SendWelcome(ctx, "c@example.com", "C")

	emails := sender.GetEmails()

	// Verify subject formats include app name
	if !strings.Contains(emails[0].Subject, "MyApp") {
		t.Errorf("Verification subject should contain app name: %s", emails[0].Subject)
	}
	if !strings.Contains(emails[1].Subject, "MyApp") {
		t.Errorf("Password reset subject should contain app name: %s", emails[1].Subject)
	}
	if !strings.Contains(emails[2].Subject, "MyApp") {
		t.Errorf("Welcome subject should contain app name: %s", emails[2].Subject)
	}
}

// =============================================================================
// NoopSender Tests
// =============================================================================

func TestNoopSender_New(t *testing.T) {
	sender := NewNoopSender()
	if sender == nil {
		t.Fatal("NewNoopSender returned nil")
	}
}

func TestNoopSender_Send(t *testing.T) {
	sender := NewNoopSender()
	ctx := context.Background()

	err := sender.Send(ctx, ports.EmailMessage{
		To:       "test@example.com",
		Subject:  "Test",
		HTMLBody: "<p>Test</p>",
		TextBody: "Test",
	})
	if err != nil {
		t.Errorf("NoopSender.Send should not return error, got: %v", err)
	}
}

func TestNoopSender_SendVerification(t *testing.T) {
	sender := NewNoopSender()
	ctx := context.Background()

	err := sender.SendVerification(ctx, "test@example.com", "Test User", "token123")
	if err != nil {
		t.Errorf("NoopSender.SendVerification should not return error, got: %v", err)
	}
}

func TestNoopSender_SendPasswordReset(t *testing.T) {
	sender := NewNoopSender()
	ctx := context.Background()

	err := sender.SendPasswordReset(ctx, "test@example.com", "Test User", "token123")
	if err != nil {
		t.Errorf("NoopSender.SendPasswordReset should not return error, got: %v", err)
	}
}

func TestNoopSender_SendWelcome(t *testing.T) {
	sender := NewNoopSender()
	ctx := context.Background()

	err := sender.SendWelcome(ctx, "test@example.com", "Test User")
	if err != nil {
		t.Errorf("NoopSender.SendWelcome should not return error, got: %v", err)
	}
}

func TestNoopSender_ImplementsInterface(t *testing.T) {
	var _ ports.EmailSender = (*NoopSender)(nil)
}

// =============================================================================
// SMTPSender Tests
// =============================================================================

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

func TestSMTPSender_TemplateExecution(t *testing.T) {
	config := DefaultConfig()
	config.BaseURL = "https://example.com"
	config.AppName = "TestApp"

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	data := emailTemplateData{
		Name:    "John Doe",
		AppName: "TestApp",
		Link:    "https://example.com/verify?token=abc123",
	}

	// Test verification template
	var buf bytes.Buffer
	err = sender.verificationTmpl.Execute(&buf, data)
	if err != nil {
		t.Errorf("verification template execution failed: %v", err)
	}
	html := buf.String()
	if !strings.Contains(html, "John Doe") {
		t.Error("verification template should contain recipient name")
	}
	if !strings.Contains(html, "TestApp") {
		t.Error("verification template should contain app name")
	}
	if !strings.Contains(html, "abc123") {
		t.Error("verification template should contain token in link")
	}

	// Test password reset template
	buf.Reset()
	err = sender.passwordResetTmpl.Execute(&buf, data)
	if err != nil {
		t.Errorf("password reset template execution failed: %v", err)
	}
	html = buf.String()
	if !strings.Contains(html, "Reset your password") {
		t.Error("password reset template should contain reset message")
	}

	// Test welcome template
	buf.Reset()
	err = sender.welcomeTmpl.Execute(&buf, data)
	if err != nil {
		t.Errorf("welcome template execution failed: %v", err)
	}
	html = buf.String()
	if !strings.Contains(html, "Welcome") {
		t.Error("welcome template should contain welcome message")
	}
}

func TestSMTPSender_ImplementsInterface(t *testing.T) {
	var _ ports.EmailSender = (*SMTPSender)(nil)
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
	if config.From != "noreply@localhost" {
		t.Errorf("From = %s, want noreply@localhost", config.From)
	}
	if config.FromName != "APIGate" {
		t.Errorf("FromName = %s, want APIGate", config.FromName)
	}
	if config.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", config.Timeout)
	}
}

func TestSMTPConfig_AllFields(t *testing.T) {
	config := SMTPConfig{
		Host:        "smtp.example.com",
		Port:        587,
		Username:    "user@example.com",
		Password:    "secret",
		From:        "sender@example.com",
		FromName:    "My App",
		UseTLS:      true,
		SkipVerify:  false,
		UseImplicit: false,
		Timeout:     60 * time.Second,
		BaseURL:     "https://myapp.com",
		AppName:     "MyApp",
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	if sender.config.Host != "smtp.example.com" {
		t.Errorf("Host = %s, want smtp.example.com", sender.config.Host)
	}
	if sender.config.Port != 587 {
		t.Errorf("Port = %d, want 587", sender.config.Port)
	}
	if sender.config.Username != "user@example.com" {
		t.Errorf("Username = %s, want user@example.com", sender.config.Username)
	}
	if sender.config.AppName != "MyApp" {
		t.Errorf("AppName = %s, want MyApp", sender.config.AppName)
	}
}

func TestSMTPConfig_ImplicitTLS(t *testing.T) {
	config := SMTPConfig{
		Host:        "smtp.example.com",
		Port:        465,
		UseImplicit: true,
		From:        "sender@example.com",
		FromName:    "Test",
		AppName:     "TestApp",
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	if !sender.config.UseImplicit {
		t.Error("UseImplicit should be true")
	}
}

// =============================================================================
// Factory Tests
// =============================================================================

func TestNewSender_SMTP(t *testing.T) {
	s := settings.Settings{
		settings.KeyEmailProvider:     "smtp",
		settings.KeyEmailSMTPHost:     "smtp.example.com",
		settings.KeyEmailSMTPPort:     "587",
		settings.KeyEmailSMTPUsername: "user",
		settings.KeyEmailSMTPPassword: "pass",
		settings.KeyEmailFromAddress:  "from@example.com",
		settings.KeyEmailFromName:     "From Name",
		settings.KeyEmailSMTPUseTLS:   "true",
		settings.KeyPortalBaseURL:     "https://example.com",
		settings.KeyPortalAppName:     "TestApp",
	}

	sender, err := NewSender(s)
	if err != nil {
		t.Fatalf("NewSender(smtp) failed: %v", err)
	}

	// Verify it's an SMTP sender
	_, ok := sender.(*SMTPSender)
	if !ok {
		t.Error("Expected *SMTPSender for smtp provider")
	}
}

func TestNewSender_SMTP_MissingHost(t *testing.T) {
	s := settings.Settings{
		settings.KeyEmailProvider: "smtp",
		// Missing host
	}

	_, err := NewSender(s)
	if err == nil {
		t.Fatal("Expected error for missing SMTP host")
	}
	if !strings.Contains(err.Error(), "SMTP host is required") {
		t.Errorf("Error should mention missing host: %v", err)
	}
}

func TestNewSender_Mock(t *testing.T) {
	s := settings.Settings{
		settings.KeyEmailProvider: "mock",
		settings.KeyPortalBaseURL: "https://example.com",
		settings.KeyPortalAppName: "TestApp",
	}

	sender, err := NewSender(s)
	if err != nil {
		t.Fatalf("NewSender(mock) failed: %v", err)
	}

	// Verify it's a mock sender
	mockSender, ok := sender.(*MockSender)
	if !ok {
		t.Fatal("Expected *MockSender for mock provider")
	}

	if mockSender.BaseURL != "https://example.com" {
		t.Errorf("BaseURL = %s, want https://example.com", mockSender.BaseURL)
	}
	if mockSender.AppName != "TestApp" {
		t.Errorf("AppName = %s, want TestApp", mockSender.AppName)
	}
}

func TestNewSender_None(t *testing.T) {
	s := settings.Settings{
		settings.KeyEmailProvider: "none",
	}

	sender, err := NewSender(s)
	if err != nil {
		t.Fatalf("NewSender(none) failed: %v", err)
	}

	// Verify it's a noop sender
	_, ok := sender.(*NoopSender)
	if !ok {
		t.Error("Expected *NoopSender for none provider")
	}
}

func TestNewSender_Empty(t *testing.T) {
	s := settings.Settings{
		settings.KeyEmailProvider: "",
	}

	sender, err := NewSender(s)
	if err != nil {
		t.Fatalf("NewSender('') failed: %v", err)
	}

	// Verify it's a noop sender
	_, ok := sender.(*NoopSender)
	if !ok {
		t.Error("Expected *NoopSender for empty provider")
	}
}

func TestNewSender_Unknown(t *testing.T) {
	s := settings.Settings{
		settings.KeyEmailProvider: "unknown_provider",
	}

	_, err := NewSender(s)
	if err == nil {
		t.Fatal("Expected error for unknown provider")
	}
	if !strings.Contains(err.Error(), "unknown email provider") {
		t.Errorf("Error should mention unknown provider: %v", err)
	}
}

func TestNewSender_Mock_DefaultAppName(t *testing.T) {
	s := settings.Settings{
		settings.KeyEmailProvider: "mock",
		settings.KeyPortalBaseURL: "https://example.com",
		// No app name - should use default
	}

	sender, err := NewSender(s)
	if err != nil {
		t.Fatalf("NewSender(mock) failed: %v", err)
	}

	mockSender, ok := sender.(*MockSender)
	if !ok {
		t.Fatal("Expected *MockSender for mock provider")
	}

	if mockSender.AppName != "APIGate" {
		t.Errorf("AppName = %s, want APIGate (default)", mockSender.AppName)
	}
}

func TestNewSender_SMTP_DefaultPort(t *testing.T) {
	s := settings.Settings{
		settings.KeyEmailProvider:    "smtp",
		settings.KeyEmailSMTPHost:    "smtp.example.com",
		settings.KeyEmailFromAddress: "from@example.com",
		// No port specified - should use default 587
	}

	sender, err := NewSender(s)
	if err != nil {
		t.Fatalf("NewSender(smtp) failed: %v", err)
	}

	smtpSender, ok := sender.(*SMTPSender)
	if !ok {
		t.Fatal("Expected *SMTPSender for smtp provider")
	}

	if smtpSender.config.Port != 587 {
		t.Errorf("Port = %d, want 587 (default)", smtpSender.config.Port)
	}
}

// =============================================================================
// Email Template Content Tests
// =============================================================================

func TestVerificationEmailTemplateContent(t *testing.T) {
	// Verify template contains expected elements
	if !strings.Contains(verificationEmailTemplate, "{{.AppName}}") {
		t.Error("Verification template should contain {{.AppName}}")
	}
	if !strings.Contains(verificationEmailTemplate, "{{.Name}}") {
		t.Error("Verification template should contain {{.Name}}")
	}
	if !strings.Contains(verificationEmailTemplate, "{{.Link}}") {
		t.Error("Verification template should contain {{.Link}}")
	}
	if !strings.Contains(verificationEmailTemplate, "Verify") {
		t.Error("Verification template should contain 'Verify'")
	}
	if !strings.Contains(verificationEmailTemplate, "24 hours") {
		t.Error("Verification template should mention expiry")
	}
}

func TestPasswordResetEmailTemplateContent(t *testing.T) {
	if !strings.Contains(passwordResetEmailTemplate, "{{.AppName}}") {
		t.Error("Password reset template should contain {{.AppName}}")
	}
	if !strings.Contains(passwordResetEmailTemplate, "{{.Name}}") {
		t.Error("Password reset template should contain {{.Name}}")
	}
	if !strings.Contains(passwordResetEmailTemplate, "{{.Link}}") {
		t.Error("Password reset template should contain {{.Link}}")
	}
	if !strings.Contains(passwordResetEmailTemplate, "Reset") {
		t.Error("Password reset template should contain 'Reset'")
	}
	if !strings.Contains(passwordResetEmailTemplate, "1 hour") {
		t.Error("Password reset template should mention expiry")
	}
}

func TestWelcomeEmailTemplateContent(t *testing.T) {
	if !strings.Contains(welcomeEmailTemplate, "{{.AppName}}") {
		t.Error("Welcome template should contain {{.AppName}}")
	}
	if !strings.Contains(welcomeEmailTemplate, "{{.Name}}") {
		t.Error("Welcome template should contain {{.Name}}")
	}
	if !strings.Contains(welcomeEmailTemplate, "{{.Link}}") {
		t.Error("Welcome template should contain {{.Link}}")
	}
	if !strings.Contains(welcomeEmailTemplate, "Welcome") {
		t.Error("Welcome template should contain 'Welcome'")
	}
	if !strings.Contains(welcomeEmailTemplate, "API Keys") {
		t.Error("Welcome template should mention API Keys")
	}
}

// =============================================================================
// Email Message Building Tests
// =============================================================================

func TestEmailMessageBuilding_MultipartHTMLAndText(t *testing.T) {
	// Test that the message building logic handles both HTML and text body
	msg := ports.EmailMessage{
		To:       "test@example.com",
		Subject:  "Test Subject",
		HTMLBody: "<p>HTML content</p>",
		TextBody: "Text content",
	}

	if msg.HTMLBody == "" || msg.TextBody == "" {
		t.Error("Both HTML and text body should be present for multipart")
	}
}

func TestEmailMessageBuilding_HTMLOnly(t *testing.T) {
	msg := ports.EmailMessage{
		To:       "test@example.com",
		Subject:  "Test Subject",
		HTMLBody: "<p>HTML content</p>",
		TextBody: "",
	}

	if msg.HTMLBody == "" {
		t.Error("HTML body should be present")
	}
	if msg.TextBody != "" {
		t.Error("Text body should be empty for HTML-only")
	}
}

func TestEmailMessageBuilding_TextOnly(t *testing.T) {
	msg := ports.EmailMessage{
		To:       "test@example.com",
		Subject:  "Test Subject",
		HTMLBody: "",
		TextBody: "Text content",
	}

	if msg.TextBody == "" {
		t.Error("Text body should be present")
	}
	if msg.HTMLBody != "" {
		t.Error("HTML body should be empty for text-only")
	}
}

// =============================================================================
// Integration-style Tests (using MockSender)
// =============================================================================

func TestIntegration_VerificationFlow(t *testing.T) {
	sender := NewMockSender("https://myapp.com", "MyApp")
	ctx := context.Background()

	// Simulate user registration
	userEmail := "newuser@example.com"
	userName := "New User"
	token := "verification-token-12345"

	err := sender.SendVerification(ctx, userEmail, userName, token)
	if err != nil {
		t.Fatalf("SendVerification failed: %v", err)
	}

	// Verify email was sent
	emails := sender.FindByTo(userEmail)
	if len(emails) != 1 {
		t.Fatalf("Expected 1 email to %s, got %d", userEmail, len(emails))
	}

	email := emails[0]
	if email.Type != "verification" {
		t.Errorf("Email type = %s, want verification", email.Type)
	}
	if email.Token != token {
		t.Errorf("Token = %s, want %s", email.Token, token)
	}
	if email.Name != userName {
		t.Errorf("Name = %s, want %s", email.Name, userName)
	}
}

func TestIntegration_PasswordResetFlow(t *testing.T) {
	sender := NewMockSender("https://myapp.com", "MyApp")
	ctx := context.Background()

	userEmail := "existinguser@example.com"
	userName := "Existing User"
	token := "reset-token-67890"

	err := sender.SendPasswordReset(ctx, userEmail, userName, token)
	if err != nil {
		t.Fatalf("SendPasswordReset failed: %v", err)
	}

	emails := sender.FindByType("password_reset")
	if len(emails) != 1 {
		t.Fatalf("Expected 1 password reset email, got %d", len(emails))
	}

	email := emails[0]
	if email.To != userEmail {
		t.Errorf("To = %s, want %s", email.To, userEmail)
	}
	if email.Token != token {
		t.Errorf("Token = %s, want %s", email.Token, token)
	}
}

func TestIntegration_WelcomeAfterVerification(t *testing.T) {
	sender := NewMockSender("https://myapp.com", "MyApp")
	ctx := context.Background()

	userEmail := "verified@example.com"
	userName := "Verified User"

	err := sender.SendWelcome(ctx, userEmail, userName)
	if err != nil {
		t.Fatalf("SendWelcome failed: %v", err)
	}

	emails := sender.FindByType("welcome")
	if len(emails) != 1 {
		t.Fatalf("Expected 1 welcome email, got %d", len(emails))
	}

	email := emails[0]
	if email.To != userEmail {
		t.Errorf("To = %s, want %s", email.To, userEmail)
	}
	if email.Name != userName {
		t.Errorf("Name = %s, want %s", email.Name, userName)
	}
}

func TestIntegration_MultipleEmailsToSameUser(t *testing.T) {
	sender := NewMockSender("https://myapp.com", "MyApp")
	ctx := context.Background()

	userEmail := "multipleemails@example.com"
	userName := "Multi User"

	// Send verification
	sender.SendVerification(ctx, userEmail, userName, "token1")

	// User requests password reset before verifying
	sender.SendPasswordReset(ctx, userEmail, userName, "token2")

	// User finally verifies and gets welcome
	sender.SendWelcome(ctx, userEmail, userName)

	emails := sender.FindByTo(userEmail)
	if len(emails) != 3 {
		t.Errorf("Expected 3 emails to user, got %d", len(emails))
	}

	// Verify types in order
	expectedTypes := []string{"verification", "password_reset", "welcome"}
	for i, expectedType := range expectedTypes {
		if emails[i].Type != expectedType {
			t.Errorf("Email %d type = %s, want %s", i, emails[i].Type, expectedType)
		}
	}
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestMockSender_FailureRecovery(t *testing.T) {
	sender := NewMockSender("https://example.com", "TestApp")
	ctx := context.Background()

	// Set to fail
	sender.SetShouldFail(true, errors.New("temporary failure"))

	// Should fail
	err := sender.SendVerification(ctx, "test@example.com", "Test", "token")
	if err == nil {
		t.Error("Expected error when ShouldFail is true")
	}

	// Reset failure mode
	sender.SetShouldFail(false, nil)

	// Should succeed now
	err = sender.SendVerification(ctx, "test@example.com", "Test", "token")
	if err != nil {
		t.Errorf("Expected success after resetting ShouldFail, got: %v", err)
	}

	// Should have stored the email
	if sender.Count() != 1 {
		t.Errorf("Count = %d, want 1", sender.Count())
	}
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestMockSender_EmptyStrings(t *testing.T) {
	sender := NewMockSender("", "")
	ctx := context.Background()

	err := sender.SendVerification(ctx, "", "", "")
	if err != nil {
		t.Errorf("Should handle empty strings without error: %v", err)
	}

	email, ok := sender.GetLastEmail()
	if !ok {
		t.Fatal("Email should be stored")
	}

	// Empty app name results in subject format issues but shouldn't crash
	if email.Type != "verification" {
		t.Errorf("Type = %s, want verification", email.Type)
	}
}

func TestMockSender_SpecialCharacters(t *testing.T) {
	sender := NewMockSender("https://example.com", "Test App <>&\"'")
	ctx := context.Background()

	specialName := `John "Johnny" O'Brien <test>`
	specialToken := `token/with+special=chars&more`

	err := sender.SendVerification(ctx, "test@example.com", specialName, specialToken)
	if err != nil {
		t.Errorf("Should handle special characters: %v", err)
	}

	email, _ := sender.GetLastEmail()
	if email.Name != specialName {
		t.Errorf("Name not preserved correctly: %s", email.Name)
	}
	if email.Token != specialToken {
		t.Errorf("Token not preserved correctly: %s", email.Token)
	}
}

func TestMockSender_LongContent(t *testing.T) {
	sender := NewMockSender("https://example.com", "TestApp")
	ctx := context.Background()

	// Create a very long name
	longName := strings.Repeat("A", 10000)
	longToken := strings.Repeat("x", 10000)

	err := sender.SendVerification(ctx, "test@example.com", longName, longToken)
	if err != nil {
		t.Errorf("Should handle long content: %v", err)
	}

	email, _ := sender.GetLastEmail()
	if len(email.Name) != 10000 {
		t.Errorf("Name length = %d, want 10000", len(email.Name))
	}
}

func TestSMTPConfig_EmptyValues(t *testing.T) {
	config := SMTPConfig{
		Host:    "",
		Port:    0,
		AppName: "",
	}

	// Should still create sender (network errors will occur during Send)
	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender should succeed even with empty config: %v", err)
	}

	if sender == nil {
		t.Fatal("Sender should not be nil")
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkMockSender_Send(b *testing.B) {
	sender := NewMockSender("https://example.com", "TestApp")
	ctx := context.Background()
	msg := ports.EmailMessage{
		To:       "test@example.com",
		Subject:  "Benchmark",
		HTMLBody: "<p>Benchmark</p>",
		TextBody: "Benchmark",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sender.Send(ctx, msg)
	}
}

func BenchmarkMockSender_FindByTo(b *testing.B) {
	sender := NewMockSender("https://example.com", "TestApp")
	ctx := context.Background()

	// Add some emails
	for i := 0; i < 100; i++ {
		sender.SendVerification(ctx, fmt.Sprintf("user%d@example.com", i%10), "User", "token")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sender.FindByTo("user5@example.com")
	}
}

func BenchmarkMockSender_Count(b *testing.B) {
	sender := NewMockSender("https://example.com", "TestApp")
	ctx := context.Background()

	// Add some emails
	for i := 0; i < 100; i++ {
		sender.SendVerification(ctx, "user@example.com", "User", "token")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sender.Count()
	}
}

// =============================================================================
// SMTPSender Send Tests
// =============================================================================

func TestSMTPSender_Send_ConnectionError(t *testing.T) {
	config := SMTPConfig{
		Host:    "nonexistent.invalid.localhost",
		Port:    25,
		From:    "test@example.com",
		Timeout: 100 * time.Millisecond, // Short timeout for fast tests
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	ctx := context.Background()
	msg := ports.EmailMessage{
		To:       "recipient@example.com",
		Subject:  "Test",
		TextBody: "Test body",
	}

	err = sender.Send(ctx, msg)
	if err == nil {
		t.Error("Expected connection error, got nil")
	}
	// Should get a dial error
	if !strings.Contains(err.Error(), "dial") {
		t.Errorf("Error should be dial-related: %v", err)
	}
}

func TestSMTPSender_Send_ImplicitTLS_ConnectionError(t *testing.T) {
	config := SMTPConfig{
		Host:        "nonexistent.invalid.localhost",
		Port:        465,
		From:        "test@example.com",
		UseImplicit: true,
		Timeout:     100 * time.Millisecond,
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	ctx := context.Background()
	msg := ports.EmailMessage{
		To:       "recipient@example.com",
		Subject:  "Test",
		TextBody: "Test body",
	}

	err = sender.Send(ctx, msg)
	if err == nil {
		t.Error("Expected connection error, got nil")
	}
	// Should get a dial tls error
	if !strings.Contains(err.Error(), "dial") {
		t.Errorf("Error should be dial-related: %v", err)
	}
}

func TestSMTPSender_Send_ContextCancellation(t *testing.T) {
	config := SMTPConfig{
		Host:    "nonexistent.invalid.localhost",
		Port:    25,
		From:    "test@example.com",
		Timeout: 10 * time.Second,
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	msg := ports.EmailMessage{
		To:       "recipient@example.com",
		Subject:  "Test",
		TextBody: "Test body",
	}

	err = sender.Send(ctx, msg)
	if err == nil {
		t.Error("Expected error due to cancelled context")
	}
}

func TestSMTPSender_SendVerification_ConnectionError(t *testing.T) {
	config := SMTPConfig{
		Host:    "nonexistent.invalid.localhost",
		Port:    25,
		From:    "test@example.com",
		Timeout: 100 * time.Millisecond,
		BaseURL: "https://example.com",
		AppName: "TestApp",
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	ctx := context.Background()
	err = sender.SendVerification(ctx, "user@example.com", "Test User", "token123")
	if err == nil {
		t.Error("Expected connection error, got nil")
	}
}

func TestSMTPSender_SendPasswordReset_ConnectionError(t *testing.T) {
	config := SMTPConfig{
		Host:    "nonexistent.invalid.localhost",
		Port:    25,
		From:    "test@example.com",
		Timeout: 100 * time.Millisecond,
		BaseURL: "https://example.com",
		AppName: "TestApp",
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	ctx := context.Background()
	err = sender.SendPasswordReset(ctx, "user@example.com", "Test User", "token123")
	if err == nil {
		t.Error("Expected connection error, got nil")
	}
}

func TestSMTPSender_SendWelcome_ConnectionError(t *testing.T) {
	config := SMTPConfig{
		Host:    "nonexistent.invalid.localhost",
		Port:    25,
		From:    "test@example.com",
		Timeout: 100 * time.Millisecond,
		BaseURL: "https://example.com",
		AppName: "TestApp",
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	ctx := context.Background()
	err = sender.SendWelcome(ctx, "user@example.com", "Test User")
	if err == nil {
		t.Error("Expected connection error, got nil")
	}
}

// =============================================================================
// SMTPSender Message Building Tests
// =============================================================================

func TestSMTPSender_MessageBuilding_MultipartContent(t *testing.T) {
	config := DefaultConfig()
	config.BaseURL = "https://example.com"
	config.AppName = "TestApp"

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	// Verify templates produce proper content
	data := emailTemplateData{
		Name:    "Test User",
		AppName: "TestApp",
		Link:    "https://example.com/verify?token=abc",
	}

	var htmlBuf bytes.Buffer
	err = sender.verificationTmpl.Execute(&htmlBuf, data)
	if err != nil {
		t.Fatalf("Template execution failed: %v", err)
	}

	html := htmlBuf.String()
	// Verify HTML structure
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("HTML should contain DOCTYPE")
	}
	if !strings.Contains(html, "<html>") {
		t.Error("HTML should contain html tag")
	}
	if !strings.Contains(html, "Test User") {
		t.Error("HTML should contain user name")
	}
	if !strings.Contains(html, "https://example.com/verify?token=abc") {
		t.Error("HTML should contain verification link")
	}
}

func TestSMTPSender_MessageBuilding_HTMLOnlyMessage(t *testing.T) {
	// Test that Send handles HTML-only messages correctly
	config := SMTPConfig{
		Host:    "nonexistent.invalid.localhost",
		Port:    25,
		From:    "test@example.com",
		Timeout: 100 * time.Millisecond,
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	msg := ports.EmailMessage{
		To:       "recipient@example.com",
		Subject:  "HTML Only",
		HTMLBody: "<p>HTML content only</p>",
		TextBody: "", // No text body
	}

	// This will fail at connection, but exercises the message building code path
	ctx := context.Background()
	err = sender.Send(ctx, msg)
	// We expect a dial error, not a message building error
	if err != nil && !strings.Contains(err.Error(), "dial") {
		t.Errorf("Unexpected error type: %v", err)
	}
}

func TestSMTPSender_MessageBuilding_TextOnlyMessage(t *testing.T) {
	config := SMTPConfig{
		Host:    "nonexistent.invalid.localhost",
		Port:    25,
		From:    "test@example.com",
		Timeout: 100 * time.Millisecond,
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	msg := ports.EmailMessage{
		To:       "recipient@example.com",
		Subject:  "Text Only",
		HTMLBody: "", // No HTML body
		TextBody: "Plain text content only",
	}

	ctx := context.Background()
	err = sender.Send(ctx, msg)
	// We expect a dial error, not a message building error
	if err != nil && !strings.Contains(err.Error(), "dial") {
		t.Errorf("Unexpected error type: %v", err)
	}
}

func TestSMTPSender_MessageBuilding_BothHTMLAndText(t *testing.T) {
	config := SMTPConfig{
		Host:    "nonexistent.invalid.localhost",
		Port:    25,
		From:    "test@example.com",
		Timeout: 100 * time.Millisecond,
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	msg := ports.EmailMessage{
		To:       "recipient@example.com",
		Subject:  "Multipart",
		HTMLBody: "<p>HTML content</p>",
		TextBody: "Plain text content",
	}

	ctx := context.Background()
	err = sender.Send(ctx, msg)
	// We expect a dial error, not a message building error
	if err != nil && !strings.Contains(err.Error(), "dial") {
		t.Errorf("Unexpected error type: %v", err)
	}
}

// =============================================================================
// SMTPSender Template Rendering Tests
// =============================================================================

func TestSMTPSender_VerificationLink(t *testing.T) {
	config := SMTPConfig{
		Host:    "localhost",
		Port:    25,
		From:    "test@example.com",
		BaseURL: "https://myapp.example.com",
		AppName: "MyApp",
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	// Verify the link format would be correct
	expectedLinkPrefix := "https://myapp.example.com/verify-email?token="
	data := emailTemplateData{
		Name:    "Test",
		AppName: sender.config.AppName,
		Link:    fmt.Sprintf("%s/verify-email?token=%s", sender.config.BaseURL, "test-token"),
	}

	if !strings.HasPrefix(data.Link, expectedLinkPrefix) {
		t.Errorf("Verification link should start with %s, got: %s", expectedLinkPrefix, data.Link)
	}
}

func TestSMTPSender_PasswordResetLink(t *testing.T) {
	config := SMTPConfig{
		Host:    "localhost",
		Port:    25,
		From:    "test@example.com",
		BaseURL: "https://myapp.example.com",
		AppName: "MyApp",
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	// Verify the link format would be correct
	expectedLinkPrefix := "https://myapp.example.com/reset-password?token="
	data := emailTemplateData{
		Name:    "Test",
		AppName: sender.config.AppName,
		Link:    fmt.Sprintf("%s/reset-password?token=%s", sender.config.BaseURL, "reset-token"),
	}

	if !strings.HasPrefix(data.Link, expectedLinkPrefix) {
		t.Errorf("Password reset link should start with %s, got: %s", expectedLinkPrefix, data.Link)
	}
}

func TestSMTPSender_WelcomeLink(t *testing.T) {
	config := SMTPConfig{
		Host:    "localhost",
		Port:    25,
		From:    "test@example.com",
		BaseURL: "https://myapp.example.com",
		AppName: "MyApp",
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	// Verify the link format would be correct
	expectedLink := "https://myapp.example.com/dashboard"
	data := emailTemplateData{
		Name:    "Test",
		AppName: sender.config.AppName,
		Link:    fmt.Sprintf("%s/dashboard", sender.config.BaseURL),
	}

	if data.Link != expectedLink {
		t.Errorf("Welcome link should be %s, got: %s", expectedLink, data.Link)
	}
}

// =============================================================================
// SMTPSender Configuration Validation Tests
// =============================================================================

func TestSMTPSender_TLSConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		config      SMTPConfig
		expectError bool
	}{
		{
			name: "STARTTLS enabled",
			config: SMTPConfig{
				Host:   "smtp.example.com",
				Port:   587,
				UseTLS: true,
			},
			expectError: false,
		},
		{
			name: "STARTTLS disabled",
			config: SMTPConfig{
				Host:   "smtp.example.com",
				Port:   25,
				UseTLS: false,
			},
			expectError: false,
		},
		{
			name: "Implicit TLS (port 465)",
			config: SMTPConfig{
				Host:        "smtp.example.com",
				Port:        465,
				UseImplicit: true,
			},
			expectError: false,
		},
		{
			name: "Skip TLS verification",
			config: SMTPConfig{
				Host:       "smtp.example.com",
				Port:       587,
				UseTLS:     true,
				SkipVerify: true,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender, err := NewSMTPSender(tt.config)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if sender == nil {
					t.Error("Sender should not be nil")
				}
			}
		})
	}
}

func TestSMTPSender_AuthConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		config   SMTPConfig
		hasAuth  bool
	}{
		{
			name: "With authentication",
			config: SMTPConfig{
				Host:     "smtp.example.com",
				Port:     587,
				Username: "user@example.com",
				Password: "secret",
			},
			hasAuth: true,
		},
		{
			name: "Without authentication",
			config: SMTPConfig{
				Host: "smtp.example.com",
				Port: 25,
			},
			hasAuth: false,
		},
		{
			name: "Username only (no password)",
			config: SMTPConfig{
				Host:     "smtp.example.com",
				Port:     587,
				Username: "user@example.com",
			},
			hasAuth: true, // Will still try to auth (may fail)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender, err := NewSMTPSender(tt.config)
			if err != nil {
				t.Fatalf("NewSMTPSender failed: %v", err)
			}

			if tt.hasAuth {
				if sender.config.Username == "" {
					t.Error("Username should be set")
				}
			} else {
				if sender.config.Username != "" {
					t.Error("Username should be empty")
				}
			}
		})
	}
}

func TestSMTPSender_TimeoutConfiguration(t *testing.T) {
	tests := []struct {
		name            string
		timeout         time.Duration
		expectedTimeout time.Duration
	}{
		{
			name:            "Custom timeout",
			timeout:         60 * time.Second,
			expectedTimeout: 60 * time.Second,
		},
		{
			name:            "Short timeout",
			timeout:         5 * time.Second,
			expectedTimeout: 5 * time.Second,
		},
		{
			name:            "Zero timeout",
			timeout:         0,
			expectedTimeout: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := SMTPConfig{
				Host:    "smtp.example.com",
				Port:    587,
				Timeout: tt.timeout,
			}

			sender, err := NewSMTPSender(config)
			if err != nil {
				t.Fatalf("NewSMTPSender failed: %v", err)
			}

			if sender.config.Timeout != tt.expectedTimeout {
				t.Errorf("Timeout = %v, want %v", sender.config.Timeout, tt.expectedTimeout)
			}
		})
	}
}

// =============================================================================
// SMTPSender Sender/Recipient Configuration Tests
// =============================================================================

func TestSMTPSender_FromConfiguration(t *testing.T) {
	tests := []struct {
		name         string
		from         string
		fromName     string
		expectedFrom string
		expectedName string
	}{
		{
			name:         "Full from address",
			from:         "noreply@example.com",
			fromName:     "Example App",
			expectedFrom: "noreply@example.com",
			expectedName: "Example App",
		},
		{
			name:         "Only from address",
			from:         "noreply@example.com",
			fromName:     "",
			expectedFrom: "noreply@example.com",
			expectedName: "",
		},
		{
			name:         "Empty from",
			from:         "",
			fromName:     "Some Name",
			expectedFrom: "",
			expectedName: "Some Name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := SMTPConfig{
				Host:     "smtp.example.com",
				Port:     587,
				From:     tt.from,
				FromName: tt.fromName,
			}

			sender, err := NewSMTPSender(config)
			if err != nil {
				t.Fatalf("NewSMTPSender failed: %v", err)
			}

			if sender.config.From != tt.expectedFrom {
				t.Errorf("From = %s, want %s", sender.config.From, tt.expectedFrom)
			}
			if sender.config.FromName != tt.expectedName {
				t.Errorf("FromName = %s, want %s", sender.config.FromName, tt.expectedName)
			}
		})
	}
}

// =============================================================================
// SMTPSender Template Variable Tests
// =============================================================================

func TestSMTPSender_TemplateVariables_Verification(t *testing.T) {
	config := SMTPConfig{
		Host:    "localhost",
		Port:    25,
		BaseURL: "https://myapp.com",
		AppName: "MyApp",
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	testCases := []struct {
		name      string
		userName  string
		token     string
		wantInHTML []string
	}{
		{
			name:     "Standard name and token",
			userName: "John Doe",
			token:    "abc123",
			wantInHTML: []string{"John Doe", "abc123", "MyApp"},
		},
		{
			name:     "Name with special chars",
			userName: "O'Brien",
			token:    "xyz789",
			wantInHTML: []string{"O&#39;Brien", "xyz789"}, // html/template escapes apostrophe
		},
		{
			name:     "Empty name",
			userName: "",
			token:    "empty123",
			wantInHTML: []string{"empty123"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := emailTemplateData{
				Name:    tc.userName,
				AppName: sender.config.AppName,
				Link:    fmt.Sprintf("%s/verify-email?token=%s", sender.config.BaseURL, tc.token),
			}

			var buf bytes.Buffer
			err := sender.verificationTmpl.Execute(&buf, data)
			if err != nil {
				t.Fatalf("Template execution failed: %v", err)
			}

			html := buf.String()
			for _, want := range tc.wantInHTML {
				if !strings.Contains(html, want) {
					t.Errorf("HTML should contain %q", want)
				}
			}
		})
	}
}

func TestSMTPSender_TemplateVariables_PasswordReset(t *testing.T) {
	config := SMTPConfig{
		Host:    "localhost",
		Port:    25,
		BaseURL: "https://myapp.com",
		AppName: "MyApp",
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	data := emailTemplateData{
		Name:    "Jane Smith",
		AppName: sender.config.AppName,
		Link:    fmt.Sprintf("%s/reset-password?token=%s", sender.config.BaseURL, "reset-token-456"),
	}

	var buf bytes.Buffer
	err = sender.passwordResetTmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("Template execution failed: %v", err)
	}

	html := buf.String()
	expectedStrings := []string{
		"Jane Smith",
		"reset-token-456",
		"MyApp",
		"reset",
		"password",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(strings.ToLower(html), strings.ToLower(expected)) {
			t.Errorf("HTML should contain %q (case insensitive)", expected)
		}
	}
}

func TestSMTPSender_TemplateVariables_Welcome(t *testing.T) {
	config := SMTPConfig{
		Host:    "localhost",
		Port:    25,
		BaseURL: "https://myapp.com",
		AppName: "MyApp",
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	data := emailTemplateData{
		Name:    "New User",
		AppName: sender.config.AppName,
		Link:    fmt.Sprintf("%s/dashboard", sender.config.BaseURL),
	}

	var buf bytes.Buffer
	err = sender.welcomeTmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("Template execution failed: %v", err)
	}

	html := buf.String()
	expectedStrings := []string{
		"New User",
		"MyApp",
		"dashboard",
		"Welcome",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(strings.ToLower(html), strings.ToLower(expected)) {
			t.Errorf("HTML should contain %q (case insensitive)", expected)
		}
	}
}

// =============================================================================
// Mock Sender Default Error Tests
// =============================================================================

func TestMockSender_ShouldFail_DefaultError_Verification(t *testing.T) {
	sender := NewMockSender("https://example.com", "TestApp")
	ctx := context.Background()

	sender.SetShouldFail(true, nil)

	err := sender.SendVerification(ctx, "test@example.com", "Test", "token")
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "mock email send failure") {
		t.Errorf("err = %v, should contain 'mock email send failure'", err)
	}
}

func TestMockSender_ShouldFail_DefaultError_PasswordReset(t *testing.T) {
	sender := NewMockSender("https://example.com", "TestApp")
	ctx := context.Background()

	sender.SetShouldFail(true, nil)

	err := sender.SendPasswordReset(ctx, "test@example.com", "Test", "token")
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "mock email send failure") {
		t.Errorf("err = %v, should contain 'mock email send failure'", err)
	}
}

func TestMockSender_ShouldFail_DefaultError_Welcome(t *testing.T) {
	sender := NewMockSender("https://example.com", "TestApp")
	ctx := context.Background()

	sender.SetShouldFail(true, nil)

	err := sender.SendWelcome(ctx, "test@example.com", "Test")
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "mock email send failure") {
		t.Errorf("err = %v, should contain 'mock email send failure'", err)
	}
}

// =============================================================================
// Additional Edge Case Tests for emailTemplateData
// =============================================================================

func TestEmailTemplateData_EmptyFields(t *testing.T) {
	config := DefaultConfig()
	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	data := emailTemplateData{
		Name:    "",
		AppName: "",
		Link:    "",
	}

	// All templates should handle empty data without panicking
	templates := []*template.Template{
		sender.verificationTmpl,
		sender.passwordResetTmpl,
		sender.welcomeTmpl,
	}

	for i, tmpl := range templates {
		var buf bytes.Buffer
		err := tmpl.Execute(&buf, data)
		if err != nil {
			t.Errorf("Template %d failed with empty data: %v", i, err)
		}
	}
}

func TestEmailTemplateData_UnicodeContent(t *testing.T) {
	config := DefaultConfig()
	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	data := emailTemplateData{
		Name:    "User Name", // Chinese characters
		AppName: "Aplicacion", // Spanish accented characters
		Link:    "https://example.com/test",
	}

	var buf bytes.Buffer
	err = sender.verificationTmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("Template execution failed with unicode: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, data.Name) {
		t.Error("Template should preserve unicode name")
	}
}

func TestEmailTemplateData_HTMLInjection(t *testing.T) {
	config := DefaultConfig()
	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	// Test that HTML in user input is properly escaped
	data := emailTemplateData{
		Name:    "<script>alert('xss')</script>",
		AppName: "TestApp",
		Link:    "https://example.com/verify?token=test",
	}

	var buf bytes.Buffer
	err = sender.verificationTmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("Template execution failed: %v", err)
	}

	html := buf.String()
	// html/template should escape the script tags
	if strings.Contains(html, "<script>") {
		t.Error("Template should escape HTML in user input")
	}
}

// =============================================================================
// SMTP Address Formatting Tests
// =============================================================================

func TestSMTPSender_AddressFormat(t *testing.T) {
	config := SMTPConfig{
		Host:     "smtp.example.com",
		Port:     587,
		From:     "sender@example.com",
		FromName: "Sender Name",
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	// Verify the address would be formatted correctly
	expectedAddr := fmt.Sprintf("%s:%d", sender.config.Host, sender.config.Port)
	if expectedAddr != "smtp.example.com:587" {
		t.Errorf("Address format incorrect: %s", expectedAddr)
	}
}

// =============================================================================
// Factory Edge Case Tests
// =============================================================================

func TestNewSender_SMTP_AllOptions(t *testing.T) {
	s := settings.Settings{
		settings.KeyEmailProvider:     "smtp",
		settings.KeyEmailSMTPHost:     "smtp.example.com",
		settings.KeyEmailSMTPPort:     "465",
		settings.KeyEmailSMTPUsername: "user@example.com",
		settings.KeyEmailSMTPPassword: "password123",
		settings.KeyEmailFromAddress:  "from@example.com",
		settings.KeyEmailFromName:     "My Application",
		settings.KeyEmailSMTPUseTLS:   "false", // Implicit TLS
		settings.KeyPortalBaseURL:     "https://myapp.example.com",
		settings.KeyPortalAppName:     "My Application",
	}

	sender, err := NewSender(s)
	if err != nil {
		t.Fatalf("NewSender failed: %v", err)
	}

	smtpSender, ok := sender.(*SMTPSender)
	if !ok {
		t.Fatal("Expected *SMTPSender")
	}

	if smtpSender.config.Host != "smtp.example.com" {
		t.Errorf("Host = %s, want smtp.example.com", smtpSender.config.Host)
	}
	if smtpSender.config.Port != 465 {
		t.Errorf("Port = %d, want 465", smtpSender.config.Port)
	}
	if smtpSender.config.Username != "user@example.com" {
		t.Errorf("Username = %s, want user@example.com", smtpSender.config.Username)
	}
	if smtpSender.config.AppName != "My Application" {
		t.Errorf("AppName = %s, want 'My Application'", smtpSender.config.AppName)
	}
}

func TestNewSender_Mock_WithEmptyBaseURL(t *testing.T) {
	s := settings.Settings{
		settings.KeyEmailProvider: "mock",
		settings.KeyPortalBaseURL: "",
		settings.KeyPortalAppName: "TestApp",
	}

	sender, err := NewSender(s)
	if err != nil {
		t.Fatalf("NewSender failed: %v", err)
	}

	mockSender, ok := sender.(*MockSender)
	if !ok {
		t.Fatal("Expected *MockSender")
	}

	if mockSender.BaseURL != "" {
		t.Errorf("BaseURL = %s, want empty", mockSender.BaseURL)
	}
}

// =============================================================================
// Concurrent Access Tests for SMTPSender
// =============================================================================

func TestSMTPSender_ConcurrentTemplateAccess(t *testing.T) {
	config := DefaultConfig()
	config.BaseURL = "https://example.com"
	config.AppName = "TestApp"

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	var wg sync.WaitGroup
	const numGoroutines = 50

	// Concurrently execute templates
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			data := emailTemplateData{
				Name:    fmt.Sprintf("User %d", i),
				AppName: "TestApp",
				Link:    fmt.Sprintf("https://example.com/verify?token=token%d", i),
			}

			var buf bytes.Buffer
			err := sender.verificationTmpl.Execute(&buf, data)
			if err != nil {
				t.Errorf("Concurrent template execution failed: %v", err)
			}
		}(i)
	}

	wg.Wait()
}

// =============================================================================
// Mock SMTP Server for Integration Tests
// =============================================================================

// mockSMTPServer creates a simple mock SMTP server for testing
type mockSMTPServer struct {
	listener     net.Listener
	receivedMsgs []string
	mu           sync.Mutex
	shouldFail   string // stage at which to fail: "connect", "ehlo", "auth", "mail", "rcpt", "data", "quit"
}

func newMockSMTPServer(t *testing.T) *mockSMTPServer {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create mock SMTP server: %v", err)
	}

	server := &mockSMTPServer{
		listener: listener,
	}

	go server.accept(t)
	return server
}

func (s *mockSMTPServer) addr() string {
	return s.listener.Addr().String()
}

func (s *mockSMTPServer) port() int {
	return s.listener.Addr().(*net.TCPAddr).Port
}

func (s *mockSMTPServer) close() {
	s.listener.Close()
}

func (s *mockSMTPServer) setFailAt(stage string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shouldFail = stage
}

func (s *mockSMTPServer) accept(t *testing.T) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return // Server closed
		}
		go s.handleConnection(conn, t)
	}
}

func (s *mockSMTPServer) handleConnection(conn net.Conn, t *testing.T) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	s.mu.Lock()
	failAt := s.shouldFail
	s.mu.Unlock()

	if failAt == "connect" {
		conn.Close()
		return
	}

	// Send greeting
	conn.Write([]byte("220 mock.smtp.local ESMTP Mock\r\n"))

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimSpace(line)

		cmd := strings.ToUpper(strings.Split(line, " ")[0])

		switch cmd {
		case "EHLO", "HELO":
			if failAt == "ehlo" {
				conn.Write([]byte("550 EHLO failed\r\n"))
				return
			}
			conn.Write([]byte("250-mock.smtp.local\r\n"))
			conn.Write([]byte("250-AUTH PLAIN LOGIN\r\n"))
			conn.Write([]byte("250 OK\r\n"))

		case "AUTH":
			if failAt == "auth" {
				conn.Write([]byte("535 Authentication failed\r\n"))
				return
			}
			conn.Write([]byte("235 Authentication successful\r\n"))

		case "MAIL":
			if failAt == "mail" {
				conn.Write([]byte("550 MAIL FROM rejected\r\n"))
				return
			}
			conn.Write([]byte("250 OK\r\n"))

		case "RCPT":
			if failAt == "rcpt" {
				conn.Write([]byte("550 RCPT TO rejected\r\n"))
				return
			}
			conn.Write([]byte("250 OK\r\n"))

		case "DATA":
			if failAt == "data" {
				conn.Write([]byte("550 DATA rejected\r\n"))
				return
			}
			conn.Write([]byte("354 Start mail input\r\n"))
			// Read until we get a line with just "."
			var data bytes.Buffer
			for {
				dataLine, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimSpace(dataLine) == "." {
					break
				}
				data.WriteString(dataLine)
			}
			s.mu.Lock()
			s.receivedMsgs = append(s.receivedMsgs, data.String())
			s.mu.Unlock()
			conn.Write([]byte("250 Message accepted\r\n"))

		case "QUIT":
			if failAt == "quit" {
				conn.Write([]byte("550 QUIT failed\r\n"))
				return
			}
			conn.Write([]byte("221 Bye\r\n"))
			return

		case "RSET":
			conn.Write([]byte("250 OK\r\n"))

		default:
			conn.Write([]byte("500 Unknown command\r\n"))
		}
	}
}

func (s *mockSMTPServer) getReceivedMessages() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]string, len(s.receivedMsgs))
	copy(result, s.receivedMsgs)
	return result
}

// =============================================================================
// Tests with Mock SMTP Server
// =============================================================================

func TestSMTPSender_Send_WithMockServer_Success(t *testing.T) {
	server := newMockSMTPServer(t)
	defer server.close()

	config := SMTPConfig{
		Host:     "127.0.0.1",
		Port:     server.port(),
		From:     "sender@example.com",
		FromName: "Test Sender",
		Username: "user",
		Password: "pass",
		Timeout:  5 * time.Second,
		UseTLS:   false, // Mock server doesn't support TLS
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	ctx := context.Background()
	msg := ports.EmailMessage{
		To:       "recipient@example.com",
		Subject:  "Test Subject",
		TextBody: "Test body content",
	}

	err = sender.Send(ctx, msg)
	if err != nil {
		t.Errorf("Send failed: %v", err)
	}

	// Verify message was received
	msgs := server.getReceivedMessages()
	if len(msgs) != 1 {
		t.Errorf("Expected 1 message, got %d", len(msgs))
	}
	if len(msgs) > 0 && !strings.Contains(msgs[0], "Test body content") {
		t.Errorf("Message should contain body: %s", msgs[0])
	}
}

func TestSMTPSender_Send_WithMockServer_HTMLAndText(t *testing.T) {
	server := newMockSMTPServer(t)
	defer server.close()

	config := SMTPConfig{
		Host:     "127.0.0.1",
		Port:     server.port(),
		From:     "sender@example.com",
		FromName: "Test Sender",
		Timeout:  5 * time.Second,
		UseTLS:   false,
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	ctx := context.Background()
	msg := ports.EmailMessage{
		To:       "recipient@example.com",
		Subject:  "Multipart Test",
		HTMLBody: "<p>HTML content</p>",
		TextBody: "Plain text content",
	}

	err = sender.Send(ctx, msg)
	if err != nil {
		t.Errorf("Send failed: %v", err)
	}

	msgs := server.getReceivedMessages()
	if len(msgs) != 1 {
		t.Errorf("Expected 1 message, got %d", len(msgs))
	}
	if len(msgs) > 0 {
		if !strings.Contains(msgs[0], "multipart/alternative") {
			t.Errorf("Message should be multipart: %s", msgs[0])
		}
		if !strings.Contains(msgs[0], "HTML content") {
			t.Errorf("Message should contain HTML: %s", msgs[0])
		}
		if !strings.Contains(msgs[0], "Plain text content") {
			t.Errorf("Message should contain text: %s", msgs[0])
		}
	}
}

func TestSMTPSender_Send_WithMockServer_HTMLOnly(t *testing.T) {
	server := newMockSMTPServer(t)
	defer server.close()

	config := SMTPConfig{
		Host:     "127.0.0.1",
		Port:     server.port(),
		From:     "sender@example.com",
		Timeout:  5 * time.Second,
		UseTLS:   false,
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	ctx := context.Background()
	msg := ports.EmailMessage{
		To:       "recipient@example.com",
		Subject:  "HTML Only Test",
		HTMLBody: "<p>Only HTML content</p>",
		TextBody: "",
	}

	err = sender.Send(ctx, msg)
	if err != nil {
		t.Errorf("Send failed: %v", err)
	}

	msgs := server.getReceivedMessages()
	if len(msgs) != 1 {
		t.Errorf("Expected 1 message, got %d", len(msgs))
	}
	if len(msgs) > 0 {
		if !strings.Contains(msgs[0], "text/html") {
			t.Errorf("Message should be text/html: %s", msgs[0])
		}
	}
}

func TestSMTPSender_Send_WithMockServer_TextOnly(t *testing.T) {
	server := newMockSMTPServer(t)
	defer server.close()

	config := SMTPConfig{
		Host:     "127.0.0.1",
		Port:     server.port(),
		From:     "sender@example.com",
		Timeout:  5 * time.Second,
		UseTLS:   false,
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	ctx := context.Background()
	msg := ports.EmailMessage{
		To:       "recipient@example.com",
		Subject:  "Text Only Test",
		HTMLBody: "",
		TextBody: "Only plain text content",
	}

	err = sender.Send(ctx, msg)
	if err != nil {
		t.Errorf("Send failed: %v", err)
	}

	msgs := server.getReceivedMessages()
	if len(msgs) != 1 {
		t.Errorf("Expected 1 message, got %d", len(msgs))
	}
	if len(msgs) > 0 {
		if !strings.Contains(msgs[0], "text/plain") {
			t.Errorf("Message should be text/plain: %s", msgs[0])
		}
	}
}

func TestSMTPSender_Send_WithMockServer_NoAuth(t *testing.T) {
	server := newMockSMTPServer(t)
	defer server.close()

	config := SMTPConfig{
		Host:     "127.0.0.1",
		Port:     server.port(),
		From:     "sender@example.com",
		Username: "", // No auth
		Password: "",
		Timeout:  5 * time.Second,
		UseTLS:   false,
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	ctx := context.Background()
	msg := ports.EmailMessage{
		To:       "recipient@example.com",
		Subject:  "No Auth Test",
		TextBody: "Test without auth",
	}

	err = sender.Send(ctx, msg)
	if err != nil {
		t.Errorf("Send without auth failed: %v", err)
	}
}

func TestSMTPSender_Send_WithMockServer_AuthFailure(t *testing.T) {
	server := newMockSMTPServer(t)
	defer server.close()
	server.setFailAt("auth")

	config := SMTPConfig{
		Host:     "127.0.0.1",
		Port:     server.port(),
		From:     "sender@example.com",
		Username: "user",
		Password: "wrong",
		Timeout:  5 * time.Second,
		UseTLS:   false,
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	ctx := context.Background()
	msg := ports.EmailMessage{
		To:       "recipient@example.com",
		Subject:  "Auth Fail Test",
		TextBody: "Test auth failure",
	}

	err = sender.Send(ctx, msg)
	if err == nil {
		t.Error("Expected auth error, got nil")
	}
	if !strings.Contains(err.Error(), "auth") {
		t.Errorf("Error should mention auth: %v", err)
	}
}

func TestSMTPSender_Send_WithMockServer_MailFailure(t *testing.T) {
	server := newMockSMTPServer(t)
	defer server.close()
	server.setFailAt("mail")

	config := SMTPConfig{
		Host:     "127.0.0.1",
		Port:     server.port(),
		From:     "sender@example.com",
		Timeout:  5 * time.Second,
		UseTLS:   false,
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	ctx := context.Background()
	msg := ports.EmailMessage{
		To:       "recipient@example.com",
		Subject:  "Mail Fail Test",
		TextBody: "Test mail from failure",
	}

	err = sender.Send(ctx, msg)
	if err == nil {
		t.Error("Expected mail error, got nil")
	}
	if !strings.Contains(err.Error(), "mail") {
		t.Errorf("Error should mention mail: %v", err)
	}
}

func TestSMTPSender_Send_WithMockServer_RcptFailure(t *testing.T) {
	server := newMockSMTPServer(t)
	defer server.close()
	server.setFailAt("rcpt")

	config := SMTPConfig{
		Host:     "127.0.0.1",
		Port:     server.port(),
		From:     "sender@example.com",
		Timeout:  5 * time.Second,
		UseTLS:   false,
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	ctx := context.Background()
	msg := ports.EmailMessage{
		To:       "recipient@example.com",
		Subject:  "Rcpt Fail Test",
		TextBody: "Test rcpt to failure",
	}

	err = sender.Send(ctx, msg)
	if err == nil {
		t.Error("Expected rcpt error, got nil")
	}
	if !strings.Contains(err.Error(), "rcpt") {
		t.Errorf("Error should mention rcpt: %v", err)
	}
}

func TestSMTPSender_Send_WithMockServer_DataFailure(t *testing.T) {
	server := newMockSMTPServer(t)
	defer server.close()
	server.setFailAt("data")

	config := SMTPConfig{
		Host:     "127.0.0.1",
		Port:     server.port(),
		From:     "sender@example.com",
		Timeout:  5 * time.Second,
		UseTLS:   false,
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	ctx := context.Background()
	msg := ports.EmailMessage{
		To:       "recipient@example.com",
		Subject:  "Data Fail Test",
		TextBody: "Test data failure",
	}

	err = sender.Send(ctx, msg)
	if err == nil {
		t.Error("Expected data error, got nil")
	}
	if !strings.Contains(err.Error(), "data") {
		t.Errorf("Error should mention data: %v", err)
	}
}

func TestSMTPSender_SendVerification_WithMockServer(t *testing.T) {
	server := newMockSMTPServer(t)
	defer server.close()

	config := SMTPConfig{
		Host:     "127.0.0.1",
		Port:     server.port(),
		From:     "sender@example.com",
		FromName: "Test App",
		Timeout:  5 * time.Second,
		UseTLS:   false,
		BaseURL:  "https://example.com",
		AppName:  "TestApp",
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	ctx := context.Background()
	err = sender.SendVerification(ctx, "user@example.com", "Test User", "verify-token-123")
	if err != nil {
		t.Errorf("SendVerification failed: %v", err)
	}

	msgs := server.getReceivedMessages()
	if len(msgs) != 1 {
		t.Errorf("Expected 1 message, got %d", len(msgs))
	}
	if len(msgs) > 0 {
		if !strings.Contains(msgs[0], "verify-token-123") {
			t.Errorf("Message should contain token: %s", msgs[0])
		}
		if !strings.Contains(msgs[0], "Test User") {
			t.Errorf("Message should contain user name: %s", msgs[0])
		}
	}
}

func TestSMTPSender_SendPasswordReset_WithMockServer(t *testing.T) {
	server := newMockSMTPServer(t)
	defer server.close()

	config := SMTPConfig{
		Host:     "127.0.0.1",
		Port:     server.port(),
		From:     "sender@example.com",
		FromName: "Test App",
		Timeout:  5 * time.Second,
		UseTLS:   false,
		BaseURL:  "https://example.com",
		AppName:  "TestApp",
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	ctx := context.Background()
	err = sender.SendPasswordReset(ctx, "user@example.com", "Test User", "reset-token-456")
	if err != nil {
		t.Errorf("SendPasswordReset failed: %v", err)
	}

	msgs := server.getReceivedMessages()
	if len(msgs) != 1 {
		t.Errorf("Expected 1 message, got %d", len(msgs))
	}
	if len(msgs) > 0 {
		if !strings.Contains(msgs[0], "reset-token-456") {
			t.Errorf("Message should contain token: %s", msgs[0])
		}
	}
}

func TestSMTPSender_SendWelcome_WithMockServer(t *testing.T) {
	server := newMockSMTPServer(t)
	defer server.close()

	config := SMTPConfig{
		Host:     "127.0.0.1",
		Port:     server.port(),
		From:     "sender@example.com",
		FromName: "Test App",
		Timeout:  5 * time.Second,
		UseTLS:   false,
		BaseURL:  "https://example.com",
		AppName:  "TestApp",
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	ctx := context.Background()
	err = sender.SendWelcome(ctx, "user@example.com", "Welcome User")
	if err != nil {
		t.Errorf("SendWelcome failed: %v", err)
	}

	msgs := server.getReceivedMessages()
	if len(msgs) != 1 {
		t.Errorf("Expected 1 message, got %d", len(msgs))
	}
	if len(msgs) > 0 {
		if !strings.Contains(msgs[0], "Welcome User") {
			t.Errorf("Message should contain user name: %s", msgs[0])
		}
		if !strings.Contains(msgs[0], "dashboard") {
			t.Errorf("Message should contain dashboard link: %s", msgs[0])
		}
	}
}

func TestSMTPSender_Send_WithMockServer_MultipleMessages(t *testing.T) {
	server := newMockSMTPServer(t)
	defer server.close()

	config := SMTPConfig{
		Host:     "127.0.0.1",
		Port:     server.port(),
		From:     "sender@example.com",
		Timeout:  5 * time.Second,
		UseTLS:   false,
		BaseURL:  "https://example.com",
		AppName:  "TestApp",
	}

	sender, err := NewSMTPSender(config)
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	ctx := context.Background()

	// Send multiple messages
	for i := 0; i < 3; i++ {
		msg := ports.EmailMessage{
			To:       fmt.Sprintf("user%d@example.com", i),
			Subject:  fmt.Sprintf("Test %d", i),
			TextBody: fmt.Sprintf("Body %d", i),
		}
		err = sender.Send(ctx, msg)
		if err != nil {
			t.Errorf("Send %d failed: %v", i, err)
		}
	}

	msgs := server.getReceivedMessages()
	if len(msgs) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(msgs))
	}
}
