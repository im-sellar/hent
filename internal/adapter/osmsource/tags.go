// Package osmsource lit un extrait OpenStreetMap et en construit un graphe
// routier. Il n'est utilisé que par graphbuild, hors ligne : le serveur ne
// voit jamais de fichier PBF.
package osmsource

import "github.com/im-sellar/hent/internal/domain"

// wayClasses associe les valeurs de highway retenues à leur classe et à leur
// exposition au trafic (0 = aucune, 255 = maximale). Ce qui n'y figure pas
// est écarté : autoroutes, voies rapides, escaliers, et tout ce qui n'est pas
// une voie.
var wayClasses = map[string]struct {
	class   domain.WayClass
	traffic uint8
}{
	"path":          {domain.WayPath, 0},
	"bridleway":     {domain.WayPath, 0},
	"track":         {domain.WayTrack, 0},
	"footway":       {domain.WayFootway, 0},
	"pedestrian":    {domain.WayFootway, 10},
	"cycleway":      {domain.WayFootway, 10},
	"living_street": {domain.WayResidential, 40},
	"service":       {domain.WayResidential, 60},
	"residential":   {domain.WayResidential, 90},
	"unclassified":  {domain.WayResidential, 120},
	"tertiary":      {domain.WayTertiary, 180},
	"secondary":     {domain.WaySecondary, 255},
}

// defaultSurfaces donne le revêtement présumé quand le tag surface est absent,
// ce qui est le cas le plus fréquent dans OSM.
var defaultSurfaces = map[domain.WayClass]domain.Surface{
	domain.WayPath:        domain.SurfaceGround,
	domain.WayTrack:       domain.SurfaceGround,
	domain.WayFootway:     domain.SurfacePaved,
	domain.WayResidential: domain.SurfacePaved,
	domain.WayTertiary:    domain.SurfacePaved,
	domain.WaySecondary:   domain.SurfacePaved,
}

var surfaceValues = map[string]domain.Surface{
	"asphalt": domain.SurfacePaved, "paved": domain.SurfacePaved,
	"concrete": domain.SurfacePaved, "paving_stones": domain.SurfacePaved,
	"cobblestone": domain.SurfacePaved, "sett": domain.SurfacePaved,

	"gravel": domain.SurfaceGravel, "fine_gravel": domain.SurfaceGravel,
	"compacted": domain.SurfaceGravel, "pebblestone": domain.SurfaceGravel,

	"ground": domain.SurfaceGround, "dirt": domain.SurfaceGround,
	"earth": domain.SurfaceGround, "grass": domain.SurfaceGround,
	"sand": domain.SurfaceGround, "mud": domain.SurfaceGround,
	"unpaved": domain.SurfaceGround, "woodchips": domain.SurfaceGround,
}

// Classify traduit les tags d'un way OSM. Le dernier retour indique si le
// tronçon est praticable à pied et doit entrer dans le graphe.
func Classify(tags map[string]string) (domain.WayClass, domain.Surface, uint8, bool) {
	spec, ok := wayClasses[tags["highway"]]
	if !ok {
		return domain.WayUnknown, domain.SurfaceUnknown, 0, false
	}

	switch tags["access"] {
	case "private", "no":
		return domain.WayUnknown, domain.SurfaceUnknown, 0, false
	}

	surface, ok := surfaceValues[tags["surface"]]
	if !ok {
		surface = defaultSurfaces[spec.class]
	}

	return spec.class, surface, spec.traffic, true
}
