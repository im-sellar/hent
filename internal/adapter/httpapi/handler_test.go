package httpapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/im-sellar/hent/internal/adapter/httpapi"
	"github.com/im-sellar/hent/internal/adapter/network/csr"
	"github.com/im-sellar/hent/internal/app/generateloop"
	"github.com/im-sellar/hent/internal/testsupport"
)

func TestPostLoops(t *testing.T) {
	srv := httptest.NewServer(testHandler(t))
	defer srv.Close()

	// Le départ est au centre de la grille de test. Un point proche d'un bord
	// enverrait les waypoints hors de la zone couverte et NearestNode
	// échouerait : le test deviendrait intermittent.
	body := `{"start":{"lat":48.135,"lon":-1.628},"distance_m":4000,
	          "activity":"trail","preferences":{"avoid_paved":0.5},"max_results":3}`

	resp, err := http.Post(srv.URL+"/v1/loops", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("statut %d, attendu 200", resp.StatusCode)
	}

	var out struct {
		Loops []struct {
			ID    string          `json:"id"`
			Score json.RawMessage `json:"score"`
			Geom  json.RawMessage `json:"geometry"`
		} `json:"loops"`
		Attribution string `json:"attribution"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}

	if len(out.Loops) == 0 {
		t.Fatal("aucune boucle renvoyée")
	}
	if out.Loops[0].ID == "" {
		t.Error("chaque boucle doit porter un identifiant, pour l'export GPX")
	}
	if !strings.Contains(out.Attribution, "OpenStreetMap") {
		t.Errorf("attribution manquante : %q", out.Attribution)
	}
}

func TestPostLoopsValidation(t *testing.T) {
	srv := httptest.NewServer(testHandler(t))
	defer srv.Close()

	cases := map[string]string{
		"distance nulle":     `{"start":{"lat":48.11,"lon":-1.67},"distance_m":0}`,
		"distance démesurée": `{"start":{"lat":48.11,"lon":-1.67},"distance_m":900000}`,
		"latitude invalide":  `{"start":{"lat":991,"lon":-1.67},"distance_m":4000}`,
		"json cassé":         `{oops`,
	}

	for nom, body := range cases {
		t.Run(nom, func(t *testing.T) {
			resp, err := http.Post(srv.URL+"/v1/loops", "application/json", strings.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("statut %d, attendu 400", resp.StatusCode)
			}
		})
	}
}

var trkptRegexp = regexp.MustCompile(`<trkpt lat="(-?[\d.]+)" lon="(-?[\d.]+)">`)

func TestGetGPX(t *testing.T) {
	srv := httptest.NewServer(testHandler(t))
	defer srv.Close()

	body := `{"start":{"lat":48.135,"lon":-1.628},"distance_m":4000,"max_results":1}`
	resp, err := http.Post(srv.URL+"/v1/loops", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Loops []struct {
			ID       string       `json:"id"`
			Geometry [][2]float64 `json:"geometry"` // [lon, lat]
		} `json:"loops"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("réponse illisible : %v", err)
	}
	resp.Body.Close()

	// Sans cette garde, l'indexation ci-dessous provoquerait un « index out of
	// range » dont le message ne dirait rien de la cause réelle.
	if len(out.Loops) == 0 {
		t.Fatal("aucune boucle renvoyée : rien à exporter en GPX")
	}
	geom := out.Loops[0].Geometry
	if len(geom) == 0 {
		t.Fatal("la géométrie renvoyée par le POST est vide")
	}

	// L'identifiant encode la requête : le GPX se régénère sans état côté
	// serveur, ce qui n'est possible que parce que la génération est
	// déterministe.
	gpx, err := http.Get(srv.URL + "/v1/loops/" + out.Loops[0].ID + ".gpx")
	if err != nil {
		t.Fatal(err)
	}
	defer gpx.Body.Close()

	if gpx.StatusCode != http.StatusOK {
		t.Fatalf("statut %d, attendu 200", gpx.StatusCode)
	}
	var buf bytes.Buffer
	buf.ReadFrom(gpx.Body)
	corps := buf.String()
	if !strings.Contains(corps, "<trkseg>") {
		t.Error("la réponse ne ressemble pas à du GPX")
	}

	// La propriété la plus importante du service : la boucle que le GPX
	// contient doit être exactement celle que le POST a renvoyée, pas une
	// boucle régénérée qui aurait divergé.
	pts := trkptRegexp.FindAllStringSubmatch(corps, -1)
	if len(pts) != len(geom) {
		t.Fatalf("%d points de trace dans le GPX, attendu %d (comme la géométrie du POST)",
			len(pts), len(geom))
	}

	// Le premier et le dernier point de toute boucle sont le départ demandé,
	// quelle que soit la boucle choisie : les comparer seuls ne détecterait
	// pas une régénération qui aurait dérivé vers une autre boucle candidate.
	// On compare donc l'intégralité du tracé.
	const epsilon = 1e-6 // les coordonnées GPX sont tronquées à 7 décimales
	for i := range pts {
		lat, err := strconv.ParseFloat(pts[i][1], 64)
		if err != nil {
			t.Fatalf("point %d : latitude illisible : %v", i, err)
		}
		lon, err := strconv.ParseFloat(pts[i][2], 64)
		if err != nil {
			t.Fatalf("point %d : longitude illisible : %v", i, err)
		}
		if math.Abs(lat-geom[i][1]) > epsilon || math.Abs(lon-geom[i][0]) > epsilon {
			t.Fatalf("point %d : GPX (%.7f, %.7f), POST (%.7f, %.7f)",
				i, lat, lon, geom[i][1], geom[i][0])
		}
	}
}

func TestHealthzEtRegions(t *testing.T) {
	srv := httptest.NewServer(testHandler(t))
	defer srv.Close()

	respHealthz, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	bodyHealthz, _ := io.ReadAll(respHealthz.Body)
	respHealthz.Body.Close()
	if respHealthz.StatusCode != http.StatusOK {
		t.Errorf("/healthz : statut %d, attendu 200", respHealthz.StatusCode)
	}
	if string(bodyHealthz) != "ok" {
		t.Errorf("/healthz : corps %q, attendu %q", bodyHealthz, "ok")
	}

	respRegions, err := http.Get(srv.URL + "/v1/regions")
	if err != nil {
		t.Fatal(err)
	}
	defer respRegions.Body.Close()
	if respRegions.StatusCode != http.StatusOK {
		t.Errorf("/v1/regions : statut %d, attendu 200", respRegions.StatusCode)
	}
	var regions struct {
		BBox        map[string]float64 `json:"bbox"`
		Attribution string             `json:"attribution"`
	}
	if err := json.NewDecoder(respRegions.Body).Decode(&regions); err != nil {
		t.Fatalf("/v1/regions : réponse illisible : %v", err)
	}
	if !strings.Contains(regions.Attribution, "OpenStreetMap") {
		t.Errorf("/v1/regions : attribution manquante, obtenu %q", regions.Attribution)
	}
	if regions.BBox["min_lat"] >= regions.BBox["max_lat"] || regions.BBox["min_lon"] >= regions.BBox["max_lon"] {
		t.Errorf("/v1/regions : bbox incohérente : %+v", regions.BBox)
	}

	respMetrics, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer respMetrics.Body.Close()
	if respMetrics.StatusCode != http.StatusOK {
		t.Errorf("/metrics : statut %d, attendu 200", respMetrics.StatusCode)
	}
	bodyMetrics, _ := io.ReadAll(respMetrics.Body)
	for _, nom := range []string{
		"hent_requests_total",
		"hent_errors_total",
		"hent_request_duration_ms_total",
		"hent_astar_explored_nodes_total",
		"hent_candidates_dropped_total",
	} {
		if !strings.Contains(string(bodyMetrics), nom) {
			t.Errorf("/metrics : métrique %q absente du corps", nom)
		}
	}
}

func testHandler(t *testing.T) http.Handler {
	t.Helper()
	return testHandlerAvecProxiesDeConfiance(t, nil)
}

func testHandlerAvecProxiesDeConfiance(t *testing.T, trustedProxies map[string]struct{}) http.Handler {
	t.Helper()

	g := testsupport.NouvelleGrille(40, 200) // ~8 km de côté autour de 48.10 / -1.68
	return httpapi.New(
		generateloop.New(g),
		csr.Provenance{BuiltAt: "2026-08-18T10:00:00Z"},
		g.BBox(),
		trustedProxies,
	)
}
