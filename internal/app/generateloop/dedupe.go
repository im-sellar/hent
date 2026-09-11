package generateloop

import "github.com/im-sellar/hent/internal/domain"

// Deux boucles partageant plus que ce seuil d'arêtes sont considérées comme
// la même proposition. Des angles de départ voisins convergent souvent vers
// le même itinéraire.
const jaccardThreshold = 0.7

func dedupe(loops []domain.Loop) []domain.Loop {
	var kept []domain.Loop
	var keptSets []map[domain.EdgeRef]struct{}

	for _, l := range loops {
		set := edgeSet(l)

		doublon := false
		for _, other := range keptSets {
			if jaccard(set, other) >= jaccardThreshold {
				doublon = true
				break
			}
		}
		if !doublon {
			kept = append(kept, l)
			keptSets = append(keptSets, set)
		}
	}
	return kept
}

func edgeSet(l domain.Loop) map[domain.EdgeRef]struct{} {
	s := make(map[domain.EdgeRef]struct{}, len(l.Edges))
	for _, e := range l.Edges {
		s[e] = struct{}{}
	}
	return s
}

// jaccard : taille de l'intersection sur taille de l'union.
func jaccard(a, b map[domain.EdgeRef]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for e := range a {
		if _, ok := b[e]; ok {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	return float64(inter) / float64(union)
}
