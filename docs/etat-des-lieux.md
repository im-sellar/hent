# État des lieux — 14 septembre 2026

Où en est `hent`, et comment reprendre.

## En un coup d'œil

**9 tâches sur 9** terminées, toutes revues et approuvées, plus une revue finale
de branche qui a corrigé six constats supplémentaires (un Critical, cinq
Important — voir plus bas). Le moteur lit OpenStreetMap, construit son graphe,
le sérialise en artefact versionné, calcule un itinéraire optimal entre deux
points, génère des boucles, calcule un score lisible et sert tout cela en HTTP
avec export GPX.

Sur la Bretagne entière : **5 980 086 nœuds, 12 517 776 arêtes**. L'artefact
`graph.bin` a grossi à environ 294 Mo (format v2, voir plus bas) et se
construit en un peu moins d'une minute. Le §12 de la conception ne pariait que
sur un département dans un VPS à 4 Go : la marge reste largement suffisante.

| # | Tâche | Code | Revue |
|---|---|---|---|
| 1 | Squelette, géométrie, garde-fou d'architecture | ✅ `435deb7` | ✅ approuvée |
| 2 | Graphe CSR et son constructeur | ✅ `03c4b9d` | ✅ approuvée |
| 3 | Profil, pondérations, fonction de coût | ✅ `f90450e` | ✅ approuvée |
| 4 | A*, index spatial, benchmark, port réseau | ✅ `bcc19a4` | ✅ approuvée |
| 5 | Lecture OSM et construction du graphe | ✅ `bc8032b` | ✅ approuvée |
| 6 | Sérialisation `graph.bin` et binaire `graphbuild` | ✅ `a089e71` | ✅ approuvée |
| 7 | Génération de boucles | ✅ `3ca036a` | ✅ approuvée |
| 8 | Score lisible | ✅ `eff04e5` | ✅ approuvée |
| 9 | API HTTP, export GPX, binaire `routed` | ✅ `bb0be7b` | ✅ approuvée |

`go vet ./... && go test ./...` passe sur l'ensemble du dépôt, `go test -race`
également.

