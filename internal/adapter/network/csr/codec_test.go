package csr_test

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
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
	if len(gotProv.Sources) != 1 {
		t.Fatalf("%d source(s), attendu 1 : la comparaison suivante ne vérifierait rien", len(gotProv.Sources))
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

// TestCodecAllerRetourProvenanceChampParChamp vérifie que chaque champ de
// domain.Provenance survit à l'aller-retour Write/ReadGraph, y compris ceux
// que TestCodecAllerRetour ne compare pas (File, ConfigHash, ordre des
// sources). Toutes les valeurs sont distinctes les unes des autres : deux
// champs partageant la même valeur laisseraient passer une interversion
// (ex. Name et File échangés), et une valeur vide laisserait passer une
// perte pure. La conversion domain.Provenance <-> provenanceHeader fait
// exactement ce travail champ par champ ; rien d'autre ne le garantit.
func TestCodecAllerRetourProvenanceChampParChamp(t *testing.T) {
	g := carre(t)
	prov := domain.Provenance{
		BuiltAt:    "builtat-valeur",
		ConfigHash: "confighash-valeur",
		Sources: []domain.Source{
			{Name: "name-0", File: "file-0", SHA256: "sha256-0", SizeBytes: 100},
			{Name: "name-1", File: "file-1", SHA256: "sha256-1", SizeBytes: 200},
		},
	}

	var buf bytes.Buffer
	if err := csr.Write(&buf, g, prov); err != nil {
		t.Fatalf("Write : %v", err)
	}

	_, got, err := csr.ReadGraph(&buf)
	if err != nil {
		t.Fatalf("ReadGraph : %v", err)
	}

	if got.BuiltAt != prov.BuiltAt {
		t.Errorf("BuiltAt = %q, attendu %q", got.BuiltAt, prov.BuiltAt)
	}
	if got.ConfigHash != prov.ConfigHash {
		t.Errorf("ConfigHash = %q, attendu %q", got.ConfigHash, prov.ConfigHash)
	}
	if len(got.Sources) != len(prov.Sources) {
		t.Fatalf("%d source(s), attendu %d : les comparaisons suivantes ne vérifieraient rien",
			len(got.Sources), len(prov.Sources))
	}
	for i, want := range prov.Sources {
		if got.Sources[i] != want {
			t.Errorf("Sources[%d] = %+v, attendu %+v", i, got.Sources[i], want)
		}
	}
}

// champSourcesBrut écrit p et renvoie la valeur JSON brute du champ "sources"
// dans l'en-tête produit. ReadGraph ne convient pas ici : depuisEnTete boucle
// sur h.Sources et n'exécute aucune itération que l'en-tête porte "null" ou
// "[]", si bien que domain.Provenance.Sources revient nil dans les deux cas
// — la distinction ne survit que dans les octets de l'en-tête, pas dans le
// type reconstruit.
func champSourcesBrut(t *testing.T, p domain.Provenance) string {
	t.Helper()

	g := carre(t)
	var buf bytes.Buffer
	if err := csr.Write(&buf, g, p); err != nil {
		t.Fatalf("Write : %v", err)
	}
	data := buf.Bytes()

	headerLen := binary.LittleEndian.Uint32(data[6:10])
	header := data[10 : 10+int(headerLen)]

	var champs map[string]json.RawMessage
	if err := json.Unmarshal(header, &champs); err != nil {
		t.Fatalf("en-tête illisible : %v", err)
	}
	return string(champs["sources"])
}

// TestVersEnTeteDistingueSourcesNilEtVide verrouille les deux cas dégénérés
// de la conversion domain.Provenance -> provenanceHeader : un Sources nil
// doit rester "null" dans l'en-tête, et ne pas se confondre avec un Sources
// vide non-nil, qui doit rester "[]". Les artefacts déjà produits avec un
// Sources nil (le cas de csr.Provenance{} dans les tests de ce fichier)
// verraient leur en-tête changer si les deux convergeaient vers la même
// représentation.
func TestVersEnTeteDistingueSourcesNilEtVide(t *testing.T) {
	sourcesNil := domain.Provenance{BuiltAt: "sans-sources"}
	sourcesVide := domain.Provenance{BuiltAt: "sources-vides", Sources: []domain.Source{}}

	if got := champSourcesBrut(t, sourcesNil); got != "null" {
		t.Errorf(`Sources nil : champ "sources" de l'en-tête = %s, attendu "null"`, got)
	}
	if got := champSourcesBrut(t, sourcesVide); got != "[]" {
		t.Errorf(`Sources vide non-nil : champ "sources" de l'en-tête = %s, attendu "[]"`, got)
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
