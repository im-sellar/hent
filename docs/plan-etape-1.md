# hent — Plan d'implémentation (étape 1)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** livrer un service HTTP qui, à partir d'un point de départ et d'une distance cible, renvoie des boucles trail réellement courables en Ille-et-Vilaine, exportables en GPX.

**Architecture:** deux binaires (`graphbuild` hors ligne, `routed` en service) autour d'un artefact `graph.bin`. Clean architecture avec ports à granularité grossière : l'A* et le parcours du graphe CSR restent confinés dans l'adaptateur, seules des données traversent les frontières.

**Tech Stack:** Go 1.25, `github.com/paulmach/osm` (lecture PBF), `golang.org/x/sync/errgroup`, stdlib pour tout le reste (`net/http`, `container/heap`, `encoding/binary`, `encoding/xml`).

**Spec:** `~/Library/CloudStorage/OneDrive-Hellowork/obsidian-vault/claude/hent/2026-08-18-hent-design.md`

**Périmètre :** étape 1 de la feuille de route (§2 de la spec) — trail à pied, anti-bitume, longueur cible, OSM seul. Les étapes 2 à 5 (dénivelé, POI, canopée, vélo) feront l'objet de plans distincts. À la fin de ce plan, le service tourne et produit des GPX exploitables.

## Global Constraints

