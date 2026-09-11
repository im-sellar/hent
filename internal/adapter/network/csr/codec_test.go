package csr_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/im-sellar/hent/internal/adapter/network/csr"
	"github.com/im-sellar/hent/internal/domain"
)

func TestCodecAllerRetour(t *testing.T) {
	g := carre(t)
	prov := csr.Provenance{
		BuiltAt:    "2026-08-18T10:00:00Z",
		ConfigHash: "abc123",
		Sources: []csr.Source{{
			Name: "geofabrik/bretagne", File: "bretagne-latest.osm.pbf",
			SHA256: "deadbeef", SizeBytes: 12345,
		}},
	}

	var buf bytes.Buffer
	if err := csr.Write(&buf, g, prov); err != nil {
		t.Fatalf("Write : %v", err)
	}

	got, gotProv, err := csr.ReadGraph(&buf)
	if err != nil {
		t.Fatalf("ReadGraph : %v", err)
	}

	if got.NumNodes() != g.NumNodes() || got.NumEdges() != g.NumEdges() {
		t.Fatalf("relu %d nœuds / %d arêtes, écrit %d / %d",
			got.NumNodes(), got.NumEdges(), g.NumNodes(), g.NumEdges())
	}
	if gotProv.Sources[0].SHA256 != "deadbeef" {
		t.Errorf("provenance perdue : %+v", gotProv)
	}

	// Les attributs et la topologie doivent survivre au tour.
	for n := domain.NodeRef(0); n < domain.NodeRef(g.NumNodes()); n++ {
		s1, e1 := g.EdgeRange(n)
		s2, e2 := got.EdgeRange(n)
		if s1 != s2 || e1 != e2 {
			t.Fatalf("nœud %d : intervalle [%d,%d) relu [%d,%d)", n, s1, e1, s2, e2)
		}
		for e := s1; e < e1; e++ {
			if g.Target(e) != got.Target(e) {
				t.Fatalf("arête %d : cible %d relue %d", e, g.Target(e), got.Target(e))
			}
			a, b := g.Attrs(e), got.Attrs(e)
			if a.Surface != b.Surface || a.Class != b.Class || a.Traffic != b.Traffic {
				t.Fatalf("arête %d : attributs %+v relus %+v", e, a, b)
			}
		}
	}

	// L'index spatial doit être reconstruit à la lecture.
	if _, ok := got.NearestNode(domain.Coord{Lat: 48.1001, Lon: -1.6801}); !ok {
		t.Error("index spatial non reconstruit après lecture")
	}
}

func TestCodecRefuseUnMauvaisFichier(t *testing.T) {
	_, _, err := csr.ReadGraph(bytes.NewReader([]byte("ce n'est pas un graphe")))
	if !errors.Is(err, csr.ErrBadMagic) {
		t.Fatalf("erreur = %v, attendu ErrBadMagic", err)
	}
}
