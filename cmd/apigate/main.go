// Package main is the entry point for APIGate.
//
//	@title						APIGate - API Monetization Proxy
//	@version					1.0
//	@description				Self-hosted API monetization solution with authentication, rate limiting, usage metering, and billing.
//	@termsOfService				https://github.com/artpar/apigate
//
//	@contact.name				APIGate Support
//	@contact.url				https://github.com/artpar/apigate/issues
//
//	@license.name				MIT
//	@license.url				https://opensource.org/licenses/MIT
//
//	@host						localhost:8080
//	@BasePath					/
//
//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						X-API-Key
//	@description				API key for authentication
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Bearer token authentication (format: "Bearer {api_key}")
package main

func main() {
	Execute()
}