- **Go 1.25** — `go 1.25` dans `go.mod`.
- **Module path :** `github.com/im-sellar/hent`. Le seul paramètre à ajuster : si le handle GitHub réel diffère, le changer partout avant la Task 1 (il est codé en dur dans le test d'architecture).
- **Emplacement du dépôt :** `/Users/amorice/DevHome/02_Perso/hent`.
- **Identité git :** l'identité globale de la machine est `amorice@hellowork.com`. Pour un dépôt perso destiné à être publié, poser une identité locale au dépôt à la Task 1.
- **Règle de dépendance :** `domain/` n'importe rien du projet. `app/` n'importe que `domain/`. `adapter/` et `platform/` peuvent importer `app/` et `domain/`. Vérifiée par test, pas par discipline.
- **Règle du modèle de coût :** tout critère se formule comme une **pénalité dans [0,1] avec un poids ≥ 0**, jamais comme une récompense. Un coût négatif casse Dijkstra et A* silencieusement.
- **Attribution :** toute réponse de l'API porte `© les contributeurs OpenStreetMap`.
- **Langue :** commentaires et messages d'erreur en français, identifiants en anglais.
- **Commits :** pas de signature `Co-Authored-By`, pas de mention d'outil.

## Structure des fichiers

| Fichier | Responsabilité |
|---|---|
| `internal/domain/geo.go` | `Coord`, `BBox`, distance orthodromique. Aucune dépendance. |
| `internal/domain/track.go` | `NodeRef`, `EdgeRef`, `Path`, `Loop`, `PathOptions`. |
| `internal/domain/profile.go` | `Surface`, `WayClass`, `Preferences` → `Weights`. |
| `internal/domain/score.go` | `Score` et son calcul à partir d'une `Loop`. |
| `internal/app/port/network.go` | interface `RouteNetwork`, définie chez le consommateur. |
| `internal/app/generateloop/generate.go` | stratégie waypoints + dichotomie + anti-réutilisation. |
| `internal/app/generateloop/dedupe.go` | déduplication Jaccard. |
| `internal/adapter/network/csr/graph.go` | structure CSR, `Builder`, accesseurs. |
| `internal/adapter/network/csr/attrs.go` | `EdgeAttrs` et fonction de coût. |
| `internal/adapter/network/csr/astar.go` | A* + file de priorité. |
| `internal/adapter/network/csr/nearest.go` | recherche du nœud le plus proche (grille spatiale). |
| `internal/adapter/network/csr/codec.go` | sérialisation / lecture de `graph.bin`. |
| `internal/adapter/osmsource/read.go` | lecture PBF, filtrage, construction du graphe. |
| `internal/adapter/osmsource/tags.go` | tags OSM → `Surface`, `WayClass`, exposition trafic. |
| `internal/adapter/httpapi/handler.go` | routes, DTO, validation. |
| `internal/adapter/gpxfile/write.go` | export GPX. |
| `internal/architecture_test.go` | garde-fou de la règle de dépendance. |
| `cmd/graphbuild/main.go` | CLI de construction de l'artefact. |
| `cmd/routed/main.go` | serveur, câblage, vérification de satisfaction des ports. |

---

## Task 1 : Squelette, géométrie, garde-fou d'architecture

**Files:**
- Create: `go.mod`, `.gitignore`, `README.md`
- Create: `internal/domain/geo.go`, `internal/domain/geo_test.go`
- Create: `internal/architecture_test.go`

**Interfaces:**
- Consumes: rien.
- Produces: `domain.Coord{Lat, Lon float64}`, `domain.BBox{Min, Max Coord}`, `domain.HaversineM(a, b Coord) float64`, `domain.BBox.Contains(Coord) bool`.

- [ ] **Step 1: Initialiser le dépôt**

```bash
mkdir -p /Users/amorice/DevHome/02_Perso/hent
cd /Users/amorice/DevHome/02_Perso/hent
git init
git config user.name "Aurélien MORICE"
git config user.email "<ton-email-perso>"   # dépôt destiné à être publié
go mod init github.com/im-sellar/hent
printf 'graph.bin\n*.osm.pbf\n/hent\n/graphbuild\n/routed\n' > .gitignore
printf '# hent\n\nGénérateur de boucles trail « nature-aware ».\n\nDonnées © les contributeurs OpenStreetMap (ODbL).\n' > README.md
```

- [ ] **Step 2: Écrire le test de géométrie (il doit échouer)**

`internal/domain/geo_test.go` :

```go
package domain_test

import (
	"math"
	"testing"

	"github.com/im-sellar/hent/internal/domain"
)

func TestHaversineM_RennesSaintMalo(t *testing.T) {
	rennes := domain.Coord{Lat: 48.1173, Lon: -1.6778}
	stMalo := domain.Coord{Lat: 48.6493, Lon: -2.0257}

	got := domain.HaversineM(rennes, stMalo)

	const want = 64000.0
	if math.Abs(got-want) > 2000 {
		t.Fatalf("Rennes→Saint-Malo = %.0f m, attendu ~%.0f m (±2 km)", got, want)
	}
}

func TestHaversineM_Symetrique(t *testing.T) {
	a := domain.Coord{Lat: 48.11, Lon: -1.67}
	b := domain.Coord{Lat: 48.20, Lon: -1.50}

	if math.Abs(domain.HaversineM(a, b)-domain.HaversineM(b, a)) > 1e-6 {
		t.Fatal("la distance doit être symétrique")
	}
}

func TestHaversineM_MemePoint(t *testing.T) {
	a := domain.Coord{Lat: 48.11, Lon: -1.67}

	if got := domain.HaversineM(a, a); got != 0 {
		t.Fatalf("distance d'un point à lui-même = %v, attendu 0", got)
	}
}

func TestBBoxContains(t *testing.T) {
	b := domain.BBox{
		Min: domain.Coord{Lat: 48.0, Lon: -2.0},
		Max: domain.Coord{Lat: 48.5, Lon: -1.0},
	}

	if !b.Contains(domain.Coord{Lat: 48.1, Lon: -1.6}) {
		t.Error("le point intérieur doit être contenu")
	}
	if b.Contains(domain.Coord{Lat: 49.0, Lon: -1.6}) {
		t.Error("le point extérieur ne doit pas être contenu")
	}
}
```

- [ ] **Step 3: Lancer les tests, vérifier l'échec**

Run: `go test ./internal/domain/`
Expected: FAIL — `undefined: domain.Coord`

- [ ] **Step 4: Implémenter la géométrie**

`internal/domain/geo.go` :

```go
// Package domain contient les types métier de hent.
// Il n'importe rien d'autre que la bibliothèque standard : c'est la couche
// dont tout le reste dépend, et qui ne dépend de rien.
package domain

import "math"

// Rayon moyen de la Terre (IUGG), en mètres.
const earthRadiusM = 6371008.8

type Coord struct {
	Lat, Lon float64
}

type BBox struct {
	Min, Max Coord
}

func (b BBox) Contains(c Coord) bool {
	return c.Lat >= b.Min.Lat && c.Lat <= b.Max.Lat &&
		c.Lon >= b.Min.Lon && c.Lon <= b.Max.Lon
}

// HaversineM retourne la distance orthodromique en mètres entre a et b.
func HaversineM(a, b Coord) float64 {
	lat1, lat2 := radians(a.Lat), radians(b.Lat)
	dLat := radians(b.Lat - a.Lat)
	dLon := radians(b.Lon - a.Lon)

	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)

	return 2 * earthRadiusM * math.Asin(math.Sqrt(h))
}

func radians(deg float64) float64 { return deg * math.Pi / 180 }
```

- [ ] **Step 5: Lancer les tests, vérifier le succès**

Run: `go test ./internal/domain/`
Expected: PASS (4 tests)

- [ ] **Step 6: Écrire le garde-fou d'architecture**

`internal/architecture_test.go`. Ce test lit les imports de chaque fichier Go et échoue si une couche interne dépend d'une couche externe. Sans dépendance externe : `go/parser` en mode `ImportsOnly` suffit.

```go
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
					if strings.HasPrefix(pkg, modulePath+"/"+b) {
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
```

- [ ] **Step 7: Vérifier que le garde-fou passe**

Run: `go test ./internal/ -run TestRegleDeDependance -v`
Expected: PASS

Pour se convaincre qu'il mord vraiment, ajouter temporairement dans `internal/domain/geo.go` un import `"github.com/im-sellar/hent/internal/app"`, relancer, constater l'échec, puis retirer l'import.

- [ ] **Step 8: Commit**

```bash
go vet ./... && go test ./...
git add -A
git commit -m "chore: squelette du projet, géométrie et garde-fou d'architecture"
```

---

## Task 2 : Le graphe CSR et son constructeur

Représentation *compressed sparse row* : un tableau d'offsets indexé par nœud, un tableau d'arêtes contiguës. Les arêtes sont **dirigées** — un tronçon bidirectionnel produit deux arêtes (voir §5 de la spec : la pente signée l'imposera à l'étape 2, autant le poser maintenant).

**Files:**
- Create: `internal/domain/track.go`
- Create: `internal/adapter/network/csr/graph.go`, `internal/adapter/network/csr/graph_test.go`

**Interfaces:**
- Consumes: `domain.Coord` (Task 1).
- Produces:
  - `domain.NodeRef uint32`, `domain.EdgeRef uint32`
  - `csr.Builder` : `NewBuilder() *Builder`, `(*Builder).AddNode(domain.Coord) domain.NodeRef`, `(*Builder).AddEdge(from, to domain.NodeRef, a EdgeAttrs)`, `(*Builder).Build() *Graph`
  - `csr.Graph` : `NumNodes() int`, `NumEdges() int`, `Coord(domain.NodeRef) domain.Coord`, `EdgeRange(domain.NodeRef) (start, end domain.EdgeRef)`, `Target(domain.EdgeRef) domain.NodeRef`, `Attrs(domain.EdgeRef) EdgeAttrs`, `BBox() domain.BBox`
  - `csr.EdgeAttrs` est défini en Task 3 ; en Task 2 il ne contient que `LengthM float64`.

- [ ] **Step 1: Écrire le test du graphe jouet**

`internal/adapter/network/csr/graph_test.go`. Le graphe de test est un carré de quatre nœuds, utilisé dans toutes les tâches suivantes.

```go
package csr_test

import (
	"testing"

	"github.com/im-sellar/hent/internal/adapter/network/csr"
	"github.com/im-sellar/hent/internal/domain"
)

// carre construit un graphe jouet : quatre nœuds aux coins d'un carré
// d'environ 1 km de côté près de Rennes, reliés en cycle, arêtes dans les
// deux sens.
//
//	3 ── 2
//	│    │
//	0 ── 1
func carre(t *testing.T) *csr.Graph {
	t.Helper()

	b := csr.NewBuilder()
	n0 := b.AddNode(domain.Coord{Lat: 48.100, Lon: -1.680})
	n1 := b.AddNode(domain.Coord{Lat: 48.100, Lon: -1.667})
	n2 := b.AddNode(domain.Coord{Lat: 48.109, Lon: -1.667})
	n3 := b.AddNode(domain.Coord{Lat: 48.109, Lon: -1.680})

	for _, pair := range [][2]domain.NodeRef{{n0, n1}, {n1, n2}, {n2, n3}, {n3, n0}} {
		from, to := pair[0], pair[1]
		length := domain.HaversineM(coordOf(b, from), coordOf(b, to))
		b.AddEdge(from, to, csr.EdgeAttrs{LengthM: length})
		b.AddEdge(to, from, csr.EdgeAttrs{LengthM: length})
	}

	return b.Build()
}

func coordOf(b *csr.Builder, n domain.NodeRef) domain.Coord { return b.Coord(n) }

func TestGraphTaille(t *testing.T) {
	g := carre(t)

	if got := g.NumNodes(); got != 4 {
		t.Errorf("NumNodes = %d, attendu 4", got)
	}
	if got := g.NumEdges(); got != 8 {
		t.Errorf("NumEdges = %d, attendu 8 (4 arêtes × 2 sens)", got)
	}
}

func TestGraphVoisins(t *testing.T) {
	g := carre(t)

	// Chaque nœud du cycle a exactement deux voisins sortants.
	for n := domain.NodeRef(0); n < 4; n++ {
		start, end := g.EdgeRange(n)
		if got := int(end - start); got != 2 {
			t.Errorf("nœud %d : %d arêtes sortantes, attendu 2", n, got)
		}
	}
}

func TestGraphCibleEtLongueur(t *testing.T) {
	g := carre(t)

	start, end := g.EdgeRange(0)
	voisins := map[domain.NodeRef]bool{}
	for e := start; e < end; e++ {
		voisins[g.Target(e)] = true
		if l := g.Attrs(e).LengthM; l <= 0 {
			t.Errorf("arête %d : longueur %v, attendu > 0", e, l)
		}
	}

	if !voisins[1] || !voisins[3] {
		t.Errorf("les voisins du nœud 0 sont %v, attendu {1, 3}", voisins)
	}
}

func TestGraphBBox(t *testing.T) {
	g := carre(t)
	b := g.BBox()

	if !b.Contains(domain.Coord{Lat: 48.105, Lon: -1.673}) {
		t.Error("la bbox doit contenir le centre du carré")
	}
	if b.Contains(domain.Coord{Lat: 49.0, Lon: -1.673}) {
		t.Error("la bbox ne doit pas contenir un point hors du carré")
	}
}
```

Note : le helper appelle `b.Coord(...)` avant `Build()`. C'est pourquoi `Builder` expose `Coord` : sans lui, le test ne pourrait pas calculer les longueurs d'arêtes qu'il fournit lui-même.

- [ ] **Step 2: Lancer, vérifier l'échec**

Run: `go test ./internal/adapter/network/csr/`
Expected: FAIL — `undefined: csr.NewBuilder`

- [ ] **Step 3: Définir les références du domaine**

`internal/domain/track.go` :

```go
package domain

// NodeRef et EdgeRef sont des identifiants opaques manipulés par le domaine
// mais interprétés uniquement par l'adaptateur réseau. Ils permettent de
// désigner des arêtes (pour interdire leur réutilisation dans une boucle)
// sans que le domaine connaisse la structure du graphe.
type NodeRef uint32

type EdgeRef uint32

// Path est un chemin résolu entre deux nœuds.
type Path struct {
	Nodes   []NodeRef
	Edges   []EdgeRef
	Coords  []Coord
	LengthM float64
	Cost    float64
}
```

- [ ] **Step 4: Implémenter le graphe CSR**

`internal/adapter/network/csr/graph.go` :

```go
// Package csr implémente le graphe routier en représentation
// « compressed sparse row » : un tableau d'offsets indexé par nœud et un
// tableau d'arêtes contiguës. Cette disposition est compacte et contiguë en
// mémoire, donc rapide à parcourir — ce qui compte, l'A* traversant des
// centaines de milliers d'arêtes par requête.
package csr

import (
	"sort"

	"github.com/im-sellar/hent/internal/domain"
)

// EdgeAttrs porte les attributs objectifs d'une arête. Aucun coût n'y est
// stocké : le coût dépend du profil et des préférences de la requête, il est
// calculé à la volée (voir attrs.go).
type EdgeAttrs struct {
	LengthM float64
}

type Graph struct {
	coords  []domain.Coord
	offsets []uint32 // len == len(coords)+1
	targets []domain.NodeRef
	attrs   []EdgeAttrs
	bbox    domain.BBox
}

func (g *Graph) NumNodes() int { return len(g.coords) }
func (g *Graph) NumEdges() int { return len(g.targets) }

func (g *Graph) Coord(n domain.NodeRef) domain.Coord { return g.coords[n] }

// EdgeRange retourne l'intervalle demi-ouvert [start, end) des arêtes
// sortantes du nœud n.
func (g *Graph) EdgeRange(n domain.NodeRef) (start, end domain.EdgeRef) {
	return domain.EdgeRef(g.offsets[n]), domain.EdgeRef(g.offsets[n+1])
}

func (g *Graph) Target(e domain.EdgeRef) domain.NodeRef { return g.targets[e] }
func (g *Graph) Attrs(e domain.EdgeRef) EdgeAttrs       { return g.attrs[e] }
func (g *Graph) BBox() domain.BBox                      { return g.bbox }

type pendingEdge struct {
	from, to domain.NodeRef
	attrs    EdgeAttrs
}

type Builder struct {
	coords []domain.Coord
	edges  []pendingEdge
}

func NewBuilder() *Builder { return &Builder{} }

func (b *Builder) AddNode(c domain.Coord) domain.NodeRef {
	b.coords = append(b.coords, c)
	return domain.NodeRef(len(b.coords) - 1)
}

func (b *Builder) Coord(n domain.NodeRef) domain.Coord { return b.coords[n] }

func (b *Builder) AddEdge(from, to domain.NodeRef, a EdgeAttrs) {
	b.edges = append(b.edges, pendingEdge{from: from, to: to, attrs: a})
}

// Build trie les arêtes par nœud source et calcule les offsets.
func (b *Builder) Build() *Graph {
	sort.Slice(b.edges, func(i, j int) bool { return b.edges[i].from < b.edges[j].from })

	g := &Graph{
		coords:  b.coords,
		offsets: make([]uint32, len(b.coords)+1),
		targets: make([]domain.NodeRef, len(b.edges)),
		attrs:   make([]EdgeAttrs, len(b.edges)),
	}

	// Comptage par nœud, puis somme préfixe.
	counts := make([]uint32, len(b.coords))
	for _, e := range b.edges {
		counts[e.from]++
	}
	var acc uint32
	for i, c := range counts {
		g.offsets[i] = acc
		acc += c
	}
	g.offsets[len(counts)] = acc

	for i, e := range b.edges {
		g.targets[i] = e.to
		g.attrs[i] = e.attrs
	}

	g.bbox = computeBBox(b.coords)
	return g
}

func computeBBox(coords []domain.Coord) domain.BBox {
	if len(coords) == 0 {
		return domain.BBox{}
	}
	b := domain.BBox{Min: coords[0], Max: coords[0]}
	for _, c := range coords[1:] {
		b.Min.Lat = min(b.Min.Lat, c.Lat)
		b.Min.Lon = min(b.Min.Lon, c.Lon)
		b.Max.Lat = max(b.Max.Lat, c.Lat)
		b.Max.Lon = max(b.Max.Lon, c.Lon)
	}
	return b
}
```

- [ ] **Step 5: Lancer, vérifier le succès**

Run: `go test ./internal/adapter/network/csr/ -v`
Expected: PASS (4 tests)

- [ ] **Step 6: Commit**

```bash
go vet ./... && go test ./...
git add -A
git commit -m "feat: graphe routier en représentation CSR"
```

---

## Task 3 : Profil, pondérations et fonction de coût

C'est ici que se joue l'invariant central de la spec (§6) : le facteur multiplicatif est toujours ≥ 1, donc le coût minimal par mètre vaut exactement 1, donc l'heuristique de l'A* (distance à vol d'oiseau) est admissible sans calibrage.

**Files:**
- Create: `internal/domain/profile.go`, `internal/domain/profile_test.go`
- Create: `internal/adapter/network/csr/attrs.go`, `internal/adapter/network/csr/attrs_test.go`
- Modify: `internal/adapter/network/csr/graph.go` (retirer la définition provisoire de `EdgeAttrs`, désormais dans `attrs.go`)

**Interfaces:**
- Consumes: `csr.EdgeAttrs` (Task 2).
- Produces:
  - `domain.Surface` : `SurfaceUnknown`, `SurfacePaved`, `SurfaceGravel`, `SurfaceGround`
  - `domain.WayClass` : `WayUnknown`, `WayPath`, `WayTrack`, `WayFootway`, `WayResidential`, `WayTertiary`, `WaySecondary`
  - `domain.Preferences{AvoidPaved float64}` et `(Preferences).Weights() Weights`
  - `domain.Weights{Paved, Traffic float64}`
  - `csr.EdgeAttrs{LengthM float64; Surface domain.Surface; Class domain.WayClass; Traffic uint8}`
  - `(EdgeAttrs).Cost(domain.Weights) float64`

- [ ] **Step 1: Écrire le test des pondérations**

`internal/domain/profile_test.go` :

```go
package domain_test

import (
	"math"
	"testing"

	"github.com/im-sellar/hent/internal/domain"
)

func TestWeightsCroissantAvecAvoidPaved(t *testing.T) {
	faible := domain.Preferences{AvoidPaved: 0.0}.Weights()
	fort := domain.Preferences{AvoidPaved: 1.0}.Weights()

	if fort.Paved <= faible.Paved {
		t.Errorf("Paved : %v à 1.0 doit dépasser %v à 0.0", fort.Paved, faible.Paved)
	}
	if fort.Traffic <= faible.Traffic {
		t.Errorf("Traffic : %v à 1.0 doit dépasser %v à 0.0", fort.Traffic, faible.Traffic)
	}
}

func TestWeightsTraficJamaisNul(t *testing.T) {
	// Même sans aversion déclarée pour le bitume, on ne veut jamais être
	// envoyé sur une route à fort trafic.
	w := domain.Preferences{AvoidPaved: 0}.Weights()

	if w.Traffic <= 0 {
		t.Errorf("Traffic = %v, attendu > 0 même à AvoidPaved = 0", w.Traffic)
	}
}

func TestWeightsToujoursPositives(t *testing.T) {
	// Invariant du §6 de la spec : jamais de poids négatif, sous peine de
	// casser silencieusement A*.
	//
	// L'assertion porte sur « est un nombre fini et positif », et non sur
	// « n'est pas négatif » : cette dernière passerait avec un NaN en sortie,
	// puisque toute comparaison avec NaN est fausse. C'est précisément le
	// piège que ce test doit attraper.
	for _, v := range []float64{-5, 0, 0.5, 1, 42,
		math.NaN(), math.Inf(1), math.Inf(-1)} {

		w := domain.Preferences{AvoidPaved: v}.Weights()

		for nom, poids := range map[string]float64{"Paved": w.Paved, "Traffic": w.Traffic} {
			if math.IsNaN(poids) || math.IsInf(poids, 0) {
				t.Errorf("AvoidPaved=%v donne un poids %s non fini : %v", v, nom, poids)
			}
			if poids < 0 {
				t.Errorf("AvoidPaved=%v donne un poids %s négatif : %v", v, nom, poids)
			}
		}
	}
}
```

- [ ] **Step 2: Écrire le test de la fonction de coût**

`internal/adapter/network/csr/attrs_test.go` :

```go
package csr_test

import (
	"math"
	"testing"

	"github.com/im-sellar/hent/internal/adapter/network/csr"
	"github.com/im-sellar/hent/internal/domain"
)

func TestCostBitumePlusCherQueSentier(t *testing.T) {
	w := domain.Preferences{AvoidPaved: 0.8}.Weights()

	sentier := csr.EdgeAttrs{LengthM: 1000, Surface: domain.SurfaceGround, Class: domain.WayPath}
	route := csr.EdgeAttrs{LengthM: 1000, Surface: domain.SurfacePaved, Class: domain.WaySecondary, Traffic: 255}

	if route.Cost(w) <= sentier.Cost(w) {
		t.Errorf("route %.0f doit coûter plus qu'un sentier %.0f", route.Cost(w), sentier.Cost(w))
	}
}

func TestCostJamaisInferieurALaLongueur(t *testing.T) {
	// Invariant central : le facteur est toujours ≥ 1, donc coût ≥ longueur.
	// C'est ce qui rend l'heuristique de l'A* admissible.
	w := domain.Preferences{AvoidPaved: 1}.Weights()

	// La dernière surface est hors énumération : elle vérifie que le cas
	// « default » de la pénalité de revêtement reste dans [0,1].
	surfaces := []domain.Surface{
		domain.SurfaceUnknown, domain.SurfacePaved,
		domain.SurfaceGravel, domain.SurfaceGround, domain.Surface(99),
	}
	for _, s := range surfaces {
		for _, traffic := range []uint8{0, 128, 255} {
			a := csr.EdgeAttrs{LengthM: 500, Surface: s, Traffic: traffic}
			got := a.Cost(w)

			// On exige un coût fini et supérieur ou égal à la longueur.
			// Tester seulement « got < a.LengthM » ne suffirait pas : un NaN
			// rendrait cette comparaison fausse et le test passerait.
			if math.IsNaN(got) || math.IsInf(got, 0) {
				t.Errorf("surface=%v trafic=%d : coût non fini (%v)", s, traffic, got)
			}
			if got < a.LengthM {
				t.Errorf("surface=%v trafic=%d : coût %.1f < longueur %.1f",
					s, traffic, got, a.LengthM)
			}
		}
	}
}

func TestCostSansPreferenceVautLaLongueur(t *testing.T) {
	// Poids nuls : le coût doit se réduire exactement à la distance.
	a := csr.EdgeAttrs{LengthM: 750, Surface: domain.SurfacePaved, Traffic: 255}

	if got := a.Cost(domain.Weights{}); got != 750 {
		t.Errorf("coût à poids nuls = %v, attendu 750", got)
	}
}
```

- [ ] **Step 3: Lancer, vérifier l'échec**

Run: `go test ./internal/domain/ ./internal/adapter/network/csr/`
Expected: FAIL — `undefined: domain.Preferences`, `unknown field Surface`

- [ ] **Step 4: Implémenter le profil**

`internal/domain/profile.go` :

```go
package domain

import "math"

type Surface uint8

const (
	SurfaceUnknown Surface = iota
	SurfacePaved
	SurfaceGravel
	SurfaceGround
)

type WayClass uint8

const (
	WayUnknown WayClass = iota
	WayPath
	WayTrack
	WayFootway
	WayResidential
	WayTertiary
	WaySecondary
)

// Preferences exprime l'intention de l'utilisateur sur une échelle 0-1.
// C'est ce que l'API expose ; les poids internes ne sortent jamais.
type Preferences struct {
	AvoidPaved float64
}

// Weights porte les coefficients du modèle de coût. Ils sont toujours ≥ 0 :
// voir la note d'invariant sur EdgeAttrs.Cost.
type Weights struct {
	Paved   float64
	Traffic float64
}

const (
	maxPavedWeight   = 6.0
	minTrafficWeight = 2.0
	maxTrafficWeight = 12.0
)

func (p Preferences) Weights() Weights {
	a := clamp01(p.AvoidPaved)
	return Weights{
		Paved: a * maxPavedWeight,
		// Le trafic garde un plancher : on ne veut jamais être envoyé sur une
		// départementale, même quand l'utilisateur ne réclame pas de chemins.
		Traffic: minTrafficWeight + a*(maxTrafficWeight-minTrafficWeight),
	}
}

func clamp01(v float64) float64 {
	switch {
	// NaN se traite en premier, et explicitement : en Go toute comparaison
	// impliquant NaN est fausse, donc un NaN traverserait les deux cas
	// suivants intact. Il contaminerait alors les poids, puis le coût, puis
	// l'A* — qui ne relaxerait plus aucune arête, sans erreur ni test rouge,
	// car « NaN < x » est également faux.
	case math.IsNaN(v):
		return 0
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}
```

- [ ] **Step 5: Implémenter les attributs et le coût**

Retirer d'abord la définition provisoire de `EdgeAttrs` de `graph.go`, puis créer `internal/adapter/network/csr/attrs.go` :

```go
package csr

import "github.com/im-sellar/hent/internal/domain"

// EdgeAttrs porte les attributs objectifs d'une arête, mesurés une fois au
// build. Aucun coût n'y est stocké : le coût dépend des préférences de la
// requête et se calcule à la volée.
type EdgeAttrs struct {
	LengthM float64
	Surface domain.Surface
	Class   domain.WayClass
	Traffic uint8 // exposition au trafic, 0 = aucune, 255 = maximale
}

// Cost applique le modèle de la spec (§6) :
//
//	coût = longueur × ( 1 + Σ wᵢ × pénalitéᵢ )
//
// INVARIANT — à ne jamais violer : chaque pénalité appartient à [0,1] et
// chaque poids est ≥ 0. Le facteur est donc toujours ≥ 1, le coût minimal par
// mètre vaut exactement 1, et l'heuristique de l'A* (distance à vol d'oiseau)
// est admissible sans calibrage.
//
// Corollaire : on n'exprime JAMAIS un critère comme une récompense. Pour
// favoriser l'ombre, on pénalise le soleil. Un coût négatif rend Dijkstra et
// A* faux — silencieusement, sans la moindre erreur, avec des chemins absurdes.
func (a EdgeAttrs) Cost(w domain.Weights) float64 {
	penalty := w.Paved*a.pavedPenalty() + w.Traffic*a.trafficPenalty()
	return a.LengthM * (1 + penalty)
}

func (a EdgeAttrs) pavedPenalty() float64 {
	switch a.Surface {
	case domain.SurfacePaved:
		return 1
	case domain.SurfaceGravel:
		return 0.3
	case domain.SurfaceGround:
		return 0
	default:
		// Revêtement non renseigné : on suppose un intermédiaire plutôt que
		// d'écarter le tronçon ou de le privilégier à tort.
		return 0.5
	}
}

func (a EdgeAttrs) trafficPenalty() float64 { return float64(a.Traffic) / 255 }
```

- [ ] **Step 6: Lancer, vérifier le succès**

Run: `go test ./... -v`
Expected: PASS — l'ensemble des tests des Tasks 1 à 3.

- [ ] **Step 7: Commit**

```bash
go vet ./... && go test ./...
git add -A
git commit -m "feat: profil, pondérations et fonction de coût"
```

---

## Task 4 : A*, index spatial et port réseau

**Files:**
- Modify: `internal/domain/track.go` (ajouter `PathOptions`)
- Create: `internal/app/port/network.go`
- Create: `internal/adapter/network/csr/astar.go`, `internal/adapter/network/csr/astar_test.go`
- Create: `internal/adapter/network/csr/nearest.go`
- Modify: `internal/adapter/network/csr/graph.go` (construire l'index spatial dans `Build`)

**Interfaces:**
- Consumes: `csr.Graph` (Task 2), `EdgeAttrs.Cost` (Task 3).
- Produces:
  - `domain.PathOptions{UsedEdges map[EdgeRef]struct{}; ReuseFactor float64; MaxNodes int}`
  - `port.RouteNetwork` : `NearestNode(domain.Coord) (domain.NodeRef, bool)`, `FindPath(context.Context, from, to domain.NodeRef, domain.Weights, domain.PathOptions) (domain.Path, error)`, `Coord(domain.NodeRef) domain.Coord`, `BBox() domain.BBox`
  - `csr.ErrNoPath`, `csr.ErrBudgetExceeded`

Note d'architecture : `csr` n'importe **pas** `app/port`. `PathOptions` vit dans `domain` — c'est un concept métier (les contraintes de recherche), pas une préoccupation d'infrastructure. Le paquet `csr` ignore l'existence du port ; c'est `cmd/routed` qui constate la satisfaction par un `var _ port.RouteNetwork = (*csr.Graph)(nil)`. C'est l'idiome Go « définir l'interface chez le consommateur », qui coïncide avec l'inversion de dépendance.

- [ ] **Step 1: Écrire les tests de l'A***

`internal/adapter/network/csr/astar_test.go`. Le dernier test est le plus important : il compare A* à un Dijkstra naïf sur des graphes tirés au sort, ce qui vérifie l'optimalité sans dépendre d'un cas particulier.

```go
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
```

- [ ] **Step 2: Lancer, vérifier l'échec**

Run: `go test ./internal/adapter/network/csr/`
Expected: FAIL — `g.FindPath undefined`

- [ ] **Step 3: Ajouter PathOptions au domaine**

Ajouter à `internal/domain/track.go` :

```go
// PathOptions porte les contraintes de recherche d'un chemin.
type PathOptions struct {
	// UsedEdges liste les arêtes déjà consommées par les segments précédents
	// d'une boucle. Leur coût est multiplié par ReuseFactor : c'est ce qui
	// évite qu'une boucle revienne sur ses pas, tout en laissant l'A* les
	// réemprunter quand il n'existe réellement pas d'alternative — un pont,
	// un col.
	UsedEdges   map[EdgeRef]struct{}
	ReuseFactor float64

	// MaxNodes plafonne l'exploration. Au-delà, la recherche est abandonnée
	// plutôt que de monopoliser le serveur. Zéro = valeur par défaut.
	MaxNodes int
}
```

Ajouter également à `Path`, dans le même fichier :

```go
	// ExploredNodes : nœuds dépilés par la recherche. Donnée d'observabilité,
	// et non résultat métier — elle rend visible l'arbitrage du modèle de
	// coût : plus les pondérations s'écartent, moins l'heuristique informe, et
	// plus A* dérive vers Dijkstra.
	ExploredNodes int
```

- [ ] **Step 4: Implémenter l'A***

`internal/adapter/network/csr/astar.go` :

```go
package csr

import (
	"container/heap"
	"context"
	"errors"
	"math"

	"github.com/im-sellar/hent/internal/domain"
)

var (
	ErrNoPath         = errors.New("aucun chemin trouvé")
	ErrBudgetExceeded = errors.New("plafond de nœuds explorés atteint")
)

const (
	defaultMaxNodes = 400_000
	noEdge          = domain.EdgeRef(math.MaxUint32)
	// Fréquence de vérification de l'annulation du contexte, en nœuds.
	ctxCheckInterval = 1024
)

type queueItem struct {
	node  domain.NodeRef
	f     float64
	index int
}

type priorityQueue []*queueItem

func (pq priorityQueue) Len() int           { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool { return pq[i].f < pq[j].f }

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index, pq[j].index = i, j
}

func (pq *priorityQueue) Push(x any) {
	item := x.(*queueItem)
	item.index = len(*pq)
	*pq = append(*pq, item)
}

func (pq *priorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*pq = old[:n-1]
	return item
}

// FindPath applique A* : on explore en priorité le nœud minimisant
// f(n) = g(n) + h(n), où h est la distance à vol d'oiseau jusqu'à la cible.
//
// h est admissible — elle ne surestime jamais — sous deux hypothèses, et non
// une seule :
//
//  1. Le coût minimal par mètre vaut exactement 1 (voir l'invariant sur
//     EdgeAttrs.Cost). C'est ce que vérifie TestFindPathEstOptimal.
//  2. La longueur d'une arête n'est jamais inférieure à la distance à vol
//     d'oiseau entre ses extrémités. Rien dans EdgeAttrs ne le contraint :
//     l'hypothèse tient parce que LengthM est calculée par HaversineM sur la
//     géométrie du tronçon, donc toujours supérieure ou égale à la corde. Une
//     source de données future qui importerait des longueurs pré-calculées
//     devrait la préserver, sous peine de rendre l'A* faux sans autre signal.
func (g *Graph) FindPath(
	ctx context.Context,
	from, to domain.NodeRef,
	w domain.Weights,
	opt domain.PathOptions,
) (domain.Path, error) {
	if int(from) >= len(g.coords) || int(to) >= len(g.coords) {
		return domain.Path{}, ErrNoPath
	}
	if err := ctx.Err(); err != nil {
		return domain.Path{}, err
	}
	if from == to {
		return domain.Path{Nodes: []domain.NodeRef{from}, Coords: []domain.Coord{g.coords[from]}}, nil
	}

	maxNodes := opt.MaxNodes
	if maxNodes <= 0 {
		maxNodes = defaultMaxNodes
	}
	reuse := opt.ReuseFactor
	if reuse < 1 {
		reuse = 1
	}

	target := g.coords[to]

	gScore := make([]float64, len(g.coords))
	parentNode := make([]domain.NodeRef, len(g.coords))
	parentEdge := make([]domain.EdgeRef, len(g.coords))
	settled := make([]bool, len(g.coords))
	for i := range gScore {
		gScore[i] = math.MaxFloat64
		parentEdge[i] = noEdge
	}
	gScore[from] = 0

	pq := &priorityQueue{}
	heap.Push(pq, &queueItem{node: from, f: domain.HaversineM(g.coords[from], target)})

	explored := 0
	for pq.Len() > 0 {
		if explored%ctxCheckInterval == 0 {
			if err := ctx.Err(); err != nil {
				return domain.Path{}, err
			}
		}

		cur := heap.Pop(pq).(*queueItem)
		if settled[cur.node] {
			continue
		}
		settled[cur.node] = true

		explored++
		if explored > maxNodes {
			return domain.Path{}, ErrBudgetExceeded
		}

		if cur.node == to {
			p := g.rebuild(from, to, parentNode, parentEdge, gScore[to])
			p.ExploredNodes = explored
			return p, nil
		}

		start, end := g.EdgeRange(cur.node)
		for e := start; e < end; e++ {
			next := g.targets[e]
			if settled[next] {
				continue
			}

			cost := g.attrs[e].Cost(w)
			if _, used := opt.UsedEdges[e]; used {
				cost *= reuse
			}

			if cand := gScore[cur.node] + cost; cand < gScore[next] {
				gScore[next] = cand
				parentNode[next] = cur.node
				parentEdge[next] = e
				heap.Push(pq, &queueItem{
					node: next,
					f:    cand + domain.HaversineM(g.coords[next], target),
				})
			}
		}
	}

	return domain.Path{}, ErrNoPath
}

func (g *Graph) rebuild(
	from, to domain.NodeRef,
	parentNode []domain.NodeRef,
	parentEdge []domain.EdgeRef,
	cost float64,
) domain.Path {
	var revNodes []domain.NodeRef
	var revEdges []domain.EdgeRef

	for n := to; n != from; n = parentNode[n] {
		revNodes = append(revNodes, n)
		revEdges = append(revEdges, parentEdge[n])
	}
	revNodes = append(revNodes, from)

	p := domain.Path{
		Nodes:  make([]domain.NodeRef, len(revNodes)),
		Coords: make([]domain.Coord, len(revNodes)),
		Edges:  make([]domain.EdgeRef, len(revEdges)),
		Cost:   cost,
	}
	for i, n := range revNodes {
		j := len(revNodes) - 1 - i
		p.Nodes[j] = n
		p.Coords[j] = g.coords[n]
	}
	for i, e := range revEdges {
		p.Edges[len(revEdges)-1-i] = e
		p.LengthM += g.attrs[e].LengthM
	}
	return p
}
```

- [ ] **Step 5: Lancer, vérifier le succès**

Run: `go test ./internal/adapter/network/csr/ -v`
Expected: PASS — en particulier `TestFindPathEstOptimal`, qui compare 50 graphes aléatoires à un Dijkstra de référence.

- [ ] **Step 6: Implémenter l'index spatial**

`internal/adapter/network/csr/nearest.go` :

```go
package csr

import (
	"math"

	"github.com/im-sellar/hent/internal/domain"
)

// Côté d'une cellule de la grille, en degrés de latitude — environ 550 m.
const cellSizeDeg = 0.005

type cellKey struct{ x, y int32 }

type spatialIndex struct {
	cells map[cellKey][]domain.NodeRef
}

func buildSpatialIndex(coords []domain.Coord) *spatialIndex {
	idx := &spatialIndex{cells: make(map[cellKey][]domain.NodeRef, len(coords)/8+1)}
	for i, c := range coords {
		k := keyOf(c)
		idx.cells[k] = append(idx.cells[k], domain.NodeRef(i))
	}
	return idx
}

func keyOf(c domain.Coord) cellKey {
	return cellKey{
		x: int32(math.Floor(c.Lon / cellSizeDeg)),
		y: int32(math.Floor(c.Lat / cellSizeDeg)),
	}
}

// NearestNode retourne le nœud le plus proche de c. La recherche part de la
// cellule contenant c et élargit l'anneau tant que rien n'est trouvé, jusqu'à
// une limite au-delà de laquelle on considère le point hors zone couverte.
func (g *Graph) NearestNode(c domain.Coord) (domain.NodeRef, bool) {
	const maxRings = 20 // ≈ 11 km

	origin := keyOf(c)

	// Le meilleur candidat s'accumule à travers les anneaux : le déclarer
	// dans la boucle perdrait les touches des anneaux précédents.
	best, bestDist := domain.NodeRef(0), math.MaxFloat64
	firstHit := int32(-1)

	for ring := int32(0); ring <= maxRings; ring++ {
		// Un nœud d'un anneau plus large peut être plus proche que celui déjà
		// trouvé — les cellules sont carrées, pas circulaires. On explore donc
		// un anneau de plus que celui de la première touche avant de conclure.
		if firstHit >= 0 && ring > firstHit+1 {
			break
		}

		for dx := -ring; dx <= ring; dx++ {
			for dy := -ring; dy <= ring; dy++ {
				// N'examiner que le bord de l'anneau : l'intérieur a déjà
				// été balayé aux itérations précédentes.
				if ring > 0 && abs32(dx) != ring && abs32(dy) != ring {
					continue
				}
				for _, n := range g.spatial.cells[cellKey{x: origin.x + dx, y: origin.y + dy}] {
					if d := domain.HaversineM(c, g.coords[n]); d < bestDist {
						best, bestDist = n, d
						if firstHit < 0 {
							firstHit = ring
						}
					}
				}
			}
		}
	}

	return best, firstHit >= 0
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}
```

Ajouter le champ `spatial *spatialIndex` à `Graph` dans `graph.go`, et sa construction en fin de `Build` :

```go
g.spatial = buildSpatialIndex(g.coords)
```

- [ ] **Step 7: Tester l'index spatial**

Ajouter à `astar_test.go` :

```go
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

	// Un leurre très éloigné, hors de portée de la recherche, occupe
	// l'indice 0. Sans lui, le nœud attendu vaudrait lui-même 0 et
	// l'assertion d'identité ne pourrait jamais échouer : elle passerait
	// même si la fonction renvoyait sa valeur par défaut.
	b.AddNode(domain.Coord{Lat: 48.6493, Lon: -2.0257})
	seul := b.AddNode(domain.Coord{Lat: 48.1000, Lon: -1.6800})
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
```

Run: `go test ./internal/adapter/network/csr/ -run TestNearest -v`
Expected: PASS

- [ ] **Step 8: Écrire le benchmark de l'A\***

C'est le garde-fou contre l'arbitrage du §6 de la spec. Le jour où une pondération est relevée et où l'exploration double, ce benchmark le dit — au lieu de laisser découvrir un délai dépassé en production.

Ajouter à `internal/adapter/network/csr/astar_test.go` :

```go
func BenchmarkFindPath(b *testing.B) {
	// Un graphe assez grand pour que l'heuristique compte réellement.
	g, n := grapheAleatoire(rand.New(rand.NewSource(42)), 4000)

	// Deux pondérations, pour rendre l'arbitrage visible : plus l'écart entre
	// le meilleur et le pire terrain se creuse, moins l'heuristique informe.
	cas := map[string]domain.Weights{
		"neutre": {},
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
```

Run: `go test ./internal/adapter/network/csr/ -run '^$' -bench BenchmarkFindPath -benchmem`

Attendu : deux lignes de résultat. Noter la valeur `nœuds/chemin` du cas `anti-bitume` — c'est la référence à surveiller. Si elle dépasse nettement celle du cas `neutre` (facteur 2 ou plus), c'est le signe que les pondérations maximales sont trop écartées et qu'elles coûteront cher en production.

- [ ] **Step 9: Déclarer le port**

`internal/app/port/network.go` :

```go
// Package port déclare les interfaces dont la couche applicative a besoin.
// Elles sont définies ici, chez le consommateur, et non chez l'implémenteur :
// c'est l'idiome Go, et c'est aussi l'inversion de dépendance.
package port

import (
	"context"

	"github.com/im-sellar/hent/internal/domain"
)

// RouteNetwork est volontairement à granularité grossière. Elle expose
// FindPath et non Neighbors : construire une boucle demande trois ou quatre
// appels, là où exposer le voisinage en ferait des millions — un coût
// d'abstraction que la boucle chaude de l'A* ne peut pas absorber.
type RouteNetwork interface {
	NearestNode(c domain.Coord) (domain.NodeRef, bool)
	FindPath(ctx context.Context, from, to domain.NodeRef,
		w domain.Weights, opt domain.PathOptions) (domain.Path, error)
	Coord(n domain.NodeRef) domain.Coord
	BBox() domain.BBox
}
```

- [ ] **Step 10: Commit**

```bash
go vet ./... && go test ./...
git add -A
git commit -m "feat: A*, index spatial, benchmark et port réseau"
```

---

## Task 5 : Lecture OSM et construction du graphe

**Files:**
- Create: `internal/adapter/osmsource/tags.go`, `internal/adapter/osmsource/tags_test.go`
- Create: `internal/adapter/osmsource/read.go`, `internal/adapter/osmsource/read_test.go`
- Create: `testdata/rennes-centre.osm.pbf` (extrait de quelques Mo, commité)
- Modify: `go.mod` (ajout de `github.com/paulmach/osm`)

**Interfaces:**
- Consumes: `csr.Builder`, `csr.EdgeAttrs` (Tasks 2-3).
- Produces:
  - `osmsource.Classify(tags map[string]string) (domain.WayClass, domain.Surface, uint8, bool)` — le booléen indique si le tronçon est retenu.
  - `osmsource.Read(ctx context.Context, path string) (*csr.Graph, Stats, error)`
  - `osmsource.Stats{Ways, Nodes, Edges int}`

- [ ] **Step 1: Préparer les données de test**

```bash
cd /Users/amorice/DevHome/02_Perso/hent
brew install osmium-tool
mkdir -p testdata data
# -L est indispensable : Geofabrik redirige « latest » vers le fichier daté du jour.
curl -L -o data/bretagne-latest.osm.pbf https://download.geofabrik.de/europe/france/bretagne-latest.osm.pbf
# Extrait resserré sur le centre de Rennes : 3,4 Mo, ~35 000 tronçons. La bbox
# est volontairement juste assez large pour couvrir les deux points du test de
# routage — un fichier binaire versionné reste dans l'historique pour toujours,
# et la suite le reparse à chaque exécution.
osmium extract --bbox -1.70,48.10,-1.655,48.13 data/bretagne-latest.osm.pbf -o testdata/rennes-centre.osm.pbf
ls -lh testdata/rennes-centre.osm.pbf
```

`data/` est ignoré par git (fichiers volumineux), `testdata/` est commité.

```bash
printf 'data/\n' >> .gitignore
```

- [ ] **Step 2: Écrire le test de classification des tags**

`internal/adapter/osmsource/tags_test.go` :

```go
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
```

- [ ] **Step 3: Lancer, vérifier l'échec**

Run: `go test ./internal/adapter/osmsource/`
Expected: FAIL — `undefined: osmsource.Classify`

- [ ] **Step 4: Implémenter la classification**

`internal/adapter/osmsource/tags.go` :

```go
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
	"path":         {domain.WayPath, 0},
	"bridleway":    {domain.WayPath, 0},
	"track":        {domain.WayTrack, 0},
	"footway":      {domain.WayFootway, 0},
	"pedestrian":   {domain.WayFootway, 10},
	"cycleway":     {domain.WayFootway, 10},
	"living_street": {domain.WayResidential, 40},
	"service":      {domain.WayResidential, 60},
	"residential":  {domain.WayResidential, 90},
	"unclassified": {domain.WayResidential, 120},
	"tertiary":     {domain.WayTertiary, 180},
	"secondary":    {domain.WaySecondary, 255},
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
```

- [ ] **Step 5: Lancer, vérifier le succès**

Run: `go test ./internal/adapter/osmsource/ -v`
Expected: PASS (3 tests, dont 12 sous-cas)

- [ ] **Step 6: Écrire le test de lecture du PBF**

`internal/adapter/osmsource/read_test.go` :

```go
package osmsource_test

import (
	"context"
	"testing"

	"github.com/im-sellar/hent/internal/adapter/osmsource"
	"github.com/im-sellar/hent/internal/domain"
)

func TestReadExtraitRennes(t *testing.T) {
	g, stats, err := osmsource.Read(context.Background(), "../../../testdata/rennes-centre.osm.pbf")
	if err != nil {
		t.Fatalf("Read : %v", err)
	}

	if stats.Ways < 1000 {
		t.Errorf("%d ways retenus, attendu au moins 1000 sur le centre de Rennes", stats.Ways)
	}
	if g.NumNodes() < 1000 {
		t.Errorf("%d nœuds, attendu au moins 1000", g.NumNodes())
	}
	// Chaque tronçon produit deux arêtes (un sens chacune).
	if g.NumEdges() != 2*stats.Edges {
		t.Errorf("%d arêtes pour %d tronçons, attendu le double", g.NumEdges(), stats.Edges)
	}

	// La bbox doit couvrir le centre de Rennes.
	if !g.BBox().Contains(domain.Coord{Lat: 48.1113, Lon: -1.6800}) {
		t.Error("la bbox ne contient pas la place de la République")
	}
}

func TestReadEtRoute(t *testing.T) {
	// Test d'intégration : sur de vraies données, deux points distants d'un
	// kilomètre doivent être reliés.
	g, _, err := osmsource.Read(context.Background(), "../../../testdata/rennes-centre.osm.pbf")
	if err != nil {
		t.Fatal(err)
	}

	from, ok1 := g.NearestNode(domain.Coord{Lat: 48.1113, Lon: -1.6800})
	to, ok2 := g.NearestNode(domain.Coord{Lat: 48.1200, Lon: -1.6700})
	if !ok1 || !ok2 {
		t.Fatal("points de départ ou d'arrivée introuvables dans le graphe")
	}

	p, err := g.FindPath(context.Background(), from, to,
		domain.Preferences{AvoidPaved: 0.5}.Weights(), domain.PathOptions{})
	if err != nil {
		t.Fatalf("aucun itinéraire entre deux points du centre de Rennes : %v", err)
	}
	if p.LengthM < 500 || p.LengthM > 5000 {
		t.Errorf("itinéraire de %.0f m, attendu entre 500 et 5000", p.LengthM)
	}
}
```

- [ ] **Step 7: Implémenter la lecture**

```bash
go get github.com/paulmach/osm@v0.9.0
```

`internal/adapter/osmsource/read.go` :

```go
package osmsource

import (
	"context"
	"fmt"
	"os"

	"github.com/paulmach/osm"
	"github.com/paulmach/osm/osmpbf"

	"github.com/im-sellar/hent/internal/adapter/network/csr"
	"github.com/im-sellar/hent/internal/domain"
)

type Stats struct {
	Ways  int // tronçons OSM retenus
	Nodes int // nœuds conservés
	Edges int // segments (chaque segment donne deux arêtes dirigées)
}

type retainedWay struct {
	nodes   []osm.NodeID
	class   domain.WayClass
	surface domain.Surface
	traffic uint8
}

// Read lit un extrait PBF et construit le graphe.
//
// Deux passes sont nécessaires : le format PBF ne garantit pas que les nœuds
// précèdent les ways qui les référencent, et on ne veut mémoriser les
// coordonnées que des nœuds réellement utilisés. La première passe collecte
// les ways praticables, la seconde les coordonnées dont ils ont besoin.
func Read(ctx context.Context, path string) (*csr.Graph, Stats, error) {
	ways, needed, err := scanWays(ctx, path)
	if err != nil {
		return nil, Stats{}, err
	}

	coords, err := scanNodes(ctx, path, needed)
	if err != nil {
		return nil, Stats{}, err
	}

	return assemble(ways, coords)
}

func scanWays(ctx context.Context, path string) ([]retainedWay, map[osm.NodeID]struct{}, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("ouverture de %s : %w", path, err)
	}
	defer f.Close()

	scanner := osmpbf.New(ctx, f, 4)
	defer scanner.Close()
	scanner.SkipNodes = true
	scanner.SkipRelations = true

	var ways []retainedWay
	needed := make(map[osm.NodeID]struct{})

	for scanner.Scan() {
		w, ok := scanner.Object().(*osm.Way)
		if !ok || len(w.Nodes) < 2 {
			continue
		}

		class, surface, traffic, keep := Classify(w.TagMap())
		if !keep {
			continue
		}

		ids := make([]osm.NodeID, len(w.Nodes))
		for i, n := range w.Nodes {
			ids[i] = n.ID
			needed[n.ID] = struct{}{}
		}
		ways = append(ways, retainedWay{
			nodes: ids, class: class, surface: surface, traffic: traffic,
		})
	}
	return ways, needed, scanner.Err()
}

func scanNodes(ctx context.Context, path string, needed map[osm.NodeID]struct{}) (map[osm.NodeID]domain.Coord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("ouverture de %s : %w", path, err)
	}
	defer f.Close()

	scanner := osmpbf.New(ctx, f, 4)
	defer scanner.Close()
	scanner.SkipWays = true
	scanner.SkipRelations = true

	coords := make(map[osm.NodeID]domain.Coord, len(needed))
	for scanner.Scan() {
		n, ok := scanner.Object().(*osm.Node)
		if !ok {
			continue
		}
		if _, want := needed[n.ID]; want {
			coords[n.ID] = domain.Coord{Lat: n.Lat, Lon: n.Lon}
		}
	}
	return coords, scanner.Err()
}

func assemble(ways []retainedWay, coords map[osm.NodeID]domain.Coord) (*csr.Graph, Stats, error) {
	b := csr.NewBuilder()
	refs := make(map[osm.NodeID]domain.NodeRef, len(coords))

	nodeRef := func(id osm.NodeID) (domain.NodeRef, bool) {
		if r, ok := refs[id]; ok {
			return r, true
		}
		c, ok := coords[id]
		if !ok {
			// Nœud hors de l'extrait : le tronçon est coupé au bord de la
			// bbox, on saute simplement ce segment.
			return 0, false
		}
		r := b.AddNode(c)
		refs[id] = r
		return r, true
	}

	var stats Stats
	for _, w := range ways {
		stats.Ways++
		for i := 0; i+1 < len(w.nodes); i++ {
			from, ok1 := nodeRef(w.nodes[i])
			to, ok2 := nodeRef(w.nodes[i+1])
			if !ok1 || !ok2 || from == to {
				continue
			}

			attrs := csr.EdgeAttrs{
				LengthM: domain.HaversineM(b.Coord(from), b.Coord(to)),
				Surface: w.surface,
				Class:   w.class,
				Traffic: w.traffic,
			}
			b.AddEdge(from, to, attrs)
			b.AddEdge(to, from, attrs)
			stats.Edges++
		}
	}
	stats.Nodes = len(refs)

	if stats.Edges == 0 {
		return nil, stats, fmt.Errorf("aucun tronçon praticable trouvé dans l'extrait")
	}
	return b.Build(), stats, nil
}
```

- [ ] **Step 8: Lancer, vérifier le succès**

Run: `go test ./internal/adapter/osmsource/ -v`
Expected: PASS — `TestReadEtRoute` calcule un vrai itinéraire dans Rennes.

- [ ] **Step 9: Commit**

```bash
go vet ./... && go test ./...
git add -A
git commit -m "feat: lecture d'un extrait OSM et construction du graphe"
```

---

## Task 6 : Sérialisation de l'artefact et binaire `graphbuild`

L'exigence de reproductibilité (§9 de la spec) n'est pas cosmétique : l'ODbL déclenche le partage à l'identique dès l'usage public, et on s'en acquitte en fournissant « les moyens de reconstruire » la base dérivée. Sans provenance dans l'en-tête, cet engagement est une fiction.

**Files:**
- Create: `internal/adapter/network/csr/codec.go`, `internal/adapter/network/csr/codec_test.go`
- Create: `cmd/graphbuild/main.go`

**Interfaces:**
- Consumes: `csr.Graph` (Task 2), `osmsource.Read` (Task 5).
- Produces:
  - `csr.Provenance{BuiltAt string; Sources []Source; ConfigHash string}`, `csr.Source{Name, File, SHA256 string; SizeBytes int64}`
  - `csr.Write(w io.Writer, g *Graph, p Provenance) error`
  - `csr.ReadGraph(r io.Reader) (*Graph, Provenance, error)`
  - `csr.ErrBadMagic`, `csr.ErrBadVersion`

- [ ] **Step 1: Écrire le test d'aller-retour**

`internal/adapter/network/csr/codec_test.go` :

```go
package csr_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/im-sellar/hent/internal/adapter/network/csr"
	"github.com/im-sellar/hent/internal/domain"
)

func TestCodecAllerRetour(t *testing.T) {
	g := carre(t)
	prov := csr.Provenance{
		BuiltAt:    "2026-08-18T10:00:00Z",
		ConfigHash: "abc123",
		Sources: []csr.Source{{
			Name: "geofabrik/bretagne", File: "bretagne-latest.osm.pbf",
			SHA256: "deadbeef", SizeBytes: 12345,
		}},
	}

	var buf bytes.Buffer
	if err := csr.Write(&buf, g, prov); err != nil {
		t.Fatalf("Write : %v", err)
	}

	got, gotProv, err := csr.ReadGraph(&buf)
	if err != nil {
		t.Fatalf("ReadGraph : %v", err)
	}

	if got.NumNodes() != g.NumNodes() || got.NumEdges() != g.NumEdges() {
		t.Fatalf("relu %d nœuds / %d arêtes, écrit %d / %d",
			got.NumNodes(), got.NumEdges(), g.NumNodes(), g.NumEdges())
	}
	if gotProv.Sources[0].SHA256 != "deadbeef" {
		t.Errorf("provenance perdue : %+v", gotProv)
	}

	// Les attributs et la topologie doivent survivre au tour.
	for n := domain.NodeRef(0); n < domain.NodeRef(g.NumNodes()); n++ {
		s1, e1 := g.EdgeRange(n)
		s2, e2 := got.EdgeRange(n)
		if s1 != s2 || e1 != e2 {
			t.Fatalf("nœud %d : intervalle [%d,%d) relu [%d,%d)", n, s1, e1, s2, e2)
		}
		for e := s1; e < e1; e++ {
			if g.Target(e) != got.Target(e) {
				t.Fatalf("arête %d : cible %d relue %d", e, g.Target(e), got.Target(e))
			}
			a, b := g.Attrs(e), got.Attrs(e)
			if a.Surface != b.Surface || a.Class != b.Class || a.Traffic != b.Traffic {
				t.Fatalf("arête %d : attributs %+v relus %+v", e, a, b)
			}
		}
	}

	// L'index spatial doit être reconstruit à la lecture.
	if _, ok := got.NearestNode(domain.Coord{Lat: 48.1001, Lon: -1.6801}); !ok {
		t.Error("index spatial non reconstruit après lecture")
	}
}

func TestCodecRefuseUnMauvaisFichier(t *testing.T) {
	_, _, err := csr.ReadGraph(bytes.NewReader([]byte("ce n'est pas un graphe")))
	if !errors.Is(err, csr.ErrBadMagic) {
		t.Fatalf("erreur = %v, attendu ErrBadMagic", err)
	}
}
```

- [ ] **Step 2: Lancer, vérifier l'échec**

Run: `go test ./internal/adapter/network/csr/ -run TestCodec`
Expected: FAIL — `undefined: csr.Write`

- [ ] **Step 3: Implémenter le codec**

`internal/adapter/network/csr/codec.go` :

```go
package csr

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/im-sellar/hent/internal/domain"
)

var (
	magic          = [4]byte{'H', 'E', 'N', 'T'}
	ErrBadMagic    = errors.New("ce fichier n'est pas un graphe hent")
	ErrBadVersion  = errors.New("version de format non prise en charge")
)

// formatVersion doit être incrémentée à chaque changement de disposition
// binaire. routed refuse de démarrer sur une version inconnue plutôt que de
// lire des octets de travers.
const formatVersion uint16 = 1

// Source décrit une donnée d'entrée avec de quoi la retrouver à l'identique.
type Source struct {
	Name      string `json:"name"`
	File      string `json:"file"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

// Provenance rend le build reproductible — exigence de l'ODbL, qui impose de
// pouvoir fournir soit la base dérivée, soit les moyens de la reconstruire.
type Provenance struct {
	BuiltAt    string   `json:"built_at"`
	Sources    []Source `json:"sources"`
	ConfigHash string   `json:"config_hash"`
}

var order = binary.LittleEndian

// Plafonds appliqués aux tailles lues dans l'en-tête avant toute allocation.
// Un artefact tronqué — graphbuild interrompu, disque plein, transfert coupé —
// ou simplement corrompu porte des compteurs arbitraires ; les allouer tels
// quels ferait réclamer plusieurs gigaoctets au démarrage du serveur. Les
// valeurs retenues laissent une marge considérable : la France entière compte
// environ dix fois moins de nœuds que le plafond.
const (
	maxHeaderLen = 1 << 20
	maxNodes     = 200_000_000
	maxEdges     = 800_000_000
)

func Write(w io.Writer, g *Graph, p Provenance) error {
	header, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("sérialisation de la provenance : %w", err)
	}

	if _, err := w.Write(magic[:]); err != nil {
		return err
	}
	for _, v := range []any{
		formatVersion,
		uint32(len(header)),
	} {
		if err := binary.Write(w, order, v); err != nil {
			return err
		}
	}
	if _, err := w.Write(header); err != nil {
		return err
	}

	if err := binary.Write(w, order, uint32(len(g.coords))); err != nil {
		return err
	}
	for _, c := range g.coords {
		if err := binary.Write(w, order, c.Lat); err != nil {
			return err
		}
		if err := binary.Write(w, order, c.Lon); err != nil {
			return err
		}
	}
	if err := binary.Write(w, order, g.offsets); err != nil {
		return err
	}

	if err := binary.Write(w, order, uint32(len(g.targets))); err != nil {
		return err
	}
	if err := binary.Write(w, order, g.targets); err != nil {
		return err
	}
	for _, a := range g.attrs {
		if err := binary.Write(w, order, float32(a.LengthM)); err != nil {
			return err
		}
		if err := binary.Write(w, order, [3]uint8{
			uint8(a.Surface), uint8(a.Class), a.Traffic,
		}); err != nil {
			return err
		}
	}
	return nil
}

func ReadGraph(r io.Reader) (*Graph, Provenance, error) {
	var gotMagic [4]byte
	if _, err := io.ReadFull(r, gotMagic[:]); err != nil {
		return nil, Provenance{}, ErrBadMagic
	}
	if gotMagic != magic {
		return nil, Provenance{}, ErrBadMagic
	}

	var version uint16
	if err := binary.Read(r, order, &version); err != nil {
		return nil, Provenance{}, err
	}
	if version != formatVersion {
		return nil, Provenance{}, fmt.Errorf("%w : fichier en version %d, binaire en version %d",
			ErrBadVersion, version, formatVersion)
	}

	var headerLen uint32
	if err := binary.Read(r, order, &headerLen); err != nil {
		return nil, Provenance{}, err
	}
	header := make([]byte, headerLen)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, Provenance{}, err
	}
	var prov Provenance
	if err := json.Unmarshal(header, &prov); err != nil {
		return nil, Provenance{}, fmt.Errorf("provenance illisible : %w", err)
	}

	var numNodes uint32
	if err := binary.Read(r, order, &numNodes); err != nil {
		return nil, prov, err
	}
	g := &Graph{
		coords:  make([]domain.Coord, numNodes),
		offsets: make([]uint32, numNodes+1),
	}
	for i := range g.coords {
		if err := binary.Read(r, order, &g.coords[i].Lat); err != nil {
			return nil, prov, err
		}
		if err := binary.Read(r, order, &g.coords[i].Lon); err != nil {
			return nil, prov, err
		}
	}
	if err := binary.Read(r, order, g.offsets); err != nil {
		return nil, prov, err
	}

	var numEdges uint32
	if err := binary.Read(r, order, &numEdges); err != nil {
		return nil, prov, err
	}
	g.targets = make([]domain.NodeRef, numEdges)
	if err := binary.Read(r, order, g.targets); err != nil {
		return nil, prov, err
	}
	g.attrs = make([]EdgeAttrs, numEdges)
	for i := range g.attrs {
		var length float32
		var packed [3]uint8
		if err := binary.Read(r, order, &length); err != nil {
			return nil, prov, err
		}
		if err := binary.Read(r, order, &packed); err != nil {
			return nil, prov, err
		}
		g.attrs[i] = EdgeAttrs{
			LengthM: float64(length),
			Surface: domain.Surface(packed[0]),
			Class:   domain.WayClass(packed[1]),
			Traffic: packed[2],
		}
	}

	g.bbox = computeBBox(g.coords)
	g.spatial = buildSpatialIndex(g.coords)
	return g, prov, nil
}
```

- [ ] **Step 4: Lancer, vérifier le succès**

Run: `go test ./internal/adapter/network/csr/ -v`
Expected: PASS

- [ ] **Step 5: Écrire le binaire graphbuild**

`cmd/graphbuild/main.go` :

```go
// Command graphbuild construit l'artefact graph.bin à partir d'un extrait
// OpenStreetMap. Il tourne hors ligne, sur le poste de développement : toutes
// les opérations coûteuses se paient ici, une fois, et jamais en production.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/im-sellar/hent/internal/adapter/network/csr"
	"github.com/im-sellar/hent/internal/adapter/osmsource"
)

