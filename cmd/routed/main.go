// Command routed sert le générateur de boucles. Il charge un artefact
// graph.bin au démarrage et ne fait plus, ensuite, que router : ni base de
// données, ni lecture de fichier OSM, ni calcul géospatial en ligne.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/im-sellar/hent/internal/adapter/httpapi"
	"github.com/im-sellar/hent/internal/adapter/network/csr"
	"github.com/im-sellar/hent/internal/app/generateloop"
	"github.com/im-sellar/hent/internal/app/port"
)

// Le graphe CSR satisfait le port réseau. C'est ici, au câblage, que la
// vérification a lieu : le paquet csr ignore l'existence du port, et le
// paquet port ignore l'existence de csr.
var _ port.RouteNetwork = (*csr.Graph)(nil)

func main() {
	graphPath := flag.String("graph", "graph.bin", "artefact produit par graphbuild")
	addr := flag.String("addr", ":8080", "adresse d'écoute")
	// Vide par défaut : sans configuration explicite, X-Forwarded-For est
	// ignoré et seule l'adresse de connexion compte pour la limitation de
	// débit. Ne renseigner que les adresses des reverse proxies effectivement
	// placés devant ce service.
	trustedProxies := flag.String("trusted-proxies", "",
		"adresses de connexion (sans port) autorisées à fournir X-Forwarded-For, séparées par des virgules")
	flag.Parse()

	if err := run(*graphPath, *addr, *trustedProxies); err != nil {
		log.Fatal(err)
	}
}

func parseTrustedProxies(s string) map[string]struct{} {
	proxies := make(map[string]struct{})
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			proxies[p] = struct{}{}
		}
	}
	return proxies
}

func run(graphPath, addr, trustedProxies string) error {
	start := time.Now()

	f, err := os.Open(graphPath)
	if err != nil {
		return err
	}
	g, prov, err := csr.ReadGraph(f)
	f.Close()
	if err != nil {
		// Notamment ErrBadVersion : mieux vaut refuser de démarrer que servir
		// des itinéraires calculés sur des octets mal interprétés.
		return err
	}
	log.Printf("graphe chargé : %d nœuds, %d arêtes, construit le %s, en %s",
		g.NumNodes(), g.NumEdges(), prov.BuiltAt, time.Since(start).Round(time.Millisecond))

	srv := &http.Server{
		Addr:              addr,
		Handler:           httpapi.New(generateloop.New(g), prov, g.BBox(), parseTrustedProxies(trustedProxies)),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("écoute sur %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("serveur : %v", err)
		}
	}()

	<-stop
	log.Print("arrêt en cours")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}
