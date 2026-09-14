package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/im-sellar/hent/internal/adapter/httpapi"
	"github.com/im-sellar/hent/internal/adapter/network/csr"
	"github.com/im-sellar/hent/internal/app/generateloop"
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
			ID string `json:"id"`
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
	if !strings.Contains(buf.String(), "<trkseg>") {
		t.Error("la réponse ne ressemble pas à du GPX")
	}
}

func TestHealthzEtRegions(t *testing.T) {
	srv := httptest.NewServer(testHandler(t))
	defer srv.Close()

	for _, path := range []string{"/healthz", "/v1/regions", "/metrics"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s : statut %d, attendu 200", path, resp.StatusCode)
		}
	}
}

func testHandler(t *testing.T) http.Handler {
	t.Helper()

	g := nouvelleGrille(40, 200) // ~8 km de côté autour de 48.10 / -1.68
	return httpapi.New(
		generateloop.New(g),
		csr.Provenance{BuiltAt: "2026-08-18T10:00:00Z"},
		g.BBox(),
	)
}
