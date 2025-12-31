package schema

import (
	"testing"
)

func TestChannelsStruct(t *testing.T) {
	channels := Channels{
		HTTP: HTTPChannel{
			Serve: HTTPServe{Enabled: true},
		},
		CLI: CLIChannel{
			Serve: CLIServe{Enabled: true},
		},
		TTY: TTYChannel{
			Serve: TTYServe{Enabled: true},
		},
		WebSocket: WebSocketChannel{
			Serve: WebSocketServe{Enabled: true},
		},
		Webhook: WebhookChannel{
			Serve: WebhookServe{Enabled: true},
		},
		GRPC: GRPCChannel{
			Serve: GRPCServe{Enabled: true},
		},
	}

	if !channels.HTTP.Serve.Enabled {
		t.Error("Channels.HTTP.Serve.Enabled not set correctly")
	}
	if !channels.CLI.Serve.Enabled {
		t.Error("Channels.CLI.Serve.Enabled not set correctly")
	}
	if !channels.TTY.Serve.Enabled {
		t.Error("Channels.TTY.Serve.Enabled not set correctly")
	}
	if !channels.WebSocket.Serve.Enabled {
		t.Error("Channels.WebSocket.Serve.Enabled not set correctly")
	}
	if !channels.Webhook.Serve.Enabled {
		t.Error("Channels.Webhook.Serve.Enabled not set correctly")
	}
	if !channels.GRPC.Serve.Enabled {
		t.Error("Channels.GRPC.Serve.Enabled not set correctly")
	}
}

func TestHTTPChannelStruct(t *testing.T) {
	httpChan := HTTPChannel{
		Serve: HTTPServe{
			Enabled:  true,
			BasePath: "/api/v1/users",
			Endpoints: []HTTPEndpoint{
				{Action: "search", Method: "GET", Path: "/search", Auth: "public"},
			},
		},
		Consume: map[string]HTTPConsumer{
			"stripe": {
				Base: "https://api.stripe.com/v1",
				Auth: HTTPAuth{
					Type:   "bearer",
					Bearer: "${STRIPE_KEY}",
				},
				Headers: map[string]string{
					"X-Custom": "value",
				},
				Methods: map[string]HTTPMethod{
					"charge": {
						Method: "POST",
						Path:   "/charges",
						Map:    map[string]string{"amount": "amount"},
					},
				},
				On: map[string]HTTPReaction{
					"created": {Method: "POST", Path: "/notify"},
				},
			},
		},
	}

	if !httpChan.Serve.Enabled {
		t.Error("HTTPChannel.Serve.Enabled not set correctly")
	}
	if httpChan.Serve.BasePath != "/api/v1/users" {
		t.Error("HTTPChannel.Serve.BasePath not set correctly")
	}
	if len(httpChan.Serve.Endpoints) != 1 {
		t.Error("HTTPChannel.Serve.Endpoints not set correctly")
	}
	if httpChan.Consume["stripe"].Base != "https://api.stripe.com/v1" {
		t.Error("HTTPChannel.Consume[stripe].Base not set correctly")
	}
}

func TestHTTPServeStruct(t *testing.T) {
	serve := HTTPServe{
		Enabled:  true,
		BasePath: "/custom",
		Endpoints: []HTTPEndpoint{
			{Action: "list", Method: "GET", Path: "/"},
			{Action: "create", Method: "POST", Path: "/", Auth: "admin"},
		},
	}

	if !serve.Enabled {
		t.Error("HTTPServe.Enabled not set correctly")
	}
	if serve.BasePath != "/custom" {
		t.Error("HTTPServe.BasePath not set correctly")
	}
	if len(serve.Endpoints) != 2 {
		t.Error("HTTPServe.Endpoints not set correctly")
	}
}

func TestHTTPEndpointStruct(t *testing.T) {
	ep := HTTPEndpoint{
		Action: "search",
		Method: "GET",
		Path:   "/search",
		Auth:   "public",
	}

	if ep.Action != "search" {
		t.Error("HTTPEndpoint.Action not set correctly")
	}
	if ep.Method != "GET" {
		t.Error("HTTPEndpoint.Method not set correctly")
	}
	if ep.Path != "/search" {
		t.Error("HTTPEndpoint.Path not set correctly")
	}
	if ep.Auth != "public" {
		t.Error("HTTPEndpoint.Auth not set correctly")
	}
}

