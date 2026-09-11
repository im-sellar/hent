# État des lieux — 11 septembre 2026 (soir)

Où en est `hent`, et comment reprendre.

## En un coup d'œil

**7 tâches sur 9** écrites, dont 6 revues et approuvées. Le moteur lit
OpenStreetMap, construit son graphe, le sérialise en artefact versionné,
calcule un itinéraire optimal entre deux points, et **génère des boucles**.

Il manque le score lisible (tâche 8) et l'API HTTP (tâche 9) pour que tout cela
soit utilisable autrement que par des tests.

Sur la Bretagne entière : **5 980 086 nœuds, 12 517 776 arêtes**, artefact de
245 Mo construit en 56 secondes, soit environ 260 Mo en mémoire. Le §12 de la
conception ne pariait que sur un département dans un VPS à 4 Go : la marge est
bien plus large que prévu.

| # | Tâche | Code | Revue |
|---|---|---|---|
| 1 | Squelette, géométrie, garde-fou d'architecture | ✅ `435deb7` | ✅ approuvée |
| 2 | Graphe CSR et son constructeur | ✅ `03c4b9d` | ✅ approuvée |
| 3 | Profil, pondérations, fonction de coût | ✅ `f90450e` | ✅ approuvée |
| 4 | A*, index spatial, benchmark, port réseau | ✅ `9b954bf` | ✅ approuvée |
| 5 | Lecture OSM et construction du graphe | ✅ `bc8032b` | ✅ approuvée |
| 6 | Sérialisation `graph.bin` et binaire `graphbuild` | ✅ `a089e71` | ✅ approuvée |
| 7 | Génération de boucles | ✅ `9e77459` | ⚠️ **pas encore relue** |
| 8 | Score lisible | — | — |
| 9 | API HTTP, export GPX, binaire `routed` | — | — |

`go vet ./... && go test ./...` passe sur les trois paquets existants.

Les SHA cités sont ceux de l'historique publié. Les commits ont été réécrits une fois, pour corriger l'adresse e-mail d'auteur, avant le premier `push` — d'anciens SHA peuvent traîner dans des notes de travail.

## Ce qui existe

```
internal/domain/geo.go               Coord, BBox, HaversineM
internal/domain/track.go             NodeRef, EdgeRef, Path
internal/domain/profile.go           Surface, WayClass, Preferences → Weights
internal/adapter/network/csr/
    graph.go                         Graph (CSR), Builder
    attrs.go                         EdgeAttrs et sa fonction de coût
internal/architecture_test.go        garde-fou de la règle de dépendance
```

L'invariant du modèle de coût — chaque pénalité dans `[0,1]`, chaque poids
positif, donc facteur toujours `≥ 1` — est verrouillé par des tests qui
exigent un résultat **fini**, et pas seulement « non négatif ». La nuance
n'est pas cosmétique : voir la section suivante.

Le garde-fou d'architecture a été vérifié comme mordant réellement : import interdit ajouté dans `domain`, échec constaté, import retiré.

## Les documents

- **`docs/design.md`** — la spec. Autorité sur toutes les décisions techniques : modèle de coût, invariant des pénalités, contrainte ODbL, feuille de route en 5 étapes.
- **Le plan d'implémentation** vit hors du dépôt, dans le vault Obsidian :
  `~/Library/CloudStorage/OneDrive-Hellowork/obsidian-vault/claude/hent/2026-08-18-hent-plan-etape1.md`
  9 tâches, tout le code Go à écrire, en TDD. C'est lui qu'on déroule.
- **Le journal d'exécution** est dans `.superpowers/sdd/2026-08-18-hent-plan-etape1/progress.md` (ignoré par git, local à la machine) : briefs, rapports, revues, et les décisions prises en cours de route.

## Décisions prises pendant la mise en route

Cinq points que le plan laissait ouverts ont été tranchés pour ne pas bloquer. Les deux premiers ont depuis été résolus pour de bon ; les trois autres restent des choix réversibles.

