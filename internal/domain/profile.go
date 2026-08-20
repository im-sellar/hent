package domain

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
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}