func main() {
	in := flag.String("in", "", "extrait OpenStreetMap au format .osm.pbf")
	out := flag.String("out", "graph.bin", "artefact à produire")
	name := flag.String("source-name", "geofabrik/bretagne", "nom de la source, inscrit dans la provenance")
	flag.Parse()

	if *in == "" {
		log.Fatal("le drapeau -in est obligatoire")
	}
	if err := run(*in, *out, *name); err != nil {
		log.Fatal(err)
	}
}

func run(in, out, sourceName string) error {
	start := time.Now()

	log.Printf("lecture de %s", in)
	g, stats, err := osmsource.Read(context.Background(), in)
	if err != nil {
		return fmt.Errorf("lecture de l'extrait : %w", err)
	}
	log.Printf("%d tronçons retenus, %d nœuds, %d arêtes dirigées",
		stats.Ways, g.NumNodes(), g.NumEdges())

	digest, size, err := hashFile(in)
	if err != nil {
		return err
	}

	prov := csr.Provenance{
		BuiltAt: time.Now().UTC().Format(time.RFC3339),
		Sources: []csr.Source{{
			Name: sourceName, File: in, SHA256: digest, SizeBytes: size,
		}},
		ConfigHash: osmsource.ConfigHash(),
	}

	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := csr.Write(f, g, prov); err != nil {
		return fmt.Errorf("écriture de l'artefact : %w", err)
	}

	info, err := f.Stat()
	if err != nil {
		return err
	}
	log.Printf("%s écrit (%.1f Mo) en %s", out, float64(info.Size())/(1<<20), time.Since(start).Round(time.Second))
	return nil
}

func hashFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	h := sha256.New()
	size, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), size, nil
}

// Le hash de configuration est calculé par osmsource à partir du contenu réel
// de ses tables de classification : voir osmsource.ConfigHash.
```

- [ ] **Step 6: Construire l'artefact réel**

```bash
# CGO_ENABLED=0 : osmpbf tire czlib dans son graphe de dépendances, qui
# exige pkg-config et zlib quand cgo est actif. Le chemin pur Go fait le même
# travail, et c'est lui qui rend le binaire statique et cross-compilable.
CGO_ENABLED=0 go build -o graphbuild ./cmd/graphbuild
./graphbuild -in testdata/rennes-centre.osm.pbf -out /tmp/rennes.bin
./graphbuild -in data/bretagne-latest.osm.pbf -out graph.bin
ls -lh graph.bin
```

Attendu : quelques dizaines de secondes pour la Bretagne, et un artefact de l'ordre de la centaine de Mo. Si le temps dépasse plusieurs minutes ou si la mémoire sature, réduire d'abord l'extrait à l'Ille-et-Vilaine avec `osmium extract --bbox -2.0,47.9,-1.0,48.7`.

- [ ] **Step 7: Commit**

```bash
go vet ./... && go test ./...
git add -A
git commit -m "feat: artefact graph.bin versionné avec provenance, et binaire graphbuild"
```

---

## Task 7 : Génération de boucles

Le cœur du produit. Un algorithme de plus court chemin ne peut pas produire une boucle — le chemin de coût minimal d'un point à lui-même est de ne pas bouger. On lui impose donc une forme : trois waypoints sur un cercle, un rayon corrigé par dichotomie, et une pénalisation des arêtes déjà consommées.

**Files:**
- Modify: `internal/domain/geo.go` (ajouter `Offset`)
- Modify: `internal/domain/track.go` (ajouter `Loop`)
- Create: `internal/app/generateloop/generate.go`, `internal/app/generateloop/generate_test.go`
- Create: `internal/app/generateloop/testnetwork_test.go`

**Interfaces:**
- Consumes: `port.RouteNetwork` (Task 4).
- Produces:
  - `domain.Offset(c Coord, distM, bearingRad float64) Coord`
  - `domain.Loop{Nodes []NodeRef; Edges []EdgeRef; Coords []Coord; LengthM float64}`
  - `generateloop.Request{Start domain.Coord; DistanceM, Tolerance float64; Prefs domain.Preferences; MaxResults, Variant int}`
  - `generateloop.New(net port.RouteNetwork) *Generator`
  - `(*Generator).Generate(ctx context.Context, req Request) ([]domain.Loop, error)`
  - `generateloop.ErrStartOutOfRange`, `generateloop.ErrNoLoopFound`

- [ ] **Step 1: Écrire le réseau de test**

`internal/app/generateloop/testnetwork_test.go`. Une grille régulière suffit et évite toute dépendance à OSM : `app/` ne doit connaître ni PBF ni CSR.

```go
package generateloop_test

