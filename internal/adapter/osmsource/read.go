package osmsource

import (
	"context"
	"fmt"
	"os"

	"github.com/paulmach/osm"
	"github.com/paulmach/osm/osmpbf"

	"github.com/im-sellar/hent/internal/adapter/network/csr"
	"github.com/im-sellar/hent/internal/domain"
)

type Stats struct {
	Ways  int // tronçons OSM retenus
	Nodes int // nœuds conservés
	Edges int // segments (chaque segment donne deux arêtes dirigées)
}

type retainedWay struct {
	nodes   []osm.NodeID
	class   domain.WayClass
	surface domain.Surface
	traffic uint8
}

// Read lit un extrait PBF et construit le graphe.
//
// Deux passes sont nécessaires : le format PBF ne garantit pas que les nœuds
// précèdent les ways qui les référencent, et on ne veut mémoriser les
// coordonnées que des nœuds réellement utilisés. La première passe collecte
// les ways praticables, la seconde les coordonnées dont ils ont besoin.
func Read(ctx context.Context, path string) (*csr.Graph, Stats, error) {
	ways, needed, err := scanWays(ctx, path)
	if err != nil {
		return nil, Stats{}, err
	}

	coords, err := scanNodes(ctx, path, needed)
	if err != nil {
		return nil, Stats{}, err
	}

	return assemble(ways, coords)
}

func scanWays(ctx context.Context, path string) ([]retainedWay, map[osm.NodeID]struct{}, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("ouverture de %s : %w", path, err)
	}
	defer f.Close()

	scanner := osmpbf.New(ctx, f, 4)
	defer scanner.Close()
	scanner.SkipNodes = true
	scanner.SkipRelations = true

	var ways []retainedWay
	needed := make(map[osm.NodeID]struct{})

	for scanner.Scan() {
		w, ok := scanner.Object().(*osm.Way)
		if !ok || len(w.Nodes) < 2 {
			continue
		}

		class, surface, traffic, keep := Classify(w.TagMap())
		if !keep {
			continue
		}

		ids := make([]osm.NodeID, len(w.Nodes))
		for i, n := range w.Nodes {
			ids[i] = n.ID
			needed[n.ID] = struct{}{}
		}
		ways = append(ways, retainedWay{
			nodes: ids, class: class, surface: surface, traffic: traffic,
		})
	}
	return ways, needed, scanner.Err()
}

func scanNodes(ctx context.Context, path string, needed map[osm.NodeID]struct{}) (map[osm.NodeID]domain.Coord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("ouverture de %s : %w", path, err)
	}
	defer f.Close()

	scanner := osmpbf.New(ctx, f, 4)
	defer scanner.Close()
	scanner.SkipWays = true
	scanner.SkipRelations = true

	coords := make(map[osm.NodeID]domain.Coord, len(needed))
	for scanner.Scan() {
		n, ok := scanner.Object().(*osm.Node)
		if !ok {
			continue
		}
		if _, want := needed[n.ID]; want {
			coords[n.ID] = domain.Coord{Lat: n.Lat, Lon: n.Lon}
		}
	}
	return coords, scanner.Err()
}

func assemble(ways []retainedWay, coords map[osm.NodeID]domain.Coord) (*csr.Graph, Stats, error) {
	b := csr.NewBuilder()
	refs := make(map[osm.NodeID]domain.NodeRef, len(coords))

	nodeRef := func(id osm.NodeID) (domain.NodeRef, bool) {
		if r, ok := refs[id]; ok {
			return r, true
		}
		c, ok := coords[id]
		if !ok {
			// Nœud hors de l'extrait : le tronçon est coupé au bord de la
			// bbox, on saute simplement ce segment.
			return 0, false
		}
		r := b.AddNode(c)
		refs[id] = r
		return r, true
	}

	var stats Stats
	for _, w := range ways {
		stats.Ways++
		for i := 0; i+1 < len(w.nodes); i++ {
			from, ok1 := nodeRef(w.nodes[i])
			to, ok2 := nodeRef(w.nodes[i+1])
			if !ok1 || !ok2 || from == to {
				continue
			}

			attrs := csr.EdgeAttrs{
				LengthM: domain.HaversineM(b.Coord(from), b.Coord(to)),
				Surface: w.surface,
				Class:   w.class,
				Traffic: w.traffic,
			}
			b.AddEdge(from, to, attrs)
			b.AddEdge(to, from, attrs)
			stats.Edges++
		}
	}
	stats.Nodes = len(refs)

	if stats.Edges == 0 {
		return nil, stats, fmt.Errorf("aucun tronçon praticable trouvé dans l'extrait")
	}
	return b.Build(), stats, nil
}
