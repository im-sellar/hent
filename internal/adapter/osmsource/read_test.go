package osmsource_test

import (
	"context"
	"testing"

	"github.com/im-sellar/hent/internal/adapter/osmsource"
	"github.com/im-sellar/hent/internal/domain"
)

func TestReadExtraitRennes(t *testing.T) {
	g, stats, err := osmsource.Read(context.Background(), "../../../testdata/rennes-centre.osm.pbf")
	if err != nil {
		t.Fatalf("Read : %v", err)
	}

	if stats.Ways < 1000 {
		t.Errorf("%d ways retenus, attendu au moins 1000 sur le centre de Rennes", stats.Ways)
	}
	if g.NumNodes() < 1000 {
		t.Errorf("%d nœuds, attendu au moins 1000", g.NumNodes())
	}
	// Chaque tronçon produit deux arêtes (un sens chacune).
	if g.NumEdges() != 2*stats.Edges {
		t.Errorf("%d arêtes pour %d tronçons, attendu le double", g.NumEdges(), stats.Edges)
	}

	// La bbox doit couvrir le centre de Rennes.
	if !g.BBox().Contains(domain.Coord{Lat: 48.1113, Lon: -1.6800}) {
		t.Error("la bbox ne contient pas la place de la République")
	}
}

func TestReadEtRoute(t *testing.T) {
	// Test d'intégration : sur de vraies données, deux points distants d'un
	// kilomètre doivent être reliés.
	g, _, err := osmsource.Read(context.Background(), "../../../testdata/rennes-centre.osm.pbf")
	if err != nil {
		t.Fatal(err)
	}

	from, ok1 := g.NearestNode(domain.Coord{Lat: 48.1113, Lon: -1.6800})
	to, ok2 := g.NearestNode(domain.Coord{Lat: 48.1200, Lon: -1.6700})
	if !ok1 || !ok2 {
		t.Fatal("points de départ ou d'arrivée introuvables dans le graphe")
	}

	p, err := g.FindPath(context.Background(), from, to,
		domain.Preferences{AvoidPaved: 0.5}.Weights(), domain.PathOptions{})
	if err != nil {
		t.Fatalf("aucun itinéraire entre deux points du centre de Rennes : %v", err)
	}
	if p.LengthM < 500 || p.LengthM > 5000 {
		t.Errorf("itinéraire de %.0f m, attendu entre 500 et 5000", p.LengthM)
	}
}
