package httpapi_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// boucleDeReference poste une requête connue et renvoie l'identifiant de la
// première boucle avec la réponse complète du POST. Les tests de cette page
// comparent systématiquement ce que rend le GET à ce qu'a rendu le POST : c'est
// la seule façon de garantir que les deux chemins ne divergent pas.
func boucleDeReference(t *testing.T, srv *httptest.Server) (string, map[string]any) {
	t.Helper()

	corps := `{"start":{"lat":48.135,"lon":-1.628},"distance_m":4000,` +
		`"tolerance":0.15,"preferences":{"avoid_paved":0.5},"max_results":2,"variant":1}`
	resp, err := http.Post(srv.URL+"/v1/loops", "application/json", strings.NewReader(corps))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var out struct {
		Loops []map[string]any `json:"loops"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("réponse du POST illisible : %v", err)
	}
	if len(out.Loops) == 0 {
		t.Fatal("aucune boucle renvoyée par le POST : les assertions qui suivent ne vérifieraient rien")
	}

	id, ok := out.Loops[0]["id"].(string)
	if !ok || id == "" {
		t.Fatalf("identifiant absent ou vide dans %v", out.Loops[0])
	}
	return id, out.Loops[0]
}

func TestGetLoopRendDuJSON(t *testing.T) {
	srv := httptest.NewServer(testHandler(t))
	defer srv.Close()

	id, _ := boucleDeReference(t, srv)

	resp, err := http.Get(srv.URL + "/v1/loops/" + id)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("statut %d, attendu 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, attendu application/json", ct)
	}

	var out struct {
		Loop        map[string]any `json:"loop"`
		Request     map[string]any `json:"request"`
		Attribution string         `json:"attribution"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("réponse illisible : %v", err)
	}

	if len(out.Loop) == 0 {
		t.Error("champ loop absent ou vide")
	}
	if len(out.Request) == 0 {
		t.Error("champ request absent ou vide")
	}
	if out.Attribution == "" {
		t.Error("attribution absente : toute réponse exposant des données OpenStreetMap doit la porter")
	}
}

// TestGetLoopRendLaMemeBoucleQueLePost est le garde-fou de cette route : sans
// lui, la représentation d'une boucle pourrait diverger entre le POST qui la
// produit et le GET qui la régénère, sans qu'aucun test ne bronche. Les deux
// chemins construisent le même DTO, ils doivent rendre les mêmes octets.
func TestGetLoopRendLaMemeBoucleQueLePost(t *testing.T) {
	srv := httptest.NewServer(testHandler(t))
	defer srv.Close()

	id, depuisPost := boucleDeReference(t, srv)

	resp, err := http.Get(srv.URL + "/v1/loops/" + id)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var out struct {
		Loop map[string]any `json:"loop"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("réponse illisible : %v", err)
	}

	attendu, err := json.Marshal(depuisPost)
	if err != nil {
		t.Fatal(err)
	}
	obtenu, err := json.Marshal(out.Loop)
	if err != nil {
		t.Fatal(err)
	}
	if string(obtenu) != string(attendu) {
		t.Errorf("le GET et le POST divergent sur la même boucle.\nPOST : %s\nGET  : %s", attendu, obtenu)
	}
}

// TestGetLoopRendLaDemandeDOrigine verrouille la raison d'être du champ
// request : l'écran de détail affiche la distance demandée à côté de la
// distance obtenue, et cette première valeur ne vit que dans l'identifiant.
func TestGetLoopRendLaDemandeDOrigine(t *testing.T) {
	srv := httptest.NewServer(testHandler(t))
	defer srv.Close()

	id, _ := boucleDeReference(t, srv)

	resp, err := http.Get(srv.URL + "/v1/loops/" + id)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var out struct {
		Request struct {
			Start       struct{ Lat, Lon float64 } `json:"start"`
			DistanceM   float64                    `json:"distance_m"`
			Tolerance   float64                    `json:"tolerance"`
			Preferences struct {
				AvoidPaved float64 `json:"avoid_paved"`
			} `json:"preferences"`
			MaxResults int `json:"max_results"`
			Variant    int `json:"variant"`
		} `json:"request"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("réponse illisible : %v", err)
	}

	// Les sept valeurs postées, deux à deux distinctes, pour qu'une perte comme
	// une interversion se voie.
	r := out.Request
	if r.Start.Lat != 48.135 {
		t.Errorf("start.lat = %v, attendu 48.135", r.Start.Lat)
	}
	if r.Start.Lon != -1.628 {
		t.Errorf("start.lon = %v, attendu -1.628", r.Start.Lon)
	}
	if r.DistanceM != 4000 {
		t.Errorf("distance_m = %v, attendu 4000", r.DistanceM)
	}
	if r.Tolerance != 0.15 {
		t.Errorf("tolerance = %v, attendu 0.15", r.Tolerance)
	}
	if r.Preferences.AvoidPaved != 0.5 {
		t.Errorf("preferences.avoid_paved = %v, attendu 0.5", r.Preferences.AvoidPaved)
	}
	if r.MaxResults != 2 {
		t.Errorf("max_results = %v, attendu 2", r.MaxResults)
	}
	if r.Variant != 1 {
		t.Errorf("variant = %v, attendu 1", r.Variant)
	}
}

// TestGetLoopAvecSuffixeRendToujoursDuGPX garde la route existante. Le suffixe
// est ce qui distingue les deux représentations, et c'est la forme qu'emploient
// déjà la documentation et les clients.
func TestGetLoopAvecSuffixeRendToujoursDuGPX(t *testing.T) {
	srv := httptest.NewServer(testHandler(t))
	defer srv.Close()

	id, _ := boucleDeReference(t, srv)

	resp, err := http.Get(srv.URL + "/v1/loops/" + id + ".gpx")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("statut %d, attendu 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/gpx+xml" {
		t.Errorf("Content-Type = %q, attendu application/gpx+xml", ct)
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, "attachment") {
		t.Errorf("Content-Disposition = %q, attendu une pièce jointe", cd)
	}

	corps, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(corps), "<trkseg>") {
		t.Error("la réponse ne ressemble pas à du GPX")
	}
}

func TestGetLoopRefuseUnIdentifiantInvalide(t *testing.T) {
	srv := httptest.NewServer(testHandler(t))
	defer srv.Close()

	for _, chemin := range []string{"/v1/loops/nimportequoi", "/v1/loops/nimportequoi.gpx"} {
		resp, err := http.Get(srv.URL + chemin)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s : statut %d, attendu 400", chemin, resp.StatusCode)
		}
	}
}

// TestGetLoopRefuseUnIndexHorsBornes vérifie les deux représentations : un
// identifiant bien formé dont l'index dépasse le nombre de boucles produites
// doit donner 404, et non une panique ni un corps vide en 200.
func TestGetLoopRefuseUnIndexHorsBornes(t *testing.T) {
	srv := httptest.NewServer(testHandler(t))
	defer srv.Close()

	id, _ := boucleDeReference(t, srv)

	// L'identifiant a la forme « index.charge » : on remplace l'index par un
	// rang qu'aucune requête à deux résultats ne peut atteindre.
	point := strings.Index(id, ".")
	if point < 0 {
		t.Fatalf("identifiant %q sans séparateur : la suite du test ne vérifierait rien", id)
	}
	horsBornes := "99" + id[point:]

	for _, chemin := range []string{"/v1/loops/" + horsBornes, "/v1/loops/" + horsBornes + ".gpx"} {
		resp, err := http.Get(srv.URL + chemin)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s : statut %d, attendu 404", chemin, resp.StatusCode)
		}
	}
}
