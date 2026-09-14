package csr

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/im-sellar/hent/internal/domain"
)

var (
	magic         = [4]byte{'H', 'E', 'N', 'T'}
	ErrBadMagic   = errors.New("ce fichier n'est pas un graphe hent")
	ErrBadVersion = errors.New("version de format non prise en charge")
)

// formatVersion doit être incrémentée à chaque changement de disposition
// binaire. routed refuse de démarrer sur une version inconnue plutôt que de
// lire des octets de travers.
//
// v2 ajoute la table des arêtes inverses (Graph.Reverse, correction I1 de la
// revue finale) et le bit de composante connexe principale par nœud
// (NearestNode, correction C1) : un artefact v1 n'a ni l'une ni l'autre, il
// est donc refusé plutôt que complété silencieusement.
const formatVersion uint16 = 2

// sourceHeader et provenanceHeader fixent la disposition de l'en-tête JSON de
// graph.bin. Les tags sont la définition du format de l'artefact : les
// modifier rend illisible tout fichier déjà produit. C'est la raison pour
// laquelle ils vivent ici et non sur domain.Provenance, que le domaine expose
// sans rien savoir de sa sérialisation.
type sourceHeader struct {
	Name      string `json:"name"`
	File      string `json:"file"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

type provenanceHeader struct {
	BuiltAt    string         `json:"built_at"`
	Sources    []sourceHeader `json:"sources"`
	ConfigHash string         `json:"config_hash"`
}

func versEnTete(p domain.Provenance) provenanceHeader {
	h := provenanceHeader{BuiltAt: p.BuiltAt, ConfigHash: p.ConfigHash}
	// Un Sources nil doit rester nil (l'en-tête sérialise alors "sources":null,
	// comme avant le déplacement de Provenance) : sans cette distinction, un
	// Sources nil et un Sources vide non-nil convergeraient vers la même
	// sortie et l'en-tête d'artefacts déjà produits avec un Sources nil
	// changerait.
	if p.Sources != nil {
		h.Sources = make([]sourceHeader, 0, len(p.Sources))
	}
	for _, s := range p.Sources {
		h.Sources = append(h.Sources, sourceHeader{
			Name: s.Name, File: s.File, SHA256: s.SHA256, SizeBytes: s.SizeBytes,
		})
	}
	return h
}

func depuisEnTete(h provenanceHeader) domain.Provenance {
	p := domain.Provenance{BuiltAt: h.BuiltAt, ConfigHash: h.ConfigHash}
	for _, s := range h.Sources {
		p.Sources = append(p.Sources, domain.Source{
			Name: s.Name, File: s.File, SHA256: s.SHA256, SizeBytes: s.SizeBytes,
		})
	}
	return p
}

var order = binary.LittleEndian

// Plafonds appliqués aux tailles lues dans l'en-tête avant toute allocation.
// Un artefact tronqué — graphbuild interrompu, disque plein, transfert coupé —
// ou simplement corrompu porte des compteurs arbitraires ; les allouer tels
// quels ferait réclamer plusieurs gigaoctets au démarrage du serveur. Les
// valeurs retenues laissent une marge considérable : la France entière compte
// environ dix fois moins de nœuds que le plafond.
const (
	maxHeaderLen = 1 << 20
	maxNodes     = 200_000_000
	maxEdges     = 800_000_000
)

// Write sérialise g et sa provenance p dans w au format binaire hent
// (en-tête magique + version + provenance JSON, puis nœuds et arêtes).
func Write(w io.Writer, g *Graph, p domain.Provenance) error {
	header, err := json.Marshal(versEnTete(p))
	if err != nil {
		return fmt.Errorf("sérialisation de la provenance : %w", err)
	}

	if _, err := w.Write(magic[:]); err != nil {
		return err
	}
	for _, v := range []any{
		formatVersion,
		uint32(len(header)),
	} {
		if err := binary.Write(w, order, v); err != nil {
			return err
		}
	}
	if _, err := w.Write(header); err != nil {
		return err
	}

	if err := binary.Write(w, order, uint32(len(g.coords))); err != nil {
		return err
	}
	for _, c := range g.coords {
		if err := binary.Write(w, order, c.Lat); err != nil {
			return err
		}
		if err := binary.Write(w, order, c.Lon); err != nil {
			return err
		}
	}
	if err := binary.Write(w, order, g.offsets); err != nil {
		return err
	}

	if err := binary.Write(w, order, uint32(len(g.targets))); err != nil {
		return err
	}
	if err := binary.Write(w, order, g.targets); err != nil {
		return err
	}
	for _, a := range g.attrs {
		// float32 suffit très largement à la précision utile ici (le mètre),
		// mais arrondit environ 43 % des longueurs vers le bas — écart de
		// l'ordre de 10⁻⁵ m sur une arête de 1 km, donc 10⁻⁷ en relatif. Cela
		// affaiblit formellement l'invariant de l'A* (Task 4) selon lequel une
		// arête n'est jamais plus courte que la corde entre ses extrémités,
		// mais sans effet observable sur des itinéraires kilométriques à
		// ±10 % de tolérance.
		if err := binary.Write(w, order, float32(a.LengthM)); err != nil {
			return err
		}
		if err := binary.Write(w, order, [3]uint8{
			uint8(a.Surface), uint8(a.Class), a.Traffic,
		}); err != nil {
			return err
		}
	}

	if err := binary.Write(w, order, g.reverse); err != nil {
		return err
	}
	// mainComponent est de longueur (numNodes+7)/8, déductible de numNodes
	// déjà écrit plus haut : pas besoin d'un préfixe de taille séparé.
	if _, err := w.Write(g.mainComponent); err != nil {
		return err
	}
	return nil
}

// ReadGraph relit un graphe et sa provenance depuis r. L'index spatial n'est
// pas sérialisé : il est reconstruit ici, pas dans Write, car il ne dépend
// que des coordonnées déjà présentes dans le flux.
func ReadGraph(r io.Reader) (*Graph, domain.Provenance, error) {
	var gotMagic [4]byte
	if _, err := io.ReadFull(r, gotMagic[:]); err != nil {
		return nil, domain.Provenance{}, ErrBadMagic
	}
	if gotMagic != magic {
		return nil, domain.Provenance{}, ErrBadMagic
	}

	var version uint16
	if err := binary.Read(r, order, &version); err != nil {
		return nil, domain.Provenance{}, err
	}
	if version != formatVersion {
		return nil, domain.Provenance{}, fmt.Errorf("%w : fichier en version %d, binaire en version %d",
			ErrBadVersion, version, formatVersion)
	}

	var headerLen uint32
	if err := binary.Read(r, order, &headerLen); err != nil {
		return nil, domain.Provenance{}, err
	}
	if headerLen > maxHeaderLen {
		return nil, domain.Provenance{}, fmt.Errorf("en-tête de %d octets, plafond %d : artefact tronqué ou corrompu",
			headerLen, maxHeaderLen)
	}
	header := make([]byte, headerLen)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, domain.Provenance{}, err
	}
	var h provenanceHeader
	if err := json.Unmarshal(header, &h); err != nil {
		return nil, domain.Provenance{}, fmt.Errorf("provenance illisible : %w", err)
	}
	prov := depuisEnTete(h)

	var numNodes uint32
	if err := binary.Read(r, order, &numNodes); err != nil {
		return nil, prov, err
	}
	if numNodes > maxNodes {
		return nil, prov, fmt.Errorf("%d nœuds annoncés, plafond %d : artefact tronqué ou corrompu",
			numNodes, maxNodes)
	}
	g := &Graph{
		coords:  make([]domain.Coord, numNodes),
		offsets: make([]uint32, numNodes+1),
	}
	for i := range g.coords {
		if err := binary.Read(r, order, &g.coords[i].Lat); err != nil {
			return nil, prov, err
		}
		if err := binary.Read(r, order, &g.coords[i].Lon); err != nil {
			return nil, prov, err
		}
	}
	if err := binary.Read(r, order, g.offsets); err != nil {
		return nil, prov, err
	}

	var numEdges uint32
	if err := binary.Read(r, order, &numEdges); err != nil {
		return nil, prov, err
	}
	if numEdges > maxEdges {
		return nil, prov, fmt.Errorf("%d arêtes annoncées, plafond %d : artefact tronqué ou corrompu",
			numEdges, maxEdges)
	}
	g.targets = make([]domain.NodeRef, numEdges)
	if err := binary.Read(r, order, g.targets); err != nil {
		return nil, prov, err
	}
	g.attrs = make([]EdgeAttrs, numEdges)
	for i := range g.attrs {
		var length float32
		var packed [3]uint8
		if err := binary.Read(r, order, &length); err != nil {
			return nil, prov, err
		}
		if err := binary.Read(r, order, &packed); err != nil {
			return nil, prov, err
		}
		g.attrs[i] = EdgeAttrs{
			LengthM: float64(length),
			Surface: domain.Surface(packed[0]),
			Class:   domain.WayClass(packed[1]),
			Traffic: packed[2],
		}
	}

	g.reverse = make([]uint32, numEdges)
	if err := binary.Read(r, order, g.reverse); err != nil {
		return nil, prov, err
	}
	g.mainComponent = make(bitset, (numNodes+7)/8)
	if _, err := io.ReadFull(r, g.mainComponent); err != nil {
		return nil, prov, err
	}

	if err := validateGraph(g); err != nil {
		return nil, prov, err
	}

	g.bbox = computeBBox(g.coords)
	g.spatial = buildSpatialIndex(g.coords)
	return g, prov, nil
}

// validateGraph vérifie la cohérence interne d'un graphe tout juste relu —
// pas la disposition qu'aurait dû produire Write, mais celle que produisent
// réellement les octets lus. Les plafonds appliqués plus haut empêchent une
// allocation démesurée ; ils ne disent rien de la cohérence des données une
// fois allouées. Un artefact complet mais corrompu (transfert interrompu puis
// repris de travers, bit retourné sur un disque sans ECC) les passerait sans
// cette étape, et ferait paniquer le service — hors d'une requête HTTP, dans
// une goroutine d'errgroup — à la première recherche de chemin.
func validateGraph(g *Graph) error {
	numNodes := uint32(len(g.coords))
	numEdges := uint32(len(g.targets))

	for i := 1; i < len(g.offsets); i++ {
		if g.offsets[i] < g.offsets[i-1] {
			return fmt.Errorf("offsets non croissants au nœud %d : artefact corrompu", i)
		}
	}
	if last := g.offsets[len(g.offsets)-1]; last != numEdges {
		return fmt.Errorf("dernier offset %d, attendu %d (nombre d'arêtes) : artefact corrompu",
			last, numEdges)
	}
	for e, t := range g.targets {
		if uint32(t) >= numNodes {
			return fmt.Errorf("arête %d cible le nœud %d, hors bornes (%d nœuds) : artefact corrompu",
				e, t, numNodes)
		}
	}
	for e, rev := range g.reverse {
		if rev >= numEdges {
			return fmt.Errorf("arête %d référence une inverse %d hors bornes (%d arêtes) : artefact corrompu",
				e, rev, numEdges)
		}
	}
	return nil
}