import (
	"context"
	"errors"
	"math"

	"github.com/im-sellar/hent/internal/domain"
)

// grille est un réseau synthétique de n×n nœuds espacés de `pas` mètres,
// relié en quatre-connexité. Il implémente port.RouteNetwork sans rien
// emprunter à l'adaptateur réel : la couche applicative se teste seule.
type grille struct {
	n      int
	pas    float64
	origin domain.Coord
	coords []domain.Coord
	// voisins[n] = arêtes sortantes, chacune (cible, longueur)
	voisins [][]arete
}

type arete struct {
	cible domain.NodeRef
	ref   domain.EdgeRef
	long  float64
}

var errPasDeChemin = errors.New("aucun chemin")

func nouvelleGrille(n int, pas float64) *grille {
	g := &grille{
		n: n, pas: pas,
		origin:  domain.Coord{Lat: 48.10, Lon: -1.68},
		coords:  make([]domain.Coord, n*n),
		voisins: make([][]arete, n*n),
	}
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			g.coords[y*n+x] = domain.Offset(
				domain.Offset(g.origin, float64(x)*pas, math.Pi/2),
				float64(y)*pas, 0)
		}
	}

	var ref domain.EdgeRef
	lier := func(a, b domain.NodeRef) {
		d := domain.HaversineM(g.coords[a], g.coords[b])
		g.voisins[a] = append(g.voisins[a], arete{cible: b, ref: ref, long: d})
		ref++
		g.voisins[b] = append(g.voisins[b], arete{cible: a, ref: ref, long: d})
		ref++
	}
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			i := domain.NodeRef(y*n + x)
			if x+1 < n {
				lier(i, i+1)
			}
			if y+1 < n {
				lier(i, i+domain.NodeRef(n))
			}
		}
	}
	return g
}

func (g *grille) Coord(n domain.NodeRef) domain.Coord { return g.coords[n] }

func (g *grille) BBox() domain.BBox {
	b := domain.BBox{Min: g.coords[0], Max: g.coords[0]}
	for _, c := range g.coords {
		b.Min.Lat, b.Min.Lon = math.Min(b.Min.Lat, c.Lat), math.Min(b.Min.Lon, c.Lon)
		b.Max.Lat, b.Max.Lon = math.Max(b.Max.Lat, c.Lat), math.Max(b.Max.Lon, c.Lon)
	}
	return b
}

func (g *grille) NearestNode(c domain.Coord) (domain.NodeRef, bool) {
	if !g.BBox().Contains(c) {
		return 0, false
	}
	best, bestD := domain.NodeRef(0), math.MaxFloat64
	for i, cc := range g.coords {
		if d := domain.HaversineM(c, cc); d < bestD {
			best, bestD = domain.NodeRef(i), d
		}
	}
	return best, true
}

// FindPath : Dijkstra simple, suffisant à l'échelle de la grille de test.
func (g *grille) FindPath(ctx context.Context, from, to domain.NodeRef,
	w domain.Weights, opt domain.PathOptions) (domain.Path, error) {

	if err := ctx.Err(); err != nil {
		return domain.Path{}, err
	}
	if from == to {
		return domain.Path{Nodes: []domain.NodeRef{from}, Coords: []domain.Coord{g.coords[from]}}, nil
	}

	reuse := opt.ReuseFactor
	if reuse < 1 {
		reuse = 1
	}

	const inf = math.MaxFloat64
	dist := make([]float64, len(g.coords))
	prevN := make([]domain.NodeRef, len(g.coords))
	prevE := make([]domain.EdgeRef, len(g.coords))
	done := make([]bool, len(g.coords))
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

		for _, a := range g.voisins[best] {
			cost := a.long
			if _, used := opt.UsedEdges[a.ref]; used {
				cost *= reuse
			}
			if cand := bestD + cost; cand < dist[a.cible] {
				dist[a.cible] = cand
				prevN[a.cible] = domain.NodeRef(best)
				prevE[a.cible] = a.ref
			}
		}
	}

	if dist[to] == inf {
		return domain.Path{}, errPasDeChemin
	}

	var revN []domain.NodeRef
	var revE []domain.EdgeRef
	for n := to; n != from; n = prevN[n] {
		revN = append(revN, n)
		revE = append(revE, prevE[n])
	}
	revN = append(revN, from)

	p := domain.Path{
		Nodes:  make([]domain.NodeRef, len(revN)),
		Coords: make([]domain.Coord, len(revN)),
		Edges:  make([]domain.EdgeRef, len(revE)),
		Cost:   dist[to],
	}
	for i, n := range revN {
		j := len(revN) - 1 - i
		p.Nodes[j], p.Coords[j] = n, g.coords[n]
	}
	for i, e := range revE {
		j := len(revE) - 1 - i
		p.Edges[j] = e
	}
	// Longueur réelle : somme des arêtes empruntées.
	for i := 0; i+1 < len(p.Nodes); i++ {
		for _, a := range g.voisins[p.Nodes[i]] {
			if a.cible == p.Nodes[i+1] {
				p.LengthM += a.long
				break
			}
		}
	}
	return p, nil
}
```

- [ ] **Step 2: Écrire les tests de propriété**

`internal/app/generateloop/generate_test.go`. Ce sont les quatre invariants de la spec (§11) — le filet de sécurité principal du projet.

```go
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
```

Ajouter le helper au réseau de test, dans `testnetwork_test.go` :

```go
func (g *grille) relies(a, b domain.NodeRef) bool {
	for _, v := range g.voisins[a] {
		if v.cible == b {
			return true
		}
	}
	return false
}
```

- [ ] **Step 3: Lancer, vérifier l'échec**

Run: `go test ./internal/app/generateloop/`
Expected: FAIL — `undefined: generateloop.New`

- [ ] **Step 4: Ajouter Offset et Loop au domaine**

Ajouter à `internal/domain/geo.go` :

```go
// mPerDegLat : longueur d'un degré de latitude, en mètres. Constante à la
// précision qui nous intéresse (le placement de waypoints tolère largement
// l'aplatissement terrestre).
const mPerDegLat = 111320.0

// Offset retourne le point situé à distM mètres de c dans la direction
// bearingRad, comptée en radians depuis le nord et dans le sens horaire.
func Offset(c Coord, distM, bearingRad float64) Coord {
	dLat := distM * math.Cos(bearingRad) / mPerDegLat
	dLon := distM * math.Sin(bearingRad) / (mPerDegLat * math.Cos(radians(c.Lat)))
	return Coord{Lat: c.Lat + dLat, Lon: c.Lon + dLon}
}
```

Ajouter à `internal/domain/track.go` :

```go
// Loop est une boucle fermée : le premier et le dernier nœud coïncident.
type Loop struct {
	Nodes   []NodeRef
	Edges   []EdgeRef
	Coords  []Coord
	LengthM float64
}
```

- [ ] **Step 5: Implémenter la génération**

`internal/app/generateloop/generate.go` :

```go
// Package generateloop porte la stratégie de génération de boucles. C'est du
// métier, pas de l'infrastructure : il ne connaît du réseau que le port
// RouteNetwork, et se teste sur une grille synthétique.
package generateloop

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"runtime"
	"sort"

	"golang.org/x/sync/errgroup"

	"github.com/im-sellar/hent/internal/app/port"
	"github.com/im-sellar/hent/internal/domain"
)

var (
	ErrStartOutOfRange = errors.New("le point de départ est hors de la zone couverte")
	ErrNoLoopFound     = errors.New("aucune boucle trouvée pour cette requête")
)

const (
	// Nombre de directions explorées. Chaque candidate coûte quatre appels de
	// plus court chemin ; elles tournent en parallèle.
	numCandidates = 20

	// Multiplicateur appliqué aux arêtes déjà consommées par les segments
	// précédents. Assez haut pour décourager le retour sur ses pas, assez bas
	// pour autoriser un passage obligé — un pont, un col.
	reuseFactor = 4.0

	// Nombre de waypoints intermédiaires. Trois, répartis à 120°, donnent une
	// vraie boucle ; deux dégénèrent en aller-retour.
	numWaypoints = 3

	// Itérations de la dichotomie sur le facteur de détour.
	maxRadiusIterations = 5

	detourLo, detourHi = 0.8, 2.5
)

type Request struct {
	Start      domain.Coord
	DistanceM  float64
	Tolerance  float64
	Prefs      domain.Preferences
	MaxResults int
	Variant    int
}

type Generator struct {
	net port.RouteNetwork
}

func New(net port.RouteNetwork) *Generator { return &Generator{net: net} }

