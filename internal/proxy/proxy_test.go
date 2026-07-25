package proxy

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"
	"time"

	"github.com/DarkTheme404/go-api-gateway/internal/circuitbreaker"
	"github.com/DarkTheme404/go-api-gateway/internal/loadbalancer"
)

func TestGatewayProxy_NoBackends(t *testing.T) {
	cb := circuitbreaker.New(5, time.Second)
	bal := loadbalancer.NewRoundRobin([]*loadbalancer.Backend{})
	gp := New(bal, cb)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	gp.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", rec.Code)
	}
}

func TestGatewayProxy_ForwardsToBackend(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("from backend"))
	}))
	defer backend.Close()

	u, _ := url.Parse(backend.URL)
	rp := httputil.NewSingleHostReverseProxy(u)
	backends := []*loadbalancer.Backend{{URL: rp, Weight: 1}}

	bal := loadbalancer.NewRoundRobin(backends)
	cb := circuitbreaker.New(5, time.Second)
	gp := New(bal, cb)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	gp.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "from backend" {
		t.Fatalf("expected 'from backend', got '%s'", rec.Body.String())
	}
}
