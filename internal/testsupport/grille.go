// Package testsupport fournit des doubles de test partagés entre plusieurs
// paquets de l'arbre internal, pour éviter qu'une même implémentation soit
// copiée d'un paquet de test à l'autre.
package testsupport

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"sync/atomic"

	"github.com/im-sellar/hent/internal/domain"
)

// Grille est un réseau synthétique de n×n nœuds espacés de `pas` mètres,
// relié en quatre-connexité. Elle implémente port.RouteNetwork sans rien
// emprunter à l'adaptateur réel, ce qui permet aux couches applicative et
// HTTP de se tester chacune sans dépendre de l'autre.
type Grille struct {
	n      int
	pas    float64
	origin domain.Coord
	coords []domain.Coord
	// voisins[n] = arêtes sortantes, chacune (cible, longueur)
	voisins  [][]arete
	explored atomic.Int64
}

type arete struct {
	cible domain.NodeRef
	ref   domain.EdgeRef
	long  float64
}

var errPasDeChemin = errors.New("aucun chemin")

func NouvelleGrille(n int, pas float64) *Grille {
	g := &Grille{
		n: n, pas: pas,
		origin:  domain.Coord{Lat: 48.10, Lon: -1.68},
		coords:  make([]domain.Coord, n*n),
		voisins: make([][]arete, n*n),
	}
	// Les nœuds sont écartés de la grille parfaite par un décalage déterministe :
	// un réseau routier réel n'est jamais régulier, et cette régularité ferait
	// retomber des directions voisines sur les mêmes nœuds, masquant l'effet des
	// paramètres de génération.
	bruit := rand.New(rand.NewSource(42))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			c := domain.Offset(
				domain.Offset(g.origin, float64(x)*pas, math.Pi/2),
				float64(y)*pas, 0)
			dx := (bruit.Float64()*2 - 1) * 0.4 * pas
			dy := (bruit.Float64()*2 - 1) * 0.4 * pas
			g.coords[y*n+x] = domain.Offset(domain.Offset(c, dx, math.Pi/2), dy, 0)
		}
	}

	var ref domain.EdgeRef
	lier := func(a, b domain.NodeRef) {
		d := domain.HaversineM(g.coords[a], g.coords[b])
		g.voisins[a] = append(g.voisins[a], arete{cible: b, ref: ref, long: d})
		ref++
		g.voisins[b] = append(g.voisins[b], arete{cible: a, ref: ref, long: d})
		ref++
	}
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			i := domain.NodeRef(y*n + x)
			if x+1 < n {
				lier(i, i+1)
			}
			if y+1 < n {
				lier(i, i+domain.NodeRef(n))
			}
		}
	}
	return g
}

func (g *Grille) Coord(n domain.NodeRef) domain.Coord { return g.coords[n] }

func (g *Grille) BBox() domain.BBox {
	b := domain.BBox{Min: g.coords[0], Max: g.coords[0]}
	for _, c := range g.coords {
		b.Min.Lat, b.Min.Lon = math.Min(b.Min.Lat, c.Lat), math.Min(b.Min.Lon, c.Lon)
		b.Max.Lat, b.Max.Lon = math.Max(b.Max.Lat, c.Lat), math.Max(b.Max.Lon, c.Lon)
	}
	return b
}

func (g *Grille) NearestNode(c domain.Coord) (domain.NodeRef, bool) {
	if !g.BBox().Contains(c) {
		return 0, false
	}
	best, bestD := domain.NodeRef(0), math.MaxFloat64
	for i, cc := range g.coords {
		if d := domain.HaversineM(c, cc); d < bestD {
			best, bestD = domain.NodeRef(i), d
		}
	}
	return best, true
}

// FindPath : Dijkstra simple, suffisant à l'échelle de la grille de test.
func (g *Grille) FindPath(ctx context.Context, from, to domain.NodeRef,
	w domain.Weights, opt domain.PathOptions) (domain.Path, error) {

	// explored alimente le compteur cumulatif quel que soit le chemin de
	// sortie, au même titre que csr.Graph : c'est ce que teste le double sur
	// ExploredNodesTotal.
	explored := 0
	defer func() { g.explored.Add(int64(explored)) }()

	if err := ctx.Err(); err != nil {
		return domain.Path{}, err
	}
	if from == to {
		return domain.Path{Nodes: []domain.NodeRef{from}, Coords: []domain.Coord{g.coords[from]}}, nil
	}

	reuse := opt.ReuseFactor
	if reuse < 1 {
		reuse = 1
	}

	const inf = math.MaxFloat64
	dist := make([]float64, len(g.coords))
	prevN := make([]domain.NodeRef, len(g.coords))
	prevE := make([]domain.EdgeRef, len(g.coords))
	done := make([]bool, len(g.coords))
	for i := range dist {
		dist[i] = inf
	}
	dist[from] = 0

	for {
		best, bestD := -1, inf
		for i, d := range dist {
			if !done[i] && d < bestD {
				best, bestD = i, d
			}
		}
		if best < 0 {
			break
		}
		done[best] = true
		explored++

		for _, a := range g.voisins[best] {
			cost := a.long
			if _, used := opt.UsedEdges[a.ref]; used {
				cost *= reuse
			}
			if cand := bestD + cost; cand < dist[a.cible] {
				dist[a.cible] = cand
				prevN[a.cible] = domain.NodeRef(best)
				prevE[a.cible] = a.ref
			}
		}
	}

	if dist[to] == inf {
		return domain.Path{}, errPasDeChemin
	}

	var revN []domain.NodeRef
	var revE []domain.EdgeRef
	for n := to; n != from; n = prevN[n] {
		revN = append(revN, n)
		revE = append(revE, prevE[n])
	}
	revN = append(revN, from)

	p := domain.Path{
		Nodes:  make([]domain.NodeRef, len(revN)),
		Coords: make([]domain.Coord, len(revN)),
		Edges:  make([]domain.EdgeRef, len(revE)),
		Cost:   dist[to],
	}
	for i, n := range revN {
		j := len(revN) - 1 - i
		p.Nodes[j], p.Coords[j] = n, g.coords[n]
	}
	for i, e := range revE {
		j := len(revE) - 1 - i
		p.Edges[j] = e
	}
	// Longueur réelle : somme des arêtes empruntées.
	for i := 0; i+1 < len(p.Nodes); i++ {
		for _, a := range g.voisins[p.Nodes[i]] {
			if a.cible == p.Nodes[i+1] {
				p.LengthM += a.long
				p.UnpavedM += a.long // la grille de test est intégralement en chemin
				break
			}
		}
	}
	p.ExploredNodes = explored
	return p, nil
}

// ExploredNodesTotal retourne le cumul des nœuds dépilés depuis la création
// de la grille, toutes recherches confondues (succès et échecs) — le même
// contrat que csr.Graph.ExploredNodesTotal.
func (g *Grille) ExploredNodesTotal() int64 { return g.explored.Load() }

// Relies indique si deux nœuds sont reliés par une arête directe.
func (g *Grille) Relies(a, b domain.NodeRef) bool {
	for _, v := range g.voisins[a] {
		if v.cible == b {
			return true
		}
	}
	return false
}
