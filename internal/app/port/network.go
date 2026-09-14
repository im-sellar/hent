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

	// ExploredNodesTotal retourne le cumul des nœuds dépilés par FindPath
	// depuis la création de l'implémentation, toutes recherches confondues
	// — succès et échecs. Le comptage vit chez l'implémenteur et non chez
	// l'appelant : une recherche qui échoue ne renvoie pas de domain.Path
	// porteur de son ExploredNodes, seul le point d'exploration lui-même
	// peut donc le cumuler sans le perdre.
	ExploredNodesTotal() int64
}
