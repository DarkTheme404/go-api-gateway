package ratelimit

import (
	"sync"
	"time"
)

// Limiter — токен-бакет rate limiter.
// Каждый клиент имеет свой лимит токенов.
type Limiter struct {
	mu       sync.Mutex
	clients  map[string]*clientLimit
	rate     float64 // токенов в секунду
	burst    int     // максимальный размер бакета
	cleanup  time.Duration
	stopChan chan struct{}
}

type clientLimit struct {
	tokens   float64
	lastTime time.Time
}

// NewLimiter создаёт новый rate limiter.
func NewLimiter(rate float64, burst int) *Limiter {
	l := &Limiter{
		clients:  make(map[string]*clientLimit),
		rate:     rate,
		burst:    burst,
		cleanup:  time.Minute,
		stopChan: make(chan struct{}),
	}
	go l.cleanupLoop()
	return l
}

// Allow проверяет, разрешён ли запрос для данного клиента.
// Алгоритм token bucket: при каждом запросе токены пополняются
// пропорционально прошедшему времени, затем消費ляется один токен.
func (l *Limiter) Allow(clientID string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	client, exists := l.clients[clientID]
	if !exists {
		client = &clientLimit{
			tokens:   float64(l.burst),
			lastTime: now,
		}
		l.clients[clientID] = client
	}

	// Пополняем токены пропорционально прошедшему времени, но не больше burst
	elapsed := now.Sub(client.lastTime).Seconds()
	client.tokens += elapsed * l.rate
	if client.tokens > float64(l.burst) {
		client.tokens = float64(l.burst)
	}
	client.lastTime = now

	if client.tokens >= 1 {
		client.tokens--
		return true
	}

	return false
}

// cleanupLoop периодически удаляет неактивных клиентов.
func (l *Limiter) cleanupLoop() {
	ticker := time.NewTicker(l.cleanup)
	defer ticker.Stop()

	for {
		select {
		case <-l.stopChan:
			return
		case <-ticker.C:
			l.cleanupStaleClients()
		}
	}
}

func (l *Limiter) cleanupStaleClients() {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := time.Now().Add(-2 * l.cleanup)
	for id, client := range l.clients {
		if client.lastTime.Before(cutoff) {
			delete(l.clients, id)
		}
	}
}

// Stop останавливает cleanup loop.
func (l *Limiter) Stop() {
	close(l.stopChan)
}

// Stats возвращает статистику по клиентам.
func (l *Limiter) Stats() (total, active int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	total = len(l.clients)
	cutoff := time.Now().Add(-time.Minute)
	for _, client := range l.clients {
		if client.lastTime.After(cutoff) {
			active++
		}
	}
	return
}
