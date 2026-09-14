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

func TestConfigHashDeterministe(t *testing.T) {
	premier := osmsource.ConfigHash()
	for i := 0; i < 10; i++ {
		if got := osmsource.ConfigHash(); got != premier {
			t.Fatalf("ConfigHash instable d'un appel à l'autre : %q puis %q", premier, got)
		}
	}
}

// TestClassifyToutesLesClassesOntUnRevetementParDefaut verrouille un invariant
// tacite entre les deux tables : lorsqu'un tronçon n'a pas de tag `surface` —
// un quart d'entre eux — Classify retombe sur defaultSurfaces. Si une classe y
// manquait, l'indexation de la map rendrait la valeur zéro, c'est-à-dire
// SurfaceUnknown, silencieusement : le tronçon serait alors pénalisé à moitié
// par le modèle de coût tout en comptant pour du chemin intégral dans le score.
// Ajouter une entrée à wayClasses sans en ajouter une à defaultSurfaces doit
// faire échouer ce test, pas produire une incohérence invisible.
func TestClassifyToutesLesClassesOntUnRevetementParDefaut(t *testing.T) {
	vues := map[domain.WayClass]string{}
	for tag := range osmsource.WayClassesPourTest() {
		class, surface, _, ok := osmsource.Classify(map[string]string{"highway": tag})
		if !ok {
			t.Fatalf("highway=%s devrait être retenu", tag)
		}
		if surface == domain.SurfaceUnknown {
			t.Errorf("highway=%s (classe %v) n'a pas de revêtement par défaut", tag, class)
		}
		vues[class] = tag
	}
	if len(vues) == 0 {
		t.Fatal("aucune classe parcourue : le test ne vérifierait rien")
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
