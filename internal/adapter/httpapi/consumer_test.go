package httpapi_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/im-sellar/hent/internal/adapter/httpapi"
	"github.com/im-sellar/hent/internal/app/port"
	"github.com/im-sellar/hent/internal/domain"
)

// generateurFactice est un double minimal de port.LoopGenerator, utilisé
// uniquement pour l'assertion de compilation de TestNewConsommeLePort.
type generateurFactice struct{}

func (generateurFactice) Generate(context.Context, domain.LoopRequest) ([]domain.Loop, error) {
	return nil, nil
}

func (generateurFactice) Stats() (exploredNodes, droppedCandidates int64) {
	return 0, 0
}

// TestNewConsommeLePort verrouille que httpapi.New dépend du port entrant
// port.LoopGenerator, et non d'une implémentation concrète.
//
// L'assertion de compilation de internal/app/port/generator_test.go garde le
// sens inverse — que le moteur satisfait le port — mais ne dit rien de savoir
// si l'adaptateur HTTP consomme réellement cette interface : remettre le type
// concret *generateloop.Generator dans la signature de New compilerait
// encore et laisserait toute la suite verte. generateurFactice, typé
// explicitement en port.LoopGenerator et non convertible vers
// *generateloop.Generator, ferme ce trou.
func TestNewConsommeLePort(t *testing.T) {
	var gen port.LoopGenerator = generateurFactice{}
	var _ http.Handler = httpapi.New(gen, domain.Provenance{}, domain.BBox{}, nil)
}
