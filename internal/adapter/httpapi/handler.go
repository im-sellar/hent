// Package httpapi expose le générateur de boucles en HTTP. Les DTO définis
// ici sont volontairement distincts des types du domaine : le contrat public
// doit pouvoir rester stable pendant que le modèle interne évolue.
package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/im-sellar/hent/internal/adapter/gpxfile"
	"github.com/im-sellar/hent/internal/adapter/network/csr"
	"github.com/im-sellar/hent/internal/app/generateloop"
	"github.com/im-sellar/hent/internal/domain"
)

const attribution = "Données © les contributeurs OpenStreetMap, sous licence ODbL"

const (
	minDistanceM = 500
	maxDistanceM = 200_000
	maxResults   = 10
	requestTTL   = 5 * time.Second
)

type coordDTO struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type prefsDTO struct {
	// Curseur d'intention, 0 à 1. Les poids internes ne sortent jamais : le
	// modèle de coût peut être entièrement revu sans casser un client.
	AvoidPaved float64 `json:"avoid_paved"`
}

type loopsRequest struct {
	Start       coordDTO `json:"start"`
	DistanceM   float64  `json:"distance_m"`
	Tolerance   float64  `json:"tolerance"`
	Activity    string   `json:"activity"`
	Preferences prefsDTO `json:"preferences"`
	MaxResults  int      `json:"max_results"`
	Variant     int      `json:"variant"`
}

func (r loopsRequest) validate() error {
	// La finitude se vérifie d'abord, et séparément des encadrements : en Go
	// toute comparaison impliquant NaN est fausse, si bien qu'un NaN
	// traverserait intact tous les tests de bornes ci-dessous — dans la
	// fonction même dont le rôle est de les faire respecter.
	//
	// Le décodeur JSON de la bibliothèque standard ne produit pas de NaN (le
	// format ne l'admet pas), cette garde n'est donc pas atteignable par
	// l'API et aucun test HTTP ne peut la couvrir. Elle protège les appelants
	// non-HTTP — un client Go, un outil en ligne de commande — qui
	// construiraient une requête directement.
	for nom, v := range map[string]float64{
		"start.lat":  r.Start.Lat,
		"start.lon":  r.Start.Lon,
		"distance_m": r.DistanceM,
		"tolerance":  r.Tolerance,
	} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("%s doit être un nombre fini", nom)
		}
	}

	switch {
	case r.Start.Lat < -90 || r.Start.Lat > 90:
		return errors.New("latitude hors bornes")
	case r.Start.Lon < -180 || r.Start.Lon > 180:
		return errors.New("longitude hors bornes")
	case r.DistanceM < minDistanceM:
		return fmt.Errorf("distance_m doit valoir au moins %d", minDistanceM)
	case r.DistanceM > maxDistanceM:
		return fmt.Errorf("distance_m ne peut dépasser %d", maxDistanceM)
	case r.Tolerance < 0 || r.Tolerance > 0.5:
		return errors.New("tolerance doit être comprise entre 0 et 0.5")
	case r.MaxResults > maxResults:
		return fmt.Errorf("max_results ne peut dépasser %d", maxResults)
	case r.Activity != "" && r.Activity != "trail":
		return errors.New(`seule l'activité "trail" est prise en charge à ce stade`)
	}
	return nil
}

func (r loopsRequest) toDomain() generateloop.Request {
	return generateloop.Request{
		Start:      domain.Coord{Lat: r.Start.Lat, Lon: r.Start.Lon},
		DistanceM:  r.DistanceM,
		Tolerance:  r.Tolerance,
		Prefs:      domain.Preferences{AvoidPaved: r.Preferences.AvoidPaved},
		MaxResults: r.MaxResults,
		Variant:    r.Variant,
	}
}

type loopDTO struct {
	ID       string       `json:"id"`
	Score    domain.Score `json:"score"`
	Geometry [][2]float64 `json:"geometry"` // [lon, lat], ordre GeoJSON
}

type loopsResponse struct {
	Loops       []loopDTO `json:"loops"`
	Attribution string    `json:"attribution"`
}

type api struct {
	gen  *generateloop.Generator
	prov csr.Provenance
	bbox domain.BBox

	requests atomic.Int64
	errors   atomic.Int64
	totalMs  atomic.Int64
}

func New(gen *generateloop.Generator, prov csr.Provenance, bbox domain.BBox) http.Handler {
	a := &api{gen: gen, prov: prov, bbox: bbox}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/loops", a.postLoops)
	mux.HandleFunc("GET /v1/loops/{id}", a.getGPX)
	mux.HandleFunc("GET /v1/regions", a.getRegions)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /metrics", a.getMetrics)

	return withRateLimit(mux)
}

