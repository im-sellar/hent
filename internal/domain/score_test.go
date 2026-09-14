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
