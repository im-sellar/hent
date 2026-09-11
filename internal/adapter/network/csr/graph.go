// Package csr implémente le graphe routier en représentation
// « compressed sparse row » : un tableau d'offsets indexé par nœud et un
// tableau d'arêtes contiguës. Cette disposition est compacte et contiguë en
// mémoire, donc rapide à parcourir — ce qui compte, l'A* traversant des
// centaines de milliers d'arêtes par requête.
package csr

import (
	"sort"

	"github.com/im-sellar/hent/internal/domain"
)

type Graph struct {
	coords  []domain.Coord
	offsets []uint32 // len == len(coords)+1
	targets []domain.NodeRef
	attrs   []EdgeAttrs
	bbox    domain.BBox
	spatial *spatialIndex
}

func (g *Graph) NumNodes() int { return len(g.coords) }
func (g *Graph) NumEdges() int { return len(g.targets) }

func (g *Graph) Coord(n domain.NodeRef) domain.Coord { return g.coords[n] }

// EdgeRange retourne l'intervalle demi-ouvert [start, end) des arêtes
// sortantes du nœud n.
func (g *Graph) EdgeRange(n domain.NodeRef) (start, end domain.EdgeRef) {
	return domain.EdgeRef(g.offsets[n]), domain.EdgeRef(g.offsets[n+1])
}

func (g *Graph) Target(e domain.EdgeRef) domain.NodeRef { return g.targets[e] }
func (g *Graph) Attrs(e domain.EdgeRef) EdgeAttrs       { return g.attrs[e] }
func (g *Graph) BBox() domain.BBox                      { return g.bbox }

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

func (b *Builder) AddEdge(from, to domain.NodeRef, a EdgeAttrs) {
	b.edges = append(b.edges, pendingEdge{from: from, to: to, attrs: a})
}

// Build trie les arêtes par nœud source et calcule les offsets.
func (b *Builder) Build() *Graph {
	sort.Slice(b.edges, func(i, j int) bool { return b.edges[i].from < b.edges[j].from })

	g := &Graph{
		coords:  b.coords,
		offsets: make([]uint32, len(b.coords)+1),
		targets: make([]domain.NodeRef, len(b.edges)),
		attrs:   make([]EdgeAttrs, len(b.edges)),
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

	for i, e := range b.edges {
		g.targets[i] = e.to
		g.attrs[i] = e.attrs
	}

	g.bbox = computeBBox(b.coords)
	g.spatial = buildSpatialIndex(g.coords)
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
