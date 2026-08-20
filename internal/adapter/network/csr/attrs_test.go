package csr_test

import (
	"math"
	"testing"

	"github.com/im-sellar/hent/internal/adapter/network/csr"
	"github.com/im-sellar/hent/internal/domain"
)

func TestCostBitumePlusCherQueSentier(t *testing.T) {
	w := domain.Preferences{AvoidPaved: 0.8}.Weights()

	sentier := csr.EdgeAttrs{LengthM: 1000, Surface: domain.SurfaceGround, Class: domain.WayPath}
	route := csr.EdgeAttrs{LengthM: 1000, Surface: domain.SurfacePaved, Class: domain.WaySecondary, Traffic: 255}

	if route.Cost(w) <= sentier.Cost(w) {
		t.Errorf("route %.0f doit coûter plus qu'un sentier %.0f", route.Cost(w), sentier.Cost(w))
	}
}

func TestCostJamaisInferieurALaLongueur(t *testing.T) {
	// Invariant central : le facteur est toujours ≥ 1, donc coût ≥ longueur.
	// C'est ce qui rend l'heuristique de l'A* admissible.
	w := domain.Preferences{AvoidPaved: 1}.Weights()

	// La dernière surface est hors énumération : elle vérifie que le cas
	// « default » de la pénalité de revêtement reste dans [0,1].
	surfaces := []domain.Surface{
		domain.SurfaceUnknown, domain.SurfacePaved,
		domain.SurfaceGravel, domain.SurfaceGround, domain.Surface(99),
	}
	for _, s := range surfaces {
		for _, traffic := range []uint8{0, 128, 255} {
			a := csr.EdgeAttrs{LengthM: 500, Surface: s, Traffic: traffic}
			got := a.Cost(w)

			// On exige un coût fini et supérieur ou égal à la longueur.
			// Tester seulement « got < a.LengthM » ne suffirait pas : un NaN
			// rendrait cette comparaison fausse et le test passerait.
			if math.IsNaN(got) || math.IsInf(got, 0) {
				t.Errorf("surface=%v trafic=%d : coût non fini (%v)", s, traffic, got)
			}
			if got < a.LengthM {
				t.Errorf("surface=%v trafic=%d : coût %.1f < longueur %.1f",
					s, traffic, got, a.LengthM)
			}
		}
	}
}

func TestCostSansPreferenceVautLaLongueur(t *testing.T) {
	// Poids nuls : le coût doit se réduire exactement à la distance.
	a := csr.EdgeAttrs{LengthM: 750, Surface: domain.SurfacePaved, Traffic: 255}

	if got := a.Cost(domain.Weights{}); got != 750 {
		t.Errorf("coût à poids nuls = %v, attendu 750", got)
	}
}
