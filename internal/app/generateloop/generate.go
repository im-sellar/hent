// Package generateloop porte la stratégie de génération de boucles. C'est du
// métier, pas de l'infrastructure : il ne connaît du réseau que le port
// RouteNetwork, et se teste sur une grille synthétique.
package generateloop

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"runtime"
	"sort"
	"sync/atomic"

	"golang.org/x/sync/errgroup"

	"github.com/im-sellar/hent/internal/app/port"
	"github.com/im-sellar/hent/internal/domain"
)

var (
	ErrStartOutOfRange = errors.New("le point de départ est hors de la zone couverte")
	ErrNoLoopFound     = errors.New("aucune boucle trouvée pour cette requête")
)

const (
	// Nombre de directions explorées. Chaque candidate coûte quatre appels de
	// plus court chemin ; elles tournent en parallèle.
	numCandidates = 20

	// Multiplicateur appliqué aux arêtes déjà consommées par les segments
	// précédents. Assez haut pour décourager le retour sur ses pas, assez bas
	// pour autoriser un passage obligé — un pont, un col.
	reuseFactor = 4.0

	// Nombre de waypoints intermédiaires. Trois, répartis à 120°, donnent une
	// vraie boucle ; deux dégénèrent en aller-retour.
	numWaypoints = 3

	// Itérations de la dichotomie sur le facteur de détour.
	maxRadiusIterations = 5

	detourLo, detourHi = 0.8, 2.5
)

type Request struct {
	Start      domain.Coord
	DistanceM  float64
	Tolerance  float64
	Prefs      domain.Preferences
	MaxResults int
	Variant    int
}

type Generator struct {
	net port.RouteNetwork

	dropped atomic.Int64
}

func New(net port.RouteNetwork) *Generator { return &Generator{net: net} }

// Stats retourne les compteurs cumulés depuis le démarrage. Les nœuds
// explorés viennent du réseau et non du Generator : appendSegment ne voit
// que les segments retenus, alors qu'une recherche qui échoue est
// précisément celle qui a exploré le plus.
func (g *Generator) Stats() (exploredNodes, droppedCandidates int64) {
	return g.net.ExploredNodesTotal(), g.dropped.Load()
}