func (a *api) postLoops(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	a.requests.Add(1)
	defer func() { a.totalMs.Add(time.Since(started).Milliseconds()) }()

	var req loopsRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		a.fail(w, http.StatusBadRequest, "corps de requête illisible")
		return
	}
	if err := req.validate(); err != nil {
		a.fail(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTTL)
	defer cancel()

	loops, err := a.gen.Generate(ctx, req.toDomain())
	if err != nil {
		switch {
		case errors.Is(err, generateloop.ErrStartOutOfRange):
			a.fail(w, http.StatusBadRequest, "le point de départ est hors de la zone couverte")
		case errors.Is(err, generateloop.ErrNoLoopFound):
			a.fail(w, http.StatusNotFound, "aucune boucle trouvée pour ces critères")
		case errors.Is(err, context.DeadlineExceeded):
			a.fail(w, http.StatusGatewayTimeout, "délai dépassé")
		default:
			log.Printf("génération : %v", err)
			a.fail(w, http.StatusInternalServerError, "erreur interne")
		}
		return
	}

	resp := loopsResponse{Attribution: attribution}
	for i, l := range loops {
		resp.Loops = append(resp.Loops, loopDTO{
			ID:       encodeID(req, i),
			Score:    domain.NewScore(l, req.DistanceM),
			Geometry: geometryOf(l),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	json.NewEncoder(w).Encode(resp)
}

// getGPX régénère la boucle à partir de l'identifiant, qui encode la requête
// et l'indice. Le serveur ne conserve aucun état entre les deux appels : c'est
// possible uniquement parce que la génération est déterministe.
func (a *api) getGPX(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSuffix(r.PathValue("id"), ".gpx")

	req, index, err := decodeID(id)
	if err != nil {
		a.fail(w, http.StatusBadRequest, "identifiant de boucle invalide")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTTL)
	defer cancel()

	loops, err := a.gen.Generate(ctx, req.toDomain())
	if err != nil || index >= len(loops) {
		a.fail(w, http.StatusNotFound, "boucle introuvable")
		return
	}

	w.Header().Set("Content-Type", "application/gpx+xml")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="hent-%.0fm.gpx"`, req.DistanceM))
	name := fmt.Sprintf("Boucle %.1f km", loops[index].LengthM/1000)
	if err := gpxfile.Write(w, loops[index], name); err != nil {
		log.Printf("export GPX : %v", err)
	}
}

func (a *api) getRegions(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"bbox": map[string]float64{
			"min_lat": a.bbox.Min.Lat, "min_lon": a.bbox.Min.Lon,
			"max_lat": a.bbox.Max.Lat, "max_lon": a.bbox.Max.Lon,
		},
		"data":        a.prov,
		"attribution": attribution,
	})
}

func (a *api) getMetrics(w http.ResponseWriter, _ *http.Request) {
	explored, dropped := a.gen.Stats()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "hent_requests_total %d\n", a.requests.Load())
	fmt.Fprintf(w, "hent_errors_total %d\n", a.errors.Load())
	fmt.Fprintf(w, "hent_request_duration_ms_total %d\n", a.totalMs.Load())
	fmt.Fprintf(w, "hent_astar_explored_nodes_total %d\n", explored)
	fmt.Fprintf(w, "hent_candidates_dropped_total %d\n", dropped)
}

func (a *api) fail(w http.ResponseWriter, code int, msg string) {
	a.errors.Add(1)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func geometryOf(l domain.Loop) [][2]float64 {
	pts := make([][2]float64, len(l.Coords))
	for i, c := range l.Coords {
		pts[i] = [2]float64{c.Lon, c.Lat}
	}
	return pts
}

func encodeID(req loopsRequest, index int) string {
	raw, _ := json.Marshal(req)
	return fmt.Sprintf("%d.%s", index, base64.RawURLEncoding.EncodeToString(raw))
}

func decodeID(id string) (loopsRequest, int, error) {
	var req loopsRequest

	index, rest, found := strings.Cut(id, ".")
	if !found {
		return req, 0, errors.New("identifiant mal formé")
	}
	var i int
	if _, err := fmt.Sscanf(index, "%d", &i); err != nil || i < 0 {
		return req, 0, errors.New("indice invalide")
	}

	raw, err := base64.RawURLEncoding.DecodeString(rest)
	if err != nil {
		return req, 0, err
	}
	if err := json.Unmarshal(raw, &req); err != nil {
		return req, 0, err
	}
	if err := req.validate(); err != nil {
		return req, 0, err
	}
	return req, i, nil
}
