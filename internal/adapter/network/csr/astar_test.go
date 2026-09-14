package csr_test

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"testing"

	"github.com/im-sellar/hent/internal/adapter/network/csr"
	"github.com/im-sellar/hent/internal/domain"
)

func TestFindPathCheminSimple(t *testing.T) {
	g := carre(t)
	w := domain.Preferences{}.Weights()

	p, err := g.FindPath(context.Background(), 0, 2, w, domain.PathOptions{})
	if err != nil {
		t.Fatalf("FindPath : %v", err)
	}

	if len(p.Nodes) != 3 {
		t.Errorf("chemin de %d nœuds, attendu 3 (0 → intermédiaire → 2)", len(p.Nodes))
	}
	if p.Nodes[0] != 0 || p.Nodes[len(p.Nodes)-1] != 2 {
		t.Errorf("le chemin va de %d à %d, attendu 0 à 2", p.Nodes[0], p.Nodes[len(p.Nodes)-1])
	}
	if len(p.Edges) != len(p.Nodes)-1 {
		t.Errorf("%d arêtes pour %d nœuds", len(p.Edges), len(p.Nodes))
	}
	if p.LengthM <= 0 {
		t.Errorf("longueur = %v, attendu > 0", p.LengthM)
	}
}

func TestFindPathMemeNoeud(t *testing.T) {
	g := carre(t)

	p, err := g.FindPath(context.Background(), 1, 1, domain.Weights{}, domain.PathOptions{})
	if err != nil {
		t.Fatalf("FindPath : %v", err)
	}
	if p.LengthM != 0 || len(p.Edges) != 0 {
		t.Errorf("chemin d'un nœud à lui-même : longueur %v, %d arêtes, attendu 0 et 0",
			p.LengthM, len(p.Edges))
	}
}

