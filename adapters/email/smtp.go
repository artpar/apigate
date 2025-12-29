// Package email provides email sending adapters.
package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/artpar/apigate/ports"
)

// SMTPConfig holds SMTP server configuration.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string // sender email address
	FromName string // sender display name

	// TLS settings
	UseTLS      bool // Use STARTTLS
	SkipVerify  bool // Skip TLS certificate verification (for testing)
	UseImplicit bool // Use implicit TLS (port 465)

	// Timeouts
	Timeout time.Duration

	// Application settings
	BaseURL string // Base URL for links in emails (e.g., "https://myapp.com")
	AppName string // Application name for email templates
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() SMTPConfig {
	return SMTPConfig{
		Host:     "localhost",
		Port:     25,
		From:     "noreply@localhost",
		FromName: "APIGate",
		UseTLS:   true,
		Timeout:  30 * time.Second,
		AppName:  "APIGate",
	}
}

// SMTPSender implements ports.EmailSender using SMTP.
type SMTPSender struct {
	config SMTPConfig

	// Compiled templates
	verificationTmpl  *template.Template
	passwordResetTmpl *template.Template
	welcomeTmpl       *template.Template
}

// NewSMTPSender creates a new SMTP email sender.
func NewSMTPSender(config SMTPConfig) (*SMTPSender, error) {
	s := &SMTPSender{config: config}

	// Parse templates
	var err error
	s.verificationTmpl, err = template.New("verification").Parse(verificationEmailTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse verification template: %w", err)
	}

	s.passwordResetTmpl, err = template.New("passwordReset").Parse(passwordResetEmailTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse password reset template: %w", err)
	}

	s.welcomeTmpl, err = template.New("welcome").Parse(welcomeEmailTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse welcome template: %w", err)
	}

	return s, nil
}

// Send sends an email via SMTP.
func (s *SMTPSender) Send(ctx context.Context, msg ports.EmailMessage) error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	// Build email message
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("From: %s <%s>\r\n", s.config.FromName, s.config.From))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", msg.To))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	buf.WriteString("MIME-Version: 1.0\r\n")

	// Multipart message if we have both HTML and text
	if msg.HTMLBody != "" && msg.TextBody != "" {
		boundary := "boundary-" + fmt.Sprintf("%d", time.Now().UnixNano())
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%s\r\n", boundary))
		buf.WriteString("\r\n")

		// Text part
		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(msg.TextBody)
		buf.WriteString("\r\n")

		// HTML part
		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString("Content-Type: text/html; charset=utf-8\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(msg.HTMLBody)
		buf.WriteString("\r\n")

		buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else if msg.HTMLBody != "" {
		buf.WriteString("Content-Type: text/html; charset=utf-8\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(msg.HTMLBody)
	} else {
		buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(msg.TextBody)
	}

	// Send the email
	if s.config.UseImplicit {
		return s.sendImplicitTLS(ctx, addr, msg.To, buf.Bytes())
	}
	return s.sendSTARTTLS(ctx, addr, msg.To, buf.Bytes())
}

// sendSTARTTLS sends email using STARTTLS (port 587/25).
func (s *SMTPSender) sendSTARTTLS(ctx context.Context, addr, to string, message []byte) error {
	// Create dialer with timeout
	dialer := &net.Dialer{Timeout: s.config.Timeout}

	// Connect
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	// STARTTLS if required
	if s.config.UseTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			tlsConfig := &tls.Config{
				ServerName:         s.config.Host,
				InsecureSkipVerify: s.config.SkipVerify,
			}
			if err := client.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("starttls: %w", err)
			}
		}
	}

	// Authenticate if credentials provided
	if s.config.Username != "" {
		auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}

	// Set sender and recipient
	if err := client.Mail(s.config.From); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("rcpt to: %w", err)
	}

	// Send message body
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err := w.Write(message); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close data: %w", err)
	}

	return client.Quit()
}

