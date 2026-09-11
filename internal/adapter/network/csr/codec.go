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
const formatVersion uint16 = 1

// Source décrit une donnée d'entrée avec de quoi la retrouver à l'identique.
type Source struct {
	Name      string `json:"name"`
	File      string `json:"file"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

// Provenance rend le build reproductible — exigence de l'ODbL, qui impose de
// pouvoir fournir soit la base dérivée, soit les moyens de la reconstruire.
type Provenance struct {
	BuiltAt    string   `json:"built_at"`
	Sources    []Source `json:"sources"`
	ConfigHash string   `json:"config_hash"`
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
func Write(w io.Writer, g *Graph, p Provenance) error {
	header, err := json.Marshal(p)
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
	return nil
}

// ReadGraph relit un graphe et sa provenance depuis r. L'index spatial n'est
// pas sérialisé : il est reconstruit ici, pas dans Write, car il ne dépend
// que des coordonnées déjà présentes dans le flux.
func ReadGraph(r io.Reader) (*Graph, Provenance, error) {
	var gotMagic [4]byte
	if _, err := io.ReadFull(r, gotMagic[:]); err != nil {
		return nil, Provenance{}, ErrBadMagic
	}
	if gotMagic != magic {
		return nil, Provenance{}, ErrBadMagic
	}

	var version uint16
	if err := binary.Read(r, order, &version); err != nil {
		return nil, Provenance{}, err
	}
	if version != formatVersion {
		return nil, Provenance{}, fmt.Errorf("%w : fichier en version %d, binaire en version %d",
			ErrBadVersion, version, formatVersion)
	}

	var headerLen uint32
	if err := binary.Read(r, order, &headerLen); err != nil {
		return nil, Provenance{}, err
	}
	if headerLen > maxHeaderLen {
		return nil, Provenance{}, fmt.Errorf("en-tête de %d octets, plafond %d : artefact tronqué ou corrompu",
			headerLen, maxHeaderLen)
	}
	header := make([]byte, headerLen)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, Provenance{}, err
	}
	var prov Provenance
	if err := json.Unmarshal(header, &prov); err != nil {
		return nil, Provenance{}, fmt.Errorf("provenance illisible : %w", err)
	}

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

	g.bbox = computeBBox(g.coords)
	g.spatial = buildSpatialIndex(g.coords)
	return g, prov, nil
}
