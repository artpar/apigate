/*
Package schema defines the core types for declarative module definitions.

A module is a self-contained unit that owns its data (schema), operations (actions),
and communication channels (HTTP, CLI, WebSocket, etc.). Each channel can both
serve (expose endpoints) and consume (call external services).

# Module Definition

A minimal module definition in YAML:

	module: user

	schema:
	  email:    { type: email, unique: true, lookup: true }
	  password: { type: secret }
	  name:     { type: string }
	  status:   { type: enum, values: [pending, active, suspended], default: pending }
	  plan:     { type: ref, to: plan }

	actions:
	  activate:   { set: { status: active } }
	  deactivate: { set: { status: suspended } }

# Field Types

Supported field types:

  - string:    Text value
  - int:       Integer value
  - float:     Floating-point value
  - bool:      Boolean value
  - timestamp: Date/time value
  - duration:  Time duration (e.g., "30s", "1h")
  - json:      JSON object/array
  - bytes:     Binary data
  - email:     Email address (validated)
  - url:       URL (validated)
  - uuid:      UUID
  - enum:      One of a set of values (requires values field)
  - ref:       Reference to another module (requires to field)
  - secret:    Sensitive data, hashed, never exposed
  - strings:   Array of strings
  - ints:      Array of integers

# Actions

Every module has implicit CRUD actions: list, get, create, update, delete.
Custom actions can be defined to modify specific fields:

	actions:
	  activate:   { set: { status: active } }
	  deactivate: { set: { status: suspended } }

# Channels

Channels define how a module communicates. Each channel supports both
serving (exposing endpoints) and consuming (calling external services):

	channels:
	  http:
	    serve: true                    # Expose REST API
	    consume:                       # Call external APIs
	      stripe:
	        base: https://api.stripe.com/v1
	        auth: { bearer: ${STRIPE_KEY} }

	  cli:
	    serve: true                    # Expose CLI commands
	    consume:                       # Call external tools
	      remote:
	        command: ssh server

	  websocket:
	    serve:
	      path: /ws
	      events: [created, updated]
	    consume:
	      feed:
	        url: wss://external/feed

	  webhook:
	    serve:
	      path: /webhooks/user
	    consume:
	      stripe:
	        events:
	          customer.created: { action: create, map: {...} }

# Path Ownership

Each module claims paths through its channel definitions. The registry
detects conflicts when multiple modules claim the same path:

  - HTTP:      /users, /users/{id}
  - CLI:       users list, users create
  - WebSocket: /ws/users
  - Webhook:   /webhooks/user

# Parsing

Load modules from YAML:

	mod, err := schema.ParseFile("modules/user.yaml")
	modules, err := schema.ParseDir("modules/")

All modules are validated on parse. Invalid modules return an error.
*/
package schema
