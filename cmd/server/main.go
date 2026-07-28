package main

import (
	"context"
	"flag"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DarkTheme404/go-api-gateway/internal/auth"
	"github.com/DarkTheme404/go-api-gateway/internal/circuitbreaker"
	"github.com/DarkTheme404/go-api-gateway/internal/config"
	"github.com/DarkTheme404/go-api-gateway/internal/loadbalancer"
	"github.com/DarkTheme404/go-api-gateway/internal/metrics"
	"github.com/DarkTheme404/go-api-gateway/internal/proxy"
	"github.com/DarkTheme404/go-api-gateway/internal/ratelimit"
	"go.uber.org/zap"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	logger, _ := zap.NewProduction()
	defer logger.Sync()
	sugar := logger.Sugar()

	cfg, err := config.Load(*configPath)
	if err != nil {
		sugar.Warnf("failed to load config, using defaults: %v", err)
		cfg = config.DefaultConfig()
	}

	backends := make([]*loadbalancer.Backend, 0, len(cfg.Backends))
	for _, b := range cfg.Backends {
		u, err := url.Parse(b.URL)
		if err != nil {
			sugar.Errorf("invalid backend URL %s: %v", b.URL, err)
			continue
		}
		rp := httputil.NewSingleHostReverseProxy(u)
		weight := b.Weight
		if weight <= 0 {
			weight = 1
		}
		backends = append(backends, &loadbalancer.Backend{URL: rp, Weight: weight})
	}

	if len(backends) == 0 {
		sugar.Fatal("no valid backends configured")
	}

	lb := loadbalancer.NewRoundRobin(backends)
	cb := circuitbreaker.New(cfg.CircuitBreaker.Threshold, time.Duration(cfg.CircuitBreaker.Timeout)*time.Second)
	limiter := ratelimit.NewLimiter(cfg.RateLimit.RPS, cfg.RateLimit.Burst)
	defer limiter.Stop()

	gwProxy := proxy.New(lb, cb)

	mux := http.NewServeMux()

	var jwtValidator *auth.JWTValidator
	if cfg.JWT.Secret != "" {
		jwtValidator = auth.NewJWTValidator(cfg.JWT.Secret, cfg.JWT.Issuer, cfg.JWT.AllowedAlg)
	}

	if cfg.Metrics.Enabled {
		mux.Handle(cfg.Metrics.Path, metrics.Handler())
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		clientID := r.RemoteAddr
		if !limiter.Allow(clientID) {
			metrics.RateLimitedTotal.WithLabelValues(clientID).Inc()
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}

		var h http.Handler = gwProxy
		if jwtValidator != nil {
			h = jwtValidator.Middleware(h)
		}
		h.ServeHTTP(w, r)

		elapsed := time.Since(start).Seconds()
		metrics.LatencyHistogram.WithLabelValues(r.Method, r.URL.Path).Observe(elapsed)
		metrics.RequestsTotal.WithLabelValues(r.Method, r.URL.Path, "200").Inc()
	})

	mux.Handle("/", handler)

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      mux,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	go func() {
		sugar.Infof("gateway listening on %s", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			sugar.Fatalf("listen error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	sugar.Info("shutting down gateway...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		sugar.Fatalf("forced shutdown: %v", err)
	}
	sugar.Info("gateway stopped")
}
