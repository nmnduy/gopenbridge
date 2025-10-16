# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Development Commands

```bash
# Build the Go binary
go build ./cmd/openbridge

# Run the proxy server
./openbridge --host 0.0.0.0 --port 8323

# Run Go tests (when tests are added)
go test ./...

# Run static analysis
go vet ./...
```

## Architecture Overview

OpenBridge is a dual-implementation API bridge (Python + Go) that translates Anthropic-style API calls to OpenAI-compatible endpoints. This allows Claude Code to work with any OpenAI-compatible LLM.

### Key Components

**Go Package** (`cmd/openbridge/`, `server/`, `proxy/`)
- Standalone binary with same CLI interface
- Standard library HTTP server with ReverseProxy
- Configuration via flags and environment variables

### Configuration Priority
1. Environment variables (highest priority)
2. YAML config file (`openbridge.yaml` or `openbridge.yml`)
3. Defaults and Hugging Face token detection

### HTTP Endpoints
- `/` - HTML home page showing server status
- `/health` - JSON health check endpoint
- `/v1/messages` - Main proxy endpoint for Anthropic-style chat calls

### Environment Variables
- `OPENAI_MODEL` - Model identifier (e.g., "zai-org/GLM-4.5:fireworks-ai")
- `OPENAI_BASE_URL` - API base URL
- `OPENAI_API_KEY` - API authentication key