// sendImplicitTLS sends email using implicit TLS (port 465).
func (s *SMTPSender) sendImplicitTLS(ctx context.Context, addr, to string, message []byte) error {
	tlsConfig := &tls.Config{
		ServerName:         s.config.Host,
		InsecureSkipVerify: s.config.SkipVerify,
	}

	// Connect with TLS
	dialer := &tls.Dialer{
		NetDialer: &net.Dialer{Timeout: s.config.Timeout},
		Config:    tlsConfig,
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("dial tls: %w", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	// Authenticate if credentials provided
	if s.config.Username != "" {
		auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}

	// Set sender and recipient
	if err := client.Mail(s.config.From); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("rcpt to: %w", err)
	}

	// Send message body
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err := w.Write(message); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close data: %w", err)
	}

	return client.Quit()
}

// SendVerification sends an email verification link.
func (s *SMTPSender) SendVerification(ctx context.Context, to, name, token string) error {
	data := emailTemplateData{
		Name:    name,
		AppName: s.config.AppName,
		Link:    fmt.Sprintf("%s/verify-email?token=%s", s.config.BaseURL, token),
	}

	var htmlBuf, textBuf bytes.Buffer
	if err := s.verificationTmpl.Execute(&htmlBuf, data); err != nil {
		return fmt.Errorf("execute verification template: %w", err)
	}

	// Generate plain text version
	textBuf.WriteString(fmt.Sprintf("Hi %s,\n\n", name))
	textBuf.WriteString(fmt.Sprintf("Welcome to %s! Please verify your email by clicking the link below:\n\n", s.config.AppName))
	textBuf.WriteString(data.Link)
	textBuf.WriteString("\n\nThis link will expire in 24 hours.\n\n")
	textBuf.WriteString(fmt.Sprintf("Thanks,\nThe %s Team", s.config.AppName))

	return s.Send(ctx, ports.EmailMessage{
		To:       to,
		Subject:  fmt.Sprintf("Verify your email for %s", s.config.AppName),
		HTMLBody: htmlBuf.String(),
		TextBody: textBuf.String(),
	})
}

// SendPasswordReset sends a password reset link.
func (s *SMTPSender) SendPasswordReset(ctx context.Context, to, name, token string) error {
	data := emailTemplateData{
		Name:    name,
		AppName: s.config.AppName,
		Link:    fmt.Sprintf("%s/reset-password?token=%s", s.config.BaseURL, token),
	}

	var htmlBuf, textBuf bytes.Buffer
	if err := s.passwordResetTmpl.Execute(&htmlBuf, data); err != nil {
		return fmt.Errorf("execute password reset template: %w", err)
	}

	// Generate plain text version
	textBuf.WriteString(fmt.Sprintf("Hi %s,\n\n", name))
	textBuf.WriteString("We received a request to reset your password. Click the link below to set a new password:\n\n")
	textBuf.WriteString(data.Link)
	textBuf.WriteString("\n\nThis link will expire in 1 hour.\n\n")
	textBuf.WriteString("If you didn't request this, you can safely ignore this email.\n\n")
	textBuf.WriteString(fmt.Sprintf("Thanks,\nThe %s Team", s.config.AppName))

	return s.Send(ctx, ports.EmailMessage{
		To:       to,
		Subject:  fmt.Sprintf("Reset your password for %s", s.config.AppName),
		HTMLBody: htmlBuf.String(),
		TextBody: textBuf.String(),
	})
}

// SendWelcome sends a welcome email after verification.
func (s *SMTPSender) SendWelcome(ctx context.Context, to, name string) error {
	data := emailTemplateData{
		Name:    name,
		AppName: s.config.AppName,
		Link:    fmt.Sprintf("%s/dashboard", s.config.BaseURL),
	}

	var htmlBuf, textBuf bytes.Buffer
	if err := s.welcomeTmpl.Execute(&htmlBuf, data); err != nil {
		return fmt.Errorf("execute welcome template: %w", err)
	}

	// Generate plain text version
	textBuf.WriteString(fmt.Sprintf("Hi %s,\n\n", name))
	textBuf.WriteString(fmt.Sprintf("Welcome to %s! Your email has been verified and your account is now active.\n\n", s.config.AppName))
	textBuf.WriteString("Here's what you can do next:\n")
	textBuf.WriteString("- Create API keys to start using the API\n")
	textBuf.WriteString("- Choose a plan that fits your needs\n")
	textBuf.WriteString("- Check out our documentation\n\n")
	textBuf.WriteString(fmt.Sprintf("Visit your dashboard: %s\n\n", data.Link))
	textBuf.WriteString(fmt.Sprintf("Thanks,\nThe %s Team", s.config.AppName))

	return s.Send(ctx, ports.EmailMessage{
		To:       to,
		Subject:  fmt.Sprintf("Welcome to %s!", s.config.AppName),
		HTMLBody: htmlBuf.String(),
		TextBody: textBuf.String(),
	})
}

