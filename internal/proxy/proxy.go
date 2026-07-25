package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/DarkTheme404/go-api-gateway/internal/circuitbreaker"
	"github.com/DarkTheme404/go-api-gateway/internal/loadbalancer"
	"github.com/DarkTheme404/go-api-gateway/internal/metrics"
)

type GatewayProxy struct {
	balancer        loadbalancer.Balancer
	breaker         *circuitbreaker.CircuitBreaker
	requestDuration metrics.RequestDuration
}

func New(balancer loadbalancer.Balancer, breaker *circuitbreaker.CircuitBreaker) *GatewayProxy {
	return &GatewayProxy{
		balancer:        balancer,
		breaker:         breaker,
		requestDuration: metrics.NewRequestDuration(),
	}
}

func (gp *GatewayProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	backend, err := gp.balancer.Next()
	if err != nil {
		http.Error(w, `{"error":"no backends available"}`, http.StatusBadGateway)
		return
	}

	err = gp.breaker.Execute(func() error {
		gp.balancer.MarkActive(backend)
		defer gp.balancer.MarkDone(backend)

		start := time.Now()
		director := func(req *http.Request) {
			backend.Director(req)
		}
		proxy := &httputil.ReverseProxy{
			Director:  director,
			Transport: http.DefaultTransport,
			ModifyResponse: func(resp *http.Response) error {
				elapsed := time.Since(start).Seconds()
				gp.requestDuration.Observe(elapsed)
				return nil
			},
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				log.Printf("proxy error: %v", err)
				http.Error(w, `{"error":"bad gateway"}`, http.StatusBadGateway)
			},
		}
		proxy.ServeHTTP(w, r)
		return nil
	})
	if err != nil {
		http.Error(w, `{"error":"service unavailable"}`, http.StatusServiceUnavailable)
	}
}
