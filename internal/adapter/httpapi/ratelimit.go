package httpapi

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	perIPRate  = 1 // requêtes par seconde
	perIPBurst = 5
	idleTTL    = 10 * time.Minute
)

type limiter struct {
	mu      sync.Mutex
	clients map[string]*client
}

type client struct {
	lim  *rate.Limiter
	seen time.Time
}

func withRateLimit(next http.Handler) http.Handler {
	l := &limiter{clients: make(map[string]*client)}
	go l.cleanup()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Les sondes de disponibilité et de métriques ne sont pas limitées :
		// elles ne coûtent rien et doivent rester joignables sous charge.
		if r.URL.Path == "/healthz" || r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		if !l.allow(clientIP(r)) {
			w.Header().Set("Retry-After", "1")
			http.Error(w, `{"error":"trop de requêtes"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *limiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	c, ok := l.clients[ip]
	if !ok {
		c = &client{lim: rate.NewLimiter(perIPRate, perIPBurst)}
		l.clients[ip] = c
	}
	c.seen = time.Now()
	return c.lim.Allow()
}

func (l *limiter) cleanup() {
	for range time.Tick(idleTTL) {
		l.mu.Lock()
		for ip, c := range l.clients {
			if time.Since(c.seen) > idleTTL {
				delete(l.clients, ip)
			}
		}
		l.mu.Unlock()
	}
}

func clientIP(r *http.Request) string {
	// Derrière Caddy, l'adresse réelle arrive dans X-Forwarded-For.
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		if host, _, err := net.SplitHostPort(fwd); err == nil {
			return host
		}
		return fwd
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
