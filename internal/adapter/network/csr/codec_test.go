package csr_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/im-sellar/hent/internal/adapter/network/csr"
	"github.com/im-sellar/hent/internal/domain"
)

func TestCodecAllerRetour(t *testing.T) {
	g := carre(t)
	prov := domain.Provenance{
		BuiltAt:    "2026-08-18T10:00:00Z",
		ConfigHash: "abc123",
		Sources: []domain.Source{{
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

	// L'index spatial doit être reconstruit à la lecture, avec les bonnes
	// coordonnées : si NearestNode ne vérifiait que « trouvé », des
	// coordonnées mal relues (nœuds permutés, latitude/longitude inversées)
	// pourraient encore répondre par coïncidence. On vise donc un nœud précis
	// — le coin 2 du carré — et on exige que ce soit exactement lui, pas le
	// nœud 0 qu'une valeur par défaut renverrait sans rien prouver.
	n, ok := got.NearestNode(domain.Coord{Lat: 48.1089, Lon: -1.6671})
	if !ok {
		t.Fatal("index spatial non reconstruit après lecture")
	}
	if n != 2 {
		t.Errorf("nœud le plus proche = %d, attendu 2", n)
	}
}

func TestCodecRefuseUnMauvaisFichier(t *testing.T) {
	_, _, err := csr.ReadGraph(bytes.NewReader([]byte("ce n'est pas un graphe")))
	if !errors.Is(err, csr.ErrBadMagic) {
		t.Fatalf("erreur = %v, attendu ErrBadMagic", err)
	}
}

func TestCodecRefuseVersionInconnue(t *testing.T) {
	g := carre(t)
	var buf bytes.Buffer
	if err := csr.Write(&buf, g, domain.Provenance{}); err != nil {
		t.Fatalf("Write : %v", err)
	}

	data := buf.Bytes()
	// La version tient sur les deux octets qui suivent la marque magique
	// « HENT » (4 octets) ; on l'altère pour simuler un format futur ou
	// incompatible.
	data[4]++

	_, _, err := csr.ReadGraph(bytes.NewReader(data))
	if !errors.Is(err, csr.ErrBadVersion) {
		t.Fatalf("erreur = %v, attendu ErrBadVersion", err)
	}
}

// offsetsLayout recalcule, à partir des octets réellement écrits par Write,
// la position de chaque section utile aux tests de corruption ci-dessous —
// plutôt que de coder en dur des décalages qui se dérégleraient au moindre
// changement de disposition binaire.
func offsetsLayout(t *testing.T, data []byte) (numNodesOffset, offsetsOffset int, numNodes uint32) {
	t.Helper()
	headerLen := binary.LittleEndian.Uint32(data[6:10])
	numNodesOffset = 10 + int(headerLen)
	numNodes = binary.LittleEndian.Uint32(data[numNodesOffset : numNodesOffset+4])
	coordsOffset := numNodesOffset + 4
	offsetsOffset = coordsOffset + int(numNodes)*16
	return numNodesOffset, offsetsOffset, numNodes
}

// TestCodecRefuseCibleHorsBornes vérifie que ReadGraph refuse un artefact
// dont une arête cible un nœud qui n'existe pas — le genre de corruption
// qu'un transfert interrompu produit et que les seuls plafonds de taille ne
// peuvent pas attraper.
func TestCodecRefuseCibleHorsBornes(t *testing.T) {
	g := carre(t)
	var buf bytes.Buffer
	if err := csr.Write(&buf, g, domain.Provenance{}); err != nil {
		t.Fatalf("Write : %v", err)
	}
	data := buf.Bytes()

	_, offsetsOffset, numNodes := offsetsLayout(t, data)
	numEdgesOffset := offsetsOffset + (int(numNodes)+1)*4
	targetsOffset := numEdgesOffset + 4

	binary.LittleEndian.PutUint32(data[targetsOffset:targetsOffset+4], numNodes+100)

	if _, _, err := csr.ReadGraph(bytes.NewReader(data)); err == nil {
		t.Fatal("ReadGraph aurait dû refuser une arête ciblant un nœud hors bornes")
	}
}

// TestCodecRefuseOffsetsNonCroissants vérifie que ReadGraph refuse un
// artefact dont les offsets ne sont pas croissants — sans quoi EdgeRange
// produirait un intervalle absurde et FindPath paniquerait ou explorerait
// n'importe quoi.
func TestCodecRefuseOffsetsNonCroissants(t *testing.T) {
	g := carre(t)
	var buf bytes.Buffer
	if err := csr.Write(&buf, g, domain.Provenance{}); err != nil {
		t.Fatalf("Write : %v", err)
	}
	data := buf.Bytes()

	_, offsetsOffset, _ := offsetsLayout(t, data)
	binary.LittleEndian.PutUint32(data[offsetsOffset:offsetsOffset+4], 1)
	binary.LittleEndian.PutUint32(data[offsetsOffset+4:offsetsOffset+8], 0)

	if _, _, err := csr.ReadGraph(bytes.NewReader(data)); err == nil {
		t.Fatal("ReadGraph aurait dû refuser des offsets non croissants")
	}
}

// TestCodecRefuseDernierOffsetIncoherent vérifie que ReadGraph refuse un
// artefact dont le dernier offset ne correspond pas au nombre d'arêtes
// annoncé — une incohérence que la monotonie seule ne détecte pas.
func TestCodecRefuseDernierOffsetIncoherent(t *testing.T) {
	g := carre(t)
	var buf bytes.Buffer
	if err := csr.Write(&buf, g, domain.Provenance{}); err != nil {
		t.Fatalf("Write : %v", err)
	}
	data := buf.Bytes()

	_, offsetsOffset, numNodes := offsetsLayout(t, data)
	lastOffsetPos := offsetsOffset + int(numNodes)*4
	last := binary.LittleEndian.Uint32(data[lastOffsetPos : lastOffsetPos+4])
	binary.LittleEndian.PutUint32(data[lastOffsetPos:lastOffsetPos+4], last+1)

	if _, _, err := csr.ReadGraph(bytes.NewReader(data)); err == nil {
		t.Fatal("ReadGraph aurait dû refuser un dernier offset incohérent avec le nombre d'arêtes")
	}
}

func TestCodecRefuseCompteurDeNoeudsDemesure(t *testing.T) {
	g := carre(t)
	var buf bytes.Buffer
	if err := csr.Write(&buf, g, domain.Provenance{}); err != nil {
		t.Fatalf("Write : %v", err)
	}
	data := buf.Bytes()

	// magic(4) + version(2) + headerLen(4) + provenance JSON, puis numNodes
	// sur 4 octets : on le remplace par un compteur délibérément aberrant,
	// comme le produirait un artefact tronqué ou corrompu.
	headerLen := binary.LittleEndian.Uint32(data[6:10])
	offset := 10 + int(headerLen)
	binary.LittleEndian.PutUint32(data[offset:offset+4], 0xFFFFFFFF)

	if _, _, err := csr.ReadGraph(bytes.NewReader(data)); err == nil {
		t.Fatal("ReadGraph aurait dû refuser un nombre de nœuds démesuré, sans tenter d'allouer")
	}
}
