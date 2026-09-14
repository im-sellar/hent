package domain

import "math"

// Score décrit une boucle en termes compréhensibles. À la différence du coût,
// qui guide l'A* et n'a de sens qu'en interne, le score est exposé par l'API :
// il permet à l'utilisateur de voir pourquoi une boucle lui est proposée et
// sur quel critère elle est moins bonne que la suivante.
type Score struct {
	DistanceM     float64 `json:"distance_m"`
	PartNonBitume float64 `json:"part_non_bitume"`
	PartTrafic    float64 `json:"part_trafic"`
	EcartCible    float64 `json:"ecart_cible"`
}

func NewScore(l Loop, targetM float64) Score {
	s := Score{DistanceM: l.LengthM}

	if l.LengthM > 0 {
		s.PartNonBitume = l.UnpavedM / l.LengthM
		s.PartTrafic = l.TrafficExposureM / l.LengthM
	}
	if targetM > 0 {
		s.EcartCible = (l.LengthM - targetM) / targetM
	}
	return s
}

// Preferable classe deux boucles : d'abord la fidélité à la distance
// demandée, puis la part de chemins. Deux boucles dans la tolérance sont
// départagées par le terrain, pas par les mètres.
func (s Score) Preferable(other Score, tolerance float64) bool {
	inTol := math.Abs(s.EcartCible) <= tolerance
	otherInTol := math.Abs(other.EcartCible) <= tolerance

	if inTol != otherInTol {
		return inTol
	}
	if inTol {
		return s.PartNonBitume > other.PartNonBitume
	}
	return math.Abs(s.EcartCible) < math.Abs(other.EcartCible)
}
