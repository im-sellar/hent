package httpapi

import (
	"reflect"
	"testing"

	"github.com/im-sellar/hent/internal/domain"
)

// TestLoopRequestApparieLesChamps verrouille la recopie champ à champ entre
// loopsRequest et domain.LoopRequest, sur le modèle de
// TestScoreDTOApparieLesChamps.
//
// domain.LoopRequest ne porte pas de tag json (le domaine n'en a pas le
// droit) et imbrique deux structs (Coord, Preferences) : la comparer par
// réflexion, champ de premier niveau par champ, couvre les deux sans
// dupliquer les types de coordDTO/prefsDTO. Toutes les valeurs sont deux à
// deux distinctes et non nulles pour qu'une perte ou une interversion se
// voie. Le nombre de champs est vérifié séparément : sans cette garde, un
// champ ajouté à domain.LoopRequest et oublié dans toDomain resterait
// invisible, puisqu'aucune comparaison ne porterait dessus.
func TestLoopRequestApparieLesChamps(t *testing.T) {
	attendu := domain.LoopRequest{
		Start:      domain.Coord{Lat: 48.117, Lon: -1.677},
		DistanceM:  12345,
		Tolerance:  0.31,
		Prefs:      domain.Preferences{AvoidPaved: 0.62},
		MaxResults: 7,
		Variant:    3,
	}

	typ := reflect.TypeOf(attendu)
	if n := typ.NumField(); n != 6 {
		t.Fatalf("domain.LoopRequest a %d champs, attendu 6 : ce test doit être étendu pour les couvrir", n)
	}

	r := loopsRequest{
		Start:       coordDTO{Lat: attendu.Start.Lat, Lon: attendu.Start.Lon},
		DistanceM:   attendu.DistanceM,
		Tolerance:   attendu.Tolerance,
		Preferences: prefsDTO{AvoidPaved: attendu.Prefs.AvoidPaved},
		MaxResults:  attendu.MaxResults,
		Variant:     attendu.Variant,
	}
	got := r.toDomain()

	valGot, valAttendu := reflect.ValueOf(got), reflect.ValueOf(attendu)
	for i := 0; i < typ.NumField(); i++ {
		nom := typ.Field(i).Name
		g, a := valGot.Field(i).Interface(), valAttendu.Field(i).Interface()
		if !reflect.DeepEqual(g, a) {
			t.Errorf("%s = %+v, attendu %+v", nom, g, a)
		}
	}
}
