package domain

// LoopRequest décrit ce qu'on demande : d'où partir, sur quelle distance, avec
// quelle tolérance et quelles préférences de terrain. MaxResults et Variant
// disent combien de propositions on veut et laquelle explorer — ce sont des
// paramètres de la demande, pas de la stratégie qui y répond.
//
// Les valeurs par défaut ne sont pas appliquées ici : les choisir est une
// décision de la couche applicative.
type LoopRequest struct {
	Start      Coord
	DistanceM  float64
	Tolerance  float64
	Prefs      Preferences
	MaxResults int
	Variant    int
}