func (g *Generator) Generate(ctx context.Context, req Request) ([]domain.Loop, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	req = req.withDefaults()

	start, ok := g.net.NearestNode(req.Start)
	if !ok {
		return nil, ErrStartOutOfRange
	}

	// Le hasard est dérivé de la requête : deux appels identiques donnent le
	// même résultat, ce qui rend l'API cachable et les GPX régénérables.
	rng := rand.New(rand.NewSource(seedOf(req)))
	angles := make([]float64, numCandidates)
	for i := range angles {
		angles[i] = float64(i)*2*math.Pi/numCandidates + rng.Float64()*0.3
	}

	results := make([]domain.Loop, numCandidates)
	found := make([]bool, numCandidates)

	group, gctx := errgroup.WithContext(ctx)
	group.SetLimit(runtime.NumCPU())
	for i, theta := range angles {
		i, theta := i, theta
		group.Go(func() error {
			loop, err := g.candidate(gctx, start, theta, req)
			if err != nil {
				if gctx.Err() != nil {
					return gctx.Err()
				}
				return nil // une direction sans issue n'est pas une erreur
			}
			results[i], found[i] = loop, true
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}

	var loops []domain.Loop
	for i, ok := range found {
		if ok {
			loops = append(loops, results[i])
		}
	}
	if len(loops) == 0 {
		return nil, ErrNoLoopFound
	}

	loops = dedupe(loops)
	sort.Slice(loops, func(a, b int) bool {
		return ecartRelatif(loops[a], req) < ecartRelatif(loops[b], req)
	})
	if len(loops) > req.MaxResults {
		loops = loops[:req.MaxResults]
	}
	return loops, nil
}

func (r Request) withDefaults() Request {
	if r.Tolerance <= 0 {
		r.Tolerance = 0.10
	}
	if r.MaxResults <= 0 {
		r.MaxResults = 5
	}
	return r
}

func ecartRelatif(l domain.Loop, req Request) float64 {
	return math.Abs(l.LengthM-req.DistanceM) / req.DistanceM
}

func seedOf(req Request) int64 {
	h := fnv.New64a()
	fmt.Fprintf(h, "%.6f|%.6f|%.1f|%.3f|%.3f|%d|%d",
		req.Start.Lat, req.Start.Lon, req.DistanceM, req.Tolerance,
		req.Prefs.AvoidPaved, req.MaxResults, req.Variant)
	return int64(h.Sum64())
}

// candidate cherche une boucle dans la direction theta, en corrigeant le rayon
// par dichotomie jusqu'à tomber dans la tolérance.
//
// Le facteur de détour traduit le fait qu'un chemin ne va pas droit : en
// relief il serpente, en plaine il file. On ne peut pas le connaître à
// l'avance, on le mesure.
func (g *Generator) candidate(ctx context.Context, start domain.NodeRef,
	theta float64, req Request) (domain.Loop, error) {

	lo, hi := detourLo, detourHi
	detour := 1.3

	for i := 0; i < maxRadiusIterations; i++ {
		radius := req.DistanceM / (2 * math.Pi * detour)

		loop, err := g.tryLoop(ctx, start, theta, radius, req)
		if err != nil {
			return domain.Loop{}, err
		}

		ecart := (loop.LengthM - req.DistanceM) / req.DistanceM
		if math.Abs(ecart) <= req.Tolerance {
			return loop, nil
		}

		// Trop long : le détour réel dépasse l'estimation, il faut réduire le
		// rayon, donc augmenter le facteur.
		if ecart > 0 {
			lo = detour
		} else {
			hi = detour
		}
		detour = (lo + hi) / 2
	}

	// Aucune itération n'a atteint la tolérance, et il n'y a délibérément pas
	// de repli sur « la moins mauvaise » : une boucle hors tolérance n'est pas
	// ce que l'utilisateur a demandé. Cette direction est abandonnée, les
	// dix-neuf autres sont explorées en parallèle — et les mesures montrent
	// qu'elles aboutissent presque toutes.
	return domain.Loop{}, ErrNoLoopFound
}

func (g *Generator) tryLoop(ctx context.Context, start domain.NodeRef,
	theta, radius float64, req Request) (domain.Loop, error) {

	center := g.net.Coord(start)
	weights := req.Prefs.Weights()
	used := make(map[domain.EdgeRef]struct{})

	loop := domain.Loop{
		Nodes:  []domain.NodeRef{start},
		Coords: []domain.Coord{center},
	}

	current := start
	for k := 0; k < numWaypoints; k++ {
		bearing := theta + float64(k)*2*math.Pi/numWaypoints
		wp, ok := g.net.NearestNode(domain.Offset(center, radius, bearing))
		if !ok || wp == current {
			continue
		}

		seg, err := g.net.FindPath(ctx, current, wp, weights, domain.PathOptions{
			UsedEdges: used, ReuseFactor: reuseFactor,
		})
		if err != nil {
			return domain.Loop{}, err
		}
		appendSegment(&loop, seg, used)
		current = wp
	}

	seg, err := g.net.FindPath(ctx, current, start, weights, domain.PathOptions{
		UsedEdges: used, ReuseFactor: reuseFactor,
	})
	if err != nil {
		return domain.Loop{}, err
	}
	appendSegment(&loop, seg, used)

	if loop.Nodes[len(loop.Nodes)-1] != start {
		return domain.Loop{}, ErrNoLoopFound
	}
	return loop, nil
}

// appendSegment recolle un segment à la boucle en évitant de dupliquer le
// nœud de jonction, et note ses arêtes comme consommées.
func appendSegment(loop *domain.Loop, seg domain.Path, used map[domain.EdgeRef]struct{}) {
	if len(seg.Nodes) > 1 {
		loop.Nodes = append(loop.Nodes, seg.Nodes[1:]...)
		loop.Coords = append(loop.Coords, seg.Coords[1:]...)
	}
	loop.Edges = append(loop.Edges, seg.Edges...)
	loop.LengthM += seg.LengthM

	for _, e := range seg.Edges {
		used[e] = struct{}{}
	}
}
```

- [ ] **Step 6: Implémenter la déduplication**

`internal/app/generateloop/dedupe.go` :

```go
package generateloop

import "github.com/im-sellar/hent/internal/domain"

// Deux boucles partageant plus que ce seuil d'arêtes sont considérées comme
// la même proposition. Des angles de départ voisins convergent souvent vers
// le même itinéraire.
const jaccardThreshold = 0.7

func dedupe(loops []domain.Loop) []domain.Loop {
	var kept []domain.Loop
	var keptSets []map[domain.EdgeRef]struct{}

	for _, l := range loops {
		set := edgeSet(l)

		doublon := false
		for _, other := range keptSets {
			if jaccard(set, other) >= jaccardThreshold {
				doublon = true
				break
			}
		}
		if !doublon {
			kept = append(kept, l)
			keptSets = append(keptSets, set)
		}
	}
	return kept
}

func edgeSet(l domain.Loop) map[domain.EdgeRef]struct{} {
	s := make(map[domain.EdgeRef]struct{}, len(l.Edges))
	for _, e := range l.Edges {
		s[e] = struct{}{}
	}
	return s
}

// jaccard : taille de l'intersection sur taille de l'union.
func jaccard(a, b map[domain.EdgeRef]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for e := range a {
		if _, ok := b[e]; ok {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	return float64(inter) / float64(union)
}
```

- [ ] **Step 7: Tester la déduplication**

Ajouter à `generate_test.go` :

```go
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
```

- [ ] **Step 8: Lancer, vérifier le succès**

```bash
# Version épinglée à dessein : x/sync exige la directive « go 1.26 » à partir
# de ses versions récentes, ce qui ferait sortir le module de la contrainte
# Go 1.25 du plan. v0.8.0 se contente de bien moins.
go get golang.org/x/sync@v0.8.0
go test ./internal/app/generateloop/ -v
```
Expected: PASS (8 tests)

- [ ] **Step 9: Commit**

```bash
go vet ./... && go test ./...
git add -A
git commit -m "feat: génération de boucles par waypoints avec dichotomie et déduplication"
```

---

## Task 8 : Score lisible

Le **coût** guide l'A* et n'a aucun sens pour un humain. Le **score** est calculé sur la boucle finie et sert à la trier et à l'expliquer. Il a besoin d'agrégats que seul l'adaptateur peut produire — ils traversent donc la frontière sous forme de données portées par `Path` et `Loop`, jamais sous forme d'un accès au graphe.

**Files:**
- Modify: `internal/domain/track.go` (agrégats sur `Path` et `Loop`)
- Modify: `internal/adapter/network/csr/astar.go` (remplir les agrégats dans `rebuild`)
- Modify: `internal/app/generateloop/generate.go` (`appendSegment` cumule, tri par score)
- Modify: `internal/app/generateloop/testnetwork_test.go` (la grille remplit aussi les agrégats)
- Create: `internal/domain/score.go`, `internal/domain/score_test.go`

**Interfaces:**
- Consumes: `domain.Loop` (Task 7).
- Produces:
  - Champs ajoutés à `domain.Path` et `domain.Loop` : `UnpavedM float64`, `TrafficExposureM float64`
  - `domain.Score{DistanceM, PartNonBitume, PartTrafic, EcartCible float64}`
  - `domain.NewScore(l Loop, targetM float64) Score`

- [ ] **Step 1: Écrire le test du score**

`internal/domain/score_test.go` :

```go
package domain_test

import (
	"math"
	"testing"

	"github.com/im-sellar/hent/internal/domain"
)

func TestNewScore(t *testing.T) {
	l := domain.Loop{
		LengthM:          10000,
		UnpavedM:         8500,
		TrafficExposureM: 500,
	}

	s := domain.NewScore(l, 10000)

	if s.DistanceM != 10000 {
		t.Errorf("DistanceM = %v, attendu 10000", s.DistanceM)
	}
	if math.Abs(s.PartNonBitume-0.85) > 1e-9 {
		t.Errorf("PartNonBitume = %v, attendu 0.85", s.PartNonBitume)
	}
	if math.Abs(s.PartTrafic-0.05) > 1e-9 {
		t.Errorf("PartTrafic = %v, attendu 0.05", s.PartTrafic)
	}
	if s.EcartCible != 0 {
		t.Errorf("EcartCible = %v, attendu 0", s.EcartCible)
	}
}

func TestNewScoreEcartSigne(t *testing.T) {
	court := domain.NewScore(domain.Loop{LengthM: 9000}, 10000)
	long := domain.NewScore(domain.Loop{LengthM: 11000}, 10000)

	if court.EcartCible >= 0 {
		t.Errorf("une boucle trop courte doit avoir un écart négatif, obtenu %v", court.EcartCible)
	}
	if long.EcartCible <= 0 {
		t.Errorf("une boucle trop longue doit avoir un écart positif, obtenu %v", long.EcartCible)
	}
}

func TestPreferable(t *testing.T) {
	const tol = 0.10

	score := func(ecart, part float64) domain.Score {
		return domain.Score{EcartCible: ecart, PartNonBitume: part}
	}

	cas := []struct {
		nom     string
		a, b    domain.Score
		attendu bool
	}{
		{"une boucle dans la tolérance l'emporte sur une boucle hors tolérance, même mieux revêtue",
			score(0.05, 0.2), score(0.30, 0.9), true},
		{"et réciproquement, hors tolérance ne l'emporte jamais sur dans la tolérance",
			score(0.30, 0.9), score(0.05, 0.2), false},
		{"deux boucles dans la tolérance se départagent sur le terrain, pas sur les mètres",
			score(0.08, 0.9), score(0.02, 0.5), true},
		{"deux boucles hors tolérance se départagent sur l'écart à la cible",
			score(0.20, 0.1), score(0.40, 0.9), true},
		{"un écart exactement égal à la tolérance compte comme dans la tolérance",
			score(tol, 0.9), score(0.30, 0.9), true},
	}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := c.a.Preferable(c.b, tol); got != c.attendu {
				t.Errorf("Preferable = %v, attendu %v", got, c.attendu)
			}
		})
	}
}

// TestPreferableEstUnOrdreStrictFaible vérifie les trois propriétés dont
// sort.Slice dépend. Une fonction de comparaison incohérente n'y provoque ni
// erreur ni panique : elle produit un ordre arbitraire, silencieusement.
func TestPreferableEstUnOrdreStrictFaible(t *testing.T) {
	const tol = 0.10

	var scores []domain.Score
	for _, ecart := range []float64{-0.30, -0.10, -0.02, 0, 0.02, 0.10, 0.30} {
		for _, part := range []float64{0, 0.5, 1} {
			scores = append(scores, domain.Score{EcartCible: ecart, PartNonBitume: part})
		}
	}

	for _, a := range scores {
		if a.Preferable(a, tol) {
			t.Fatalf("irréflexivité violée : %+v se préfère à lui-même", a)
		}
	}

	for _, a := range scores {
		for _, b := range scores {
			if a.Preferable(b, tol) && b.Preferable(a, tol) {
				t.Fatalf("asymétrie violée entre %+v et %+v", a, b)
			}
		}
	}

	for _, a := range scores {
		for _, b := range scores {
			for _, c := range scores {
				if a.Preferable(b, tol) && b.Preferable(c, tol) && !a.Preferable(c, tol) {
					t.Fatalf("transitivité violée : %+v puis %+v puis %+v", a, b, c)
				}
			}
		}
	}
}

func TestNewScoreCibleNulle(t *testing.T) {
	// Ne doit ni diviser par zéro ni produire un écart non fini.
	s := domain.NewScore(domain.Loop{LengthM: 5000}, 0)

	if math.IsNaN(s.EcartCible) || math.IsInf(s.EcartCible, 0) {
		t.Errorf("EcartCible = %v, attendu une valeur finie", s.EcartCible)
	}
}

func TestNewScoreBoucleVide(t *testing.T) {
	// Ne doit pas diviser par zéro.
	s := domain.NewScore(domain.Loop{}, 10000)

	if s.PartNonBitume != 0 || s.PartTrafic != 0 {
		t.Errorf("boucle vide : %+v, attendu des parts nulles", s)
	}
}
```

- [ ] **Step 2: Lancer, vérifier l'échec**

Run: `go test ./internal/domain/ -run TestNewScore`
Expected: FAIL — `undefined: domain.NewScore`

- [ ] **Step 3: Ajouter les agrégats et le score**

Ajouter les champs à `Path` et `Loop` dans `internal/domain/track.go` :

```go
// Dans Path et dans Loop, ajouter :

	// Agrégats qui traversent la frontière sous forme de données : la couche
	// métier calcule le score sans jamais accéder au graphe. Sur un Path, ils
	// sont calculés par l'adaptateur pendant la reconstruction du chemin ; sur
	// une Loop, ils sont cumulés segment par segment par la couche applicative
	// à partir des Path qui la composent.
	UnpavedM         float64 // longueur cumulée hors revêtement dur
	TrafficExposureM float64 // longueur pondérée par l'exposition au trafic
```

`internal/domain/score.go` :

```go
package domain

import "math"

// Score décrit une boucle en termes compréhensibles. À la différence du coût,
// qui guide l'A* et n'a de sens qu'en interne, le score est exposé par l'API :
// il permet à l'utilisateur de voir pourquoi une boucle lui est proposée et
// sur quel critère elle est moins bonne que la suivante.
type Score struct {
	DistanceM     float64 `json:"distance_m"`
	PartNonBitume float64 `json:"part_non_bitume"`
	PartTrafic    float64 `json:"part_trafic"`
	EcartCible    float64 `json:"ecart_cible"`
}

func NewScore(l Loop, targetM float64) Score {
	s := Score{DistanceM: l.LengthM}

	if l.LengthM > 0 {
		s.PartNonBitume = l.UnpavedM / l.LengthM
		s.PartTrafic = l.TrafficExposureM / l.LengthM
	}
	if targetM > 0 {
		s.EcartCible = (l.LengthM - targetM) / targetM
	}
	return s
}

// Preferable classe deux boucles : d'abord la fidélité à la distance
// demandée, puis la part de chemins. Deux boucles dans la tolérance sont
// départagées par le terrain, pas par les mètres.
func (s Score) Preferable(other Score, tolerance float64) bool {
	inTol := math.Abs(s.EcartCible) <= tolerance
	otherInTol := math.Abs(other.EcartCible) <= tolerance

	if inTol != otherInTol {
		return inTol
	}
	if inTol {
		return s.PartNonBitume > other.PartNonBitume
	}
	return math.Abs(s.EcartCible) < math.Abs(other.EcartCible)
}
```

- [ ] **Step 4: Remplir les agrégats dans l'adaptateur**

Dans `internal/adapter/network/csr/astar.go`, méthode `rebuild`, remplacer la boucle de cumul des arêtes par :

```go
	for i, e := range revEdges {
		p.Edges[len(revEdges)-1-i] = e

		a := g.attrs[e]
		p.LengthM += a.LengthM
		if a.Surface != domain.SurfacePaved {
			p.UnpavedM += a.LengthM
		}
		p.TrafficExposureM += a.LengthM * float64(a.Traffic) / 255
	}
```

Dans `internal/app/generateloop/generate.go`, `appendSegment` :

```go
	loop.LengthM += seg.LengthM
	loop.UnpavedM += seg.UnpavedM
	loop.TrafficExposureM += seg.TrafficExposureM
```

Et remplacer le tri de `Generate` :

```go
	sort.Slice(loops, func(a, b int) bool {
		sa := domain.NewScore(loops[a], req.DistanceM)
		sb := domain.NewScore(loops[b], req.DistanceM)
		return sa.Preferable(sb, req.Tolerance)
	})
```

La fonction `ecartRelatif` n'est alors plus utilisée : la supprimer.

Dans `internal/app/generateloop/testnetwork_test.go`, à la fin du calcul de longueur de `FindPath`, cumuler aussi les agrégats pour que la grille reste un double fidèle :

```go
	for i := 0; i+1 < len(p.Nodes); i++ {
		for _, a := range g.voisins[p.Nodes[i]] {
			if a.cible == p.Nodes[i+1] {
				p.LengthM += a.long
				p.UnpavedM += a.long // la grille de test est intégralement en chemin
				break
			}
		}
	}
```

- [ ] **Step 5: Lancer, vérifier le succès**

Run: `go test ./... -v`
Expected: PASS — l'ensemble des tests des Tasks 1 à 8.

- [ ] **Step 6: Commit**

```bash
go vet ./... && go test ./...
git add -A
git commit -m "feat: score lisible d'une boucle et tri par préférence"
```

---

## Task 9 : API HTTP, export GPX et binaire `routed`

**Files:**
- Create: `internal/adapter/gpxfile/write.go`, `internal/adapter/gpxfile/write_test.go`
- Create: `internal/adapter/httpapi/handler.go`, `internal/adapter/httpapi/handler_test.go`
- Create: `internal/adapter/httpapi/ratelimit.go`
- Create: `cmd/routed/main.go`
- Modify: `go.mod` (ajout de `golang.org/x/time`)

**Interfaces:**
- Consumes: `generateloop.Generator` (Task 7), `domain.NewScore` (Task 8), `csr.ReadGraph` (Task 6).
- Produces:
  - `gpxfile.Write(w io.Writer, l domain.Loop, name string) error`
  - `httpapi.New(gen *generateloop.Generator, prov csr.Provenance, bbox domain.BBox) http.Handler`

- [ ] **Step 1: Écrire le test de l'export GPX**

`internal/adapter/gpxfile/write_test.go` :

```go
package gpxfile_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/im-sellar/hent/internal/adapter/gpxfile"
	"github.com/im-sellar/hent/internal/domain"
)