func (g *Generator) Generate(ctx context.Context, req Request) ([]domain.Loop, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	req = req.withDefaults()

	start, ok := g.net.NearestNode(req.Start)
	if !ok {
		return nil, ErrStartOutOfRange
	}

	// Le hasard est dérivé de la requête : deux appels identiques donnent le
	// même résultat, ce qui rend l'API cachable et les GPX régénérables.
	rng := rand.New(rand.NewSource(seedOf(req)))
	angles := make([]float64, numCandidates)
	for i := range angles {
		angles[i] = float64(i)*2*math.Pi/numCandidates + rng.Float64()*0.3
	}

	results := make([]domain.Loop, numCandidates)
	found := make([]bool, numCandidates)

	group, gctx := errgroup.WithContext(ctx)
	group.SetLimit(runtime.NumCPU())
	for i, theta := range angles {
		i, theta := i, theta
		group.Go(func() error {
			loop, err := g.candidate(gctx, start, theta, req)
			if err != nil {
				if gctx.Err() != nil {
					return gctx.Err()
				}
				g.dropped.Add(1) // une direction sans issue n'est pas une erreur
				return nil
			}
			results[i], found[i] = loop, true
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}

	var loops []domain.Loop
	for i, ok := range found {
		if ok {
			loops = append(loops, results[i])
		}
	}
	if len(loops) == 0 {
		return nil, ErrNoLoopFound
	}

	loops = dedupe(loops)
	sort.Slice(loops, func(a, b int) bool {
		sa := domain.NewScore(loops[a], req.DistanceM)
		sb := domain.NewScore(loops[b], req.DistanceM)
		return sa.Preferable(sb, req.Tolerance)
	})
	if len(loops) > req.MaxResults {
		loops = loops[:req.MaxResults]
	}
	return loops, nil
}

func (r Request) withDefaults() Request {
	if r.Tolerance <= 0 {
		r.Tolerance = 0.10
	}
	if r.MaxResults <= 0 {
		r.MaxResults = 5
	}
	return r
}

func seedOf(req Request) int64 {
	h := fnv.New64a()
	fmt.Fprintf(h, "%.6f|%.6f|%.1f|%.3f|%.3f|%d|%d",
		req.Start.Lat, req.Start.Lon, req.DistanceM, req.Tolerance,
		req.Prefs.AvoidPaved, req.MaxResults, req.Variant)
	return int64(h.Sum64())
}

// candidate cherche une boucle dans la direction theta, en corrigeant le rayon
// par dichotomie jusqu'à tomber dans la tolérance.
//
// Le facteur de détour traduit le fait qu'un chemin ne va pas droit : en
// relief il serpente, en plaine il file. On ne peut pas le connaître à
// l'avance, on le mesure.
func (g *Generator) candidate(ctx context.Context, start domain.NodeRef,
	theta float64, req Request) (domain.Loop, error) {

	lo, hi := detourLo, detourHi
	detour := 1.3

	for i := 0; i < maxRadiusIterations; i++ {
		radius := req.DistanceM / (2 * math.Pi * detour)

		loop, err := g.tryLoop(ctx, start, theta, radius, req)
		if err != nil {
			return domain.Loop{}, err
		}

		ecart := (loop.LengthM - req.DistanceM) / req.DistanceM
		if math.Abs(ecart) <= req.Tolerance {
			return loop, nil
		}

		// Trop long : le détour réel dépasse l'estimation, il faut réduire le
		// rayon, donc augmenter le facteur.
		if ecart > 0 {
			lo = detour
		} else {
			hi = detour
		}
		detour = (lo + hi) / 2
	}

	// Aucune itération n'a atteint la tolérance, et il n'y a délibérément pas
	// de repli sur « la moins mauvaise » : une boucle hors tolérance n'est pas
	// ce que l'utilisateur a demandé. Cette direction est abandonnée, les
	// dix-neuf autres sont explorées en parallèle — et les mesures montrent
	// qu'elles aboutissent presque toutes.
	return domain.Loop{}, ErrNoLoopFound
}

func (g *Generator) tryLoop(ctx context.Context, start domain.NodeRef,
	theta, radius float64, req Request) (domain.Loop, error) {

	center := g.net.Coord(start)
	weights := req.Prefs.Weights()
	used := make(map[domain.EdgeRef]struct{})

	loop := domain.Loop{
		Nodes:  []domain.NodeRef{start},
		Coords: []domain.Coord{center},
	}

	current := start
	for k := 0; k < numWaypoints; k++ {
		bearing := theta + float64(k)*2*math.Pi/numWaypoints
		wp, ok := g.net.NearestNode(domain.Offset(center, radius, bearing))
		if !ok || wp == current {
			continue
		}

		seg, err := g.net.FindPath(ctx, current, wp, weights, domain.PathOptions{
			UsedEdges: used, ReuseFactor: reuseFactor,
		})
		if err != nil {
			return domain.Loop{}, err
		}
		appendSegment(&loop, seg, used)
		current = wp
	}

	seg, err := g.net.FindPath(ctx, current, start, weights, domain.PathOptions{
		UsedEdges: used, ReuseFactor: reuseFactor,
	})
	if err != nil {
		return domain.Loop{}, err
	}
	appendSegment(&loop, seg, used)

	if loop.Nodes[len(loop.Nodes)-1] != start {
		return domain.Loop{}, ErrNoLoopFound
	}
	return loop, nil
}

// appendSegment recolle un segment à la boucle en évitant de dupliquer le
// nœud de jonction, et note ses arêtes comme consommées.
func appendSegment(loop *domain.Loop, seg domain.Path, used map[domain.EdgeRef]struct{}) {
	if len(seg.Nodes) > 1 {
		loop.Nodes = append(loop.Nodes, seg.Nodes[1:]...)
		loop.Coords = append(loop.Coords, seg.Coords[1:]...)
	}
	loop.Edges = append(loop.Edges, seg.Edges...)
	loop.LengthM += seg.LengthM
	loop.UnpavedM += seg.UnpavedM
	loop.TrafficExposureM += seg.TrafficExposureM

	for _, e := range seg.Edges {
		used[e] = struct{}{}
	}
}