Les SHA cités sont ceux de l'historique publié. Le SHA de la tâche 4, cité par
erreur dans une version précédente de ce document (`9b954bf`, qui n'existe pas
dans l'historique), a été corrigé en `bcc19a4`.

## Ce qui existe

```
internal/domain/geo.go               Coord, BBox, HaversineM
internal/domain/track.go             NodeRef, EdgeRef, Path, Loop
internal/domain/profile.go           Surface, WayClass, Preferences → Weights
internal/domain/score.go             Score lisible d'une boucle, ordre de préférence
internal/adapter/network/csr/
    graph.go                         Graph (CSR), Builder, composante connexe principale
    attrs.go                         EdgeAttrs et sa fonction de coût
    astar.go                         A*, table des arêtes inverses (Graph.Reverse)
    nearest.go                       Index spatial, NearestNode (routable uniquement)
    codec.go                         Sérialisation graph.bin (format v2) et sa validation
internal/adapter/osmsource/          Lecture d'un extrait .osm.pbf, classification des voies
internal/adapter/gpxfile/            Export GPX
internal/adapter/httpapi/            Handler HTTP, DTO, limitation de débit
internal/app/generateloop/           Génération de boucles par waypoints et dichotomie
internal/app/port/                   RouteNetwork, interface consommée par app/
internal/testsupport/                Grille synthétique, double de test de RouteNetwork
internal/architecture_test.go        garde-fou de la règle de dépendance
cmd/graphbuild/                      construit graph.bin hors ligne
cmd/routed/                          sert le générateur en HTTP
```

L'invariant du modèle de coût — chaque pénalité dans `[0,1]`, chaque poids
positif, donc facteur toujours `≥ 1` — est verrouillé par des tests qui
exigent un résultat **fini**, et pas seulement « non négatif ».

Le garde-fou d'architecture a été vérifié comme mordant réellement : import interdit ajouté dans `domain`, échec constaté, import retiré.

## L'API HTTP

`cmd/routed` charge `graph.bin` au démarrage et sert :

- `POST /v1/loops` — génère jusqu'à `max_results` boucles pour un point de
  départ, une distance et des préférences ;
- `GET /v1/loops/{id}.gpx` — régénère et exporte une boucle précise en GPX,
  sans état côté serveur : c'est le déterminisme de la génération qui le
  permet ;
- `GET /v1/regions` — bbox couverte et provenance des données (nom de source,
  SHA256, `config_hash`), pour honorer l'obligation ODbL de reconstructibilité ;
- `GET /healthz`, `GET /metrics`.

La limitation de débit (1 req/s, rafale de 5 par IP) ne lit `X-Forwarded-For`
que derrière un proxy explicitement déclaré (`-trusted-proxies`), sans quoi
seule l'adresse de connexion compte.

## Revue finale de branche (14 septembre 2026)

Avant de considérer l'étape 1 terminée, une revue a porté sur l'ensemble de la
branche plutôt que tâche par tâche, et a trouvé six défauts qu'aucune revue
unitaire ne pouvait voir — dont deux structurels. Ils ont tous été corrigés,
et le rapport complet (mesures, sabotages de test, `fichier:ligne`) vit dans
`.superpowers/sdd/2026-08-18-hent-plan-etape1/final-review.md` et
`final-fix-report.md` (locaux, non versionnés).

**C1 (Critical) — `NearestNode` accrochait des impasses.** Le graphe Bretagne
compte 8 240 composantes connexes ; la principale en rassemble 96,9 %. Sans
vérification de routabilité, un départ tombant sur un fragment isolé rendait
**toutes** les recherches impossibles — c'était le cas du point d'exemple du
§10 (`48.117 / -1.677`), qui accrochait une composante de 3 nœuds. Corrigé en
étiquetant la composante connexe principale au build (union-find) et en
restreignant `NearestNode` aux nœuds qui en font partie. Depuis la correction,
ce point trouve à nouveau des boucles.

**I1 (Important) — le mécanisme anti-aller-retour ne marche pas au retour.**
Un tronçon bidirectionnel porte deux `EdgeRef` (un par sens) ; la pénalité de
réutilisation ne s'appliquait qu'à celui réellement emprunté, jamais à son
inverse. Sur une boucle réelle (Rennes, 5 km), 22 à 40 % du tracé était un
aller-retour non pénalisé. Corrigé en associant chaque arête à son inverse
(`Graph.Reverse`) et en pénalisant les deux. Mesuré après correction sur une
boucle de 18 km depuis Rennes : la part de tracé repassée en sens inverse
tombe à moins de 3 % (elle reste non nulle par construction — un pont, un col,
un aller-retour parfois nécessaire — mais n'est plus systématique).

**I2 — un artefact corrompu faisait paniquer le service.** `ReadGraph`
vérifie désormais la cohérence interne (offsets croissants, dernier offset
égal au nombre d'arêtes, cibles dans les bornes) et non plus seulement la
taille des compteurs.

**I3 — ce document.** Il annonçait 8 tâches sur 9 et un SHA inexistant alors
que les neuf étaient livrées. C'est la correction que vous lisez.

**I4 — le contrat JSON de `/v1/regions` dépendait du format binaire.**
`httpapi` sérialisait directement `csr.Provenance`, dont les tags viennent du
codec `graph.bin`. Un DTO propre à `httpapi` (`provenanceDTO`) découple
maintenant les deux, comme pour le reste des réponses.

**I5 — la limitation de débit comptait des requêtes, pas du CPU.** À 200 km de
distance maximale acceptée, une poignée de requêtes suffisait à saturer un
VPS à 2 vCPU avant même d'atteindre la limite en nombre. La distance maximale
est ramenée à 50 km — largement au-dessus d'une boucle de trail réelle — ce
qui ferme la porte sans toucher au limiteur ni à la génération.

**Conséquence côté format.** C1 et I1 changent tous deux la disposition
binaire de `graph.bin` (table des arêtes inverses, bit de composante
principale par nœud) : la version de format passe à **2**. Un artefact
construit avant cette revue est refusé au démarrage plutôt que mal
interprété — il faut relancer `graphbuild`.

## Performance mesurée

Boucle de 18 km depuis Rennes (`48.1173 / -1.6778`), `avoid_paved` 0.8 :
**270 à 390 ms** sur le graphe Bretagne complet. C'est le chiffre qui répond à
la question laissée ouverte à la tâche 9 (voir plus bas) sur du matériel de
développement.

Sur du matériel représentatif du VPS visé par le §12 (`GOMAXPROCS=2`), la
même mesure monte à **738 ms en médiane, 1,17 s au maximum** — au-delà des
500 ms visés par la conception. Ce n'est pas encore corrigé : les leviers
identifiés sont ceux du §6 (réduire `numCandidates` de 20 à 12, plafonner les
pondérations maximales), à trancher avant de considérer le critère de
performance du §1 comme atteint sur la cible de déploiement réelle.

## Les documents

- **`docs/design.md`** — la spec. Autorité sur toutes les décisions techniques : modèle de coût, invariant des pénalités, contrainte ODbL, feuille de route en 5 étapes.
- **Le plan d'implémentation** vit hors du dépôt, dans le vault Obsidian :
  `~/Library/CloudStorage/OneDrive-Hellowork/obsidian-vault/claude/hent/2026-08-18-hent-plan-etape1.md`
  9 tâches, tout le code Go à écrire, en TDD. C'est lui qui a été déroulé.
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
- `internal/adapter/network/csr/graph.go` — les offsets sont typés `[]uint32` plutôt que `domain.EdgeRef` ; `EdgeRange` convertit déjà, aucun bug latent, seulement une intention moins lisible.
- `internal/adapter/network/csr/graph.go` — `AddEdge`/`AddNode` ne valident toujours pas les bornes côté `Builder` (alimenté uniquement par `osmsource`, sans risque). `ReadGraph`, en revanche, valide désormais la cohérence de ce qu'il lit (voir I2 ci-dessus) : c'est lui qui ingère des données externes.
- `internal/adapter/osmsource` — le chemin qui saute un segment à nœud manquant (extrait hors bbox) n'est pas testé unitairement ; trois lignes sans état, `osmium` produit des extraits auto-contenus.

## Reprendre — y compris depuis une autre machine

### Par où commencer

L'étape 1 de la feuille de route est **complète et revue**. La suite dépend
de la feuille de route du §12 de `docs/design.md` : artefacts de déploiement
(unité systemd, `Caddyfile` d'exemple, cible de build reproductible),
garde-fous de disponibilité (`/healthz` qui distingue « prêt » de « graphe
chargé mais incohérent », sémaphore global de générations concurrentes,
`recover()` autour de la génération, `IdleTimeout`), budget mémoire mesuré
(RSS réel après chargement, pas seulement estimé), et un lien vers le dépôt
dans `/v1/regions` et le README pour clore l'engagement ODbL de
reconstructibilité. Le travail de performance mentionné plus haut (500 ms
visés, 738 ms mesurés sur `GOMAXPROCS=2`) en fait partie.

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

Un artefact `graph.bin` construit avant la revue finale du 14 septembre 2026
(format v1) est refusé par `routed` : `graphbuild` doit être relancé pour
produire un artefact au format v2 (table des arêtes inverses, composante
connexe principale).

### Un piège à connaître avant de toucher aux dépendances

`go get` et `go mod tidy` remontent d'eux-mêmes la directive `go` du module à
1.26, parce que la version récente de `golang.org/x/sync` l'exige. La
dépendance est donc épinglée à `v0.8.0`, compatible avec Go 1.25.

Si la directive change sous vos pieds après un `go mod tidy`, c'est ça. Soit
rétablir l'épinglage, soit assumer le passage à Go 1.26 — ce qui contredirait
la contrainte inscrite dans le plan d'implémentation.

### La question laissée ouverte — tranchée

Le benchmark de l'A* donnait 646 nœuds explorés par chemin en pondération neutre
contre 1809 en anti-bitume, soit un rapport de 2,8× — au-delà du seuil d'alerte
inscrit dans la conception, mais mesuré sur un graphe aléatoire où l'heuristique
est structurellement handicapée. La tâche 9 a tranché par la mesure sur le
réseau réel : voir « Performance mesurée » ci-dessus. Verdict : le critère est
atteint sur poste de développement, pas encore sur la cible VPS visée par le
§12 — à reprendre avant la mise en production.

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

**Le même motif, une troisième fois, trouvé en revue finale de branche.** Le
mécanisme anti-aller-retour (I1 ci-dessus) était lui aussi verrouillé par un
test aveugle : `TestFindPathPenaliseLaReutilisation` construisait son ensemble
d'arêtes déjà utilisées à partir du trajet aller, le seul cas où le mécanisme
fonctionne réellement. Le pendant en sens inverse
(`TestFindPathPenaliseLaReutilisationEnSensInverse`) comble ce point mort.

## Constats mineurs différés

Relevés en revue, non bloquants, à balayer avant de passer aux étapes suivantes :

- `internal/architecture_test.go` — la comparaison de préfixe d'import ne vérifie pas la frontière de segment : une future couche `internal/app2` serait faussement vue comme important `internal/app`.
- `internal/domain/geo_test.go` — `TestBBoxContains` ne couvre pas les points situés exactement sur `Min` ou `Max`, alors que `Contains` est inclusive.
- `internal/adapter/network/csr/graph.go` — offsets typés `[]uint32` au lieu de `domain.EdgeRef` ; aucun bug latent.
- `internal/adapter/httpapi/handler.go` — la garde de finitude ajoutée à la
  validation n'est couverte par aucun test HTTP : le décodeur JSON de la
  bibliothèque standard ne peut pas produire de `NaN`. Elle protège les
  appelants non-HTTP. Signalé comme tel dans le commentaire du code plutôt
  que déguisé en sécurité vérifiée.
- `internal/adapter/httpapi/handler.go` — `GET /v1/loops/{id}` sans le
  suffixe `.gpx` rend aussi du GPX ; inoffensif, mais hors du contrat
  documenté.
- `internal/adapter/httpapi/handler.go` — `/metrics` agrège toutes les
  erreurs sous un seul compteur (pas de p95, 429 non comptés) : pas assez fin
  pour répondre aux questions de suivi du §12.
- `cmd/routed/main.go` — ni `ReadTimeout` ni `IdleTimeout` ; le chargement du
  graphe Bretagne prend une vingtaine de secondes avant que le port n'écoute,
  à documenter pour la procédure de mise à jour par `rsync` + redémarrage.
