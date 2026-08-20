package domain

import "math"

type Surface uint8

const (
	SurfaceUnknown Surface = iota
	SurfacePaved
	SurfaceGravel
	SurfaceGround
)

type WayClass uint8

const (
	WayUnknown WayClass = iota
	WayPath
	WayTrack
	WayFootway
	WayResidential
	WayTertiary
	WaySecondary
)

// Preferences exprime l'intention de l'utilisateur sur une échelle 0-1.
// C'est ce que l'API expose ; les poids internes ne sortent jamais.
type Preferences struct {
	AvoidPaved float64
}

// Weights porte les coefficients du modèle de coût. Ils sont toujours ≥ 0 :
// voir la note d'invariant sur EdgeAttrs.Cost.
type Weights struct {
	Paved   float64
	Traffic float64
}

const (
	maxPavedWeight   = 6.0
	minTrafficWeight = 2.0
	maxTrafficWeight = 12.0
)

func (p Preferences) Weights() Weights {
	a := clamp01(p.AvoidPaved)
	return Weights{
		Paved: a * maxPavedWeight,
		// Le trafic garde un plancher : on ne veut jamais être envoyé sur une
		// départementale, même quand l'utilisateur ne réclame pas de chemins.
		Traffic: minTrafficWeight + a*(maxTrafficWeight-minTrafficWeight),
	}
}

func clamp01(v float64) float64 {
	switch {
	// NaN se traite en premier, et explicitement : en Go toute comparaison
	// impliquant NaN est fausse, donc un NaN traverserait les deux cas
	// suivants intact. Il contaminerait alors les poids, puis le coût, puis
	// l'A* — qui ne relaxerait plus aucune arête, sans erreur ni test rouge,
	// car « NaN < x » est également faux.
	case math.IsNaN(v):
		return 0
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}
