package csr

import (
	"container/heap"
	"context"
	"errors"
	"math"

	"github.com/im-sellar/hent/internal/domain"
)

var (
	ErrNoPath         = errors.New("aucun chemin trouvé")
	ErrBudgetExceeded = errors.New("plafond de nœuds explorés atteint")
)

const (
	defaultMaxNodes = 400_000
	// Fréquence de vérification de l'annulation du contexte, en nœuds.
	ctxCheckInterval = 1024
)

type queueItem struct {
	node  domain.NodeRef
	f     float64
	index int
}

type priorityQueue []*queueItem

func (pq priorityQueue) Len() int           { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool { return pq[i].f < pq[j].f }

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index, pq[j].index = i, j
}

func (pq *priorityQueue) Push(x any) {
	item := x.(*queueItem)
	item.index = len(*pq)
	*pq = append(*pq, item)
}

func (pq *priorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*pq = old[:n-1]
	return item
}

// FindPath applique A* : on explore en priorité le nœud minimisant
// f(n) = g(n) + h(n), où h est la distance à vol d'oiseau jusqu'à la cible.
//
// h est admissible — elle ne surestime jamais — sous deux hypothèses, et non
// une seule :
//
//  1. Le coût minimal par mètre vaut exactement 1 (voir l'invariant sur
//     EdgeAttrs.Cost). C'est ce que vérifie TestFindPathEstOptimal.
//  2. La longueur d'une arête n'est jamais inférieure à la distance à vol
//     d'oiseau entre ses extrémités. Rien dans EdgeAttrs ne le contraint :
//     l'hypothèse tient parce que LengthM est calculée par HaversineM sur la
//     géométrie du tronçon, donc toujours supérieure ou égale à la corde. Une
//     source de données future qui importerait des longueurs pré-calculées
//     devrait la préserver, sous peine de rendre l'A* faux sans autre signal.
//
// Sur le retour anticipé from == to, ExploredNodes vaut 0 : aucun nœud n'a
// été dépilé de la file. C'est correct et volontaire — poser 1 fausserait la
// métrique pour un cas qui n'a rien exploré.
func (g *Graph) FindPath(
	ctx context.Context,
	from, to domain.NodeRef,
	w domain.Weights,
	opt domain.PathOptions,
) (domain.Path, error) {
	// explored alimente exploredTotal quel que soit le chemin de sortie —
	// succès, échec ou annulation — via le defer ci-dessous : c'est la seule
	// façon de compter aussi les recherches qui échouent après avoir
	// beaucoup exploré, alors qu'elles ne renvoient pas de Path porteur du
	// compte.
	explored := 0
	defer func() { g.exploredTotal.Add(int64(explored)) }()

	if int(from) >= len(g.coords) || int(to) >= len(g.coords) {
		return domain.Path{}, ErrNoPath
	}
	if err := ctx.Err(); err != nil {
		return domain.Path{}, err
	}
	if from == to {
		return domain.Path{Nodes: []domain.NodeRef{from}, Coords: []domain.Coord{g.coords[from]}}, nil
	}

	maxNodes := opt.MaxNodes
	if maxNodes <= 0 {
		maxNodes = defaultMaxNodes
	}
	reuse := opt.ReuseFactor
	if reuse < 1 {
		reuse = 1
	}

	target := g.coords[to]

	gScore := make([]float64, len(g.coords))
	parentNode := make([]domain.NodeRef, len(g.coords))
	parentEdge := make([]domain.EdgeRef, len(g.coords))
	settled := make([]bool, len(g.coords))
	for i := range gScore {
		gScore[i] = math.MaxFloat64
	}
	gScore[from] = 0

	pq := &priorityQueue{}
	heap.Push(pq, &queueItem{node: from, f: domain.HaversineM(g.coords[from], target)})

	for pq.Len() > 0 {
		if explored%ctxCheckInterval == 0 {
			if err := ctx.Err(); err != nil {
				return domain.Path{}, err
			}
		}

		cur := heap.Pop(pq).(*queueItem)
		if settled[cur.node] {
			continue
		}
		settled[cur.node] = true
		explored++

		if cur.node == to {
			p := g.rebuild(from, to, parentNode, parentEdge, gScore[to])
			p.ExploredNodes = explored
			return p, nil
		}

		if explored > maxNodes {
			return domain.Path{}, ErrBudgetExceeded
		}

		start, end := g.EdgeRange(cur.node)
		for e := start; e < end; e++ {
			next := g.targets[e]
			if settled[next] {
				continue
			}

			cost := g.attrs[e].Cost(w)
			if _, used := opt.UsedEdges[e]; used {
				cost *= reuse
			}

			if cand := gScore[cur.node] + cost; cand < gScore[next] {
				gScore[next] = cand
				parentNode[next] = cur.node
				parentEdge[next] = e
				heap.Push(pq, &queueItem{
					node: next,
					f:    cand + domain.HaversineM(g.coords[next], target),
				})
			}
		}
	}

	return domain.Path{}, ErrNoPath
}

func (g *Graph) rebuild(
	from, to domain.NodeRef,
	parentNode []domain.NodeRef,
	parentEdge []domain.EdgeRef,
	cost float64,
) domain.Path {
	var revNodes []domain.NodeRef
	var revEdges []domain.EdgeRef

	for n := to; n != from; n = parentNode[n] {
		revNodes = append(revNodes, n)
		revEdges = append(revEdges, parentEdge[n])
	}
	revNodes = append(revNodes, from)

	p := domain.Path{
		Nodes:  make([]domain.NodeRef, len(revNodes)),
		Coords: make([]domain.Coord, len(revNodes)),
		Edges:  make([]domain.EdgeRef, len(revEdges)),
		Cost:   cost,
	}
	for i, n := range revNodes {
		j := len(revNodes) - 1 - i
		p.Nodes[j] = n
		p.Coords[j] = g.coords[n]
	}
	for i, e := range revEdges {
		p.Edges[len(revEdges)-1-i] = e

		a := g.attrs[e]
		p.LengthM += a.LengthM
		if a.Surface != domain.SurfacePaved {
			p.UnpavedM += a.LengthM
		}
		p.TrafficExposureM += a.LengthM * float64(a.Traffic) / 255
	}
	return p
}
