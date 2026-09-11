package generateloop_test

import (
	"context"
	"math"
	"testing"

	"github.com/im-sellar/hent/internal/app/generateloop"
	"github.com/im-sellar/hent/internal/domain"
)

func reseauEtRequete() (*grille, generateloop.Request) {
	// 40×40 nœuds espacés de 200 m : environ 8 km de côté, assez pour une
	// boucle de 4 km.
	g := nouvelleGrille(40, 200)
	depart := g.Coord(domain.NodeRef(40*20 + 20)) // au centre

	return g, generateloop.Request{
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
			if !g.relies(l.Nodes[j], l.Nodes[j+1]) {
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
