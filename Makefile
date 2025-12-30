.PHONY: build run test clean docker webui webui-install all

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)"

# Default: build everything
all: webui build

# Build webui and sync to embed location
webui:
	@echo "Building webui..."
	cd webui && npm run build
	@echo "Syncing webui to embed location..."
	rm -rf core/channel/http/webui/dist
	cp -r webui/dist core/channel/http/webui/
	@echo "Webui build complete"

# Install webui dependencies
webui-install:
	cd webui && npm install

# Build Go binary (embeds webui assets)
build:
	CGO_ENABLED=1 go build $(LDFLAGS) -o bin/apigate ./cmd/apigate

run: build
	./bin/apigate -config configs/apigate.example.yaml

dev:
	go run ./cmd/apigate -config configs/apigate.example.yaml

test:
	go test -v ./...

clean:
	rm -rf bin/
	rm -f apigate
	rm -rf webui/dist
	rm -rf core/channel/http/webui/dist

docker:
	docker build -t apigate:$(VERSION) .

docker-run:
	docker compose up --build

# Create a release
release:
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/apigate-linux-amd64 ./cmd/apigate
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/apigate-linux-arm64 ./cmd/apigate
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/apigate-darwin-amd64 ./cmd/apigate
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/apigate-darwin-arm64 ./cmd/apigate
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/apigate-windows-amd64.exe ./cmd/apigate

# Generate API key for testing
genkey:
	@openssl rand -hex 32 | sed 's/^/ak_/'

help:
	@echo "Available targets:"
	@echo "  all        - Build webui + Go binary (default)"
	@echo "  webui      - Build webui and sync to embed location"
	@echo "  webui-install - Install webui npm dependencies"
	@echo "  build      - Build Go binary only"
	@echo "  run        - Build and run"
	@echo "  dev        - Run with go run"
	@echo "  test       - Run tests"
	@echo "  docker     - Build Docker image"
	@echo "  docker-run - Run with docker-compose"
	@echo "  release    - Build for all platforms"
	@echo "  clean      - Remove build artifacts"
