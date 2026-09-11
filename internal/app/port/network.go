// Package port déclare les interfaces dont la couche applicative a besoin.
// Elles sont définies ici, chez le consommateur, et non chez l'implémenteur :
// c'est l'idiome Go, et c'est aussi l'inversion de dépendance.
package port

import (
	"context"

	"github.com/im-sellar/hent/internal/domain"
)

// RouteNetwork est volontairement à granularité grossière. Elle expose
// FindPath et non Neighbors : construire une boucle demande trois ou quatre
// appels, là où exposer le voisinage en ferait des millions — un coût
// d'abstraction que la boucle chaude de l'A* ne peut pas absorber.
type RouteNetwork interface {
	NearestNode(c domain.Coord) (domain.NodeRef, bool)
	FindPath(ctx context.Context, from, to domain.NodeRef,
		w domain.Weights, opt domain.PathOptions) (domain.Path, error)
	Coord(n domain.NodeRef) domain.Coord
	BBox() domain.BBox
}
