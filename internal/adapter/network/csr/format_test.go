package csr_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/im-sellar/hent/internal/adapter/network/csr"
	"github.com/im-sellar/hent/internal/domain"
)

var majArtefact = flag.Bool("maj-artefact", false,
	"réécrit l'artefact témoin au lieu de le relire")

const cheminArtefact = "testdata/artefact-v2.bin"

// provenanceTemoin est la provenance exacte encodée dans l'artefact versionné.
// Toute divergence entre cette valeur et ce que ReadGraph en retire signale un
// changement du format de graph.bin.
func provenanceTemoin() csr.Provenance {
	return csr.Provenance{
		BuiltAt: "2026-09-14T08:30:00Z",
		Sources: []csr.Source{{
			Name:      "bretagne",
			File:      "bretagne-latest.osm.pbf",
			SHA256:    "3f786850e387550fdab836ed7e6dc881de23001b",
			SizeBytes: 421337,
		}},
		ConfigHash: "9e107d9d372bb6826bd81d3542a419d6",
	}
}

// TestFormatArtefactStable relit un artefact produit avant le durcissement de
// l'architecture. Un test qui écrirait puis relirait avec le même code ne
// verrait pas une rupture de format : seul un fichier figé le peut.
func TestFormatArtefactStable(t *testing.T) {
	if *majArtefact {
		ecrireArtefactTemoin(t)
		return
	}

	f, err := os.Open(cheminArtefact)
	if err != nil {
		t.Fatalf("artefact témoin absent (%v) — le régénérer avec : go test ./internal/adapter/network/csr/ -run TestFormatArtefactStable -maj-artefact", err)
	}
	defer f.Close()

	g, prov, err := csr.ReadGraph(f)
	if err != nil {
		t.Fatalf("relecture de l'artefact témoin : %v", err)
	}

	if g.NumNodes() != 4 {
		t.Errorf("NumNodes = %d, attendu 4", g.NumNodes())
	}

	attendue := provenanceTemoin()
	if prov.BuiltAt != attendue.BuiltAt {
		t.Errorf("BuiltAt = %q, attendu %q", prov.BuiltAt, attendue.BuiltAt)
	}
	if prov.ConfigHash != attendue.ConfigHash {
		t.Errorf("ConfigHash = %q, attendu %q", prov.ConfigHash, attendue.ConfigHash)
	}
	if len(prov.Sources) != 1 {
		t.Fatalf("%d source(s), attendu 1 : les comparaisons suivantes ne vérifieraient rien",
			len(prov.Sources))
	}
	if prov.Sources[0] != attendue.Sources[0] {
		t.Errorf("Sources[0] = %+v, attendu %+v", prov.Sources[0], attendue.Sources[0])
	}
}

func ecrireArtefactTemoin(t *testing.T) {
	t.Helper()

	b := csr.NewBuilder()
	n := make([]domain.NodeRef, 4)
	coords := []domain.Coord{
		{Lat: 48.100, Lon: -1.680}, {Lat: 48.100, Lon: -1.667},
		{Lat: 48.109, Lon: -1.667}, {Lat: 48.109, Lon: -1.680},
	}
	for i, c := range coords {
		n[i] = b.AddNode(c)
	}
	for i := 0; i < 4; i++ {
		from, to := n[i], n[(i+1)%4]
		l := domain.HaversineM(coords[i], coords[(i+1)%4])
		b.AddEdge(from, to, csr.EdgeAttrs{LengthM: l})
		b.AddEdge(to, from, csr.EdgeAttrs{LengthM: l})
	}

	var buf bytes.Buffer
	if err := csr.Write(&buf, b.Build(), provenanceTemoin()); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(cheminArtefact), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cheminArtefact, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("artefact témoin réécrit : %s (%d octets)", cheminArtefact, buf.Len())
}
