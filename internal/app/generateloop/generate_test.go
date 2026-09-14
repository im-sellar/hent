package generateloop_test

import (
	"context"
	"math"
	"testing"

	"github.com/im-sellar/hent/internal/app/generateloop"
	"github.com/im-sellar/hent/internal/domain"
	"github.com/im-sellar/hent/internal/testsupport"
)

func reseauEtRequete() (*testsupport.Grille, domain.LoopRequest) {
	// 40×40 nœuds espacés de 200 m : environ 8 km de côté, assez pour une
	// boucle de 4 km.
	g := testsupport.NouvelleGrille(40, 200)
	depart := g.Coord(domain.NodeRef(40*20 + 20)) // au centre

	return g, domain.LoopRequest{
		Start:      depart,
		DistanceM:  4000,
		Tolerance:  0.15,
		Prefs:      domain.Preferences{AvoidPaved: 0.5},
		MaxResults: 5,
	}
}

func TestGenerateRevientAuDepart(t *testing.T) {
	g, req := reseauEtRequete()

	loops, err := generateloop.New(g).Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate : %v", err)
	}
	if len(loops) == 0 {
		t.Fatal("aucune boucle produite")
	}

	for i, l := range loops {
		if l.Nodes[0] != l.Nodes[len(l.Nodes)-1] {
			t.Errorf("boucle %d : part de %d et finit à %d", i, l.Nodes[0], l.Nodes[len(l.Nodes)-1])
		}
	}
}

func TestGenerateRespecteLaTolerance(t *testing.T) {
	g, req := reseauEtRequete()

	loops, err := generateloop.New(g).Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if len(loops) == 0 {
		t.Fatal("aucune boucle produite : les assertions qui suivent ne vérifieraient rien")
	}

	for i, l := range loops {
		ecart := math.Abs(l.LengthM-req.DistanceM) / req.DistanceM
		if ecart > req.Tolerance {
			t.Errorf("boucle %d : %.0f m pour %.0f demandés (écart %.1f %%, toléré %.1f %%)",
				i, l.LengthM, req.DistanceM, ecart*100, req.Tolerance*100)
		}
	}
}

func TestGenerateProduitDesBouclesConnexes(t *testing.T) {
	g, req := reseauEtRequete()

	loops, err := generateloop.New(g).Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if len(loops) == 0 {
		t.Fatal("aucune boucle produite : les assertions qui suivent ne vérifieraient rien")
	}

	for i, l := range loops {
		if len(l.Edges) != len(l.Nodes)-1 {
			t.Errorf("boucle %d : %d arêtes pour %d nœuds", i, len(l.Edges), len(l.Nodes))
		}
		if len(l.Coords) != len(l.Nodes) {
			t.Errorf("boucle %d : %d coordonnées pour %d nœuds", i, len(l.Coords), len(l.Nodes))
		}
		for j := 0; j+1 < len(l.Nodes); j++ {
			if !g.Relies(l.Nodes[j], l.Nodes[j+1]) {
				t.Fatalf("boucle %d : les nœuds %d et %d ne sont pas voisins",
					i, l.Nodes[j], l.Nodes[j+1])
			}
		}
	}
}

func TestGenerateEstDeterministe(t *testing.T) {
	// Propriété exigée par l'API : même requête, même résultat. C'est ce qui
	// rend les réponses cachables et les GPX régénérables sans état.
	g, req := reseauEtRequete()

	a, err := generateloop.New(g).Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	b, err := generateloop.New(g).Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if len(a) == 0 {
		t.Fatal("aucune boucle produite : les assertions qui suivent ne vérifieraient rien")
	}
	if len(a) != len(b) {
		t.Fatalf("%d boucles puis %d", len(a), len(b))
	}
	for i := range a {
		if a[i].LengthM != b[i].LengthM || len(a[i].Nodes) != len(b[i].Nodes) {
			t.Fatalf("boucle %d diffère entre deux appels identiques", i)
		}
	}
}

func TestGenerateVariantDonneAutreChose(t *testing.T) {
	g, req := reseauEtRequete()

	a, err := generateloop.New(g).Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	req.Variant = 1
	b, err := generateloop.New(g).Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if len(a) == 0 || len(b) == 0 {
		t.Fatal("aucune boucle produite : les assertions qui suivent ne vérifieraient rien")
	}

	// On compare les ensembles d'arêtes, et non la longueur ni le nombre de
	// nœuds : sur une grille régulière, deux boucles bel et bien différentes
	// ont très souvent ces deux valeurs identiques, ce qui rendrait le test
	// intermittent.
	if r := recouvrement(a[0], b[0]); r > 0.9 {
		t.Errorf("un variant différent devrait produire une autre boucle (recouvrement %.0f %%)", r*100)
	}
}

func TestGenerateDepartHorsZone(t *testing.T) {
	g, req := reseauEtRequete()
	req.Start = domain.Coord{Lat: 43.30, Lon: 5.37} // Marseille

	if _, err := generateloop.New(g).Generate(context.Background(), req); err == nil {
		t.Fatal("un départ hors de la zone couverte doit échouer explicitement")
	}
}