func TestFindPathEviteLeBitume(t *testing.T) {
	// Deux itinéraires de 0 à 2 : le direct est bitumé et à fort trafic,
	// le détour est en terre et plus long en distance brute.
	b := csr.NewBuilder()
	n0 := b.AddNode(domain.Coord{Lat: 48.100, Lon: -1.680})
	n1 := b.AddNode(domain.Coord{Lat: 48.100, Lon: -1.670}) // sur la route
	n2 := b.AddNode(domain.Coord{Lat: 48.100, Lon: -1.660})
	n3 := b.AddNode(domain.Coord{Lat: 48.106, Lon: -1.670}) // par les chemins

	route := csr.EdgeAttrs{LengthM: 740, Surface: domain.SurfacePaved,
		Class: domain.WaySecondary, Traffic: 255}
	chemin := csr.EdgeAttrs{LengthM: 900, Surface: domain.SurfaceGround,
		Class: domain.WayPath}

	for _, e := range []struct {
		a, b domain.NodeRef
		at   csr.EdgeAttrs
	}{
		{n0, n1, route}, {n1, n2, route},
		{n0, n3, chemin}, {n3, n2, chemin},
	} {
		b.AddEdge(e.a, e.b, e.at)
		b.AddEdge(e.b, e.a, e.at)
	}
	g := b.Build()

	sansPreference, err := g.FindPath(context.Background(), n0, n2,
		domain.Weights{}, domain.PathOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if sansPreference.Nodes[1] != n1 {
		t.Error("à poids nuls, le trajet le plus court en distance doit passer par la route")
	}

	avecPreference, err := g.FindPath(context.Background(), n0, n2,
		domain.Preferences{AvoidPaved: 1}.Weights(), domain.PathOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if avecPreference.Nodes[1] != n3 {
		t.Error("avec AvoidPaved=1, le trajet doit passer par les chemins")
	}
}

func TestFindPathAucunChemin(t *testing.T) {
	b := csr.NewBuilder()
	n0 := b.AddNode(domain.Coord{Lat: 48.10, Lon: -1.68})
	n1 := b.AddNode(domain.Coord{Lat: 48.20, Lon: -1.50}) // isolé
	g := b.Build()

	_, err := g.FindPath(context.Background(), n0, n1, domain.Weights{}, domain.PathOptions{})
	if !errors.Is(err, csr.ErrNoPath) {
		t.Fatalf("erreur = %v, attendu ErrNoPath", err)
	}
}

func TestFindPathRespecteLeContexte(t *testing.T) {
	g := carre(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := g.FindPath(ctx, 0, 2, domain.Weights{}, domain.PathOptions{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("erreur = %v, attendu context.Canceled", err)
	}
}

// cancelAfterCalls annule le contexte après un nombre fixe d'appels à Err(),
// et non après une durée : ça rend le déclenchement du contrôle périodique
// d'annulation, au milieu de l'exploration, déterministe.
type cancelAfterCalls struct {
	context.Context
	calls     int
	threshold int
}

func (c *cancelAfterCalls) Err() error {
	c.calls++
	if c.calls > c.threshold {
		return context.Canceled
	}
	return c.Context.Err()
}

// grapheChaine construit n nœuds alignés et reliés en ligne, dans les deux
// sens. La géométrie étant rectiligne, l'heuristique est exacte et l'A*
// n'explore jamais qu'un seul nœud de front à la fois : chaque itération de
// la boucle principale correspond à exactement un nœud dépilé, sans repli.
func grapheChaine(t *testing.T, n int) *csr.Graph {
	t.Helper()

	b := csr.NewBuilder()
	nodes := make([]domain.NodeRef, n)
	for i := 0; i < n; i++ {
		nodes[i] = b.AddNode(domain.Coord{Lat: 48.0, Lon: -1.8 + float64(i)*0.0005})
	}
	for i := 0; i < n-1; i++ {
		length := domain.HaversineM(b.Coord(nodes[i]), b.Coord(nodes[i+1]))
		b.AddEdge(nodes[i], nodes[i+1], csr.EdgeAttrs{LengthM: length})
		b.AddEdge(nodes[i+1], nodes[i], csr.EdgeAttrs{LengthM: length})
	}
	return b.Build()
}

// TestFindPathVerifieLeContexteEnCoursDExploration couvre le contrôle
// d'annulation exécuté tous les ctxCheckInterval nœuds à l'intérieur de la
// boucle d'exploration — et non le garde d'entrée, qui ne s'exerce qu'avant
// que la recherche ne commence. Le contexte reste valide au moment de
// l'appel puis s'annule après un nombre d'interrogations choisi pour tomber
// exactement sur le contrôle périodique, une fois l'exploration commencée.
func TestFindPathVerifieLeContexteEnCoursDExploration(t *testing.T) {
	const n = 3000
	g := grapheChaine(t, n)

	// threshold : 1 pour le garde d'entrée, 1 pour le contrôle périodique à
	// explored == 0. Le troisième appel — celui à explored == 1024 — doit
	// être le premier à voir le contexte annulé.
	ctx := &cancelAfterCalls{Context: context.Background(), threshold: 2}

	_, err := g.FindPath(ctx, 0, domain.NodeRef(n-1), domain.Weights{}, domain.PathOptions{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("erreur = %v, attendu context.Canceled", err)
	}
	if ctx.calls < 3 {
		t.Fatalf("ctx.Err() appelé %d fois, attendu au moins 3 : "+
			"le contrôle interne à la boucle n'a pas été exercé", ctx.calls)
	}
}

func TestFindPathDepasseLeBudget(t *testing.T) {
	g := carre(t)

	_, err := g.FindPath(context.Background(), 0, 2, domain.Weights{},
		domain.PathOptions{MaxNodes: 1})
	if !errors.Is(err, csr.ErrBudgetExceeded) {
		t.Fatalf("erreur = %v, attendu ErrBudgetExceeded", err)
	}
}

// TestExploredNodesTotalCompteLesEchecs verrouille la correction : une
// recherche qui échoue explore quand même, et doit alimenter le cumul au même
// titre qu'une recherche réussie. Compter uniquement les segments retenus (ce
// qu'un appelant en amont ferait via le seul domain.Path renvoyé en succès)
// omettrait précisément les recherches les plus coûteuses, puisqu'un échec ne
// porte pas son ExploredNodes.
func TestExploredNodesTotalCompteLesEchecs(t *testing.T) {
	g := carre(t)
	avant := g.ExploredNodesTotal()

	_, err := g.FindPath(context.Background(), 0, 2, domain.Weights{},
		domain.PathOptions{MaxNodes: 1})
	if !errors.Is(err, csr.ErrBudgetExceeded) {
		t.Fatalf("erreur = %v, attendu ErrBudgetExceeded", err)
	}

	apres := g.ExploredNodesTotal()
	if apres <= avant {
		t.Errorf("ExploredNodesTotal après un échec = %d, attendu strictement supérieur à %d avant",
			apres, avant)
	}
}

// TestFindPathBudgetNInterrompPasUnCheminTrouve couvre le cas où le plafond
// est atteint exactement au nœud qui porte la cible : le chemin est acquis,
// il ne doit pas être jeté pour une question de comptage. Le plafond est
// dérivé de ExploredNodes plutôt que codé en dur, pour rester valide quel
// que soit l'ordre dans lequel la file départage ses candidats à égalité.
func TestFindPathBudgetNInterrompPasUnCheminTrouve(t *testing.T) {
	g := carre(t)
	w := domain.Weights{}

	sansPlafond, err := g.FindPath(context.Background(), 0, 2, w, domain.PathOptions{})
	if err != nil {
		t.Fatal(err)
	}

	avecPlafondExact, err := g.FindPath(context.Background(), 0, 2, w,
		domain.PathOptions{MaxNodes: sansPlafond.ExploredNodes - 1})
	if err != nil {
		t.Fatalf("FindPath avec MaxNodes juste sous ExploredNodes : %v, attendu succès", err)
	}
	if avecPlafondExact.ExploredNodes != sansPlafond.ExploredNodes {
		t.Errorf("ExploredNodes = %d avec plafond exact, attendu %d (identique au cas sans plafond)",
			avecPlafondExact.ExploredNodes, sansPlafond.ExploredNodes)
	}
}

func TestFindPathExploredNodesPositif(t *testing.T) {
	g := carre(t)

	p, err := g.FindPath(context.Background(), 0, 2, domain.Weights{}, domain.PathOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if p.ExploredNodes <= 0 {
		t.Errorf("ExploredNodes = %d, attendu > 0 sur un chemin non trivial", p.ExploredNodes)
	}
}

func TestFindPathPenaliseLaReutilisation(t *testing.T) {
	g := carre(t)
	w := domain.Weights{}

	direct, err := g.FindPath(context.Background(), 0, 1, w, domain.PathOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// On interdit (très fortement) les arêtes du premier trajet : l'A* doit
	// faire le tour du carré dans l'autre sens.
	used := map[domain.EdgeRef]struct{}{}
	for _, e := range direct.Edges {
		used[e] = struct{}{}
	}

	detour, err := g.FindPath(context.Background(), 0, 1, w,
		domain.PathOptions{UsedEdges: used, ReuseFactor: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if len(detour.Nodes) <= len(direct.Nodes) {
		t.Errorf("le détour fait %d nœuds, le direct %d : la réutilisation n'a pas été pénalisée",
			len(detour.Nodes), len(direct.Nodes))
	}
}

// TestFindPathPenaliseLaReutilisationEnSensInverse est le pendant du test
// précédent dans le sens qui compte réellement : un tronçon bidirectionnel
// porte deux EdgeRef distincts, un par sens, et c'est précisément le sens
// inverse qu'emprunte un aller-retour (D→W1 puis, plus tard, W1→D). Un test
// qui ne construit UsedEdges qu'à partir du trajet aller — comme celui du
// dessus — ne peut pas voir un mécanisme qui échoue uniquement au retour.
func TestFindPathPenaliseLaReutilisationEnSensInverse(t *testing.T) {
	g := carre(t)
	w := domain.Weights{}

	aller, err := g.FindPath(context.Background(), 0, 1, w, domain.PathOptions{})
	if err != nil {
		t.Fatal(err)
	}

	used := map[domain.EdgeRef]struct{}{}
	for _, e := range aller.Edges {
		used[e] = struct{}{}
	}

	// Le trajet retour emprunte le même tronçon en sens inverse : les EdgeRef
	// utilisées ici (1 → 0) ne sont jamais celles de `used` (0 → 1). Seule la
	// table des arêtes inverses peut relier les deux.
	retour, err := g.FindPath(context.Background(), 1, 0, w,
		domain.PathOptions{UsedEdges: used, ReuseFactor: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if len(retour.Nodes) <= len(aller.Nodes) {
		t.Errorf("le retour fait %d nœuds, l'aller %d : le demi-tour n'a pas été pénalisé",
			len(retour.Nodes), len(aller.Nodes))
	}
}

// TestFindPathEstOptimal compare A* à un Dijkstra naïf sur des graphes tirés
// au sort. C'est le test qui garantit que l'heuristique reste admissible :
// si un jour une pénalité passe sous 1, ce test tombe.
func TestFindPathEstOptimal(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	w := domain.Preferences{AvoidPaved: 0.7}.Weights()

	for iter := 0; iter < 50; iter++ {
		g, n := grapheAleatoire(rng, 60)

		from := domain.NodeRef(rng.Intn(n))
		to := domain.NodeRef(rng.Intn(n))

		got, errA := g.FindPath(context.Background(), from, to, w, domain.PathOptions{})
		want, okD := dijkstraNaif(g, from, to, w)

		if !okD {
			if errA == nil {
				t.Fatalf("itération %d : A* trouve un chemin là où Dijkstra n'en trouve pas", iter)
			}
			continue
		}
		if errA != nil {
			t.Fatalf("itération %d : A* échoue (%v) là où Dijkstra trouve %.2f", iter, errA, want)
		}
		if math.Abs(got.Cost-want) > 1e-6 {
			t.Fatalf("itération %d : A* donne %.4f, optimal %.4f", iter, got.Cost, want)
		}
	}
}

// grapheAleatoire produit une grille bruitée avec des revêtements variés.
func grapheAleatoire(rng *rand.Rand, n int) (*csr.Graph, int) {
	b := csr.NewBuilder()
	for i := 0; i < n; i++ {
		b.AddNode(domain.Coord{
			Lat: 48.0 + rng.Float64()*0.2,
			Lon: -1.8 + rng.Float64()*0.2,
		})
	}
	surfaces := []domain.Surface{
		domain.SurfaceUnknown, domain.SurfacePaved,
		domain.SurfaceGravel, domain.SurfaceGround,
	}
	for i := 0; i < n*3; i++ {
		x := domain.NodeRef(rng.Intn(n))
		y := domain.NodeRef(rng.Intn(n))
		if x == y {
			continue
		}
		a := csr.EdgeAttrs{
			LengthM: domain.HaversineM(b.Coord(x), b.Coord(y)),
			Surface: surfaces[rng.Intn(len(surfaces))],
			Traffic: uint8(rng.Intn(256)),
		}
		b.AddEdge(x, y, a)
		b.AddEdge(y, x, a)
	}
	return b.Build(), n
}

// dijkstraNaif : référence O(n²), volontairement bête et évidemment correcte.
func dijkstraNaif(g *csr.Graph, from, to domain.NodeRef, w domain.Weights) (float64, bool) {
	const inf = math.MaxFloat64
	dist := make([]float64, g.NumNodes())
	done := make([]bool, g.NumNodes())
	for i := range dist {
		dist[i] = inf
	}
	dist[from] = 0

	for {
		best, bestD := -1, inf
		for i, d := range dist {
			if !done[i] && d < bestD {
				best, bestD = i, d
			}
		}
		if best < 0 {
			break
		}
		done[best] = true

		start, end := g.EdgeRange(domain.NodeRef(best))
		for e := start; e < end; e++ {
			next := g.Target(e)
			if cand := bestD + g.Attrs(e).Cost(w); cand < dist[next] {
				dist[next] = cand
			}
		}
	}
	if dist[to] == inf {
		return 0, false
	}
	return dist[to], true
}

func TestNearestNode(t *testing.T) {
	g := carre(t)

	// On vise le nœud 2, et non le nœud 0 : `best` étant initialisé à zéro,
	// une assertion « le résultat vaut 0 » passerait aussi bien si la
	// fonction ne trouvait rien et renvoyait sa valeur par défaut.
	n, ok := g.NearestNode(domain.Coord{Lat: 48.1089, Lon: -1.6671})
	if !ok {
		t.Fatal("aucun nœud trouvé près du coin nord-est")
	}
	if n != 2 {
		t.Errorf("nœud le plus proche = %d, attendu 2", n)
	}
}

func TestNearestNodeHorsZone(t *testing.T) {
	g := carre(t)

	if _, ok := g.NearestNode(domain.Coord{Lat: 43.30, Lon: 5.37}); ok {
		t.Error("Marseille ne doit pas trouver de nœud dans un carré près de Rennes")
	}
}

// TestNearestNodeNoeudIsole couvre le cas d'un unique nœud entouré de cellules
// vides — la situation typique d'un départ de trail en zone peu dense. Une
// recherche par anneaux qui réinitialise son meilleur candidat à chaque
// itération perd la touche du premier anneau et conclut à tort « hors zone ».
func TestNearestNodeNoeudIsole(t *testing.T) {
	b := csr.NewBuilder()

	// Un leurre très éloigné et sans arête — un singleton, la plus petite
	// composante connexe possible — occupe l'indice 0. Sans lui, le nœud
	// attendu vaudrait lui-même 0 et l'assertion d'identité ne pourrait
	// jamais échouer : elle passerait même si la fonction renvoyait sa valeur
	// par défaut.
	b.AddNode(domain.Coord{Lat: 48.6493, Lon: -2.0257})
	seul := b.AddNode(domain.Coord{Lat: 48.1000, Lon: -1.6800})
	// seul a besoin d'une arête pour appartenir à une composante connexe de
	// taille supérieure à celle du leurre, donc d'être retenu comme routable
	// par NearestNode — voir C1. Le compagnon n'a pas à être proche
	// géométriquement : seule la connexité compte ici.
	compagnon := b.AddNode(domain.Coord{Lat: 48.6500, Lon: -2.0300})
	longueur := domain.HaversineM(b.Coord(seul), b.Coord(compagnon))
	b.AddEdge(seul, compagnon, csr.EdgeAttrs{LengthM: longueur})
	b.AddEdge(compagnon, seul, csr.EdgeAttrs{LengthM: longueur})
	g := b.Build()

	// À une centaine de mètres : même cellule ou cellule immédiatement
	// voisine, et rien d'autre alentour sur des kilomètres.
	n, ok := g.NearestNode(domain.Coord{Lat: 48.1008, Lon: -1.6805})
	if !ok {
		t.Fatal("un nœud isolé doit être trouvé, pas déclaré hors zone")
	}
	if n != seul {
		t.Errorf("nœud trouvé = %d, attendu %d", n, seul)
	}
}

// TestNearestNodeIgnoreUneComposanteIsolee couvre C1 : un nœud géométriquement
// plus proche mais qui n'appartient à aucune arête ne mène nulle part, et ne
// doit jamais être préféré à un nœud plus loin mais routable.
func TestNearestNodeIgnoreUneComposanteIsolee(t *testing.T) {
	b := csr.NewBuilder()

	// isole est à quelques mètres du point visé, mais sans la moindre arête :
	// c'est le cas d'une impasse ou d'une cour coupée du réseau routier.
	isole := b.AddNode(domain.Coord{Lat: 48.1000, Lon: -1.6800})
	// a et c sont plus loin mais reliés entre eux, donc routables.
	a := b.AddNode(domain.Coord{Lat: 48.1006, Lon: -1.6806})
	c := b.AddNode(domain.Coord{Lat: 48.1007, Lon: -1.6807})
	longueur := domain.HaversineM(b.Coord(a), b.Coord(c))
	b.AddEdge(a, c, csr.EdgeAttrs{LengthM: longueur})
	b.AddEdge(c, a, csr.EdgeAttrs{LengthM: longueur})
	g := b.Build()

	n, ok := g.NearestNode(domain.Coord{Lat: 48.1000, Lon: -1.6801})
	if !ok {
		t.Fatal("aucun nœud routable trouvé, alors que a et c sont à proximité")
	}
	if n == isole {
		t.Fatalf("NearestNode a accroché le nœud isolé %d, qui ne mène nulle part", isole)
	}
	if n != a {
		t.Errorf("nœud trouvé = %d, attendu %d (le nœud routable le plus proche)", n, a)
	}
}

func BenchmarkFindPath(b *testing.B) {
	// Un graphe assez grand pour que l'heuristique compte réellement.
	g, n := grapheAleatoire(rand.New(rand.NewSource(42)), 4000)

	// Deux pondérations, pour rendre l'arbitrage visible : plus l'écart entre
	// le meilleur et le pire terrain se creuse, moins l'heuristique informe.
	cas := map[string]domain.Weights{
		"neutre":      {},
		"anti-bitume": domain.Preferences{AvoidPaved: 1}.Weights(),
	}

	for nom, w := range cas {
		b.Run(nom, func(b *testing.B) {
			var explored int64
			var trouves int64

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				from := domain.NodeRef(i % n)
				to := domain.NodeRef((i * 7919) % n) // pas fixe premier : évite les paires corrélées

				p, err := g.FindPath(context.Background(), from, to, w, domain.PathOptions{})
				if err != nil {
					continue
				}
				explored += int64(p.ExploredNodes)
				trouves++
			}
			b.StopTimer()

			if trouves > 0 {
				b.ReportMetric(float64(explored)/float64(trouves), "nœuds/chemin")
			}
		})
	}
}

// ligne construit A—B—C, bidirectionnel, pour que le chemin de A à C soit forcé
// et ses longueurs connues. Le carré ne convient pas : ses deux itinéraires
// entre coins opposés sont de longueur égale, et le choix entre eux est
// arbitraire.
func ligne(t *testing.T) (*csr.Graph, [3]domain.NodeRef, [2]float64) {
	t.Helper()

	b := csr.NewBuilder()
	n := [3]domain.NodeRef{
		b.AddNode(domain.Coord{Lat: 48.100, Lon: -1.680}),
		b.AddNode(domain.Coord{Lat: 48.100, Lon: -1.670}),
		b.AddNode(domain.Coord{Lat: 48.100, Lon: -1.650}),
	}
	var longueurs [2]float64
	for i := 0; i < 2; i++ {
		l := domain.HaversineM(coordOf(b, n[i]), coordOf(b, n[i+1]))
		longueurs[i] = l
		b.AddEdge(n[i], n[i+1], csr.EdgeAttrs{LengthM: l})
		b.AddEdge(n[i+1], n[i], csr.EdgeAttrs{LengthM: l})
	}
	return b.Build(), n, longueurs
}

func TestFindPathCompteLeTraceRepasse(t *testing.T) {
	g, n, longueurs := ligne(t)

	// Seul le premier tronçon a déjà servi. ReuseFactor 1 laisse l'itinéraire
	// intact : ce qui est mesuré ici est le compte, pas l'évitement.
	used := map[domain.EdgeRef]struct{}{}
	depart, err := g.FindPath(context.Background(), n[0], n[1], domain.Weights{}, domain.PathOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range depart.Edges {
		used[e] = struct{}{}
	}

	p, err := g.FindPath(context.Background(), n[0], n[2], domain.Weights{},
		domain.PathOptions{UsedEdges: used, ReuseFactor: 1})
	if err != nil {
		t.Fatal(err)
	}

	if math.Abs(p.RetracedM-longueurs[0]) > 1e-6 {
		t.Errorf("RetracedM = %v, attendu %v (le premier tronçon seulement)", p.RetracedM, longueurs[0])
	}
	if p.RetracedM <= 0 || p.RetracedM >= p.LengthM {
		t.Errorf("RetracedM = %v hors de ]0, %v[ : le compte est dégénéré", p.RetracedM, p.LengthM)
	}
}

// TestFindPathCompteLeTraceRepasseEnSensInverse est celui qui compte : un
// aller-retour emprunte l'EdgeRef jumelle, jamais celle qui est dans UsedEdges.
// Un compte qui ignorerait la table des arêtes inverses passerait le test
// précédent et raterait le seul cas qui se produit vraiment.
func TestFindPathCompteLeTraceRepasseEnSensInverse(t *testing.T) {
	g, n, longueurs := ligne(t)

	used := map[domain.EdgeRef]struct{}{}
	retourDeja, err := g.FindPath(context.Background(), n[1], n[0], domain.Weights{}, domain.PathOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range retourDeja.Edges {
		used[e] = struct{}{}
	}

	p, err := g.FindPath(context.Background(), n[0], n[2], domain.Weights{},
		domain.PathOptions{UsedEdges: used, ReuseFactor: 1})
	if err != nil {
		t.Fatal(err)
	}

	if math.Abs(p.RetracedM-longueurs[0]) > 1e-6 {
		t.Errorf("RetracedM = %v, attendu %v : le sens inverse n'est pas compté", p.RetracedM, longueurs[0])
	}
}

func TestFindPathSansReutilisationNeRepasseSurRien(t *testing.T) {
	g, n, _ := ligne(t)

	p, err := g.FindPath(context.Background(), n[0], n[2], domain.Weights{}, domain.PathOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if p.LengthM <= 0 {
		t.Fatalf("LengthM = %v : le chemin est vide, le reste du test ne prouve rien", p.LengthM)
	}
	if p.RetracedM != 0 {
		t.Errorf("RetracedM = %v, attendu 0 sans arête déjà consommée", p.RetracedM)
	}
}
