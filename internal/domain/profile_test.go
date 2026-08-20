package domain_test

import (
	"math"
	"testing"

	"github.com/im-sellar/hent/internal/domain"
)

func TestWeightsCroissantAvecAvoidPaved(t *testing.T) {
	faible := domain.Preferences{AvoidPaved: 0.0}.Weights()
	fort := domain.Preferences{AvoidPaved: 1.0}.Weights()

	if fort.Paved <= faible.Paved {
		t.Errorf("Paved : %v à 1.0 doit dépasser %v à 0.0", fort.Paved, faible.Paved)
	}
	if fort.Traffic <= faible.Traffic {
		t.Errorf("Traffic : %v à 1.0 doit dépasser %v à 0.0", fort.Traffic, faible.Traffic)
	}
}

func TestWeightsTraficJamaisNul(t *testing.T) {
	// Même sans aversion déclarée pour le bitume, on ne veut jamais être
	// envoyé sur une route à fort trafic.
	w := domain.Preferences{AvoidPaved: 0}.Weights()

	if w.Traffic <= 0 {
		t.Errorf("Traffic = %v, attendu > 0 même à AvoidPaved = 0", w.Traffic)
	}
}

func TestWeightsToujoursPositives(t *testing.T) {
	// Invariant du §6 de la spec : jamais de poids négatif, sous peine de
	// casser silencieusement A*.
	//
	// L'assertion porte sur « est un nombre fini et positif », et non sur
	// « n'est pas négatif » : cette dernière passerait avec un NaN en sortie,
	// puisque toute comparaison avec NaN est fausse. C'est précisément le
	// piège que ce test doit attraper.
	for _, v := range []float64{-5, 0, 0.5, 1, 42,
		math.NaN(), math.Inf(1), math.Inf(-1)} {

		w := domain.Preferences{AvoidPaved: v}.Weights()

		for nom, poids := range map[string]float64{"Paved": w.Paved, "Traffic": w.Traffic} {
			if math.IsNaN(poids) || math.IsInf(poids, 0) {
				t.Errorf("AvoidPaved=%v donne un poids %s non fini : %v", v, nom, poids)
			}
			if poids < 0 {
				t.Errorf("AvoidPaved=%v donne un poids %s négatif : %v", v, nom, poids)
			}
		}
	}
}
