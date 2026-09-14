package port_test

import (
	"testing"

	"github.com/im-sellar/hent/internal/app/generateloop"
	"github.com/im-sellar/hent/internal/app/port"
	"github.com/im-sellar/hent/internal/testsupport"
)

// TestGeneratorImplementeLePort verrouille le contrat entrant. Sans cette
// assertion, une signature de Generator pourrait diverger du port sans que
// rien ne le signale avant l'assemblage dans cmd/.
func TestGeneratorImplementeLePort(t *testing.T) {
	var _ port.LoopGenerator = generateloop.New(testsupport.NouvelleGrille(4, 200))
}
