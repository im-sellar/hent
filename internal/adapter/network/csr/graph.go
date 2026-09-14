// Package csr implémente le graphe routier en représentation
// « compressed sparse row » : un tableau d'offsets indexé par nœud et un
// tableau d'arêtes contiguës. Cette disposition est compacte et contiguë en
// mémoire, donc rapide à parcourir — ce qui compte, l'A* traversant des
// centaines de milliers d'arêtes par requête.
package csr

import (
	"sort"
	"sync/atomic"

	"github.com/im-sellar/hent/internal/domain"
)

type Graph struct {
	coords  []domain.Coord
	offsets []uint32 // len == len(coords)+1
	targets []domain.NodeRef
	attrs   []EdgeAttrs
	bbox    domain.BBox
	spatial *spatialIndex

	// reverse[e] est la référence de l'arête inverse du même tronçon
	// bidirectionnel — voir Reverse.
	reverse []uint32

	// mainComponent marque, un bit par nœud, l'appartenance à la plus grande
	// composante connexe du graphe — voir computeMainComponent et NearestNode.
	mainComponent bitset

	// exploredTotal cumule les nœuds dépilés par TOUTES les recherches, y
	// compris celles qui échouent — ce sont elles qui explorent le plus.
	exploredTotal atomic.Int64
}

func (g *Graph) NumNodes() int { return len(g.coords) }
func (g *Graph) NumEdges() int { return len(g.targets) }

func (g *Graph) Coord(n domain.NodeRef) domain.Coord { return g.coords[n] }

// ExploredNodesTotal retourne le cumul des nœuds dépilés depuis le
// chargement du graphe, toutes recherches confondues (succès et échecs).
func (g *Graph) ExploredNodesTotal() int64 { return g.exploredTotal.Load() }

// EdgeRange retourne l'intervalle demi-ouvert [start, end) des arêtes
// sortantes du nœud n.
func (g *Graph) EdgeRange(n domain.NodeRef) (start, end domain.EdgeRef) {
	return domain.EdgeRef(g.offsets[n]), domain.EdgeRef(g.offsets[n+1])
}

func (g *Graph) Target(e domain.EdgeRef) domain.NodeRef { return g.targets[e] }
func (g *Graph) Attrs(e domain.EdgeRef) EdgeAttrs       { return g.attrs[e] }
func (g *Graph) BBox() domain.BBox                      { return g.bbox }

// Reverse retourne la référence de l'arête inverse de e — celle qui parcourt
// le même tronçon dans l'autre sens. Un tronçon bidirectionnel donne toujours
// deux EdgeRef distincts ; c'est cette table qui permet de les traiter comme
// une seule unité de réutilisation dans FindPath, faute de quoi emprunter un
// tronçon dans un sens puis dans l'autre échapperait à la pénalité anti-va-
// et-vient (voir astar.go).
func (g *Graph) Reverse(e domain.EdgeRef) domain.EdgeRef { return domain.EdgeRef(g.reverse[e]) }

type pendingEdge struct {
	from, to domain.NodeRef
	attrs    EdgeAttrs
}

type Builder struct {
	coords []domain.Coord
	edges  []pendingEdge
}

func NewBuilder() *Builder { return &Builder{} }

func (b *Builder) AddNode(c domain.Coord) domain.NodeRef {
	b.coords = append(b.coords, c)
	return domain.NodeRef(len(b.coords) - 1)
}

func (b *Builder) Coord(n domain.NodeRef) domain.Coord { return b.coords[n] }

// AddEdge ajoute une arête dirigée de from vers to.
//
// Contrat sur lequel Build s'appuie pour reconstituer la table des arêtes
// inverses (voir Graph.Reverse) : un tronçon bidirectionnel doit être ajouté
// par deux appels consécutifs, d'abord (from, to) puis (to, from). C'est ce
// que font tous les appelants actuels — osmsource.assemble et les graphes de
// test — et Build en déduit l'appariement sans avoir à le faire porter par
// pendingEdge.
func (b *Builder) AddEdge(from, to domain.NodeRef, a EdgeAttrs) {
	b.edges = append(b.edges, pendingEdge{from: from, to: to, attrs: a})
}

