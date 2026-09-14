package port

import (
	"context"
	"errors"

	"github.com/im-sellar/hent/internal/domain"
)

// ErrStartOutOfRange et ErrNoLoopFound sont les échecs que Generate peut
// renvoyer, distincts d'une erreur inattendue : l'appelant les distingue par
// errors.Is pour choisir le code HTTP à renvoyer. Elles vivent ici et non chez
// une implémentation concrète, pour qu'une autre stratégie de génération
// puisse les produire sans que l'appelant ait à la connaître.
var (
	ErrStartOutOfRange = errors.New("le point de départ est hors de la zone couverte")
	ErrNoLoopFound     = errors.New("aucune boucle trouvée pour cette requête")
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
