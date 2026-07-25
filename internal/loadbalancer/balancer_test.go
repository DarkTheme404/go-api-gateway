package loadbalancer

import (
	"net/http/httputil"
	"net/url"
	"testing"
)

func makeBackends(count int) []*Backend {
	backends := make([]*Backend, count)
	for i := 0; i < count; i++ {
		u, _ := url.Parse("http://localhost:900" + string(rune('0'+i)))
		rp := httputil.NewSingleHostReverseProxy(u)
		backends[i] = &Backend{URL: rp, Weight: 1}
	}
	return backends
}

func TestRoundRobin_DistributesEvenly(t *testing.T) {
	backends := makeBackends(3)
	rb := NewRoundRobin(backends)

	seen := make(map[*httputil.ReverseProxy]int)
	for i := 0; i < 9; i++ {
		b, err := rb.Next()
		if err != nil {
			t.Fatal(err)
		}
		seen[b]++
	}

	for _, b := range backends {
		if seen[b.URL] != 3 {
			t.Fatalf("expected 3 calls to each backend, got %v", seen)
		}
	}
}

func TestRoundRobin_NoBackends(t *testing.T) {
	rb := NewRoundRobin([]*Backend{})
	_, err := rb.Next()
	if err != ErrNoBackends {
		t.Fatalf("expected ErrNoBackends, got %v", err)
	}
}

func TestRoundRobin_SkipsUnhealthy(t *testing.T) {
	backends := makeBackends(3)
	backends[0].Healthy = false
	backends[1].Healthy = false
	rb := NewRoundRobin(backends)

	for i := 0; i < 5; i++ {
		b, err := rb.Next()
		if err != nil {
			t.Fatal(err)
		}
		if b != backends[2].URL {
			t.Fatalf("expected backend 2, got different")
		}
	}
}

func TestLeastConnections_PicksLowest(t *testing.T) {
	backends := makeBackends(3)
	lb := NewLeastConnections(backends)

	backends[0].ActiveConns = 5
	backends[1].ActiveConns = 2
	backends[2].ActiveConns = 8

	b, err := lb.Next()
	if err != nil {
		t.Fatal(err)
	}
	if b != backends[1].URL {
		t.Fatal("expected least connections backend")
	}
}

func TestLeastConnections_NoBackends(t *testing.T) {
	lb := NewLeastConnections([]*Backend{})
	_, err := lb.Next()
	if err != ErrNoBackends {
		t.Fatalf("expected ErrNoBackends, got %v", err)
	}
}

func TestMarkActiveAndDone(t *testing.T) {
	backends := makeBackends(1)
	rb := NewRoundRobin(backends)

	b, _ := rb.Next()
	rb.MarkActive(b)
	rb.MarkActive(b)

	if backends[0].ActiveConns != 2 {
		t.Fatalf("expected 2 active, got %d", backends[0].ActiveConns)
	}

	rb.MarkDone(b)
	if backends[0].ActiveConns != 1 {
		t.Fatalf("expected 1 active, got %d", backends[0].ActiveConns)
	}
}
