package port

import (
	"context"

	"github.com/im-sellar/hent/internal/domain"
)

// LoopGenerator est le contrat entrant : ce qu'un adaptateur peut demander au
// moteur, sans connaître la stratégie qui le réalise.
//
// Port grossier, comme RouteNetwork : générer et rendre compte de son travail
// décrivent une seule chose — un moteur de boucles observable. Les découper
// donnerait deux interfaces qu'aucun appelant ne prend séparément.
type LoopGenerator interface {
	Generate(ctx context.Context, req domain.LoopRequest) ([]domain.Loop, error)

	// Stats cumule depuis le démarrage : nœuds dépilés toutes recherches
	// confondues, et candidates écartées. Alimente /metrics.
	Stats() (exploredNodes, droppedCandidates int64)
}
