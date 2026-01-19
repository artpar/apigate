# Protocols

APIGate supports multiple protocols for different use cases.

---

## HTTP (Default)

Standard request/response pattern:

```bash
apigate routes create \
  --path "/api/v1/*" \
  --upstream backend \
  --protocol http
```

- Buffered request/response
- Full transform support
- Usage metering on completion

---

## HTTP Streaming

Streaming responses without buffering:

```bash
apigate routes update <id> --protocol http_stream
```

- Response streamed to client
- No response body transforms
- Usage metered by bytes or duration

---

## Server-Sent Events (SSE)

For event streams:

```bash
apigate routes update <id> --protocol sse
```

- Long-lived connections
- Event-based metering available
- Automatic reconnection support

### SSE Metering

```yaml
metering_expr: "sseEvents"  # Count events
metering_expr: "responseBytes"  # Count bytes
```

---

## WebSocket

Bidirectional communication:

```bash
apigate routes update <id> --protocol websocket
```

- Full duplex messaging
- Connection-level auth
- Message-based metering

---

## Protocol Selection

| Protocol | Use Case | Transforms | Metering |
|----------|----------|------------|----------|
| `http` | REST APIs | Full | Request-based |
| `http_stream` | File downloads | Headers only | Byte-based |
| `sse` | Real-time updates | Headers only | Event-based |
| `websocket` | Chat, gaming | Headers only | Message-based |

---

## See Also

- [[Routes]] - Route configuration
- [[Proxying]] - Proxy overview
- [[Transformations]] - Transform support by protocol
