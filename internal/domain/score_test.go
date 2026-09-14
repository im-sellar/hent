package domain_test

import (
	"math"
	"testing"

	"github.com/im-sellar/hent/internal/domain"
)

func TestNewScore(t *testing.T) {
	l := domain.Loop{
		LengthM:          10000,
		UnpavedM:         8500,
		TrafficExposureM: 500,
	}

	s := domain.NewScore(l, 10000)

	if s.DistanceM != 10000 {
		t.Errorf("DistanceM = %v, attendu 10000", s.DistanceM)
	}
	if math.Abs(s.PartNonBitume-0.85) > 1e-9 {
		t.Errorf("PartNonBitume = %v, attendu 0.85", s.PartNonBitume)
	}
	if math.Abs(s.PartTrafic-0.05) > 1e-9 {
		t.Errorf("PartTrafic = %v, attendu 0.05", s.PartTrafic)
	}
	if s.EcartCible != 0 {
		t.Errorf("EcartCible = %v, attendu 0", s.EcartCible)
	}
}

func TestNewScoreEcartSigne(t *testing.T) {
	court := domain.NewScore(domain.Loop{LengthM: 9000}, 10000)
	long := domain.NewScore(domain.Loop{LengthM: 11000}, 10000)

	if court.EcartCible >= 0 {
		t.Errorf("une boucle trop courte doit avoir un écart négatif, obtenu %v", court.EcartCible)
	}
	if long.EcartCible <= 0 {
		t.Errorf("une boucle trop longue doit avoir un écart positif, obtenu %v", long.EcartCible)
	}
}

func TestNewScoreBoucleVide(t *testing.T) {
	// Ne doit pas diviser par zéro.
	s := domain.NewScore(domain.Loop{}, 10000)

	if s.PartNonBitume != 0 || s.PartTrafic != 0 {
		t.Errorf("boucle vide : %+v, attendu des parts nulles", s)
	}
}

func TestNewScoreCibleNulle(t *testing.T) {
	// Ne doit ni diviser par zéro ni produire un écart non fini.
	s := domain.NewScore(domain.Loop{LengthM: 5000}, 0)

	if math.IsNaN(s.EcartCible) || math.IsInf(s.EcartCible, 0) {
		t.Errorf("EcartCible = %v, attendu une valeur finie", s.EcartCible)
	}
}

func TestPreferable(t *testing.T) {
	const tol = 0.10

	score := func(ecart, part float64) domain.Score {
		return domain.Score{EcartCible: ecart, PartNonBitume: part}
	}

	cas := []struct {
		nom     string
		a, b    domain.Score
		attendu bool
	}{
		{"une boucle dans la tolérance l'emporte sur une boucle hors tolérance, même mieux revêtue",
			score(0.05, 0.2), score(0.30, 0.9), true},
		{"et réciproquement, hors tolérance ne l'emporte jamais sur dans la tolérance",
			score(0.30, 0.9), score(0.05, 0.2), false},
		{"deux boucles dans la tolérance se départagent sur le terrain, pas sur les mètres",
			score(0.08, 0.9), score(0.02, 0.5), true},
		{"deux boucles hors tolérance se départagent sur l'écart à la cible",
			score(0.20, 0.1), score(0.40, 0.9), true},
		{"un écart exactement égal à la tolérance compte comme dans la tolérance",
			score(tol, 0.9), score(0.30, 0.9), true},
	}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := c.a.Preferable(c.b, tol); got != c.attendu {
				t.Errorf("Preferable = %v, attendu %v", got, c.attendu)
			}
		})
	}
}

// TestPreferableEstUnOrdreStrictFaible vérifie les trois propriétés dont
// sort.Slice dépend. Une fonction de comparaison incohérente n'y provoque ni
// erreur ni panique : elle produit un ordre arbitraire, silencieusement.
func TestPreferableEstUnOrdreStrictFaible(t *testing.T) {
	const tol = 0.10

	var scores []domain.Score
	for _, ecart := range []float64{-0.30, -0.10, -0.02, 0, 0.02, 0.10, 0.30} {
		for _, part := range []float64{0, 0.5, 1} {
			scores = append(scores, domain.Score{EcartCible: ecart, PartNonBitume: part})
		}
	}

	for _, a := range scores {
		if a.Preferable(a, tol) {
			t.Fatalf("irréflexivité violée : %+v se préfère à lui-même", a)
		}
	}

	for _, a := range scores {
		for _, b := range scores {
			if a.Preferable(b, tol) && b.Preferable(a, tol) {
				t.Fatalf("asymétrie violée entre %+v et %+v", a, b)
			}
		}
	}

	for _, a := range scores {
		for _, b := range scores {
			for _, c := range scores {
				if a.Preferable(b, tol) && b.Preferable(c, tol) && !a.Preferable(c, tol) {
					t.Fatalf("transitivité violée : %+v puis %+v puis %+v", a, b, c)
				}
			}
		}
	}
}

func TestNewScorePartRetracee(t *testing.T) {
	l := domain.Loop{LengthM: 10000, RetracedM: 2500}

	s := domain.NewScore(l, 10000)

	if math.Abs(s.PartRetracee-0.25) > 1e-9 {
		t.Errorf("PartRetracee = %v, attendu 0.25", s.PartRetracee)
	}
}

// TestNewScorePartRetraceeResteFinie garde la division : une boucle de longueur
// nulle produirait NaN, et NaN traverse toutes les comparaisons sans en faire
// échouer aucune — y compris celles qui sont censées le rattraper.
func TestNewScorePartRetraceeResteFinie(t *testing.T) {
	for _, l := range []domain.Loop{
		{},
		{RetracedM: 500},
		{LengthM: 10000, RetracedM: 10000},
		{LengthM: 1e-9, RetracedM: 1e-9},
	} {
		s := domain.NewScore(l, 10000)
		if math.IsNaN(s.PartRetracee) || math.IsInf(s.PartRetracee, 0) {
			t.Errorf("PartRetracee = %v pour %+v : la valeur doit rester finie", s.PartRetracee, l)
		}
		if s.PartRetracee < 0 || s.PartRetracee > 1 {
			t.Errorf("PartRetracee = %v pour %+v : hors de [0, 1]", s.PartRetracee, l)
		}
	}
}
