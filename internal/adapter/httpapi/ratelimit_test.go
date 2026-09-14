package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// TestLimiteDeDebitBloqueAuDelaDuBurst est le test le plus sensible de la
// surface publique : sans lui, rien ne vérifie que la limitation de débit
// existe réellement. perIPBurst vaut 5, on en envoie strictement plus — en
// envoyer cinq ou moins vérifierait l'inverse de ce qu'on annonce.
func TestLimiteDeDebitBloqueAuDelaDuBurst(t *testing.T) {
	srv := httptest.NewServer(testHandler(t))
	defer srv.Close()

	const tentatives = 6
	var dernier *http.Response
	for i := 0; i < tentatives; i++ {
		resp, err := http.Post(srv.URL+"/v1/loops", "application/json", strings.NewReader(`{oops`))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		dernier = resp
	}
	if dernier.StatusCode != http.StatusTooManyRequests {
		t.Errorf("statut %d à la %de tentative, attendu %d", dernier.StatusCode, tentatives, http.StatusTooManyRequests)
	}
}

// TestLimiteDeDebitExempteHealthz vérifie l'exemption délibérée : la sonde
// de disponibilité doit rester joignable même quand un client a épuisé son
// crédit sur le reste de l'API.
func TestLimiteDeDebitExempteHealthz(t *testing.T) {
	srv := httptest.NewServer(testHandler(t))
	defer srv.Close()

	for i := 0; i < 10; i++ {
		resp, err := http.Post(srv.URL+"/v1/loops", "application/json", strings.NewReader(`{oops`))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("/healthz sous charge : statut %d, attendu 200 (sonde exemptée)", resp.StatusCode)
	}
}

// TestLimiteDeDebitIgnoreXFFSansProxyDeConfiance verrouille le correctif de
// sécurité : sans proxy de confiance déclaré, forger X-Forwarded-For avec une
// valeur différente à chaque requête ne doit donner aucune clé de limiteur
// neuve. Avant correction, ce test échoue : l'en-tête était honoré sans
// condition et la limitation s'annulait.
func TestLimiteDeDebitIgnoreXFFSansProxyDeConfiance(t *testing.T) {
	srv := httptest.NewServer(testHandler(t))
	defer srv.Close()

	client := &http.Client{}
	var dernier *http.Response
	for i := 0; i < 6; i++ {
		req, err := http.NewRequest(http.MethodPost, srv.URL+"/v1/loops", strings.NewReader(`{oops`))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("X-Forwarded-For", "203.0.113."+strconv.Itoa(i))

		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		dernier = resp
	}
	if dernier.StatusCode != http.StatusTooManyRequests {
		t.Errorf("statut %d malgré un X-Forwarded-For forgé et changeant, attendu %d : "+
			"l'en-tête ne doit pas permettre de contourner la limite sans proxy de confiance déclaré",
			dernier.StatusCode, http.StatusTooManyRequests)
	}
}

// TestLimiteDeDebitUtiliseLeProxyDeConfiance est le pendant positif : une
// fois un proxy explicitement déclaré de confiance, X-Forwarded-For est
// honoré et deux adresses forgées distinctes obtiennent des compteurs
// distincts — aucune des six requêtes n'est bloquée.
func TestLimiteDeDebitUtiliseLeProxyDeConfiance(t *testing.T) {
	// httptest.NewServer se lie par défaut sur la boucle locale IPv4 : c'est
	// l'adresse de connexion que verra le serveur pour chaque requête.
	srv := httptest.NewServer(testHandlerAvecProxiesDeConfiance(t, map[string]struct{}{"127.0.0.1": {}}))
	defer srv.Close()

	client := &http.Client{}
	for i := 0; i < 6; i++ {
		req, err := http.NewRequest(http.MethodPost, srv.URL+"/v1/loops", strings.NewReader(`{oops`))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("X-Forwarded-For", "203.0.113."+strconv.Itoa(i))

		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests {
			t.Fatalf("requête %d bloquée alors que chaque appel porte une adresse "+
				"X-Forwarded-For distincte derrière un proxy de confiance", i)
		}
	}
}
