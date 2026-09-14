package httpapi

import "testing"

// TestLoopRequestApparieLesChamps verrouille la recopie champ à champ entre
// loopsRequest et domain.LoopRequest.
//
// Sept champs, sept types simples : une perte ou une interversion (ex.
// Tolerance et Preferences.AvoidPaved, deux float64) compile et ne se voit
// pas au premier regard. Toutes les valeurs sont deux à deux distinctes et
// non nulles pour que les deux défauts se voient.
func TestLoopRequestApparieLesChamps(t *testing.T) {
	r := loopsRequest{
		Start:       coordDTO{Lat: 48.117, Lon: -1.677},
		DistanceM:   12345,
		Tolerance:   0.31,
		Preferences: prefsDTO{AvoidPaved: 0.62},
		MaxResults:  7,
		Variant:     3,
	}

	got := r.toDomain()

	switch {
	case got.Start.Lat != r.Start.Lat:
		t.Errorf("Start.Lat = %v, attendu %v", got.Start.Lat, r.Start.Lat)
	case got.Start.Lon != r.Start.Lon:
		t.Errorf("Start.Lon = %v, attendu %v", got.Start.Lon, r.Start.Lon)
	case got.DistanceM != r.DistanceM:
		t.Errorf("DistanceM = %v, attendu %v", got.DistanceM, r.DistanceM)
	case got.Tolerance != r.Tolerance:
		t.Errorf("Tolerance = %v, attendu %v", got.Tolerance, r.Tolerance)
	case got.Prefs.AvoidPaved != r.Preferences.AvoidPaved:
		t.Errorf("Prefs.AvoidPaved = %v, attendu %v", got.Prefs.AvoidPaved, r.Preferences.AvoidPaved)
	case got.MaxResults != r.MaxResults:
		t.Errorf("MaxResults = %v, attendu %v", got.MaxResults, r.MaxResults)
	case got.Variant != r.Variant:
		t.Errorf("Variant = %v, attendu %v", got.Variant, r.Variant)
	}
}