func TestGenerateRespecteLeContexte(t *testing.T) {
	g, req := reseauEtRequete()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := generateloop.New(g).Generate(ctx, req); err == nil {
		t.Fatal("un contexte annulé doit interrompre la génération")
	}
}

func TestGenerateNeRenvoiePasDeDoublons(t *testing.T) {
	g, req := reseauEtRequete()

	loops, err := generateloop.New(g).Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if len(loops) == 0 {
		t.Fatal("aucune boucle produite : les assertions qui suivent ne vérifieraient rien")
	}

	for i := 0; i < len(loops); i++ {
		for j := i + 1; j < len(loops); j++ {
			if recouvrement(loops[i], loops[j]) >= 0.7 {
				t.Errorf("boucles %d et %d se recouvrent à %.0f %%",
					i, j, recouvrement(loops[i], loops[j])*100)
			}
		}
	}
}

// TestStatsRemonteLesNoeudsExplores vérifie que Stats() n'est pas câblé sur
// une source qui resterait bloquée à zéro : le compteur vient du réseau
// (testsupport.Grille ici), pas d'un total accumulé dans le Generator.
func TestStatsRemonteLesNoeudsExplores(t *testing.T) {
	g, req := reseauEtRequete()
	gen := generateloop.New(g)

	if _, err := gen.Generate(context.Background(), req); err != nil {
		t.Fatal(err)
	}

	explored, _ := gen.Stats()
	if explored <= 0 {
		t.Fatalf("nœuds explorés = %d, attendu > 0 après une génération", explored)
	}
}

func recouvrement(a, b domain.Loop) float64 {
	sa := map[domain.EdgeRef]struct{}{}
	for _, e := range a.Edges {
		sa[e] = struct{}{}
	}
	inter := 0
	sb := map[domain.EdgeRef]struct{}{}
	for _, e := range b.Edges {
		sb[e] = struct{}{}
	}
	for e := range sa {
		if _, ok := sb[e]; ok {
			inter++
		}
	}
	if len(sa) == 0 || len(sb) == 0 {
		return 0
	}
	return float64(inter) / float64(len(sa)+len(sb)-inter)
}

// reseauToutRepasse déclare chaque mètre parcouru comme déjà emprunté. Une
// boucle dont les agrégats sont correctement cumulés affiche alors exactement
// 100 % de tracé repassé — quel que soit le nombre de segments recollés, et
// sans dépendre de la topologie de la grille. Un cumul oublié donnerait zéro.
type reseauToutRepasse struct{ *testsupport.Grille }

func (r reseauToutRepasse) FindPath(ctx context.Context, from, to domain.NodeRef,
	w domain.Weights, opt domain.PathOptions) (domain.Path, error) {

	p, err := r.Grille.FindPath(ctx, from, to, w, opt)
	p.RetracedM = p.LengthM
	return p, err
}

func TestGenerateCumuleLeTraceRepasse(t *testing.T) {
	g, req := reseauEtRequete()

	loops, err := generateloop.New(reseauToutRepasse{g}).Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate : %v", err)
	}
	if len(loops) == 0 {
		t.Fatal("aucune boucle produite : les assertions qui suivent ne vérifieraient rien")
	}

	for i, l := range loops {
		if l.LengthM <= 0 {
			t.Fatalf("boucle %d de longueur nulle : le rapport qui suit n'aurait pas de sens", i)
		}
		if math.Abs(l.RetracedM-l.LengthM) > 1e-6 {
			t.Errorf("boucle %d : RetracedM = %v pour LengthM = %v, attendu l'égalité",
				i, l.RetracedM, l.LengthM)
		}
		if part := domain.NewScore(l, req.DistanceM).PartRetracee; math.Abs(part-1) > 1e-9 {
			t.Errorf("boucle %d : PartRetracee = %v, attendu 1", i, part)
		}
	}
}

// TestGenerateNeRepassePasSurLaGrille est le pendant du test précédent sur le
// réseau nu : la grille synthétique offre partout une alternative, donc une
// boucle correcte n'y repasse jamais. C'est ce qui distingue un cumul juste
// d'un cumul qui additionnerait tout.
func TestGenerateNeRepassePasSurLaGrille(t *testing.T) {
	g, req := reseauEtRequete()

	loops, err := generateloop.New(g).Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(loops) == 0 {
		t.Fatal("aucune boucle produite : les assertions qui suivent ne vérifieraient rien")
	}

	for i, l := range loops {
		if l.RetracedM < 0 || l.RetracedM > l.LengthM {
			t.Errorf("boucle %d : RetracedM = %v hors de [0, %v]", i, l.RetracedM, l.LengthM)
		}
		if part := domain.NewScore(l, req.DistanceM).PartRetracee; part > 0.15 {
			t.Errorf("boucle %d : %.1f %% de tracé repassé sur une grille régulière, attendu presque zéro",
				i, part*100)
		}
	}
}