func TestWriteGPX(t *testing.T) {
	l := domain.Loop{
		Coords: []domain.Coord{
			{Lat: 48.1173, Lon: -1.6778},
			{Lat: 48.1200, Lon: -1.6700},
			{Lat: 48.1173, Lon: -1.6778},
		},
		LengthM: 1500,
	}

	var buf bytes.Buffer
	if err := gpxfile.Write(&buf, l, "Boucle test"); err != nil {
		t.Fatalf("Write : %v", err)
	}
	out := buf.String()

	for _, attendu := range []string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<gpx`, `creator="hent"`, `<trkseg>`,
		`lat="48.1173"`, `lon="-1.6778"`,
		`<name>Boucle test</name>`,
		`OpenStreetMap`, // l'attribution voyage avec le fichier
	} {
		if !strings.Contains(out, attendu) {
			t.Errorf("le GPX ne contient pas %q", attendu)
		}
	}

	if n := strings.Count(out, "<trkpt "); n != 3 {
		t.Errorf("%d points de trace, attendu 3", n)
	}
}

func TestWriteGPXBoucleVide(t *testing.T) {
	var buf bytes.Buffer
	if err := gpxfile.Write(&buf, domain.Loop{}, "vide"); err == nil {
		t.Fatal("exporter une boucle sans point doit échouer")
	}
}
```

- [ ] **Step 2: Implémenter l'export GPX**

`internal/adapter/gpxfile/write.go` :

```go
// Package gpxfile sérialise une boucle au format GPX 1.1, lisible par les
// montres et applications de trace.
package gpxfile

import (
	"errors"
	"fmt"
	"io"

	"github.com/im-sellar/hent/internal/domain"
)

// Attribution : l'ODbL impose de créditer la source, et le fichier circule
// indépendamment de l'API — l'attribution doit donc voyager avec lui.
const attribution = "Données © les contributeurs OpenStreetMap, sous licence ODbL"

var ErrEmptyLoop = errors.New("boucle sans point, rien à exporter")

func Write(w io.Writer, l domain.Loop, name string) error {
	if len(l.Coords) == 0 {
		return ErrEmptyLoop
	}

	if _, err := fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="hent" xmlns="http://www.topografix.com/GPX/1/1">
  <metadata>
    <name>%s</name>
    <desc>%s — %.0f m</desc>
    <copyright author="OpenStreetMap contributors"><license>https://opendatacommons.org/licenses/odbl/</license></copyright>
  </metadata>
  <trk>
    <name>%s</name>
    <trkseg>
`, escape(name), attribution, l.LengthM, escape(name)); err != nil {
		return err
	}

	for _, c := range l.Coords {
		// %.7f et non %.7g : le second compte des chiffres SIGNIFICATIFS, pas
		// des décimales. Sur une latitude à deux chiffres entiers il n'en
		// resterait que cinq, soit près d'un mètre d'erreur — inutilement
		// dégradé alors que la précision exacte ne coûte rien.
		if _, err := fmt.Fprintf(w, "      <trkpt lat=\"%.7f\" lon=\"%.7f\"></trkpt>\n", c.Lat, c.Lon); err != nil {
			return err
		}
	}

	_, err := io.WriteString(w, "    </trkseg>\n  </trk>\n</gpx>\n")
	return err
}

func escape(s string) string {
	var out []rune
	for _, r := range s {
		switch r {
		case '&':
			out = append(out, []rune("&amp;")...)
		case '<':
			out = append(out, []rune("&lt;")...)
		case '>':
			out = append(out, []rune("&gt;")...)
		default:
			out = append(out, r)
		}
	}
	return string(out)
}
```

- [ ] **Step 3: Lancer, vérifier le succès**

Run: `go test ./internal/adapter/gpxfile/ -v`
Expected: PASS

Note sur le verbe de formatage : `%.7f` donne sept **décimales**, soit environ un centimètre — la précision utile pour du GPS. `%.7g`, qui paraît équivalent, donne sept chiffres *significatifs* : sur une latitude à deux chiffres entiers, il n'en reste que cinq, et l'erreur atteint le mètre.

- [ ] **Step 4: Écrire le test de l'API**

`internal/adapter/httpapi/handler_test.go` :

```go
package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/im-sellar/hent/internal/adapter/httpapi"
)

func TestPostLoops(t *testing.T) {
	srv := httptest.NewServer(testHandler(t))
	defer srv.Close()

	// Le départ est au centre de la grille de test. Un point proche d'un bord
	// enverrait les waypoints hors de la zone couverte et NearestNode
	// échouerait : le test deviendrait intermittent.
	body := `{"start":{"lat":48.135,"lon":-1.628},"distance_m":4000,
	          "activity":"trail","preferences":{"avoid_paved":0.5},"max_results":3}`

	resp, err := http.Post(srv.URL+"/v1/loops", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("statut %d, attendu 200", resp.StatusCode)
	}

	var out struct {
		Loops []struct {
			ID    string          `json:"id"`
			Score json.RawMessage `json:"score"`
			Geom  json.RawMessage `json:"geometry"`
		} `json:"loops"`
		Attribution string `json:"attribution"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}

	if len(out.Loops) == 0 {
		t.Fatal("aucune boucle renvoyée")
	}
	if out.Loops[0].ID == "" {
		t.Error("chaque boucle doit porter un identifiant, pour l'export GPX")
	}
	if !strings.Contains(out.Attribution, "OpenStreetMap") {
		t.Errorf("attribution manquante : %q", out.Attribution)
	}
}

func TestPostLoopsValidation(t *testing.T) {
	srv := httptest.NewServer(testHandler(t))
	defer srv.Close()

	cases := map[string]string{
		"distance nulle":     `{"start":{"lat":48.11,"lon":-1.67},"distance_m":0}`,
		"distance démesurée": `{"start":{"lat":48.11,"lon":-1.67},"distance_m":900000}`,
		"latitude invalide":  `{"start":{"lat":991,"lon":-1.67},"distance_m":4000}`,
		"json cassé":         `{oops`,
	}

	for nom, body := range cases {
		t.Run(nom, func(t *testing.T) {
			resp, err := http.Post(srv.URL+"/v1/loops", "application/json", strings.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("statut %d, attendu 400", resp.StatusCode)
			}
		})
	}
}

func TestGetGPX(t *testing.T) {
	srv := httptest.NewServer(testHandler(t))
	defer srv.Close()

	body := `{"start":{"lat":48.135,"lon":-1.628},"distance_m":4000,"max_results":1}`
	resp, err := http.Post(srv.URL+"/v1/loops", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Loops []struct {
			ID string `json:"id"`
		} `json:"loops"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("réponse illisible : %v", err)
	}
	resp.Body.Close()

	// Sans cette garde, l'indexation ci-dessous provoquerait un « index out of
	// range » dont le message ne dirait rien de la cause réelle.
	if len(out.Loops) == 0 {
		t.Fatal("aucune boucle renvoyée : rien à exporter en GPX")
	}

	// L'identifiant encode la requête : le GPX se régénère sans état côté
	// serveur, ce qui n'est possible que parce que la génération est
	// déterministe.
	gpx, err := http.Get(srv.URL + "/v1/loops/" + out.Loops[0].ID + ".gpx")
	if err != nil {
		t.Fatal(err)
	}
	defer gpx.Body.Close()

	if gpx.StatusCode != http.StatusOK {
		t.Fatalf("statut %d, attendu 200", gpx.StatusCode)
	}
	var buf bytes.Buffer
	buf.ReadFrom(gpx.Body)
	if !strings.Contains(buf.String(), "<trkseg>") {
		t.Error("la réponse ne ressemble pas à du GPX")
	}
}

func TestHealthzEtRegions(t *testing.T) {
	srv := httptest.NewServer(testHandler(t))
	defer srv.Close()

	for _, path := range []string{"/healthz", "/v1/regions", "/metrics"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s : statut %d, attendu 200", path, resp.StatusCode)
		}
	}
}
```

`testHandler` construit un handler sur un réseau synthétique. Créer `internal/adapter/httpapi/testnetwork_test.go` en **copiant** le fichier `internal/app/generateloop/testnetwork_test.go` (type `grille`), en changeant seulement la ligne `package` pour `httpapi_test`. La grille est déjà centrée près de Rennes (origine 48,100 / −1,680) et s'étend sur environ 7,8 km de côté ; les tests partent de son centre. Ajouter ensuite :

```go
func testHandler(t *testing.T) http.Handler {
	t.Helper()

	g := nouvelleGrille(40, 200) // ~8 km de côté autour de 48.10 / -1.68
	return httpapi.New(
		generateloop.New(g),
		csr.Provenance{BuiltAt: "2026-08-18T10:00:00Z"},
		g.BBox(),
	)
}
```

- [ ] **Step 5: Implémenter le handler**

`internal/adapter/httpapi/handler.go` :

```go
// Package httpapi expose le générateur de boucles en HTTP. Les DTO définis
// ici sont volontairement distincts des types du domaine : le contrat public
// doit pouvoir rester stable pendant que le modèle interne évolue.
package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/im-sellar/hent/internal/adapter/gpxfile"
	"github.com/im-sellar/hent/internal/adapter/network/csr"
	"github.com/im-sellar/hent/internal/app/generateloop"
	"github.com/im-sellar/hent/internal/domain"
)

const attribution = "Données © les contributeurs OpenStreetMap, sous licence ODbL"

const (
	minDistanceM = 500
	maxDistanceM = 200_000
	maxResults   = 10
	requestTTL   = 5 * time.Second
)

type coordDTO struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type prefsDTO struct {
	// Curseur d'intention, 0 à 1. Les poids internes ne sortent jamais : le
	// modèle de coût peut être entièrement revu sans casser un client.
	AvoidPaved float64 `json:"avoid_paved"`
}

type loopsRequest struct {
	Start       coordDTO `json:"start"`
	DistanceM   float64  `json:"distance_m"`
	Tolerance   float64  `json:"tolerance"`
	Activity    string   `json:"activity"`
	Preferences prefsDTO `json:"preferences"`
	MaxResults  int      `json:"max_results"`
	Variant     int      `json:"variant"`
}

func (r loopsRequest) validate() error {
	// La finitude se vérifie d'abord, et séparément des encadrements : en Go
	// toute comparaison impliquant NaN est fausse, si bien qu'un NaN
	// traverserait intact tous les tests de bornes ci-dessous — dans la
	// fonction même dont le rôle est de les faire respecter.
	//
	// Le décodeur JSON de la bibliothèque standard ne produit pas de NaN (le
	// format ne l'admet pas), cette garde n'est donc pas atteignable par
	// l'API et aucun test HTTP ne peut la couvrir. Elle protège les appelants
	// non-HTTP — un client Go, un outil en ligne de commande — qui
	// construiraient une requête directement.
	for nom, v := range map[string]float64{
		"start.lat":  r.Start.Lat,
		"start.lon":  r.Start.Lon,
		"distance_m": r.DistanceM,
		"tolerance":  r.Tolerance,
	} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("%s doit être un nombre fini", nom)
		}
	}

	switch {
	case r.Start.Lat < -90 || r.Start.Lat > 90:
		return errors.New("latitude hors bornes")
	case r.Start.Lon < -180 || r.Start.Lon > 180:
		return errors.New("longitude hors bornes")
	case r.DistanceM < minDistanceM:
		return fmt.Errorf("distance_m doit valoir au moins %d", minDistanceM)
	case r.DistanceM > maxDistanceM:
		return fmt.Errorf("distance_m ne peut dépasser %d", maxDistanceM)
	case r.Tolerance < 0 || r.Tolerance > 0.5:
		return errors.New("tolerance doit être comprise entre 0 et 0.5")
	case r.MaxResults > maxResults:
		return fmt.Errorf("max_results ne peut dépasser %d", maxResults)
	case r.Activity != "" && r.Activity != "trail":
		return errors.New(`seule l'activité "trail" est prise en charge à ce stade`)
	}
	return nil
}

func (r loopsRequest) toDomain() generateloop.Request {
	return generateloop.Request{
		Start:      domain.Coord{Lat: r.Start.Lat, Lon: r.Start.Lon},
		DistanceM:  r.DistanceM,
		Tolerance:  r.Tolerance,
		Prefs:      domain.Preferences{AvoidPaved: r.Preferences.AvoidPaved},
		MaxResults: r.MaxResults,
		Variant:    r.Variant,
	}
}

type loopDTO struct {
	ID       string       `json:"id"`
	Score    domain.Score `json:"score"`
	Geometry [][2]float64 `json:"geometry"` // [lon, lat], ordre GeoJSON
}

type loopsResponse struct {
	Loops       []loopDTO `json:"loops"`
	Attribution string    `json:"attribution"`
}

type api struct {
	gen  *generateloop.Generator
	prov csr.Provenance
	bbox domain.BBox

	requests atomic.Int64
	errors   atomic.Int64
	totalMs  atomic.Int64
}

func New(gen *generateloop.Generator, prov csr.Provenance, bbox domain.BBox, trustedProxies map[string]struct{}) http.Handler {
	a := &api{gen: gen, prov: prov, bbox: bbox}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/loops", a.postLoops)
	mux.HandleFunc("GET /v1/loops/{id}", a.getGPX)
	mux.HandleFunc("GET /v1/regions", a.getRegions)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /metrics", a.getMetrics)

	return withRateLimit(mux, trustedProxies)
}

