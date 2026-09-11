package csr

import (
	"math"

	"github.com/im-sellar/hent/internal/domain"
)

// Côté d'une cellule de la grille, en degrés de latitude — environ 550 m.
const cellSizeDeg = 0.005

type cellKey struct{ x, y int32 }

type spatialIndex struct {
	cells map[cellKey][]domain.NodeRef
}

func buildSpatialIndex(coords []domain.Coord) *spatialIndex {
	idx := &spatialIndex{cells: make(map[cellKey][]domain.NodeRef, len(coords)/8+1)}
	for i, c := range coords {
		k := keyOf(c)
		idx.cells[k] = append(idx.cells[k], domain.NodeRef(i))
	}
	return idx
}

func keyOf(c domain.Coord) cellKey {
	return cellKey{
		x: int32(math.Floor(c.Lon / cellSizeDeg)),
		y: int32(math.Floor(c.Lat / cellSizeDeg)),
	}
}

// NearestNode retourne le nœud le plus proche de c. La recherche part de la
// cellule contenant c et élargit l'anneau tant que rien n'est trouvé, jusqu'à
// une limite au-delà de laquelle on considère le point hors zone couverte.
func (g *Graph) NearestNode(c domain.Coord) (domain.NodeRef, bool) {
	const maxRings = 20 // ≈ 11 km

	origin := keyOf(c)

	// Le meilleur candidat s'accumule à travers les anneaux : le déclarer
	// dans la boucle perdrait les touches des anneaux précédents.
	best, bestDist := domain.NodeRef(0), math.MaxFloat64
	firstHit := int32(-1)

	for ring := int32(0); ring <= maxRings; ring++ {
		// Un nœud d'un anneau plus large peut être plus proche que celui déjà
		// trouvé — les cellules sont carrées, pas circulaires. On explore donc
		// un anneau de plus que celui de la première touche avant de conclure.
		if firstHit >= 0 && ring > firstHit+1 {
			break
		}

		for dx := -ring; dx <= ring; dx++ {
			for dy := -ring; dy <= ring; dy++ {
				// N'examiner que le bord de l'anneau : l'intérieur a déjà
				// été balayé aux itérations précédentes.
				if ring > 0 && abs32(dx) != ring && abs32(dy) != ring {
					continue
				}
				for _, n := range g.spatial.cells[cellKey{x: origin.x + dx, y: origin.y + dy}] {
					if d := domain.HaversineM(c, g.coords[n]); d < bestDist {
						best, bestDist = n, d
						if firstHit < 0 {
							firstHit = ring
						}
					}
				}
			}
		}
	}

	return best, firstHit >= 0
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}
