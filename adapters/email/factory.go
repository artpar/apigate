package email

import (
	"fmt"

	"github.com/artpar/apigate/domain/settings"
	"github.com/artpar/apigate/ports"
)

// NewSender creates an email sender based on settings.
func NewSender(s settings.Settings) (ports.EmailSender, error) {
	provider := s.Get(settings.KeyEmailProvider)

	switch provider {
	case "smtp":
		config := SMTPConfig{
			Host:       s.Get(settings.KeyEmailSMTPHost),
			Port:       s.GetInt(settings.KeyEmailSMTPPort, 587),
			Username:   s.Get(settings.KeyEmailSMTPUsername),
			Password:   s.Get(settings.KeyEmailSMTPPassword),
			From:       s.Get(settings.KeyEmailFromAddress),
			FromName:   s.Get(settings.KeyEmailFromName),
			UseTLS:     s.GetBool(settings.KeyEmailSMTPUseTLS),
			BaseURL:    s.Get(settings.KeyPortalBaseURL),
			AppName:    s.GetOrDefault(settings.KeyPortalAppName, "APIGate"),
		}
		if config.Host == "" {
			return nil, fmt.Errorf("SMTP host is required")
		}
		return NewSMTPSender(config)

	case "mock":
		baseURL := s.Get(settings.KeyPortalBaseURL)
		appName := s.GetOrDefault(settings.KeyPortalAppName, "APIGate")
		return NewMockSender(baseURL, appName), nil

	case "none", "":
		return NewNoopSender(), nil

	default:
		return nil, fmt.Errorf("unknown email provider: %s", provider)
	}
}
