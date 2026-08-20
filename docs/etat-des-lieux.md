# État des lieux — 20 août 2026

Où en est `hent`, et comment reprendre.

## En un coup d'œil

**3 tâches terminées sur 9** pour l'étape 1, toutes revues et approuvées. Le graphe se construit, se parcourt, et sait chiffrer le coût d'une arête selon les préférences de l'utilisateur. Il ne sait pas encore lire OpenStreetMap ni calculer d'itinéraire.

| # | Tâche | Code | Revue |
|---|---|---|---|
| 1 | Squelette, géométrie, garde-fou d'architecture | ✅ `435deb7` | ✅ approuvée |
| 2 | Graphe CSR et son constructeur | ✅ `03c4b9d` | ✅ approuvée |
| 3 | Profil, pondérations, fonction de coût | ✅ `f90450e` | ✅ approuvée |
| 4 | A*, index spatial, benchmark, port réseau | — | — |
| 5 | Lecture OSM et construction du graphe | — | — |
| 6 | Sérialisation `graph.bin` et binaire `graphbuild` | — | — |
| 7 | Génération de boucles | — | — |
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

## Reprendre

Reprendre le plan à la **Task 4** : A*, index spatial, benchmark et port réseau.

C'est la tâche la plus lourde du plan — quatre fichiers, dix étapes — et la
plus exigeante : son test de propriété compare l'A* à un Dijkstra naïf sur
cinquante graphes tirés au sort. Si ce test échoue, le diagnostic porte sur
l'admissibilité de l'heuristique, pas sur une coquille.

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
