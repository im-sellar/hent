package httpapi

import (
	"encoding/json"
	"testing"

	"github.com/im-sellar/hent/internal/domain"
)

// TestScoreDTOApparieLesChamps verrouille la recopie champ à champ entre
// domain.Score et sa représentation publique.
//
// Les cinq champs sont des float64 : intervertir deux d'entre eux compile, et
// le témoin de contrat ne le verrait pas non plus — sur la grille de test,
// part_trafic et part_retracee valent tous deux zéro. Seules des valeurs deux à
// deux distinctes rendent l'erreur visible.
func TestScoreDTOApparieLesChamps(t *testing.T) {
	s := domain.Score{
		DistanceM:     17400,
		PartNonBitume: 0.55,
		PartTrafic:    0.023,
		PartRetracee:  0.004,
		EcartCible:    -0.032,
	}

	brut, err := json.Marshal(scoreDTOOf(s))
	if err != nil {
		t.Fatal(err)
	}

	var obtenu map[string]float64
	if err := json.Unmarshal(brut, &obtenu); err != nil {
		t.Fatal(err)
	}

	attendu := map[string]float64{
		"distance_m":      17400,
		"part_non_bitume": 0.55,
		"part_trafic":     0.023,
		"part_retracee":   0.004,
		"ecart_cible":     -0.032,
	}
	if len(obtenu) != len(attendu) {
		t.Fatalf("%d champs sérialisés, attendu %d : %s", len(obtenu), len(attendu), brut)
	}
	for nom, valeur := range attendu {
		got, présent := obtenu[nom]
		if !présent {
			t.Errorf("champ %q absent de la sortie", nom)
			continue
		}
		if got != valeur {
			t.Errorf("%s = %v, attendu %v", nom, got, valeur)
		}
	}
}
