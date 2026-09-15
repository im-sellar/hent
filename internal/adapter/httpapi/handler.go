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
	"github.com/im-sellar/hent/internal/app/port"
	"github.com/im-sellar/hent/internal/domain"
)

const attribution = "Données © les contributeurs OpenStreetMap, sous licence ODbL"

const (
	minDistanceM = 500
	// Plafonné à 50 km, largement au-dessus d'une boucle de trail réelle,
	// plutôt qu'aux 200 km d'origine : le §12 dimensionne le rate limiting
	// sur le nombre de requêtes, pas sur le CPU qu'elles consomment, et une
	// boucle acceptée jusqu'à 200 km sature la machine bien avant d'atteindre
	// la limite en nombre (I5 de la revue finale). Réduire la distance
	// maximale ferme la porte sans toucher au limiteur ni à la génération.
	maxDistanceM = 50_000
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

func (r loopsRequest) toDomain() domain.LoopRequest {
	return domain.LoopRequest{
		Start:      domain.Coord{Lat: r.Start.Lat, Lon: r.Start.Lon},
		DistanceM:  r.DistanceM,
		Tolerance:  r.Tolerance,
		Prefs:      domain.Preferences{AvoidPaved: r.Preferences.AvoidPaved},
		MaxResults: r.MaxResults,
		Variant:    r.Variant,
	}
}

// scoreDTO fixe le contrat public du score. Les noms exposés ici sont
// indépendants de ceux du domaine : c'est ce qui permet de renommer un champ
// métier sans casser un client, et inversement.
type scoreDTO struct {
	DistanceM     float64 `json:"distance_m"`
	PartNonBitume float64 `json:"part_non_bitume"`
	PartTrafic    float64 `json:"part_trafic"`
	PartRetracee  float64 `json:"part_retracee"`
	EcartCible    float64 `json:"ecart_cible"`
}

func scoreDTOOf(s domain.Score) scoreDTO {
	return scoreDTO{
		DistanceM:     s.DistanceM,
		PartNonBitume: s.PartNonBitume,
		PartTrafic:    s.PartTrafic,
		PartRetracee:  s.PartRetracee,
		EcartCible:    s.EcartCible,
	}
}

type loopDTO struct {
	ID       string       `json:"id"`
	Score    scoreDTO     `json:"score"`
	Geometry [][2]float64 `json:"geometry"` // [lon, lat], ordre GeoJSON
}

type loopsResponse struct {
	Loops       []loopDTO `json:"loops"`
	Attribution string    `json:"attribution"`
}

// loopResponse est la réponse de GET /v1/loops/{id}. Elle rend la demande à
// côté de la boucle : l'identifiant l'encode déjà, et un client qui ouvre une
// boucle par son lien n'a pas d'autre moyen de savoir quelle distance avait
// été demandée — la déduire de l'écart à la cible serait un calcul à rebours.
//
// loopsRequest y sert de représentation de la demande plutôt qu'un type jumeau :
// c'est le même contrat, décrit une fois, lu à l'entrée et rendu à la sortie.
type loopResponse struct {
	Loop        loopDTO      `json:"loop"`
	Request     loopsRequest `json:"request"`
	Attribution string       `json:"attribution"`
}

// sourceDTO et provenanceDTO découplent le contrat JSON de /v1/regions des
// tags de sérialisation propres au format binaire de graph.bin, portés par
// l'adaptateur csr. Les laisser fuir jusqu'ici ferait dépendre l'API publique
// d'un choix de sérialisation interne, exactement ce que l'en-tête du paquet
// promet d'éviter pour tout le reste des réponses.
type sourceDTO struct {
	Name      string `json:"name"`
	File      string `json:"file"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

type provenanceDTO struct {
	BuiltAt    string      `json:"built_at"`
	Sources    []sourceDTO `json:"sources"`
	ConfigHash string      `json:"config_hash"`
}

func provenanceDTOOf(p domain.Provenance) provenanceDTO {
	sources := make([]sourceDTO, len(p.Sources))
	for i, s := range p.Sources {
		sources[i] = sourceDTO{Name: s.Name, File: s.File, SHA256: s.SHA256, SizeBytes: s.SizeBytes}
	}
	return provenanceDTO{BuiltAt: p.BuiltAt, Sources: sources, ConfigHash: p.ConfigHash}
}

type api struct {
	gen  port.LoopGenerator
	prov domain.Provenance
	bbox domain.BBox

	requests atomic.Int64
	errors   atomic.Int64
	totalMs  atomic.Int64
}

// New construit le handler HTTP. trustedProxies liste les adresses de
// connexion (sans port) autorisées à fournir X-Forwarded-For pour la
// limitation de débit — vide, l'en-tête est ignoré et seule l'adresse de
// connexion compte, ce qui est le comportement sûr par défaut.
func New(gen port.LoopGenerator, prov domain.Provenance, bbox domain.BBox, trustedProxies map[string]struct{}) http.Handler {
	a := &api{gen: gen, prov: prov, bbox: bbox}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/loops", a.postLoops)
	mux.HandleFunc("GET /v1/loops/{id}", a.getLoop)
	mux.HandleFunc("GET /v1/regions", a.getRegions)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /metrics", a.getMetrics)

	return withRateLimit(mux, trustedProxies)
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
		case errors.Is(err, port.ErrStartOutOfRange):
			a.fail(w, http.StatusBadRequest, "le point de départ est hors de la zone couverte")
		case errors.Is(err, port.ErrNoLoopFound):
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
			Score:    scoreDTOOf(domain.NewScore(l, req.DistanceM)),
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
// getLoop régénère une boucle depuis son identifiant et la rend dans l'une des
// deux représentations : GPX si le chemin porte le suffixe .gpx, JSON sinon.
// Le mux de net/http ne sait pas router sur un suffixe — un joker occupe un
// segment entier — d'où ce branchement ici plutôt que deux routes.
//
// Aucun état n'est conservé entre le POST qui a produit l'identifiant et cet
// appel : la génération étant déterministe, rejouer la demande encodée redonne
// exactement les mêmes boucles.
func (a *api) getLoop(w http.ResponseWriter, r *http.Request) {
	brut := r.PathValue("id")
	enGPX := strings.HasSuffix(brut, ".gpx")
	id := strings.TrimSuffix(brut, ".gpx")

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

	if !enGPX {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		json.NewEncoder(w).Encode(loopResponse{
			Loop: loopDTO{
				ID:       encodeID(req, index),
				Score:    scoreDTOOf(domain.NewScore(loops[index], req.DistanceM)),
				Geometry: geometryOf(loops[index]),
			},
			Request:     req,
			Attribution: attribution,
		})
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
		"data":        provenanceDTOOf(a.prov),
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