func TestHTTPConsumerStruct(t *testing.T) {
	consumer := HTTPConsumer{
		Base: "https://api.example.com",
		Auth: HTTPAuth{
			Type:     "basic",
			Username: "user",
			Password: "pass",
		},
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Methods: map[string]HTTPMethod{
			"get_user": {
				Method: "GET",
				Path:   "/users/{id}",
			},
		},
		On: map[string]HTTPReaction{
			"user_created": {
				Method: "POST",
				Path:   "/webhooks/user",
			},
		},
	}

	if consumer.Base != "https://api.example.com" {
		t.Error("HTTPConsumer.Base not set correctly")
	}
	if consumer.Auth.Type != "basic" {
		t.Error("HTTPConsumer.Auth.Type not set correctly")
	}
	if len(consumer.Headers) != 1 {
		t.Error("HTTPConsumer.Headers not set correctly")
	}
	if len(consumer.Methods) != 1 {
		t.Error("HTTPConsumer.Methods not set correctly")
	}
	if len(consumer.On) != 1 {
		t.Error("HTTPConsumer.On not set correctly")
	}
}

func TestHTTPAuthStruct(t *testing.T) {
	tests := []struct {
		name string
		auth HTTPAuth
	}{
		{
			name: "bearer auth",
			auth: HTTPAuth{
				Type:   "bearer",
				Bearer: "token123",
			},
		},
		{
			name: "basic auth",
			auth: HTTPAuth{
				Type:     "basic",
				Username: "user",
				Password: "pass",
			},
		},
		{
			name: "header auth",
			auth: HTTPAuth{
				Type:   "header",
				Header: map[string]string{"X-API-Key": "secret"},
			},
		},
		{
			name: "query auth",
			auth: HTTPAuth{
				Type:  "query",
				Query: map[string]string{"api_key": "secret"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := tt.auth
			if auth.Type == "" {
				t.Error("HTTPAuth.Type should be set")
			}
		})
	}
}

func TestHTTPMethodStruct(t *testing.T) {
	method := HTTPMethod{
		Method: "POST",
		Path:   "/charge",
		Map:    map[string]string{"amount": "amount_cents"},
		Response: HTTPResponse{
			Set:     map[string]string{"charge_id": "id"},
			Extract: map[string]string{"status": "status"},
		},
	}

	if method.Method != "POST" {
		t.Error("HTTPMethod.Method not set correctly")
	}
	if method.Path != "/charge" {
		t.Error("HTTPMethod.Path not set correctly")
	}
	if len(method.Map) != 1 {
		t.Error("HTTPMethod.Map not set correctly")
	}
	if len(method.Response.Set) != 1 {
		t.Error("HTTPMethod.Response.Set not set correctly")
	}
}

func TestHTTPResponseStruct(t *testing.T) {
	resp := HTTPResponse{
		Set:     map[string]string{"local_field": "response_field"},
		Extract: map[string]string{"data": "$.results"},
	}

	if len(resp.Set) != 1 {
		t.Error("HTTPResponse.Set not set correctly")
	}
	if len(resp.Extract) != 1 {
		t.Error("HTTPResponse.Extract not set correctly")
	}
}

func TestHTTPReactionStruct(t *testing.T) {
	reaction := HTTPReaction{
		Method: "POST",
		Path:   "/callback",
		Map:    map[string]string{"id": "record_id"},
		Response: HTTPResponse{
			Set: map[string]string{"result": "ok"},
		},
	}

	if reaction.Method != "POST" {
		t.Error("HTTPReaction.Method not set correctly")
	}
	if reaction.Path != "/callback" {
		t.Error("HTTPReaction.Path not set correctly")
	}
}

func TestCLIChannelStruct(t *testing.T) {
	cliChan := CLIChannel{
		Serve: CLIServe{
			Enabled: true,
			Command: "users",
			Commands: []CLICommand{
				{
					Action: "list",
					Name:   "ls",
					Output: "table",
					Columns: []string{"id", "name", "email"},
				},
			},
		},
		Consume: map[string]CLIConsumer{
			"ssh": {
				Command: "ssh",
				Methods: map[string]CLIMethod{
					"exec": {
						Args:  []string{"-t", "host"},
						Parse: "lines",
					},
				},
			},
		},
	}

	if !cliChan.Serve.Enabled {
		t.Error("CLIChannel.Serve.Enabled not set correctly")
	}
	if cliChan.Serve.Command != "users" {
		t.Error("CLIChannel.Serve.Command not set correctly")
	}
	if len(cliChan.Serve.Commands) != 1 {
		t.Error("CLIChannel.Serve.Commands not set correctly")
	}
	if cliChan.Consume["ssh"].Command != "ssh" {
		t.Error("CLIChannel.Consume[ssh].Command not set correctly")
	}
}

func TestCLIServeStruct(t *testing.T) {
	serve := CLIServe{
		Enabled: true,
		Command: "myapp",
		Commands: []CLICommand{
			{Action: "run"},
		},
	}

	if !serve.Enabled {
		t.Error("CLIServe.Enabled not set correctly")
	}
	if serve.Command != "myapp" {
		t.Error("CLIServe.Command not set correctly")
	}
}

func TestCLICommandStruct(t *testing.T) {
	cmd := CLICommand{
		Action:  "create",
		Name:    "add",
		Args:    []CLIArg{{Name: "name", Required: true}},
		Flags:   []CLIFlag{{Param: "email", Short: "e", Required: true}},
		Output:  "detail",
		Columns: []string{"id", "name"},
		Confirm: "Are you sure?",
	}

	if cmd.Action != "create" {
		t.Error("CLICommand.Action not set correctly")
	}
	if cmd.Name != "add" {
		t.Error("CLICommand.Name not set correctly")
	}
	if len(cmd.Args) != 1 {
		t.Error("CLICommand.Args not set correctly")
	}
	if len(cmd.Flags) != 1 {
		t.Error("CLICommand.Flags not set correctly")
	}
	if cmd.Confirm != "Are you sure?" {
		t.Error("CLICommand.Confirm not set correctly")
	}
}

func TestCLIArgStruct(t *testing.T) {
	arg := CLIArg{
		Name:        "id",
		Required:    true,
		Description: "The user ID",
	}

	if arg.Name != "id" {
		t.Error("CLIArg.Name not set correctly")
	}
	if !arg.Required {
		t.Error("CLIArg.Required not set correctly")
	}
	if arg.Description != "The user ID" {
		t.Error("CLIArg.Description not set correctly")
	}
}

func TestCLIFlagStruct(t *testing.T) {
	flag := CLIFlag{
		Param:       "email",
		Name:        "email-address",
		Short:       "e",
		Type:        "string",
		Default:     "default@example.com",
		Required:    true,
		Prompt:      true,
		Description: "User email address",
	}

	if flag.Param != "email" {
		t.Error("CLIFlag.Param not set correctly")
	}
	if flag.Name != "email-address" {
		t.Error("CLIFlag.Name not set correctly")
	}
	if flag.Short != "e" {
		t.Error("CLIFlag.Short not set correctly")
	}
	if flag.Type != "string" {
		t.Error("CLIFlag.Type not set correctly")
	}
	if flag.Default != "default@example.com" {
		t.Error("CLIFlag.Default not set correctly")
	}
	if !flag.Required {
		t.Error("CLIFlag.Required not set correctly")
	}
	if !flag.Prompt {
		t.Error("CLIFlag.Prompt not set correctly")
	}
}

func TestCLIConsumerStruct(t *testing.T) {
	consumer := CLIConsumer{
		Command: "docker",
		Methods: map[string]CLIMethod{
			"ps": {
				Args:  []string{"ps", "-a"},
				Parse: "json",
			},
		},
	}

	if consumer.Command != "docker" {
		t.Error("CLIConsumer.Command not set correctly")
	}
	if len(consumer.Methods) != 1 {
		t.Error("CLIConsumer.Methods not set correctly")
	}
}

func TestCLIMethodStruct(t *testing.T) {
	method := CLIMethod{
		Args:  []string{"run", "--rm"},
		Parse: "json",
		Stdin: true,
	}

	if len(method.Args) != 2 {
		t.Error("CLIMethod.Args not set correctly")
	}
	if method.Parse != "json" {
		t.Error("CLIMethod.Parse not set correctly")
	}
	if !method.Stdin {
		t.Error("CLIMethod.Stdin not set correctly")
	}
}

func TestTTYChannelStruct(t *testing.T) {
	ttyChan := TTYChannel{
		Serve: TTYServe{
			Enabled:    true,
			Prompt:     "> ",
			Welcome:    "Welcome!",
			History:    true,
			Completion: true,
			Commands: []TTYCommand{
				{Name: "help", Action: "help", Aliases: []string{"?", "h"}},
			},
		},
	}

	if !ttyChan.Serve.Enabled {
		t.Error("TTYChannel.Serve.Enabled not set correctly")
	}
	if ttyChan.Serve.Prompt != "> " {
		t.Error("TTYChannel.Serve.Prompt not set correctly")
	}
	if ttyChan.Serve.Welcome != "Welcome!" {
		t.Error("TTYChannel.Serve.Welcome not set correctly")
	}
	if !ttyChan.Serve.History {
		t.Error("TTYChannel.Serve.History not set correctly")
	}
	if !ttyChan.Serve.Completion {
		t.Error("TTYChannel.Serve.Completion not set correctly")
	}
}

func TestTTYServeStruct(t *testing.T) {
	serve := TTYServe{
		Enabled:    true,
		Prompt:     "$ ",
		Welcome:    "Hello",
		History:    true,
		Completion: false,
		Commands:   []TTYCommand{},
	}

	if !serve.Enabled {
		t.Error("TTYServe.Enabled not set correctly")
	}
}

func TestTTYCommandStruct(t *testing.T) {
	cmd := TTYCommand{
		Name:        "quit",
		Action:      "exit",
		Args:        []CLIArg{{Name: "code"}},
		Description: "Exit the shell",
		Aliases:     []string{"exit", "q"},
	}

	if cmd.Name != "quit" {
		t.Error("TTYCommand.Name not set correctly")
	}
	if cmd.Action != "exit" {
		t.Error("TTYCommand.Action not set correctly")
	}
	if len(cmd.Aliases) != 2 {
		t.Error("TTYCommand.Aliases not set correctly")
	}
}

func TestWebSocketChannelStruct(t *testing.T) {
	wsChan := WebSocketChannel{
		Serve: WebSocketServe{
			Enabled:  true,
			Path:     "/ws",
			Auth:     "jwt",
			Protocol: "json",
			Events:   []string{"created", "updated"},
			Inbound: map[string]WSMessage{
				"subscribe": {Params: []string{"channel"}},
			},
			Outbound: map[string]WSMessage{
				"notification": {Params: []string{"message"}},
			},
		},
		Consume: map[string]WebSocketConsumer{
			"feed": {
				URL:       "wss://example.com/feed",
				Reconnect: true,
			},
		},
	}

	if !wsChan.Serve.Enabled {
		t.Error("WebSocketChannel.Serve.Enabled not set correctly")
	}
	if wsChan.Serve.Path != "/ws" {
		t.Error("WebSocketChannel.Serve.Path not set correctly")
	}
	if wsChan.Consume["feed"].URL != "wss://example.com/feed" {
		t.Error("WebSocketChannel.Consume[feed].URL not set correctly")
	}
}

func TestWebSocketServeStruct(t *testing.T) {
	serve := WebSocketServe{
		Enabled:  true,
		Path:     "/realtime",
		Auth:     "api_key",
		Protocol: "msgpack",
		Events:   []string{"change"},
		Inbound:  map[string]WSMessage{"ping": {}},
		Outbound: map[string]WSMessage{"pong": {}},
	}

	if !serve.Enabled {
		t.Error("WebSocketServe.Enabled not set correctly")
	}
	if serve.Protocol != "msgpack" {
		t.Error("WebSocketServe.Protocol not set correctly")
	}
}

func TestWSMessageStruct(t *testing.T) {
	msg := WSMessage{
		Params:  []string{"channel", "data"},
		Auth:    "user",
		Handler: "handle_message",
	}

	if len(msg.Params) != 2 {
		t.Error("WSMessage.Params not set correctly")
	}
	if msg.Auth != "user" {
		t.Error("WSMessage.Auth not set correctly")
	}
	if msg.Handler != "handle_message" {
		t.Error("WSMessage.Handler not set correctly")
	}
}

func TestWebSocketConsumerStruct(t *testing.T) {
	consumer := WebSocketConsumer{
		URL:       "wss://api.example.com/ws",
		Reconnect: true,
		Auth: HTTPAuth{
			Type:   "bearer",
			Bearer: "token",
		},
		OnMessage:    WSHandler{Action: "process"},
		OnConnect:    WSHandler{Action: "on_connect"},
		OnDisconnect: WSHandler{Action: "on_disconnect"},
	}

	if consumer.URL != "wss://api.example.com/ws" {
		t.Error("WebSocketConsumer.URL not set correctly")
	}
	if !consumer.Reconnect {
		t.Error("WebSocketConsumer.Reconnect not set correctly")
	}
}

func TestWSHandlerStruct(t *testing.T) {
	handler := WSHandler{
		Action:    "notify_clients",
		Map:       map[string]string{"event": "type"},
		Broadcast: true,
		Channel:   "updates",
	}

	if handler.Action != "notify_clients" {
		t.Error("WSHandler.Action not set correctly")
	}
	if !handler.Broadcast {
		t.Error("WSHandler.Broadcast not set correctly")
	}
	if handler.Channel != "updates" {
		t.Error("WSHandler.Channel not set correctly")
	}
}

func TestWebhookChannelStruct(t *testing.T) {
	webhookChan := WebhookChannel{
		Serve: WebhookServe{
			Enabled: true,
			Path:    "/webhooks/incoming",
		},
		Consume: map[string]WebhookConsumer{
			"stripe": {
				Secret: "whsec_xxx",
				Events: map[string]WebhookHandler{
					"payment.completed": {
						Action: "create",
						Lookup: map[string]string{"stripe_id": "id"},
					},
				},
			},
		},
	}

	if !webhookChan.Serve.Enabled {
		t.Error("WebhookChannel.Serve.Enabled not set correctly")
	}
	if webhookChan.Consume["stripe"].Secret != "whsec_xxx" {
		t.Error("WebhookChannel.Consume[stripe].Secret not set correctly")
	}
}

func TestWebhookServeStruct(t *testing.T) {
	serve := WebhookServe{
		Enabled: true,
		Path:    "/hooks",
	}

	if !serve.Enabled {
		t.Error("WebhookServe.Enabled not set correctly")
	}
	if serve.Path != "/hooks" {
		t.Error("WebhookServe.Path not set correctly")
	}
}

func TestWebhookConsumerStruct(t *testing.T) {
	consumer := WebhookConsumer{
		Secret: "secret123",
		Events: map[string]WebhookHandler{
			"user.created": {
				Action: "sync_user",
			},
		},
	}

	if consumer.Secret != "secret123" {
		t.Error("WebhookConsumer.Secret not set correctly")
	}
	if len(consumer.Events) != 1 {
		t.Error("WebhookConsumer.Events not set correctly")
	}
}

func TestWebhookHandlerStruct(t *testing.T) {
	handler := WebhookHandler{
		Action: "update",
		Lookup: map[string]string{"external_id": "id"},
		Map:    map[string]string{"name": "data.name"},
		Set:    map[string]string{"synced": "true"},
		Then: []WebhookThen{
			{Emit: "synced"},
			{Notify: "admin"},
			{Call: "send_notification"},
		},
	}

	if handler.Action != "update" {
		t.Error("WebhookHandler.Action not set correctly")
	}
	if len(handler.Lookup) != 1 {
		t.Error("WebhookHandler.Lookup not set correctly")
	}
	if len(handler.Then) != 3 {
		t.Error("WebhookHandler.Then not set correctly")
	}
}

func TestWebhookThenStruct(t *testing.T) {
	then := WebhookThen{
		Emit:   "event.processed",
		Notify: "channel",
		Call:   "post_process",
	}

	if then.Emit != "event.processed" {
		t.Error("WebhookThen.Emit not set correctly")
	}
	if then.Notify != "channel" {
		t.Error("WebhookThen.Notify not set correctly")
	}
	if then.Call != "post_process" {
		t.Error("WebhookThen.Call not set correctly")
	}
}

func TestGRPCChannelStruct(t *testing.T) {
	grpcChan := GRPCChannel{
		Serve: GRPCServe{
			Enabled: true,
			Service: "UserService",
			Methods: map[string]GRPCMethod{
				"GetUser": {Name: "GetUser", Stream: "none"},
			},
		},
		Consume: map[string]GRPCConsumer{
			"auth": {
				Target: "localhost:50051",
				TLS: GRPCTLSConfig{
					Enabled:  true,
					Insecure: false,
				},
			},
		},
	}

	if !grpcChan.Serve.Enabled {
		t.Error("GRPCChannel.Serve.Enabled not set correctly")
	}
	if grpcChan.Serve.Service != "UserService" {
		t.Error("GRPCChannel.Serve.Service not set correctly")
	}
	if grpcChan.Consume["auth"].Target != "localhost:50051" {
		t.Error("GRPCChannel.Consume[auth].Target not set correctly")
	}
}

func TestGRPCServeStruct(t *testing.T) {
	serve := GRPCServe{
		Enabled: true,
		Service: "MyService",
		Methods: map[string]GRPCMethod{
			"List": {Name: "List", Stream: "server"},
		},
	}

	if !serve.Enabled {
		t.Error("GRPCServe.Enabled not set correctly")
	}
	if serve.Service != "MyService" {
		t.Error("GRPCServe.Service not set correctly")
	}
}

func TestGRPCMethodStruct(t *testing.T) {
	method := GRPCMethod{
		Name:   "StreamUpdates",
		Stream: "bidirectional",
	}

	if method.Name != "StreamUpdates" {
		t.Error("GRPCMethod.Name not set correctly")
	}
	if method.Stream != "bidirectional" {
		t.Error("GRPCMethod.Stream not set correctly")
	}
}

func TestGRPCConsumerStruct(t *testing.T) {
	consumer := GRPCConsumer{
		Target: "api.example.com:443",
		TLS: GRPCTLSConfig{
			Enabled:  true,
			Insecure: false,
			CACert:   "/path/to/ca.crt",
		},
		Methods: map[string]GRPCClientMethod{
			"Validate": {
				Map:      map[string]string{"token": "auth_token"},
				Response: map[string]string{"valid": "is_valid"},
			},
		},
	}

	if consumer.Target != "api.example.com:443" {
		t.Error("GRPCConsumer.Target not set correctly")
	}
	if !consumer.TLS.Enabled {
		t.Error("GRPCConsumer.TLS.Enabled not set correctly")
	}
	if consumer.TLS.CACert != "/path/to/ca.crt" {
		t.Error("GRPCConsumer.TLS.CACert not set correctly")
	}
}

func TestGRPCTLSConfigStruct(t *testing.T) {
	tls := GRPCTLSConfig{
		Enabled:  true,
		Insecure: true,
		CACert:   "/certs/ca.pem",
	}

	if !tls.Enabled {
		t.Error("GRPCTLSConfig.Enabled not set correctly")
	}
	if !tls.Insecure {
		t.Error("GRPCTLSConfig.Insecure not set correctly")
	}
	if tls.CACert != "/certs/ca.pem" {
		t.Error("GRPCTLSConfig.CACert not set correctly")
	}
}

func TestGRPCClientMethodStruct(t *testing.T) {
	method := GRPCClientMethod{
		Map:      map[string]string{"id": "user_id"},
		Response: map[string]string{"name": "user_name"},
	}

	if len(method.Map) != 1 {
		t.Error("GRPCClientMethod.Map not set correctly")
	}
	if len(method.Response) != 1 {
		t.Error("GRPCClientMethod.Response not set correctly")
	}
}
