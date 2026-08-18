package csr_test

import (
	"testing"

	"github.com/im-sellar/hent/internal/adapter/network/csr"
	"github.com/im-sellar/hent/internal/domain"
)

// carre construit un graphe jouet : quatre nœuds aux coins d'un carré
// d'environ 1 km de côté près de Rennes, reliés en cycle, arêtes dans les
// deux sens.
//
//	3 ── 2
//	│    │
//	0 ── 1
func carre(t *testing.T) *csr.Graph {
	t.Helper()

	b := csr.NewBuilder()
	n0 := b.AddNode(domain.Coord{Lat: 48.100, Lon: -1.680})
	n1 := b.AddNode(domain.Coord{Lat: 48.100, Lon: -1.667})
	n2 := b.AddNode(domain.Coord{Lat: 48.109, Lon: -1.667})
	n3 := b.AddNode(domain.Coord{Lat: 48.109, Lon: -1.680})

	for _, pair := range [][2]domain.NodeRef{{n0, n1}, {n1, n2}, {n2, n3}, {n3, n0}} {
		from, to := pair[0], pair[1]
		length := domain.HaversineM(coordOf(b, from), coordOf(b, to))
		b.AddEdge(from, to, csr.EdgeAttrs{LengthM: length})
		b.AddEdge(to, from, csr.EdgeAttrs{LengthM: length})
	}

	return b.Build()
}

func coordOf(b *csr.Builder, n domain.NodeRef) domain.Coord { return b.Coord(n) }

func TestGraphTaille(t *testing.T) {
	g := carre(t)

	if got := g.NumNodes(); got != 4 {
		t.Errorf("NumNodes = %d, attendu 4", got)
	}
	if got := g.NumEdges(); got != 8 {
		t.Errorf("NumEdges = %d, attendu 8 (4 arêtes × 2 sens)", got)
	}
}

func TestGraphVoisins(t *testing.T) {
	g := carre(t)

	// Chaque nœud du cycle a exactement deux voisins sortants.
	for n := domain.NodeRef(0); n < 4; n++ {
		start, end := g.EdgeRange(n)
		if got := int(end - start); got != 2 {
			t.Errorf("nœud %d : %d arêtes sortantes, attendu 2", n, got)
		}
	}
}

func TestGraphCibleEtLongueur(t *testing.T) {
	g := carre(t)

	start, end := g.EdgeRange(0)
	voisins := map[domain.NodeRef]bool{}
	for e := start; e < end; e++ {
		voisins[g.Target(e)] = true
		if l := g.Attrs(e).LengthM; l <= 0 {
			t.Errorf("arête %d : longueur %v, attendu > 0", e, l)
		}
	}

	if !voisins[1] || !voisins[3] {
		t.Errorf("les voisins du nœud 0 sont %v, attendu {1, 3}", voisins)
	}
}

func TestGraphBBox(t *testing.T) {
	g := carre(t)
	b := g.BBox()

	if !b.Contains(domain.Coord{Lat: 48.105, Lon: -1.673}) {
		t.Error("la bbox doit contenir le centre du carré")
	}
	if b.Contains(domain.Coord{Lat: 49.0, Lon: -1.673}) {
		t.Error("la bbox ne doit pas contenir un point hors du carré")
	}
}
