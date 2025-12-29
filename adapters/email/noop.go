package email

import (
	"context"

	"github.com/artpar/apigate/ports"
)

// NoopSender is a no-op email sender for when email is disabled.
type NoopSender struct{}

// NewNoopSender creates a new no-op email sender.
func NewNoopSender() *NoopSender {
	return &NoopSender{}
}

// Send does nothing.
func (s *NoopSender) Send(ctx context.Context, msg ports.EmailMessage) error {
	return nil
}

// SendVerification does nothing.
func (s *NoopSender) SendVerification(ctx context.Context, to, name, token string) error {
	return nil
}

// SendPasswordReset does nothing.
func (s *NoopSender) SendPasswordReset(ctx context.Context, to, name, token string) error {
	return nil
}

// SendWelcome does nothing.
func (s *NoopSender) SendWelcome(ctx context.Context, to, name string) error {
	return nil
}
