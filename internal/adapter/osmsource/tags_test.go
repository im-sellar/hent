package osmsource_test

import (
	"testing"

	"github.com/im-sellar/hent/internal/adapter/osmsource"
	"github.com/im-sellar/hent/internal/domain"
)

func TestClassifyRetientLesChemins(t *testing.T) {
	cases := []struct {
		nom     string
		tags    map[string]string
		class   domain.WayClass
		surface domain.Surface
	}{
		{"sentier", map[string]string{"highway": "path"},
			domain.WayPath, domain.SurfaceGround},
		{"chemin d'exploitation", map[string]string{"highway": "track"},
			domain.WayTrack, domain.SurfaceGround},
		{"trottoir", map[string]string{"highway": "footway"},
			domain.WayFootway, domain.SurfacePaved},
		{"rue résidentielle", map[string]string{"highway": "residential"},
			domain.WayResidential, domain.SurfacePaved},
		{"sentier explicitement gravillonné", map[string]string{"highway": "path", "surface": "gravel"},
			domain.WayPath, domain.SurfaceGravel},
		{"route explicitement en terre", map[string]string{"highway": "residential", "surface": "ground"},
			domain.WayResidential, domain.SurfaceGround},
	}

	for _, c := range cases {
		t.Run(c.nom, func(t *testing.T) {
			class, surface, _, ok := osmsource.Classify(c.tags)
			if !ok {
				t.Fatal("le tronçon devrait être retenu")
			}
			if class != c.class {
				t.Errorf("classe = %v, attendu %v", class, c.class)
			}
			if surface != c.surface {
				t.Errorf("revêtement = %v, attendu %v", surface, c.surface)
			}
		})
	}
}

func TestClassifyEcarte(t *testing.T) {
	cases := []struct {
		nom  string
		tags map[string]string
	}{
		{"autoroute", map[string]string{"highway": "motorway"}},
		{"voie rapide", map[string]string{"highway": "trunk"}},
		{"pas une voie", map[string]string{"building": "yes"}},
		{"accès privé", map[string]string{"highway": "track", "access": "private"}},
		{"accès interdit", map[string]string{"highway": "path", "access": "no"}},
		{"escalier", map[string]string{"highway": "steps"}},
	}

	for _, c := range cases {
		t.Run(c.nom, func(t *testing.T) {
			if _, _, _, ok := osmsource.Classify(c.tags); ok {
				t.Error("le tronçon devrait être écarté")
			}
		})
	}
}

func TestClassifyExpositionTrafic(t *testing.T) {
	_, _, traficRoute, _ := osmsource.Classify(map[string]string{"highway": "secondary"})
	_, _, traficSentier, _ := osmsource.Classify(map[string]string{"highway": "path"})

	if traficSentier != 0 {
		t.Errorf("exposition d'un sentier = %d, attendu 0", traficSentier)
	}
	if traficRoute <= traficSentier {
		t.Errorf("exposition d'une départementale (%d) doit dépasser celle d'un sentier (%d)",
			traficRoute, traficSentier)
	}
}
