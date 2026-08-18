package domain_test

import (
	"math"
	"testing"

	"github.com/im-sellar/hent/internal/domain"
)

func TestHaversineM_RennesSaintMalo(t *testing.T) {
	rennes := domain.Coord{Lat: 48.1173, Lon: -1.6778}
	stMalo := domain.Coord{Lat: 48.6493, Lon: -2.0257}

	got := domain.HaversineM(rennes, stMalo)

	const want = 64000.0
	if math.Abs(got-want) > 2000 {
		t.Fatalf("Rennes→Saint-Malo = %.0f m, attendu ~%.0f m (±2 km)", got, want)
	}
}

func TestHaversineM_Symetrique(t *testing.T) {
	a := domain.Coord{Lat: 48.11, Lon: -1.67}
	b := domain.Coord{Lat: 48.20, Lon: -1.50}

	if math.Abs(domain.HaversineM(a, b)-domain.HaversineM(b, a)) > 1e-6 {
		t.Fatal("la distance doit être symétrique")
	}
}

func TestHaversineM_MemePoint(t *testing.T) {
	a := domain.Coord{Lat: 48.11, Lon: -1.67}

	if got := domain.HaversineM(a, a); got != 0 {
		t.Fatalf("distance d'un point à lui-même = %v, attendu 0", got)
	}
}

func TestBBoxContains(t *testing.T) {
	b := domain.BBox{
		Min: domain.Coord{Lat: 48.0, Lon: -2.0},
		Max: domain.Coord{Lat: 48.5, Lon: -1.0},
	}

	if !b.Contains(domain.Coord{Lat: 48.1, Lon: -1.6}) {
		t.Error("le point intérieur doit être contenu")
	}
	if b.Contains(domain.Coord{Lat: 49.0, Lon: -1.6}) {
		t.Error("le point extérieur ne doit pas être contenu")
	}
}