// Build trie les arêtes par nœud source, calcule les offsets, la table des
// arêtes inverses et la composante connexe principale.
func (b *Builder) Build() *Graph {
	// Le tri porte sur les indices plutôt que sur b.edges directement : il
	// faut pouvoir retrouver, après coup, à quel indice trié a atterri
	// l'arête inverse de chaque arête (voir la boucle de reconstruction de
	// reverse ci-dessous). Un tri instable romprait cet appariement.
	order := make([]int, len(b.edges))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		return b.edges[order[i]].from < b.edges[order[j]].from
	})

	g := &Graph{
		coords:  b.coords,
		offsets: make([]uint32, len(b.coords)+1),
		targets: make([]domain.NodeRef, len(b.edges)),
		attrs:   make([]EdgeAttrs, len(b.edges)),
		reverse: make([]uint32, len(b.edges)),
	}

	// Comptage par nœud, puis somme préfixe.
	counts := make([]uint32, len(b.coords))
	for _, e := range b.edges {
		counts[e.from]++
	}
	var acc uint32
	for i, c := range counts {
		g.offsets[i] = acc
		acc += c
	}
	g.offsets[len(counts)] = acc

	// newIndex[i] retrouve, pour l'indice original i dans b.edges, sa
	// position après tri — nécessaire pour traduire i^1 (son inverse, par le
	// contrat de AddEdge) en référence d'arête dans le graphe trié.
	newIndex := make([]uint32, len(b.edges))
	for pos, orig := range order {
		newIndex[orig] = uint32(pos)
	}

	for pos, orig := range order {
		e := b.edges[orig]
		g.targets[pos] = e.to
		g.attrs[pos] = e.attrs
		g.reverse[pos] = newIndex[orig^1]
	}

	g.bbox = computeBBox(b.coords)
	g.spatial = buildSpatialIndex(g.coords)
	g.mainComponent = computeMainComponent(g)
	return g
}

func computeBBox(coords []domain.Coord) domain.BBox {
	if len(coords) == 0 {
		return domain.BBox{}
	}
	b := domain.BBox{Min: coords[0], Max: coords[0]}
	for _, c := range coords[1:] {
		b.Min.Lat = min(b.Min.Lat, c.Lat)
		b.Min.Lon = min(b.Min.Lon, c.Lon)
		b.Max.Lat = max(b.Max.Lat, c.Lat)
		b.Max.Lon = max(b.Max.Lon, c.Lon)
	}
	return b
}

// bitset est un tableau de bits compact, un bit par nœud : le graphe
// Bretagne en compte 6 millions, un []bool coûterait huit fois plus de
// mémoire qu'un tableau de booléens n'apporte de simplicité ici.
type bitset []byte

func newBitset(n int) bitset { return make(bitset, (n+7)/8) }

func (bs bitset) set(i int) { bs[i/8] |= 1 << uint(i%8) }

func (bs bitset) get(i int) bool { return bs[i/8]&(1<<uint(i%8)) != 0 }

// computeMainComponent étiquette la plus grande composante connexe du graphe
// par une union-find sur les arêtes. Le graphe réel n'est pas connexe : le
// réseau routier de la Bretagne compte plusieurs milliers de fragments
// (impasses, cours, tronçons coupés au bord de l'extrait), et un nœud qui n'y
// appartient pas ne mène nulle part — c'est l'information que NearestNode
// utilise pour ne jamais accrocher un point de départ dans une impasse sans
// issue.
func computeMainComponent(g *Graph) bitset {
	n := g.NumNodes()
	parent := make([]int32, n)
	size := make([]int32, n)
	for i := range parent {
		parent[i] = int32(i)
		size[i] = 1
	}

	var find func(x int32) int32
	find = func(x int32) int32 {
		for parent[x] != x {
			parent[x] = parent[parent[x]] // compression de chemin par demi-saut
			x = parent[x]
		}
		return x
	}
	union := func(a, b int32) {
		ra, rb := find(a), find(b)
		if ra == rb {
			return
		}
		if size[ra] < size[rb] {
			ra, rb = rb, ra
		}
		parent[rb] = ra
		size[ra] += size[rb]
	}

	for node := 0; node < n; node++ {
		start, end := g.EdgeRange(domain.NodeRef(node))
		for e := start; e < end; e++ {
			union(int32(node), int32(g.targets[e]))
		}
	}

	var mainRoot int32 = -1
	var mainSize int32
	for i := 0; i < n; i++ {
		if find(int32(i)) == int32(i) && size[i] > mainSize {
			mainRoot, mainSize = int32(i), size[i]
		}
	}

	bits := newBitset(n)
	for i := 0; i < n; i++ {
		if find(int32(i)) == mainRoot {
			bits.set(i)
		}
	}
	return bits
}
