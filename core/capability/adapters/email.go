package adapters

import (
	"context"

	"github.com/artpar/apigate/core/capability"
	"github.com/artpar/apigate/ports"
)

// EmailAdapter wraps a ports.EmailSender to implement capability.EmailProvider.
type EmailAdapter struct {
	name  string
	inner ports.EmailSender
}

// WrapEmail creates a capability.EmailProvider from a ports.EmailSender.
func WrapEmail(name string, inner ports.EmailSender) *EmailAdapter {
	return &EmailAdapter{name: name, inner: inner}
}

func (a *EmailAdapter) Name() string {
	return a.name
}

func (a *EmailAdapter) Send(ctx context.Context, msg capability.EmailMessage) error {
	return a.inner.Send(ctx, ports.EmailMessage{
		To:       msg.To,
		Subject:  msg.Subject,
		HTMLBody: msg.HTMLBody,
		TextBody: msg.TextBody,
	})
}

func (a *EmailAdapter) SendTemplate(ctx context.Context, to, templateID string, vars map[string]string) error {
	// The existing EmailSender doesn't have SendTemplate.
	// We could implement this by looking up a template and rendering it.
	// For now, return not implemented.
	return ErrNotImplemented
}

func (a *EmailAdapter) TestConnection(ctx context.Context) error {
	// Most email providers don't have a direct test connection.
	// This could be implemented as a test email send.
	return nil
}

// Ensure EmailAdapter implements capability.EmailProvider
var _ capability.EmailProvider = (*EmailAdapter)(nil)
