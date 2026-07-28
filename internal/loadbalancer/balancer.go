package loadbalancer

import (
	"errors"
	"net/http/httputil"
	"sync"
	"sync/atomic"
)

var ErrNoBackends = errors.New("no backends available")

type Backend struct {
	URL            *httputil.ReverseProxy
	Weight         int
	ActiveConns    int64
	Healthy        bool
}

type Balancer interface {
	Next() (*httputil.ReverseProxy, error)
	MarkActive(backend *httputil.ReverseProxy)
	MarkDone(backend *httputil.ReverseProxy)
}

type RoundRobinBalancer struct {
	backends []*Backend
	current  uint64
}

func NewRoundRobin(backends []*Backend) *RoundRobinBalancer {
	for _, b := range backends {
		b.Healthy = true
	}
	return &RoundRobinBalancer{backends: backends}
}

// Next выбирает следующий бэкенд по кругу, пропуская нездоровые.
// Используем атомарный счётчик вместо мьютекса для минимальной задержки.
func (rb *RoundRobinBalancer) Next() (*httputil.ReverseProxy, error) {
	if len(rb.backends) == 0 {
		return nil, ErrNoBackends
	}

	n := len(rb.backends)
	for i := 0; i < n; i++ {
		idx := atomic.AddUint64(&rb.current, 1) % uint64(n)
		b := rb.backends[idx]
		if b.Healthy {
			return b.URL, nil
		}
	}

	return nil, ErrNoBackends
}

func (rb *RoundRobinBalancer) MarkActive(backend *httputil.ReverseProxy) {
	for _, b := range rb.backends {
		if b.URL == backend {
			atomic.AddInt64(&b.ActiveConns, 1)
			return
		}
	}
}

func (rb *RoundRobinBalancer) MarkDone(backend *httputil.ReverseProxy) {
	for _, b := range rb.backends {
		if b.URL == backend {
			atomic.AddInt64(&b.ActiveConns, -1)
			return
		}
	}
}

type LeastConnectionsBalancer struct {
	backends []*Backend
	mu       sync.Mutex
}

func NewLeastConnections(backends []*Backend) *LeastConnectionsBalancer {
	for _, b := range backends {
		b.Healthy = true
	}
	return &LeastConnectionsBalancer{backends: backends}
}

// Next выбирает бэкенд с наименьшим числом активных соединений.
// Подходит для длинных запросов (websocket, streaming).
func (lb *LeastConnectionsBalancer) Next() (*httputil.ReverseProxy, error) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if len(lb.backends) == 0 {
		return nil, ErrNoBackends
	}

	var best *Backend
	bestConns := int64(-1)

	for _, b := range lb.backends {
		if !b.Healthy {
			continue
		}
		if best == nil || b.ActiveConns < bestConns {
			best = b
			bestConns = b.ActiveConns
		}
	}

	if best == nil {
		return nil, ErrNoBackends
	}

	return best.URL, nil
}

func (lb *LeastConnectionsBalancer) MarkActive(backend *httputil.ReverseProxy) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	for _, b := range lb.backends {
		if b.URL == backend {
			b.ActiveConns++
			return
		}
	}
}

func (lb *LeastConnectionsBalancer) MarkDone(backend *httputil.ReverseProxy) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	for _, b := range lb.backends {
		if b.URL == backend {
			b.ActiveConns--
			return
		}
	}
}
