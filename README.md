# Go API Gateway

High-performance API gateway in Go with rate limiting, JWT auth, circuit breaker, and load balancing.

## Architecture

```
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│                              GO API GATEWAY                                                   │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                              │
│   CLIENT LAYER                     GATEWAY LAYER                       BACKEND SERVICES     │
│                                                                                              │
│   ┌──────────────┐               ┌────────────────────────┐          ┌──────────────────┐    │
│   │              │               │                        │          │                  │    │
│   │  Web Apps    │──────────────▶│    Reverse Proxy       │─────────▶│  Backend 1       │    │
│   │              │               │    (net/http)          │          │  :9001           │    │
│   └──────────────┘               │                        │          │  weight: 1       │    │
│                                  │  ┌──────────────────┐ │          └──────────────────┘    │
│   ┌──────────────┐               │  │                  │ │                                   │
│   │              │──────────────▶│  │  MIDDLEWARE      │ │          ┌──────────────────┐    │
│   │  Mobile Apps │               │  │  PIPELINE        │ │          │                  │    │
│   │              │               │  │                  │ │          │  Backend 2       │    │
│   └──────────────┘               │  │  ┌────────────┐ │ │          │  :9002           │    │
│                                  │  │  │    1.      │ │ │          │  weight: 1       │    │
│   ┌──────────────┐               │  │  │ Rate Limit │─┼─┤── PASS ─▶└──────────────────┘    │
│   │              │──────────────▶│  │  │ Token      │ │ │          ┌──────────────────┐    │
│   │  3rd Party   │               │  │  │ Bucket     │ │ │          │                  │    │
│   │  APIs        │               │  │  └────────────┘ │ │          │  Backend 3       │    │
│   │              │               │  │        │         │ │          │  :9003           │    │
│   └──────────────┘               │  │        ▼         │ │          │  weight: 2       │    │
│                                  │  │  ┌────────────┐ │ │          └──────────────────┘    │
│   ┌──────────────┐               │  │  │    2.      │ │ │                                   │
│   │   Micro-     │──────────────▶│  │  │ JWT Auth   │─┼─┤── DENY ──▶ 401 Unauthorized      │
│   │   services   │               │  │  │ Validate   │ │ │                                   │
│   └──────────────┘               │  │  │ Algorithms │ │ │                                   │
│                                  │  │  └────────────┘ │ │                                   │
│                                  │  │        │         │ │          ┌──────────────────┐    │
│                                  │  │        ▼         │ │          │                  │    │
│                                  │  │  ┌────────────┐ │ │          │   Prometheus     │    │
│                                  │  │  │    3.      │ │ │          │   :8080/metrics  │    │
│                                  │  │  │ Circuit    │─┼─┤──────────│─▶                │    │
│                                  │  │  │ Breaker    │ │ │          │   gateway_        │    │
│                                  │  │  │            │ │ │          │   requests_total  │    │
│                                  │  │  │ CLOSED ───▶│─┼─┤          │   rate_limited_   │    │
│                                  │  │  │    │       │ │ │          │   total           │    │
│                                  │  │  │    ▼       │ │ │          │   latency_seconds │    │
│                                  │  │  │ OPEN ─────▶│─┼─┤── REJECT▶│   circuit_        │    │
│                                  │  │  │    │       │ │ │          │   breaker_state   │    │
│                                  │  │  │    ▼       │ │ │          │                  │    │
│                                  │  │  │ HALF-OPEN  │ │ │          └──────────────────┘    │
│                                  │  │  └────────────┘ │ │                                   │
│                                  │  │        │         │ │                                   │
│                                  │  │        ▼         │ │                                   │
│                                  │  │  ┌────────────┐ │ │                                   │
│                                  │  │  │    4.      │ │ │                                   │
│                                  │  │  │ Load       │─┼─┘                                   │
│                                  │  │  │ Balancer   │ │                                     │
│                                  │  │  │            │ │                                     │
│                                  │  │  │ Round      │ │                                     │
│                                  │  │  │ Robin  ────┼─┼─────▶ Backends                      │
│                                  │  │  │            │ │                                     │
│                                  │  │  │ Least Conn │ │                                     │
│                                  │  │  │ ───────────┼─┼─────▶ Backends                      │
│                                  │  │  └────────────┘ │                                     │
│                                  │  └──────────────────┘                                     │
│                                  │  :8080                                                    │
│                                  └──────────────────────────────────────────────────────────┘
│                                                                                              │
│   ┌─────────────────────────────────────────────────────────────────────────────────────────┐ │
│   │                              REQUEST FLOW                                               │ │
│   │                                                                                         │ │
│   │   Client ──▶ Gateway ──▶ Rate Limit ──▶ JWT Auth ──▶ Circuit Breaker ──▶ Load Balancer │ │
│   │      │                                                     │                │           │ │
│   │      │                                                     │                │           │ │
│   │      │         ◀── Response ◀── Backend ◀─────────────────┘────────────────┘           │ │
│   │      │                                                                                │ │
│   │      └── Metrics ──▶ Prometheus ──▶ Grafana                                            │ │
│   └─────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                              │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Features

- **Rate Limiting** — Token bucket per client, sliding window
- **JWT Authentication** — Token validation with configurable algorithms
- **Circuit Breaker** — Closed → Open → Half-Open state machine, configurable threshold and timeout
- **Load Balancing** — Round-robin and least connections strategies
- **Prometheus Metrics** — `gateway_requests_total`, `gateway_rate_limited_total`, `gateway_latency_seconds`, `gateway_circuit_breaker_state`
- **Graceful Shutdown** — SIGINT/SIGTERM handling

## Quick Start

```bash
# Run
go run cmd/server/main.go -config config.yaml

# Test
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/v1/resource

# Metrics
curl http://localhost:8080/metrics
```

## Configuration

Edit `config.yaml`:

```yaml
listen_addr: ":8080"
server:
  read_timeout: 30
  write_timeout: 30
backends:
  - url: "http://localhost:9001"
    weight: 1
  - url: "http://localhost:9002"
    weight: 1
jwt:
  secret: "your-secret-key"
  issuer: "go-api-gateway"
  allowed_algorithms: ["HS256"]
rate_limit:
  rps: 100
  burst: 200
circuit_breaker:
  threshold: 5
  timeout: 30
metrics:
  enabled: true
  path: "/metrics"
```

## Docker

```bash
docker build -t go-api-gateway .
docker run -p 8080:8080 go-api-gateway
```

## Testing

```bash
go test ./...
```

## Project Structure

```
cmd/server/main.go            — Entry point, HTTP server, graceful shutdown
internal/auth/jwt.go          — JWT validation middleware
internal/circuitbreaker/      — Circuit breaker (closed/open/half-open)
internal/config/config.go     — YAML config loading
internal/loadbalancer/        — RoundRobin and LeastConnections
internal/metrics/metrics.go   — Prometheus metrics
internal/proxy/proxy.go       — Reverse proxy with load balancer
internal/ratelimit/limiter.go — Token bucket rate limiter
```