1. ~~**Identité git**~~ — **résolu.** Les commits ont été réécrits sous `aurelien.morice@ik.me` avant toute publication, et le dépôt porte cette identité en local.
2. ~~**Chemin du module**~~ — **résolu.** La supposition initiale (`github.com/amorice/hent`) était fausse : le dépôt est `github.com/im-sellar/hent`. Le chemin a été corrigé partout, y compris là où il est codé en dur dans `internal/architecture_test.go`, et dans le plan d'implémentation.
3. ~~**Tests HTTP de la Task 9**~~ — **appliqué au plan.** Le point de départ des tests partait de 160 m du bord de la grille synthétique ; les waypoints d'une boucle de 4 km en seraient sortis et le test aurait échoué par intermittence. Il passe à `48.135 / -1.628`, au centre.
4. ~~**`TestGenerateVariantDonneAutreChose` (Task 7)**~~ — **appliqué au plan.** Il comparait deux boucles par leur longueur et leur nombre de nœuds ; sur une grille régulière, deux boucles distinctes ont très souvent ces deux valeurs identiques. Il compare désormais les ensembles d'arêtes.
5. **Sérialisation (Task 6)** — l'écriture élément par élément via `binary.Write` est conservée telle que le plan la spécifie. `graphbuild` tourne hors ligne, sa lenteur ne touche jamais le service. Si le build de la Bretagne dépasse deux minutes, passer à un `bufio.Writer` avec encodage en tampon.

Les points 3 et 4 étaient de vrais défauts du plan, trouvés avant exécution ; ils y ont été corrigés le 19 août, il n'y a plus rien à reporter au moment d'attaquer les Tasks 7 et 9.

## Constats mineurs laissés de côté

- `internal/architecture_test.go:38` — la comparaison de préfixe d'import ne vérifie pas la frontière de segment. Une future couche `internal/app2` serait faussement détectée comme important `internal/app`. Aucun impact sur les quatre couches actuelles.
- `internal/domain/geo_test.go` — `TestBBoxContains` ne couvre pas les points situés exactement sur `Min` ou `Max`, alors que `Contains` utilise des comparaisons inclusives.

## Reprendre — y compris depuis une autre machine

### Par où commencer

1. **Faire relire la tâche 7.** Son code est commité et sa suite est verte,
   mais aucun relecteur ne l'a examinée : la session s'est arrêtée entre
   l'implémentation et la revue. Diff à relire : `b15cbe8..9e77459`.
2. Puis la **tâche 8** (score lisible), puis la **tâche 9** (API, export GPX,
   serveur).

### Où vivent les documents

| Quoi | Où | Suit-il la machine ? |
|---|---|---|
| Conception | `docs/design.md` | oui, dans le dépôt |
| Cet état des lieux | `docs/etat-des-lieux.md` | oui, dans le dépôt |
| Plan d'implémentation | vault Obsidian, `claude/hent/2026-08-18-hent-plan-etape1.md` | oui, OneDrive |
| Journal d'exécution et décisions | vault Obsidian, `claude/hent/2026-09-11-hent-journal-execution.md` | oui, OneDrive |
| Briefs, rapports, revues détaillés | `.superpowers/sdd/` | **non**, ignoré par git |

Le journal d'exécution contient toutes les décisions prises en cours de route
et leur justification. C'est le document à lire avant de reprendre.

### Ce qu'il faut réinstaller sur une nouvelle machine

Rien n'est nécessaire pour faire tourner la suite de tests : l'extrait de
Rennes est versionné dans `testdata/`.

Pour reconstruire un artefact régional en revanche :

```sh
brew install osmium-tool
curl -L -o data/bretagne-latest.osm.pbf \
  https://download.geofabrik.de/europe/france/bretagne-latest.osm.pbf
CGO_ENABLED=0 go build -o graphbuild ./cmd/graphbuild
./graphbuild -in data/bretagne-latest.osm.pbf -out graph.bin
```

`CGO_ENABLED=0` n'est pas optionnel : sans lui, le build réclame `pkg-config`
et `zlib` à cause d'une dépendance transitive activée par cgo. Le chemin pur Go
fait le même travail et rend le binaire statique et cross-compilable, comme le
prévoit la conception.

### Un piège à connaître avant de toucher aux dépendances

`go get` et `go mod tidy` remontent d'eux-mêmes la directive `go` du module à
1.26, parce que la version récente de `golang.org/x/sync` l'exige. La
dépendance est donc épinglée à `v0.8.0`, compatible avec Go 1.25.

Si la directive change sous vos pieds après un `go mod tidy`, c'est ça. Soit
rétablir l'épinglage, soit assumer le passage à Go 1.26 — ce qui contredirait
la contrainte inscrite dans le plan d'implémentation.

