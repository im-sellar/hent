package httpapi

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	perIPRate  = 1 // requêtes par seconde
	perIPBurst = 5
	idleTTL    = 10 * time.Minute

	// Plafond du nombre de clients suivis simultanément. Dimensionné très
	// au-dessus d'un trafic légitime pour ce service, et très en dessous de
	// ce qui mettrait la mémoire en péril.
	maxTrackedClients = 50_000
)

type limiter struct {
	mu      sync.Mutex
	clients map[string]*client
}

type client struct {
	lim  *rate.Limiter
	seen time.Time
}

func withRateLimit(next http.Handler, trustedProxies map[string]struct{}) http.Handler {
	l := &limiter{clients: make(map[string]*client)}
	go l.cleanup()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Les sondes de disponibilité et de métriques ne sont pas limitées :
		// elles ne coûtent rien et doivent rester joignables sous charge.
		if r.URL.Path == "/healthz" || r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		if !l.allow(clientIP(r, trustedProxies)) {
			w.Header().Set("Retry-After", "1")
			http.Error(w, `{"error":"trop de requêtes"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// allow décide du sort d'une requête et borne la taille de la table.
//
// Le plafond n'est pas cosmétique : chaque entrée porte un rate.Limiter, et
// sans borne un client qui fait varier son adresse — ou son en-tête, si un
// proxy de confiance est mal configuré — fait enfler la map jusqu'à la
// mémoire disponible. Le nettoyage périodique ne suffit pas : il ne passe que
// toutes les dix minutes, et tout ce qui arrive entre deux passages
// s'accumule.
//
// Au-delà du plafond, on refuse plutôt que d'allouer : mieux vaut dégrader le
// service pour des clients inconnus que tomber pour tout le monde.
func (l *limiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	c, ok := l.clients[ip]
	if !ok {
		if len(l.clients) >= maxTrackedClients {
			return false
		}
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

// clientIP identifie le client pour la limitation de débit.
//
// X-Forwarded-For n'est lu QUE si la connexion vient d'un proxy déclaré de
// confiance. C'est une règle de sécurité, pas une commodité : n'importe quel
// client peut envoyer cet en-tête, et un reverse proxy l'AJOUTE à ce qu'il a
// reçu au lieu de l'écraser. Lui faire confiance sans condition revient à
// offrir une clé de limiteur neuve à chaque requête — soit à supprimer la
// limitation tout en croyant l'avoir.
//
// Quand l'en-tête est digne de confiance, c'est sa DERNIÈRE entrée qu'on
// retient : les précédentes ont été fournies par le client.
func clientIP(r *http.Request, trustedProxies map[string]struct{}) string {
	remote := hostOf(r.RemoteAddr)

	if _, trusted := trustedProxies[remote]; trusted {
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			parts := strings.Split(fwd, ",")
			return hostOf(strings.TrimSpace(parts[len(parts)-1]))
		}
	}
	return remote
}

func hostOf(addr string) string {
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}
