package schema

// Channels defines all communication channels for a module.
// Each channel can both serve (expose) and consume (call) endpoints.
type Channels struct {
	// HTTP channel for REST API.
	HTTP HTTPChannel `yaml:"http,omitempty"`

	// CLI channel for command-line interface.
	CLI CLIChannel `yaml:"cli,omitempty"`

	// TTY channel for interactive terminal sessions.
	TTY TTYChannel `yaml:"tty,omitempty"`

	// WebSocket channel for real-time communication.
	WebSocket WebSocketChannel `yaml:"websocket,omitempty"`

	// Webhook channel for event notifications.
	Webhook WebhookChannel `yaml:"webhook,omitempty"`

	// GRPC channel for gRPC service.
	GRPC GRPCChannel `yaml:"grpc,omitempty"`
}

// --------------------------------------------------------------------------
// HTTP Channel
// --------------------------------------------------------------------------

// HTTPChannel defines HTTP REST API configuration.
type HTTPChannel struct {
	// Serve exposes HTTP endpoints. Set to true for default CRUD endpoints.
	Serve HTTPServe `yaml:"serve,omitempty"`

	// Consume defines external HTTP APIs this module calls.
	Consume map[string]HTTPConsumer `yaml:"consume,omitempty"`
}

// HTTPServe defines HTTP endpoints this module exposes.
type HTTPServe struct {
	// Enabled indicates whether to serve HTTP endpoints.
	// If true, default CRUD endpoints are created.
	Enabled bool `yaml:"enabled,omitempty"`

	// BasePath overrides the default /{plural} path.
	BasePath string `yaml:"base_path,omitempty"`

	// Endpoints defines custom endpoint configurations.
	Endpoints []HTTPEndpoint `yaml:"endpoints,omitempty"`
}

// HTTPEndpoint defines a single HTTP endpoint.
type HTTPEndpoint struct {
	// Action is the action this endpoint triggers.
	Action string `yaml:"action"`

	// Method is the HTTP method (GET, POST, PATCH, DELETE).
	Method string `yaml:"method"`

	// Path is the endpoint path, relative to BasePath.
	Path string `yaml:"path"`

	// Auth overrides the default auth for this endpoint.
	Auth string `yaml:"auth,omitempty"`
}

// HTTPConsumer defines an external HTTP API to consume.
type HTTPConsumer struct {
	// Base is the base URL of the external API.
	Base string `yaml:"base"`

	// Auth defines authentication for the external API.
	Auth HTTPAuth `yaml:"auth,omitempty"`

	// Headers are default headers to send with every request.
	Headers map[string]string `yaml:"headers,omitempty"`

	// Methods defines callable methods on this external API.
	Methods map[string]HTTPMethod `yaml:"methods,omitempty"`

	// On defines reactions to internal events.
	On map[string]HTTPReaction `yaml:"on,omitempty"`
}

