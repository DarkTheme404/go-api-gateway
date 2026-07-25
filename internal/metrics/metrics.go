package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_requests_total",
			Help: "Total number of requests processed by the gateway",
		},
		[]string{"method", "path", "status"},
	)

	RateLimitedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_rate_limited_total",
			Help: "Total number of requests rejected by rate limiter",
		},
		[]string{"client_id"},
	)

	LatencyHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_latency_seconds",
			Help:    "Request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	CircuitBreakerState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gateway_circuit_breaker_state",
			Help: "Current state of circuit breaker (0=closed, 1=open, 2=half-open)",
		},
		[]string{"name"},
	)
)

func init() {
	prometheus.MustRegister(RequestsTotal, RateLimitedTotal, LatencyHistogram, CircuitBreakerState)
}

type RequestDuration struct {
	histogram *prometheus.HistogramVec
}

func NewRequestDuration() RequestDuration {
	return RequestDuration{
		histogram: LatencyHistogram,
	}
}

func (rd RequestDuration) Observe(seconds float64) {
	rd.histogram.WithLabelValues("", "").Observe(seconds)
}

func Handler() http.Handler {
	return promhttp.Handler()
}