func (a *api) postLoops(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	a.requests.Add(1)
	defer func() { a.totalMs.Add(time.Since(started).Milliseconds()) }()

	var req loopsRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		a.fail(w, http.StatusBadRequest, "corps de requête illisible")
		return
	}
	if err := req.validate(); err != nil {
		a.fail(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTTL)
	defer cancel()

	loops, err := a.gen.Generate(ctx, req.toDomain())
	if err != nil {
		switch {
		case errors.Is(err, generateloop.ErrStartOutOfRange):
			a.fail(w, http.StatusBadRequest, "le point de départ est hors de la zone couverte")
		case errors.Is(err, generateloop.ErrNoLoopFound):
			a.fail(w, http.StatusNotFound, "aucune boucle trouvée pour ces critères")
		case errors.Is(err, context.DeadlineExceeded):
			a.fail(w, http.StatusGatewayTimeout, "délai dépassé")
		default:
			log.Printf("génération : %v", err)
			a.fail(w, http.StatusInternalServerError, "erreur interne")
		}
		return
	}

	resp := loopsResponse{Attribution: attribution}
	for i, l := range loops {
		resp.Loops = append(resp.Loops, loopDTO{
			ID:       encodeID(req, i),
			Score:    domain.NewScore(l, req.DistanceM),
			Geometry: geometryOf(l),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	json.NewEncoder(w).Encode(resp)
}

// getGPX régénère la boucle à partir de l'identifiant, qui encode la requête
// et l'indice. Le serveur ne conserve aucun état entre les deux appels : c'est
// possible uniquement parce que la génération est déterministe.
func (a *api) getGPX(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSuffix(r.PathValue("id"), ".gpx")

	req, index, err := decodeID(id)
	if err != nil {
		a.fail(w, http.StatusBadRequest, "identifiant de boucle invalide")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTTL)
	defer cancel()

	loops, err := a.gen.Generate(ctx, req.toDomain())
	if err != nil || index >= len(loops) {
		a.fail(w, http.StatusNotFound, "boucle introuvable")
		return
	}

	w.Header().Set("Content-Type", "application/gpx+xml")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="hent-%.0fm.gpx"`, req.DistanceM))
	name := fmt.Sprintf("Boucle %.1f km", loops[index].LengthM/1000)
	if err := gpxfile.Write(w, loops[index], name); err != nil {
		log.Printf("export GPX : %v", err)
	}
}

func (a *api) getRegions(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"bbox": map[string]float64{
			"min_lat": a.bbox.Min.Lat, "min_lon": a.bbox.Min.Lon,
			"max_lat": a.bbox.Max.Lat, "max_lon": a.bbox.Max.Lon,
		},
		"data":        a.prov,
		"attribution": attribution,
	})
}

func (a *api) getMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "hent_requests_total %d\n", a.requests.Load())
	fmt.Fprintf(w, "hent_errors_total %d\n", a.errors.Load())
	fmt.Fprintf(w, "hent_request_duration_ms_total %d\n", a.totalMs.Load())
}

func (a *api) fail(w http.ResponseWriter, code int, msg string) {
	a.errors.Add(1)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func geometryOf(l domain.Loop) [][2]float64 {
	pts := make([][2]float64, len(l.Coords))
	for i, c := range l.Coords {
		pts[i] = [2]float64{c.Lon, c.Lat}
	}
	return pts
}

func encodeID(req loopsRequest, index int) string {
	raw, _ := json.Marshal(req)
	return fmt.Sprintf("%d.%s", index, base64.RawURLEncoding.EncodeToString(raw))
}

func decodeID(id string) (loopsRequest, int, error) {
	var req loopsRequest

	index, rest, found := strings.Cut(id, ".")
	if !found {
		return req, 0, errors.New("identifiant mal formé")
	}
	var i int
	if _, err := fmt.Sscanf(index, "%d", &i); err != nil || i < 0 {
		return req, 0, errors.New("indice invalide")
	}

	raw, err := base64.RawURLEncoding.DecodeString(rest)
	if err != nil {
		return req, 0, err
	}
	if err := json.Unmarshal(raw, &req); err != nil {
		return req, 0, err
	}
	if err := req.validate(); err != nil {
		return req, 0, err
	}
	return req, i, nil
}
```

Ajouter `"context"` à la liste des imports du fichier.

- [ ] **Step 6: Implémenter la limitation de débit**

Chaque requête consomme des centaines de millisecondes de CPU : sans limite, c'est un déni de service en une ligne de `curl`.

`internal/adapter/httpapi/ratelimit.go` :

```go
package httpapi

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	perIPRate  = 1 // requêtes par seconde
	perIPBurst = 5
	idleTTL    = 10 * time.Minute

	// Plafond du nombre de clients suivis simultanément. Dimensionné très
	// au-dessus d'un trafic légitime pour ce service, et très en dessous de
	// ce qui mettrait la mémoire en péril.
	maxTrackedClients = 50_000
)

type limiter struct {
	mu      sync.Mutex
	clients map[string]*client
}

type client struct {
	lim  *rate.Limiter
	seen time.Time
}

func withRateLimit(next http.Handler, trustedProxies map[string]struct{}) http.Handler {
	l := &limiter{clients: make(map[string]*client)}
	go l.cleanup()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Les sondes de disponibilité et de métriques ne sont pas limitées :
		// elles ne coûtent rien et doivent rester joignables sous charge.
		if r.URL.Path == "/healthz" || r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		if !l.allow(clientIP(r, trustedProxies)) {
			w.Header().Set("Retry-After", "1")
			http.Error(w, `{"error":"trop de requêtes"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// allow décide du sort d'une requête et borne la taille de la table.
//
// Le plafond n'est pas cosmétique : chaque entrée porte un rate.Limiter, et
// sans borne un client qui fait varier son adresse — ou son en-tête, si un
// proxy de confiance est mal configuré — fait enfler la map jusqu'à la
// mémoire disponible. Le nettoyage périodique ne suffit pas : il ne passe que
// toutes les dix minutes, et tout ce qui arrive entre deux passages
// s'accumule.
//
// Au-delà du plafond, on refuse plutôt que d'allouer : mieux vaut dégrader le
// service pour des clients inconnus que tomber pour tout le monde.
func (l *limiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	c, ok := l.clients[ip]
	if !ok {
		if len(l.clients) >= maxTrackedClients {
			return false
		}
		c = &client{lim: rate.NewLimiter(perIPRate, perIPBurst)}
		l.clients[ip] = c
	}
	c.seen = time.Now()
	return c.lim.Allow()
}

func (l *limiter) cleanup() {
	for range time.Tick(idleTTL) {
		l.mu.Lock()
		for ip, c := range l.clients {
			if time.Since(c.seen) > idleTTL {
				delete(l.clients, ip)
			}
		}
		l.mu.Unlock()
	}
}

// clientIP identifie le client pour la limitation de débit.
//
// X-Forwarded-For n'est lu QUE si la connexion vient d'un proxy déclaré de
// confiance. C'est une règle de sécurité, pas une commodité : n'importe quel
// client peut envoyer cet en-tête, et un reverse proxy l'AJOUTE à ce qu'il a
// reçu au lieu de l'écraser. Lui faire confiance sans condition revient à
// offrir une clé de limiteur neuve à chaque requête — soit à supprimer la
// limitation tout en croyant l'avoir.
//
// Quand l'en-tête est digne de confiance, c'est sa DERNIÈRE entrée qu'on
// retient : les précédentes ont été fournies par le client.
func clientIP(r *http.Request, trustedProxies map[string]struct{}) string {
	remote := hostOf(r.RemoteAddr)

	if _, trusted := trustedProxies[remote]; trusted {
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			parts := strings.Split(fwd, ",")
			return hostOf(strings.TrimSpace(parts[len(parts)-1]))
		}
	}
	return remote
}

func hostOf(addr string) string {
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}
```

Le burst de 5 laisse passer les tests, qui enchaînent quelques requêtes. Si `TestPostLoopsValidation` se met à renvoyer des 429, augmenter `perIPBurst` plutôt que de contourner la limite dans le test.

- [ ] **Step 7: Lancer les tests de l'API**

```bash
# Version épinglée à dessein : x/time à partir de v0.15.0 exige la directive
# « go 1.26 », ce qui ferait sortir le module de la contrainte Go 1.25 du plan.
# v0.14.0 se contente de 1.24. Même précaution que pour x/sync, épinglé à v0.8.0.
go get golang.org/x/time@v0.14.0
go test ./internal/adapter/httpapi/ ./internal/adapter/gpxfile/ -v
```
Expected: PASS

- [ ] **Step 8: Exposer les métriques d'exploration**

La spec (§12) exige trois mesures, pas seulement le temps de réponse : **nœuds explorés par A\*** et **taux de candidates abandonnées** sont ce qui permet de régler les pondérations autrement qu'à l'aveugle. Sans elles, l'arbitrage coût/performance du §6 reste invisible jusqu'au premier timeout en production.

`domain.Path` porte déjà `ExploredNodes` depuis la Task 4 : la donnée remonte de l'A* sans qu'aucune signature ne change. Il reste à l'agréger et à l'exposer.

**D'abord, le comptage des nœuds explorés change de place.** Le placer dans
`appendSegment` ne compterait que les segments *retenus* — or une recherche qui
échoue est précisément celle qui a vidé son tas après avoir exploré le plus
largement. La métrique censée surveiller le coût de l'exploration omettrait donc
systématiquement les cas les plus coûteux, et sous-estimerait d'autant.

Le comptage appartient à l'adaptateur, qui voit toutes les recherches. Ajouter à
`internal/adapter/network/csr/graph.go` :

```go
	// exploredTotal cumule les nœuds dépilés par TOUTES les recherches, y
	// compris celles qui échouent — ce sont elles qui explorent le plus.
	exploredTotal atomic.Int64
```

et, dans `FindPath` (`astar.go`), incrémenter ce compteur sur **chaque** chemin
de sortie, succès comme échec, juste avant le `return`. Exposer ensuite :

```go
// ExploredNodesTotal retourne le cumul des nœuds dépilés depuis le chargement
// du graphe, toutes recherches confondues.
func (g *Graph) ExploredNodesTotal() int64 { return g.exploredTotal.Load() }
```

Dans `internal/app/generateloop/generate.go`, ajouter le compteur de candidates
abandonnées au `Generator` :

```go
type Generator struct {
	net port.RouteNetwork

	exploredNodes atomic.Int64
	dropped       atomic.Int64
}

// Stats retourne les compteurs cumulés depuis le démarrage.
func (g *Generator) Stats() (exploredNodes, droppedCandidates int64) {
	return g.exploredNodes.Load(), g.dropped.Load()
}
```

Ajouter `"sync/atomic"` aux imports. Incrémenter `dropped` dans le `group.Go` de `Generate`, là où une direction sans issue est ignorée :

```go
			loop, err := g.candidate(gctx, start, theta, req)
			if err != nil {
				if gctx.Err() != nil {
					return gctx.Err()
				}
				g.dropped.Add(1) // une direction sans issue n'est pas une erreur
				return nil
			}
```

Et cumuler les nœuds explorés dans `appendSegment`, qui devient une méthode pour accéder au compteur :

```go
func (g *Generator) appendSegment(loop *domain.Loop, seg domain.Path, used map[domain.EdgeRef]struct{}) {
	g.exploredNodes.Add(int64(seg.ExploredNodes))

	if len(seg.Nodes) > 1 {
		loop.Nodes = append(loop.Nodes, seg.Nodes[1:]...)
		loop.Coords = append(loop.Coords, seg.Coords[1:]...)
	}
	loop.Edges = append(loop.Edges, seg.Edges...)
	loop.LengthM += seg.LengthM
	loop.UnpavedM += seg.UnpavedM
	loop.TrafficExposureM += seg.TrafficExposureM

	for _, e := range seg.Edges {
		used[e] = struct{}{}
	}
}
```

Adapter les deux appels dans `tryLoop` : `appendSegment(&loop, seg, used)` devient `g.appendSegment(&loop, seg, used)`.

Enfin, dans `internal/adapter/httpapi/handler.go`, compléter `getMetrics` :

```go
func (a *api) getMetrics(w http.ResponseWriter, _ *http.Request) {
	explored, dropped := a.gen.Stats()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "hent_requests_total %d\n", a.requests.Load())
	fmt.Fprintf(w, "hent_errors_total %d\n", a.errors.Load())
	fmt.Fprintf(w, "hent_request_duration_ms_total %d\n", a.totalMs.Load())
	fmt.Fprintf(w, "hent_astar_explored_nodes_total %d\n", explored)
	fmt.Fprintf(w, "hent_candidates_dropped_total %d\n", dropped)
}
```

Vérifier que tout compile et que la suite passe :

Run: `go test ./... && go vet ./...`
Expected: PASS

- [ ] **Step 9: Écrire le serveur**

`cmd/routed/main.go` :

```go
// Command routed sert le générateur de boucles. Il charge un artefact
// graph.bin au démarrage et ne fait plus, ensuite, que router : ni base de
// données, ni lecture de fichier OSM, ni calcul géospatial en ligne.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/im-sellar/hent/internal/adapter/httpapi"
	"github.com/im-sellar/hent/internal/adapter/network/csr"
	"github.com/im-sellar/hent/internal/app/generateloop"
	"github.com/im-sellar/hent/internal/app/port"
)

// Le graphe CSR satisfait le port réseau. C'est ici, au câblage, que la
// vérification a lieu : le paquet csr ignore l'existence du port, et le
// paquet port ignore l'existence de csr.
var _ port.RouteNetwork = (*csr.Graph)(nil)

func main() {
	graphPath := flag.String("graph", "graph.bin", "artefact produit par graphbuild")
	addr := flag.String("addr", ":8080", "adresse d'écoute")
	flag.Parse()

	if err := run(*graphPath, *addr); err != nil {
		log.Fatal(err)
	}
}

func run(graphPath, addr string) error {
	start := time.Now()

	f, err := os.Open(graphPath)
	if err != nil {
		return err
	}
	g, prov, err := csr.ReadGraph(f)
	f.Close()
	if err != nil {
		// Notamment ErrBadVersion : mieux vaut refuser de démarrer que servir
		// des itinéraires calculés sur des octets mal interprétés.
		return err
	}
	log.Printf("graphe chargé : %d nœuds, %d arêtes, construit le %s, en %s",
		g.NumNodes(), g.NumEdges(), prov.BuiltAt, time.Since(start).Round(time.Millisecond))

	srv := &http.Server{
		Addr:              addr,
		Handler:           httpapi.New(generateloop.New(g), prov, g.BBox()),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("écoute sur %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("serveur : %v", err)
		}
	}()

	<-stop
	log.Print("arrêt en cours")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}
```

- [ ] **Step 10: Essai de bout en bout**

```bash
CGO_ENABLED=0 go build -o routed ./cmd/routed
./routed -graph graph.bin &
sleep 2

curl -s localhost:8080/healthz
curl -s localhost:8080/v1/regions | head -c 400

curl -s -X POST localhost:8080/v1/loops \
  -H 'Content-Type: application/json' \
  -d '{"start":{"lat":48.1173,"lon":-1.6778},"distance_m":12000,
       "activity":"trail","preferences":{"avoid_paved":0.8},"max_results":3}' \
  | python3 -m json.tool | head -40
```

Récupérer un `id` de la réponse, puis :

```bash
curl -s "localhost:8080/v1/loops/<ID>.gpx" -o boucle.gpx
head -20 boucle.gpx
```

Ouvrir `boucle.gpx` dans un visualiseur de trace et vérifier à l'œil que la boucle est plausible : elle ferme, elle évite les grands axes, elle fait à peu près la distance demandée.

- [ ] **Step 11: Mesurer**

```bash
time curl -s -X POST localhost:8080/v1/loops -H 'Content-Type: application/json' \
  -d '{"start":{"lat":48.1173,"lon":-1.6778},"distance_m":18000,"preferences":{"avoid_paved":0.8}}' \
  -o /dev/null
curl -s localhost:8080/metrics
```

Objectif de la spec : moins de 500 ms. Si le temps est très supérieur, ne pas optimiser au hasard — regarder d'abord `hent_request_duration_ms_total` rapporté au nombre de requêtes, puis réduire `numCandidates`, et enfin abaisser les pondérations maximales (`maxPavedWeight`, `maxTrafficWeight`), qui gonflent l'exploration de l'A* en rendant l'heuristique moins informative.

- [ ] **Step 12: Commit**

```bash
go vet ./... && go test ./...
git add -A
git commit -m "feat: API HTTP, export GPX et serveur routed"
```

---

## Après ce plan

Le service tourne en local et produit des GPX. Trois suites possibles, par ordre de valeur :

1. **Le mettre en ligne** — VPS, systemd, Caddy pour le TLS. C'est ce qui transforme le projet en service public, avec l'obligation d'attribution déjà en place et `graphbuild` open source qui couvre l'exigence ODbL de reproductibilité.
2. **Étape 2 de la feuille de route** — le dénivelé, via RGE ALTI. C'est le critère qui manque le plus vite quand on court pour de vrai.
3. **Améliorer la génération** — quand les boucles paraîtront naïves, remplacer `generateloop` par une recherche locale sur la formulation *orienteering*. La spec (§14) prévoit ce remplacement : ports et API ne bougent pas.
