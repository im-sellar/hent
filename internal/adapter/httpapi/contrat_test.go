package httpapi_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var majTemoins = flag.Bool("maj-temoins", false,
	"réécrit les témoins de contrat au lieu de les comparer")

// TestContratJSONInchange fige les réponses publiques octet pour octet.
//
// Il ne vérifie aucune règle métier : son seul rôle est de tomber si une
// réorganisation interne modifie ce que voit un client. C'est le filet du
// durcissement d'architecture, et il doit être vert avant que le premier type
// ne soit déplacé.
//
// Les réponses sont déterministes : la génération est amorcée par un hachage
// de la requête, et la provenance du handler de test porte une date figée.
func TestContratJSONInchange(t *testing.T) {
	srv := httptest.NewServer(testHandler(t))
	defer srv.Close()

	cas := []struct {
		nom     string
		methode string
		chemin  string
		corps   string
	}{
		{
			nom: "loops", methode: http.MethodPost, chemin: "/v1/loops",
			corps: `{"start":{"lat":48.135,"lon":-1.628},"distance_m":4000,` +
				`"tolerance":0.15,"preferences":{"avoid_paved":0.5},"max_results":3}`,
		},
		{nom: "regions", methode: http.MethodGet, chemin: "/v1/regions"},
	}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			req, err := http.NewRequest(c.methode, srv.URL+c.chemin, strings.NewReader(c.corps))
			if err != nil {
				t.Fatal(err)
			}
			if c.corps != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("statut %d, attendu 200 : le témoin ne prouverait rien sur une erreur",
					resp.StatusCode)
			}

			brut, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if len(brut) == 0 {
				t.Fatal("réponse vide : le témoin ne prouverait rien")
			}

			// Réindentation : le témoin reste lisible en revue de code, et une
			// différence se voit ligne à ligne plutôt qu'en un seul bloc.
			var indente bytes.Buffer
			if err := json.Indent(&indente, brut, "", "  "); err != nil {
				t.Fatalf("réponse illisible comme JSON : %v", err)
			}
			obtenu := append(indente.Bytes(), '\n')

			chemin := filepath.Join("testdata", "contrat-"+c.nom+".json")
			if *majTemoins {
				if err := os.WriteFile(chemin, obtenu, 0o644); err != nil {
					t.Fatal(err)
				}
				t.Logf("témoin réécrit : %s", chemin)
				return
			}

			attendu, err := os.ReadFile(chemin)
			if err != nil {
				t.Fatalf("témoin absent (%v) — le régénérer avec : go test ./internal/adapter/httpapi/ -run TestContratJSONInchange -maj-temoins", err)
			}
			if !bytes.Equal(obtenu, attendu) {
				t.Errorf("le contrat public a changé.\n--- attendu ---\n%s\n--- obtenu ---\n%s",
					attendu, obtenu)
			}
		})
	}
}
