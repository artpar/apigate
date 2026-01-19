# Installation

Deploy APIGate in your environment.

---

## Requirements

| Requirement | Minimum | Recommended |
|-------------|---------|-------------|
| **OS** | Linux, macOS, Windows | Linux (Ubuntu 22.04+) |
| **Memory** | 256 MB | 512 MB+ |
| **Disk** | 100 MB | 1 GB+ (for usage data) |
| **Go** | 1.21+ (for building) | Latest |

---

## Option 1: Download Binary

Download the latest release for your platform:

```bash
# Linux (amd64)
curl -L https://github.com/artpar/apigate/releases/latest/download/apigate-linux-amd64 -o apigate
chmod +x apigate

# macOS (Intel)
curl -L https://github.com/artpar/apigate/releases/latest/download/apigate-darwin-amd64 -o apigate
chmod +x apigate

# macOS (Apple Silicon)
curl -L https://github.com/artpar/apigate/releases/latest/download/apigate-darwin-arm64 -o apigate
chmod +x apigate
```

---

## Option 2: Build from Source

```bash
# Clone repository
git clone https://github.com/artpar/apigate.git
cd apigate

# Build
go build -o apigate ./cmd/apigate

# Verify
./apigate version
```

---

## Option 3: Docker

```bash
# Pull image
docker pull artpar/apigate:latest

# Run with volume for data persistence
docker run -d \
  --name apigate \
  -p 8080:8080 \
  -v apigate-data:/data \
  artpar/apigate:latest
```

### Docker Compose

```yaml
version: '3.8'

services:
  apigate:
    image: artpar/apigate:latest
    ports:
      - "8080:8080"
    volumes:
      - apigate-data:/data
    environment:
      - APIGATE_DATABASE_DSN=/data/apigate.db
      - APIGATE_SERVER_PORT=8080

volumes:
  apigate-data:
```

---

## Initial Setup

After installation, run the setup wizard:

```bash
# Interactive setup
./apigate init

# Or start the server and use web setup
./apigate serve
# Then visit http://localhost:8080
```

The setup wizard will:
1. Create the database
2. Set up the admin account
3. Configure basic settings

---

## Verify Installation

```bash
# Check version
./apigate version

# Check health
curl http://localhost:8080/health

# Expected response:
# {"status":"healthy","database":"connected"}
```

---

## Directory Structure

After running, APIGate creates:

```
./
├── apigate.db          # SQLite database
├── apigate             # Binary
└── logs/               # Log files (if configured)
```

---

## Next Steps

- [[Quick-Start]] - Set up your first API proxy
- [[Configuration]] - Configure environment variables
- [[First-Customer]] - Onboard your first customer
