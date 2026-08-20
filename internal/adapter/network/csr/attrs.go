package csr

import "github.com/im-sellar/hent/internal/domain"

// EdgeAttrs porte les attributs objectifs d'une arête, mesurés une fois au
// build. Aucun coût n'y est stocké : le coût dépend des préférences de la
// requête et se calcule à la volée.
type EdgeAttrs struct {
	LengthM float64
	Surface domain.Surface
	Class   domain.WayClass
	Traffic uint8 // exposition au trafic, 0 = aucune, 255 = maximale
}

// Cost applique le modèle de la spec (§6) :
//
//	coût = longueur × ( 1 + Σ wᵢ × pénalitéᵢ )
//
// INVARIANT — à ne jamais violer : chaque pénalité appartient à [0,1] et
// chaque poids est ≥ 0. Le facteur est donc toujours ≥ 1, le coût minimal par
// mètre vaut exactement 1, et l'heuristique de l'A* (distance à vol d'oiseau)
// est admissible sans calibrage.
//
// Corollaire : on n'exprime JAMAIS un critère comme une récompense. Pour
// favoriser l'ombre, on pénalise le soleil. Un coût négatif rend Dijkstra et
// A* faux — silencieusement, sans la moindre erreur, avec des chemins absurdes.
func (a EdgeAttrs) Cost(w domain.Weights) float64 {
	penalty := w.Paved*a.pavedPenalty() + w.Traffic*a.trafficPenalty()
	return a.LengthM * (1 + penalty)
}

func (a EdgeAttrs) pavedPenalty() float64 {
	switch a.Surface {
	case domain.SurfacePaved:
		return 1
	case domain.SurfaceGravel:
		return 0.3
	case domain.SurfaceGround:
		return 0
	default:
		// Revêtement non renseigné : on suppose un intermédiaire plutôt que
		// d'écarter le tronçon ou de le privilégier à tort.
		return 0.5
	}
}

func (a EdgeAttrs) trafficPenalty() float64 { return float64(a.Traffic) / 255 }
