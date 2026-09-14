// Package osmsource lit un extrait OpenStreetMap et en construit un graphe
// routier. Il n'est utilisé que par graphbuild, hors ligne : le serveur ne
// voit jamais de fichier PBF.
package osmsource

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/im-sellar/hent/internal/domain"
)

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

// ConfigHash résume le contenu réel des tables de classification ci-dessus.
//
// Il entre dans la provenance de l'artefact, et c'est ce qui donne corps à
// l'engagement ODbL de fournir « les moyens de reconstruire » : deux graphes
// bâtis avec des règles différentes doivent porter des empreintes différentes,
// sans quoi rien ne distingue deux artefacts qui n'ont pas la même origine.
//
// Les clés sont triées avant d'être hachées : l'ordre d'itération d'une map Go
// est délibérément aléatoire, et hacher sans trier produirait une empreinte
// différente à chaque exécution — ce qui détruirait la propriété recherchée
// tout en en donnant l'apparence.
func ConfigHash() string {
	h := sha256.New()

	classes := make([]string, 0, len(wayClasses))
	for k := range wayClasses {
		classes = append(classes, k)
	}
	sort.Strings(classes)
	for _, k := range classes {
		spec := wayClasses[k]
		fmt.Fprintf(h, "way\t%s\t%d\t%d\n", k, spec.class, spec.traffic)
	}

	surfaces := make([]string, 0, len(surfaceValues))
	for k := range surfaceValues {
		surfaces = append(surfaces, k)
	}
	sort.Strings(surfaces)
	for _, k := range surfaces {
		fmt.Fprintf(h, "surface\t%s\t%d\n", k, surfaceValues[k])
	}

	defaults := make([]int, 0, len(defaultSurfaces))
	for k := range defaultSurfaces {
		defaults = append(defaults, int(k))
	}
	sort.Ints(defaults)
	for _, k := range defaults {
		fmt.Fprintf(h, "default\t%d\t%d\n", k, defaultSurfaces[domain.WayClass(k)])
	}

	return hex.EncodeToString(h.Sum(nil)[:8])
}

// WayClassesPourTest expose les valeurs de highway retenues, afin que les
// tests puissent vérifier la cohérence entre les tables sans les dupliquer —
// une copie finirait par diverger de l'original, et le test perdrait son sens.
func WayClassesPourTest() map[string]struct{} {
	tags := make(map[string]struct{}, len(wayClasses))
	for k := range wayClasses {
		tags[k] = struct{}{}
	}
	return tags
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
