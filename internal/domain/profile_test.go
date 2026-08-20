package domain_test

import (
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
	for _, v := range []float64{-5, 0, 0.5, 1, 42} {
		w := domain.Preferences{AvoidPaved: v}.Weights()
		if w.Paved < 0 || w.Traffic < 0 {
			t.Errorf("AvoidPaved=%v donne des poids négatifs : %+v", v, w)
		}
	}
}