### La question laissée ouverte

Le benchmark de l'A* donne 646 nœuds explorés par chemin en pondération neutre
contre 1809 en anti-bitume, soit un rapport de 2,8× — au-delà du seuil d'alerte
inscrit dans la conception. Mais cette mesure porte sur un graphe aléatoire, où
la distance à vol d'oiseau n'est pas corrélée à la topologie : l'heuristique y
est structurellement handicapée et le chiffre ne prédit pas le comportement sur
un vrai réseau.

**C'est à la tâche 9 qu'il faudra trancher**, lors de l'essai de bout en bout :
si le temps de réponse dépasse les 500 ms visés par la conception, il faudra
abaisser les pondérations maximales — au prix d'itinéraires moins tranchés dans
leur évitement du bitume.

L'invariant à ne jamais perdre de vue, quel que soit l'ordre choisi ensuite : **tout critère se formule comme une pénalité positive, jamais comme une récompense**. Un coût négatif rend l'A* faux silencieusement, sans erreur, avec des itinéraires absurdes. C'est expliqué au §6 de `docs/design.md`.

## Défauts du plan corrigés en cours de route

Le plan d'implémentation n'est pas parole d'évangile : trois défauts y ont été
trouvés par la revue et corrigés **à la source**, pour qu'ils ne survivent pas
à une réexécution.

**Le plus instructif — le bornage aveugle au `NaN`.** La fonction qui ramène
les préférences dans `[0,1]` s'écrivait :

```go
switch {
case v < 0: return 0
case v > 1: return 1
default:    return v
}
```

En Go, **toute** comparaison impliquant `NaN` est fausse. Un `NaN` traversait
donc les deux cas et sortait intact, contaminait les pondérations, puis le
coût (`longueur × (1 + NaN) = NaN`), et l'A* aurait cessé de relaxer la
moindre arête — sans erreur, sans test rouge, puisque `NaN < x` est également
faux.

Le vrai enseignement est ailleurs : **les deux tests censés verrouiller cet
invariant étaient aveugles pour la même raison.** Ils vérifiaient
`poids < 0` et `coût < longueur`, deux comparaisons également fausses pour
`NaN`. Une assertion « n'est pas négatif » ne dit rien d'un `NaN` — il faut
exiger « est fini et positif ». Une assertion aveugle est plus dangereuse
qu'une assertion absente : elle donne l'illusion de la couverture.

Le même motif a ensuite été trouvé, par recherche, dans la validation de
l'API (Task 9), où les encadrements de latitude, longitude, distance et
tolérance étaient tous aveugles au `NaN` — dans la fonction dont le rôle est
précisément de valider. Corrigé par anticipation.

**Deux tests intermittents par construction.** Le plan faisait partir les
boucles de test à 160 m du bord de la grille synthétique, d'où les waypoints
d'une boucle de 4 km seraient sortis ; et il comparait deux boucles par leur
longueur et leur nombre de nœuds, deux valeurs très souvent identiques sur une
grille régulière. Les deux sont corrigés dans le plan.

## Constats mineurs différés

Relevés en revue, non bloquants, à balayer avant de considérer l'étape 1 terminée :

- `internal/architecture_test.go` — la comparaison de préfixe d'import ne vérifie pas la frontière de segment : une future couche `internal/app2` serait faussement vue comme important `internal/app`.
- `internal/domain/geo_test.go` — `TestBBoxContains` ne couvre pas les points situés exactement sur `Min` ou `Max`, alors que `Contains` est inclusive.
- `internal/adapter/network/csr/graph.go` — `AddEdge`/`AddNode` ne valident pas les bornes : un `NodeRef` hors bornes panique dans `Build()` au lieu de remonter une erreur. Sans risque tant que `csr` n'ingère pas de données externes.
- `internal/adapter/network/csr/graph_test.go:30` — coquille de rédaction dans un commentaire.
- `internal/adapter/httpapi/handler.go` (à venir en Task 9) — la garde de
  finitude ajoutée à la validation n'est couverte par aucun test : le décodeur
  JSON de la bibliothèque standard ne peut pas produire de `NaN`, le format ne
  l'admettant pas. Elle protège les appelants non-HTTP. Signalé comme tel dans
  le commentaire du code plutôt que déguisé en sécurité vérifiée.
