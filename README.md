# Go API Gateway

Высокопроизводительный API-шлюз на Go с rate limiting, auth и load balancing.

## Архитектура

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Clients   │────▶│   Gateway   │────▶│  Backends   │
│             │     │             │     │  (services) │
└─────────────┘     └──────┬──────┘     └─────────────┘
                           │
                    ┌──────┴──────┐
                    │ Middleware  │
                    │ ─────────── │
                    │ Rate Limit  │
                    │ Auth (JWT)  │
                    │ Circuit Brk │
                    │ Load Balance│
                    └─────────────┘
```

## Возможности

### Rate Limiting
- Token bucket algorithm
- Per-client limits
- Sliding window counters
- Redis-backed distributed limits

### Authentication
- JWT token validation
- API key management
- OAuth2 integration
- Role-based access control

### Load Balancing
- Round-robin
- Weighted round-robin
- Least connections
- Consistent hashing

### Circuit Breaker
- State machine (closed → open → half-open)
- Configurable thresholds
- Automatic recovery
- Fallback responses

### Middleware Stack

```go
// Конфигурация middleware
gateway := NewGateway(Config{
    Middlewares: []Middleware{
        NewRateLimiter(100, 200),        // 100 RPS, burst 200
        NewAuthMiddleware(jwtSecret),    // JWT validation
        NewCircuitBreaker(5, 30*time.Second), // 5 failures → 30s open
        NewLoadBalancer(backends),       // Round-robin
    },
})
```

## Быстрый старт

```bash
# Запуск
go run cmd/gateway/main.go

# Тест
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/v1/resource
```

## Метрики

| Метрика | Описание |
|---------|----------|
| `gateway_requests_total` | Всего запросов |
| `gateway_rate_limited_total` | Отброшены по rate limit |
| `gateway_circuit_breaker_state` | Состояние circuit breaker |
| `gateway_latency_seconds` | Задержка шлюза |
