package port_test

import (
	"github.com/im-sellar/hent/internal/app/generateloop"
	"github.com/im-sellar/hent/internal/app/port"
)

// Cette assertion verrouille le contrat entrant. Sans elle, une signature de
// Generator pourrait diverger du port sans que rien ne le signale avant
// l'assemblage dans cmd/.
//
// Le compilateur la tranche seul : nul besoin d'instancier une grille ni de
// lancer quoi que ce soit.
var _ port.LoopGenerator = (*generateloop.Generator)(nil)