// emailTemplateData holds data for email templates.
type emailTemplateData struct {
	Name    string
	AppName string
	Link    string
}

// Ensure interface compliance.
var _ ports.EmailSender = (*SMTPSender)(nil)

// -----------------------------------------------------------------------------
// Email Templates
// -----------------------------------------------------------------------------

var verificationEmailTemplate = strings.TrimSpace(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Verify your email</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { text-align: center; padding: 20px 0; }
        .content { background: #f9f9f9; padding: 30px; border-radius: 8px; }
        .button { display: inline-block; background: #007bff; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; margin: 20px 0; }
        .footer { text-align: center; padding: 20px; color: #666; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.AppName}}</h1>
        </div>
        <div class="content">
            <h2>Verify your email address</h2>
            <p>Hi {{.Name}},</p>
            <p>Thanks for signing up for {{.AppName}}! Please click the button below to verify your email address:</p>
            <p style="text-align: center;">
                <a href="{{.Link}}" class="button">Verify Email</a>
            </p>
            <p>Or copy and paste this link into your browser:</p>
            <p style="word-break: break-all; color: #666;">{{.Link}}</p>
            <p>This link will expire in 24 hours.</p>
        </div>
        <div class="footer">
            <p>If you didn't create an account, you can safely ignore this email.</p>
        </div>
    </div>
</body>
</html>
`)

var passwordResetEmailTemplate = strings.TrimSpace(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Reset your password</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { text-align: center; padding: 20px 0; }
        .content { background: #f9f9f9; padding: 30px; border-radius: 8px; }
        .button { display: inline-block; background: #dc3545; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; margin: 20px 0; }
        .footer { text-align: center; padding: 20px; color: #666; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.AppName}}</h1>
        </div>
        <div class="content">
            <h2>Reset your password</h2>
            <p>Hi {{.Name}},</p>
            <p>We received a request to reset your password. Click the button below to set a new password:</p>
            <p style="text-align: center;">
                <a href="{{.Link}}" class="button">Reset Password</a>
            </p>
            <p>Or copy and paste this link into your browser:</p>
            <p style="word-break: break-all; color: #666;">{{.Link}}</p>
            <p><strong>This link will expire in 1 hour.</strong></p>
        </div>
        <div class="footer">
            <p>If you didn't request a password reset, you can safely ignore this email.</p>
            <p>Your password will remain unchanged.</p>
        </div>
    </div>
</body>
</html>
`)

var welcomeEmailTemplate = strings.TrimSpace(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Welcome!</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { text-align: center; padding: 20px 0; }
        .content { background: #f9f9f9; padding: 30px; border-radius: 8px; }
        .button { display: inline-block; background: #28a745; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; margin: 20px 0; }
        .footer { text-align: center; padding: 20px; color: #666; font-size: 14px; }
        .features { background: white; padding: 20px; border-radius: 5px; margin: 20px 0; }
        .features li { margin: 10px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.AppName}}</h1>
        </div>
        <div class="content">
            <h2>Welcome to {{.AppName}}!</h2>
            <p>Hi {{.Name}},</p>
            <p>Your email has been verified and your account is now active. You're ready to get started!</p>
            <div class="features">
                <h3>Here's what you can do:</h3>
                <ul>
                    <li><strong>Create API Keys</strong> - Generate keys to start making API calls</li>
                    <li><strong>Choose a Plan</strong> - Select a plan that fits your usage needs</li>
                    <li><strong>View Usage</strong> - Monitor your API usage and costs</li>
                </ul>
            </div>
            <p style="text-align: center;">
                <a href="{{.Link}}" class="button">Go to Dashboard</a>
            </p>
        </div>
        <div class="footer">
            <p>Need help? Check out our documentation or contact support.</p>
        </div>
    </div>
</body>
</html>
`)