// HTTPAuth defines authentication for an HTTP consumer.
type HTTPAuth struct {
	// Type is the auth type: "bearer", "basic", "header", "query".
	Type string `yaml:"type,omitempty"`

	// Bearer token (can use ${ENV_VAR} syntax).
	Bearer string `yaml:"bearer,omitempty"`

	// Basic auth credentials.
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`

	// Header adds a custom auth header.
	Header map[string]string `yaml:"header,omitempty"`

	// Query adds auth as query parameter.
	Query map[string]string `yaml:"query,omitempty"`
}

// HTTPMethod defines a callable method on an external API.
type HTTPMethod struct {
	// Method is the HTTP method.
	Method string `yaml:"method"`

	// Path is the URL path (can include {field} placeholders).
	Path string `yaml:"path"`

	// Map defines how to map module fields to request body/query.
	Map map[string]string `yaml:"map,omitempty"`

	// Response defines how to handle the response.
	Response HTTPResponse `yaml:"response,omitempty"`
}

// HTTPResponse defines how to handle an external API response.
type HTTPResponse struct {
	// Set maps response fields to module fields.
	Set map[string]string `yaml:"set,omitempty"`

	// Extract defines fields to extract from the response.
	Extract map[string]string `yaml:"extract,omitempty"`
}

// HTTPReaction defines what to do when an internal event occurs.
type HTTPReaction struct {
	// Method is the HTTP method to use.
	Method string `yaml:"method"`

	// Path is the URL path.
	Path string `yaml:"path"`

	// Map defines how to map event data to request.
	Map map[string]string `yaml:"map,omitempty"`

	// Response defines how to handle the response.
	Response HTTPResponse `yaml:"response,omitempty"`
}

// --------------------------------------------------------------------------
// CLI Channel
// --------------------------------------------------------------------------

// CLIChannel defines command-line interface configuration.
type CLIChannel struct {
	// Serve exposes CLI commands.
	Serve CLIServe `yaml:"serve,omitempty"`

	// Consume defines external CLI tools/processes to call.
	Consume map[string]CLIConsumer `yaml:"consume,omitempty"`
}

// CLIServe defines CLI commands this module exposes.
type CLIServe struct {
	// Enabled indicates whether to expose CLI commands.
	// If true, default CRUD commands are created.
	Enabled bool `yaml:"enabled,omitempty"`

	// Command overrides the default command name (defaults to plural).
	Command string `yaml:"command,omitempty"`

	// Commands defines custom command configurations.
	Commands []CLICommand `yaml:"commands,omitempty"`
}

// CLICommand defines a single CLI command.
type CLICommand struct {
	// Action is the action this command triggers.
	Action string `yaml:"action"`

	// Name overrides the command name (defaults to action name).
	Name string `yaml:"name,omitempty"`

	// Args defines positional arguments.
	Args []CLIArg `yaml:"args,omitempty"`

	// Flags defines command flags.
	Flags []CLIFlag `yaml:"flags,omitempty"`

	// Output format: "table", "detail", "json", "yaml".
	Output string `yaml:"output,omitempty"`

	// Columns for table output.
	Columns []string `yaml:"columns,omitempty"`

	// Confirm message for destructive actions.
	Confirm string `yaml:"confirm,omitempty"`
}

// CLIArg defines a positional CLI argument.
type CLIArg struct {
	// Name of the argument.
	Name string `yaml:"name"`

	// Required indicates this argument must be provided.
	Required bool `yaml:"required,omitempty"`

	// Description for help text.
	Description string `yaml:"description,omitempty"`
}

// CLIFlag defines a CLI flag.
type CLIFlag struct {
	// Param is the action input this flag maps to.
	Param string `yaml:"param,omitempty"`

	// Name overrides the flag name.
	Name string `yaml:"name,omitempty"`

	// Short is the single-letter shorthand.
	Short string `yaml:"short,omitempty"`

	// Type is the flag type: "string", "int", "bool".
	Type string `yaml:"type,omitempty"`

	// Default value.
	Default string `yaml:"default,omitempty"`

	// Required indicates this flag must be provided.
	Required bool `yaml:"required,omitempty"`

	// Prompt indicates to prompt for value if not provided.
	Prompt bool `yaml:"prompt,omitempty"`

	// Description for help text.
	Description string `yaml:"description,omitempty"`
}

// CLIConsumer defines an external CLI tool to call.
type CLIConsumer struct {
	// Command is the base command to run.
	Command string `yaml:"command"`

	// Methods defines callable operations.
	Methods map[string]CLIMethod `yaml:"methods,omitempty"`
}

// CLIMethod defines a callable CLI operation.
type CLIMethod struct {
	// Args are the arguments to pass.
	Args []string `yaml:"args,omitempty"`

	// Parse defines how to parse output: "json", "yaml", "lines", "table".
	Parse string `yaml:"parse,omitempty"`

	// Stdin indicates to send data to stdin.
	Stdin bool `yaml:"stdin,omitempty"`
}

// --------------------------------------------------------------------------
// TTY Channel
// --------------------------------------------------------------------------

// TTYChannel defines interactive terminal session configuration.
type TTYChannel struct {
	// Serve exposes an interactive terminal interface.
	Serve TTYServe `yaml:"serve,omitempty"`
}

// TTYServe defines TTY server configuration.
type TTYServe struct {
	// Enabled indicates whether to expose TTY interface.
	Enabled bool `yaml:"enabled,omitempty"`

	// Prompt is the interactive prompt string.
	Prompt string `yaml:"prompt,omitempty"`

	// Welcome is the message shown when session starts.
	Welcome string `yaml:"welcome,omitempty"`

	// History enables command history.
	History bool `yaml:"history,omitempty"`

	// Completion enables tab completion.
	Completion bool `yaml:"completion,omitempty"`

	// Commands defines available interactive commands.
	Commands []TTYCommand `yaml:"commands,omitempty"`
}

// TTYCommand defines an interactive terminal command.
type TTYCommand struct {
	// Name is the command name.
	Name string `yaml:"name"`

	// Action is the action this command triggers.
	Action string `yaml:"action"`

	// Args defines expected arguments.
	Args []CLIArg `yaml:"args,omitempty"`

	// Description for help text.
	Description string `yaml:"description,omitempty"`

	// Aliases are alternative command names.
	Aliases []string `yaml:"aliases,omitempty"`
}

// --------------------------------------------------------------------------
// WebSocket Channel
// --------------------------------------------------------------------------

// WebSocketChannel defines WebSocket configuration.
type WebSocketChannel struct {
	// Serve runs a WebSocket server.
	Serve WebSocketServe `yaml:"serve,omitempty"`

	// Consume connects to external WebSocket servers.
	Consume map[string]WebSocketConsumer `yaml:"consume,omitempty"`
}

// WebSocketServe defines WebSocket server configuration.
type WebSocketServe struct {
	// Enabled indicates whether to run a WebSocket server.
	Enabled bool `yaml:"enabled,omitempty"`

	// Path is the WebSocket endpoint path (e.g., "/ws").
	Path string `yaml:"path,omitempty"`

	// Auth defines authentication: "none", "api_key", "jwt".
	Auth string `yaml:"auth,omitempty"`

	// Protocol defines message format: "json", "protobuf", "msgpack".
	Protocol string `yaml:"protocol,omitempty"`

	// Events lists events this module broadcasts.
	Events []string `yaml:"events,omitempty"`

	// Inbound defines messages clients can send.
	Inbound map[string]WSMessage `yaml:"inbound,omitempty"`

	// Outbound defines messages server can send.
	Outbound map[string]WSMessage `yaml:"outbound,omitempty"`
}

// WSMessage defines a WebSocket message type.
type WSMessage struct {
	// Params lists the message parameters.
	Params []string `yaml:"params,omitempty"`

	// Auth required to send this message.
	Auth string `yaml:"auth,omitempty"`

	// Handler is the action to execute when this message is received.
	Handler string `yaml:"handler,omitempty"`
}

// WebSocketConsumer defines an external WebSocket to connect to.
type WebSocketConsumer struct {
	// URL of the WebSocket server.
	URL string `yaml:"url"`

	// Reconnect indicates whether to auto-reconnect.
	Reconnect bool `yaml:"reconnect,omitempty"`

	// Auth for connection.
	Auth HTTPAuth `yaml:"auth,omitempty"`

	// OnMessage defines what to do when a message is received.
	OnMessage WSHandler `yaml:"on_message,omitempty"`

	// OnConnect defines what to do when connected.
	OnConnect WSHandler `yaml:"on_connect,omitempty"`

	// OnDisconnect defines what to do when disconnected.
	OnDisconnect WSHandler `yaml:"on_disconnect,omitempty"`
}

// WSHandler defines a WebSocket event handler.
type WSHandler struct {
	// Action to execute.
	Action string `yaml:"action,omitempty"`

	// Map defines how to map message data to action input.
	Map map[string]string `yaml:"map,omitempty"`

	// Broadcast indicates to broadcast to subscribers.
	Broadcast bool `yaml:"broadcast,omitempty"`

	// Channel to broadcast to.
	Channel string `yaml:"channel,omitempty"`
}

// --------------------------------------------------------------------------
// Webhook Channel
// --------------------------------------------------------------------------

// WebhookChannel defines webhook configuration.
type WebhookChannel struct {
	// Serve exposes webhook endpoints for others to call.
	Serve WebhookServe `yaml:"serve,omitempty"`

	// Consume processes incoming webhooks from external services.
	Consume map[string]WebhookConsumer `yaml:"consume,omitempty"`
}

// WebhookServe defines webhook endpoints this module exposes.
type WebhookServe struct {
	// Enabled indicates whether to expose webhook endpoints.
	Enabled bool `yaml:"enabled,omitempty"`

	// Path is the webhook endpoint path.
	Path string `yaml:"path,omitempty"`
}

// WebhookConsumer defines how to process external webhooks.
type WebhookConsumer struct {
	// Secret for webhook signature verification.
	Secret string `yaml:"secret,omitempty"`

	// Events maps external event names to handlers.
	Events map[string]WebhookHandler `yaml:"events,omitempty"`
}

// WebhookHandler defines how to handle an incoming webhook event.
type WebhookHandler struct {
	// Action to execute: "create", "update", "delete", or custom.
	Action string `yaml:"action"`

	// Lookup finds existing record to update/delete.
	Lookup map[string]string `yaml:"lookup,omitempty"`

	// Map defines how to map webhook payload to action input.
	Map map[string]string `yaml:"map,omitempty"`

	// Set defines fields to set directly.
	Set map[string]string `yaml:"set,omitempty"`

	// Then defines follow-up actions.
	Then []WebhookThen `yaml:"then,omitempty"`
}

// WebhookThen defines a follow-up action after webhook processing.
type WebhookThen struct {
	// Emit emits an internal event.
	Emit string `yaml:"emit,omitempty"`

	// Notify sends to a WebSocket channel.
	Notify string `yaml:"notify,omitempty"`

	// Call invokes an action.
	Call string `yaml:"call,omitempty"`
}

// --------------------------------------------------------------------------
// gRPC Channel
// --------------------------------------------------------------------------

// GRPCChannel defines gRPC configuration.
type GRPCChannel struct {
	// Serve exposes a gRPC service.
	Serve GRPCServe `yaml:"serve,omitempty"`

	// Consume calls external gRPC services.
	Consume map[string]GRPCConsumer `yaml:"consume,omitempty"`
}

// GRPCServe defines gRPC server configuration.
type GRPCServe struct {
	// Enabled indicates whether to expose gRPC service.
	Enabled bool `yaml:"enabled,omitempty"`

	// Service name override.
	Service string `yaml:"service,omitempty"`

	// Methods maps actions to gRPC method names.
	Methods map[string]GRPCMethod `yaml:"methods,omitempty"`
}

// GRPCMethod defines a gRPC method configuration.
type GRPCMethod struct {
	// Name of the gRPC method.
	Name string `yaml:"name"`

	// Stream type: "none", "server", "client", "bidirectional".
	Stream string `yaml:"stream,omitempty"`
}

// GRPCConsumer defines an external gRPC service to call.
type GRPCConsumer struct {
	// Target is the gRPC server address.
	Target string `yaml:"target"`

	// TLS configuration.
	TLS GRPCTLSConfig `yaml:"tls,omitempty"`

	// Methods defines callable methods.
	Methods map[string]GRPCClientMethod `yaml:"methods,omitempty"`
}

// GRPCTLSConfig defines TLS settings for gRPC.
type GRPCTLSConfig struct {
	// Enabled indicates whether to use TLS.
	Enabled bool `yaml:"enabled,omitempty"`

	// Insecure skips certificate verification.
	Insecure bool `yaml:"insecure,omitempty"`

	// CACert is the CA certificate path.
	CACert string `yaml:"ca_cert,omitempty"`
}

// GRPCClientMethod defines a gRPC client method.
type GRPCClientMethod struct {
	// Map defines how to map module fields to request.
	Map map[string]string `yaml:"map,omitempty"`

	// Response defines how to handle the response.
	Response map[string]string `yaml:"response,omitempty"`
}
