# LLM Chat Service API

A lightweight Go web service that provides an anonymous chat interface powered by Groq's LLM API. Features streaming responses via SSE and WebSocket, in-memory conversation history management, and Redis-based token caching.

## Features

- **Streaming Support**: Server-Sent Events (SSE) and WebSocket support for real-time responses
- **Conversation History**: In-memory storage with automatic trimming to last 20 exchanges
- **Token Caching**: Redis-based cache to avoid recomputing token counts
- **Structured Logging**: JSON logging with request/response tracking
- **Metrics**: Prometheus metrics endpoint for monitoring
- **Concurrency Safe**: Handles 50+ simultaneous requests with mutex-protected state
- **Health Checks**: Simple health endpoint for monitoring

## Architecture

The service follows a clean, layered architecture:

- **API Layer** (`internal/api/`): HTTP handlers, middleware, and routing
- **Service Layer** (`internal/service/`): Business logic for chat processing
- **Storage Layer** (`internal/storage/`): In-memory history and Redis cache
- **LLM Layer** (`internal/llm/`): Groq API client with streaming support
- **Config/Logging**: Environment-based configuration and structured logging

**Trade-offs**:
- In-memory storage is simple and fast but ephemeral (lost on restart)
- Redis adds resilience for caching but introduces deployment complexity
- Single global conversation simplifies architecture but limits scalability

## Prerequisites

- Go 1.21+
- Docker and Docker Compose (for containerized deployment)
- Groq API key ([Get one here](https://console.groq.com/))

## Quick Start

### Using Docker Compose (Recommended)

1. Clone the repository:
```bash
git clone https://github.com/AmmrFX/LLM-Chat-Service-API
cd llm-chat-service
```

2. Set your Groq API key:
```bash
export GROQ_API_KEY=
```

3. Start the services:
```bash
docker compose up --build
```

The service will be available at `http://localhost:8000`

### Local Development

1. Install dependencies:
```bash
go mod download
```

2. Start Redis (optional, for token caching):
```bash
docker run -d -p 6379:6379 redis:7-alpine
```

3. Set environment variables:
```bash
export GROQ_API_KEY=
export PORT=8000
export REDIS_ADDR=localhost:6379
export MAX_TOKENS=1024
export MAX_EXCHANGES=20
```

4. Run the service:
```bash
go run cmd/main.go
```

## API Documentation


### Chat (JSON Response)

```bash
curl -X POST http://localhost:8000/chat \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {"role": "user", "content": "Hello, how are you?"}
    ],
    "stream": false
  }'
```

### Chat (SSE Streaming)

```bash
curl -X POST http://localhost:8000/chat \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "messages": [
      {"role": "user", "content": "Tell me a short story"}
    ],
    "stream": true
  }'
```

### Chat (WebSocket)

Connect via WebSocket and send:
```json
{
  "messages": [
    {"role": "user", "content": "Hello"}
  ],
  "stream": true
}
```

Receive streaming tokens as JSON messages.

### Metrics

```bash
curl http://localhost:8000/metrics
```

Returns Prometheus metrics.

## Request Format

```json
{
  "messages": [
    {"role": "user", "content": "Your message here"},
    {"role": "assistant", "content": "Previous response"},
    {"role": "user", "content": "Follow-up question"}
  ],
  "stream": true
}
```

**Validation Rules**:
- Messages array cannot be empty
- Roles must be exactly "user" or "assistant" (case-sensitive)
- Content cannot be empty
- Last message must be from "user"

## Response Format

**Non-streaming**:
```json
{
  "response": "Full response text"
}
```

**SSE Streaming**:
```
data: token1
data: token2
data: [DONE]
```

**WebSocket**:
```json
{"token": "token1"}
{"token": "token2"}
{"done": "true"}
```

## Error Responses

```json
{
  "error": "Error message description"
}
```

Status codes:
- `400`: Bad Request (validation errors)
- `502`: Bad Gateway (LLM API errors)
- `500`: Internal Server Error

## Testing

Run unit tests:
```bash
go test ./...
```

Run with coverage:
```bash
go test -cover ./...
```

Run integration tests (requires GROQ_API_KEY):
```bash
go test -tags=integration ./tests/...
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8000` | HTTP server port |
| `GROQ_API_KEY` | *required* | Groq API key |
| `REDIS_ADDR` | `redis:6379` | Redis address |
| `REDIS_PASSWORD` | `` | Redis password |
| `MAX_TOKENS` | `1024` | Maximum tokens per request |
| `MAX_EXCHANGES` | `20` | Maximum conversation exchanges to keep |

## Project Structure

```
llm-chat-service/
├── cmd/
│   └── main.go              # Application entry point
├── internal/
│   ├── api/                 # HTTP handlers, middleware, routing
│   ├── service/             # Business logic
│   ├── storage/             # Memory and Redis storage
│   ├── llm/                 # Groq API client
│   ├── config/              # Configuration loading
│   └── logging/             # Structured logging
├── tests/                   # Integration tests
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
├── openapi.yaml
└── README.md
```

## License

MIT

