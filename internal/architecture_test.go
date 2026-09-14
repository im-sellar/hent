package internal_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const modulePath = "github.com/im-sellar/hent"

// Une couche ne doit jamais importer les couches listées en face d'elle.
var forbiddenImports = map[string][]string{
	"internal/domain": {"internal/app", "internal/adapter", "internal/platform"},
	"internal/app":    {"internal/adapter", "internal/platform"},
}

func TestRegleDeDependance(t *testing.T) {
	root := projectRoot(t)

	for layer, banned := range forbiddenImports {
		dir := filepath.Join(root, layer)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue // la couche n'existe pas encore
		}

		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
				return err
			}

			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}

			for _, imp := range file.Imports {
				pkg := strings.Trim(imp.Path.Value, `"`)
				for _, b := range banned {
					if dependanceInterdite(pkg, b) {
						rel, _ := filepath.Rel(root, path)
						t.Errorf("%s importe %s : la couche %s ne doit pas dépendre de %s",
							rel, pkg, layer, b)
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// dependanceInterdite dit si le chemin d'import pkg tombe dans la couche
// donnée. La frontière de segment est vérifiée explicitement : internal/app2
// n'appartient pas à internal/app.
func dependanceInterdite(pkg, couche string) bool {
	prefixe := modulePath + "/" + couche
	return pkg == prefixe || strings.HasPrefix(pkg, prefixe+"/")
}

// TestDependanceInterditeRespecteLesSegments verrouille la comparaison de
// chemins d'import. Une comparaison de préfixe nue confond internal/app et
// internal/app2 : le test qui garantit l'architecture serait alors le seul
// endroit du dépôt où une approximation resterait invisible.
func TestDependanceInterditeRespecteLesSegments(t *testing.T) {
	cas := []struct {
		pkg      string
		couche   string
		interdit bool
	}{
		{modulePath + "/internal/app", "internal/app", true},
		{modulePath + "/internal/app/port", "internal/app", true},
		{modulePath + "/internal/app/generateloop", "internal/app", true},
		{modulePath + "/internal/app2", "internal/app", false},
		{modulePath + "/internal/application", "internal/app", false},
		{modulePath + "/internal/appareil/photo", "internal/app", false},
		{modulePath + "/internal/domain", "internal/app", false},
		{"encoding/json", "internal/app", false},
	}

	for _, c := range cas {
		if got := dependanceInterdite(c.pkg, c.couche); got != c.interdit {
			t.Errorf("dependanceInterdite(%q, %q) = %v, attendu %v",
				c.pkg, c.couche, got, c.interdit)
		}
	}
}

// couplagesToleres recense les dépendances entre adaptateurs déjà présentes
// quand la règle a été introduite. Chacune est une dette identifiée, pas un
// oubli : la liste existe pour qu'aucune nouvelle ne s'ajoute en silence.
var couplagesToleres = map[string]struct{}{
	// osmsource n'existe que pour construire le graphe CSR en flux : cette
	// dépendance est son produit, pas un couplage accidentel. La rompre
	// demanderait un port GraphBuilder et une refonte du chemin d'ingestion de
	// six millions de nœuds, qu'aucun témoin de performance n'encadre.
	"osmsource -> network": {},

	// httpapi compose gpxfile pour l'export GPX. Un port LoopExporter serait
	// plus propre et peu coûteux ; il est hors du périmètre de ce chantier.
	"httpapi -> gpxfile": {},
}

// TestAdaptateursCloisonnes interdit qu'un adaptateur en importe un autre.
//
// La règle par couche ne l'attrape pas : adapter → adapter reste dans la même
// couche. C'est par ce trou que la signature de httpapi.New en est venue à
// nommer csr.Provenance, faisant dépendre l'adaptateur entrant du format
// binaire de l'artefact. Un type partagé par deux adaptateurs appartient au
// domaine ; c'est cmd/ qui les assemble, pas eux qui se connaissent.
func TestAdaptateursCloisonnes(t *testing.T) {
	root := projectRoot(t)
	racineAdaptateurs := filepath.Join(root, "internal/adapter")

	entrees, err := os.ReadDir(racineAdaptateurs)
	if err != nil {
		t.Fatal(err)
	}

	var familles []string
	for _, e := range entrees {
		if e.IsDir() {
			familles = append(familles, e.Name())
		}
	}
	if len(familles) < 2 {
		t.Fatalf("%d famille(s) d'adaptateurs : la règle ne vérifierait rien", len(familles))
	}

	vus := map[string]struct{}{}

	for _, famille := range familles {
		dir := filepath.Join(racineAdaptateurs, famille)

		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
				return err
			}

			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}

			for _, imp := range file.Imports {
				pkg := strings.Trim(imp.Path.Value, `"`)
				for _, autre := range familles {
					if autre == famille {
						continue
					}
					if !dependanceInterdite(pkg, "internal/adapter/"+autre) {
						continue
					}

					couplage := famille + " -> " + autre
					vus[couplage] = struct{}{}
					if _, tolere := couplagesToleres[couplage]; tolere {
						continue
					}

					rel, _ := filepath.Rel(root, path)
					t.Errorf("%s importe %s : l'adaptateur %s ne doit pas connaître l'adaptateur %s — un type partagé appartient au domaine",
						rel, pkg, famille, autre)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	// Une exception qui n'a plus lieu d'être doit disparaître, sinon la liste
	// ne fait que grossir et cesse de vouloir dire quelque chose.
	for couplage := range couplagesToleres {
		if _, encore := vus[couplage]; !encore {
			t.Errorf("le couplage toléré %q n'existe plus dans le code : retirer l'exception", couplage)
		}
	}
}

func projectRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod introuvable en remontant depuis le répertoire de test")
		}
		dir = parent
	}
}
