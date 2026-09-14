> Revue de l'ensemble de la branche, après les neuf tâches. Elle a trouvé deux
> défauts majeurs qu'aucune revue tâche par tâche ne pouvait voir, parce qu'ils
> ne se manifestent qu'en faisant tourner le service sur le graphe réel.

# Revue finale — branche étape 1 (`1d75f77..bb0be7b`)

*Méthode : le diff de 5 462 lignes a été lu en plusieurs passes, mais l'essentiel de
la revue s'est faite sur l'arbre de travail plutôt que sur le fichier de diff — pour
une branche qui crée 43 fichiers, lire le résultat vaut mieux que lire l'ajout. La
suite (68 tests, `go vet`, `gofmt`) a été exécutée dans une copie isolée
(`git archive` vers un répertoire temporaire) ; le dépôt n'a jamais été modifié.
Cinq sondes de mesure ont été écrites dans cette copie puis supprimées : elles sont
citées ci-dessous avec leurs chiffres, et **elles ont été déterminantes** — deux des
constats les plus graves ne se voient pas à la lecture.*

---

## Verdict global

**Corrections requises avant fusion.**

Un constat Critical, cinq Important. Le code est de bonne facture — la discipline
sur le `NaN`, sur l'admissibilité de l'heuristique et sur la limitation de débit est
au-dessus de ce qu'on voit habituellement. Mais deux mécanismes centraux de la
conception sont inopérants en production sans qu'aucun test ne le dise, et ils ne se
révèlent qu'en faisant tourner le service sur l'artefact réel.

---

## Ce qui est solide

- **La discipline sur le `NaN` est réellement tenue, pas seulement annoncée.**
  `clamp01` traite `math.IsNaN` en premier cas explicite
  (`internal/domain/profile.go:62`), `loopsRequest.validate` vérifie la finitude
  *avant et séparément* des encadrements (`handler.go:66-75`), et surtout les
  **assertions** correspondantes exigent « fini et positif » et non « non négatif »
  (`profile_test.go:46`, `attrs_test.go:41`). C'est la partie difficile : une
  assertion aveugle au `NaN` est plus dangereuse qu'une assertion absente.

- **L'invariant d'admissibilité est documenté avec ses *deux* hypothèses**
  (`astar.go:56-71`) : coût minimal par mètre = 1, *et* longueur d'arête ≥ corde.
  La seconde est celle qu'on oublie ; elle est nommée, et la conséquence d'une
  source future qui l'enfreindrait est écrite. `TestFindPathEstOptimal` compare A*
  à un Dijkstra naïf sur 50 graphes tirés au sort : c'est la bonne forme de test
  pour cet invariant.

- **La limitation de débit est du travail sérieux.** `X-Forwarded-For` n'est lu que
  derrière un proxy explicitement déclaré, c'est la *dernière* entrée qui est
  retenue, la table de clients est plafonnée, et le raisonnement est écrit au-dessus
  du code (`ratelimit.go:55-104`). Les quatre tests, dont le pendant négatif
  (`TestLimiteDeDebitIgnoreXFFSansProxyDeConfiance`), verrouillent le tout.

- **`TestPreferableEstUnOrdreStrictFaible`** vérifie irréflexivité, asymétrie et
  transitivité — la propriété dont `sort.Slice` dépend et dont la violation ne
  produit ni erreur ni panique.

- **`ExploredNodesTotal` est compté chez l'implémenteur, via `defer`**, avec la
  raison écrite (`astar.go:78-84`, `port/network.go:23-29`) : un échec n'emporte pas
  de `Path`, donc l'appelant ne peut pas le compter. C'est le genre de décision
  qu'on ne retrouve pas six mois plus tard si elle n'est pas justifiée sur place.

- **Le codec borne les tailles lues avant d'allouer** (`codec.go:42-52`), avec le
  raisonnement et un test.

- **La frontière de couches n'est pas décorative.** `port.RouteNetwork` est resté à
  granularité grossière (`FindPath`, pas `Neighbors`), le profil traverse en
  scalaires (`domain.Weights`) et non en fonction, et `testsupport.Grille` permet à
  `app/` et `httpapi/` de se tester **sans** l'adaptateur réel. C'est exactement ce
  que le §4 promettait.

- **Le déterminisme est vérifié de bout en bout**, et `TestGetGPX` compare
  l'*intégralité* du tracé, pas seulement les extrémités — avec le commentaire qui
  explique pourquoi les extrémités ne prouveraient rien.

- `go vet` et `gofmt` propres, 68 tests verts en ~2 s, aucune ligne d'attribution
  dans les 32 messages de commit, `go 1.25` intact dans `go.mod` avec le piège
  `x/sync` documenté.

---

## Constats

### Critical

#### C1 — `NearestNode` accroche sur un fragment isolé : le service répond 404 là où il existe des boucles à dix mètres

`internal/adapter/network/csr/nearest.go:37`, utilisé par
`internal/app/generateloop/generate.go:79` (départ) et `:214` (waypoints).

`NearestNode` retourne le nœud géométriquement le plus proche **sans jamais vérifier
qu'il est routable**. Le graphe Bretagne réel compte 8 240 composantes connexes ; la
principale en rassemble 96,9 %. Un point de départ qui tombe sur un des fragments —
une impasse piétonne, une cour, un tronçon coupé — rend **toutes** les recherches
impossibles, et l'utilisateur reçoit :

```
HTTP 404  {"error":"aucune boucle trouvée pour ces critères"}
```

Ce n'est pas une question de critères. Mesures sur `graph.bin` (5 980 086 nœuds) :

| Point de départ | Nœud accroché | Composante |
|---|---|---|
| **48.117 / −1.677 — l'exemple littéral du §10 de la conception** | à 9 m | **3 nœuds** |
| 48.1173 / −1.6778 (celui des tests du dépôt) | à 6 m | 5 793 041 nœuds |
| Brest, Vannes, Saint-Malo, Cesson | 2 à 25 m | 5 793 041 nœuds |

Sur 40 départs tirés au hasard dans l'agglomération rennaise, **1 échoue
intégralement** (2,5 %), en 30 ms, sans qu'aucun A\* n'ait exploré plus d'un nœud.
Cinquante mètres de déplacement séparent un service qui marche d'un service qui
refuse.

C'est aussi pourquoi les mesures citées dans le brief (« 18 km depuis Rennes en
270-390 ms, 17 422 m ») sont exactes — j'ai reproduit `17 422 m` au mètre près — mais
prises sur `48.1173 / −1.6778`. L'exemple de la conception, lui, ne rend rien.

**Pourquoi c'est Critical :** le service rend une réponse fausse (« aucune boucle »
alors qu'il y en a) pour une classe entière de requêtes légitimes, avec un message
qui envoie l'utilisateur sur une mauvaise piste, et sans recours visible. Aucun test
ne peut l'attraper : `testsupport.Grille` est connexe par construction, et l'extrait
de Rennes est assez petit pour que les deux points testés soient dans la composante
principale.

**Correction.** Étiqueter les composantes au build (un parcours en largeur sur
12,5 M d'arêtes coûte quelques secondes, hors ligne), ne garder que l'identifiant de
composante ou même un seul bit « composante principale » par nœud, le sérialiser, et
faire porter la recherche de proximité sur les seuls nœuds routables. Variante
minimale sans changer le format : faire retourner à `NearestNode` les *k* plus
proches et laisser `tryLoop` réessayer — mais ça ne fait que déplacer le problème sur
les waypoints. La première solution est la bonne, et elle prépare le §12 (une
métrique « part des requêtes accrochées hors composante principale » devient
possible).

---

### Important

#### I1 — Le mécanisme anti-aller-retour est inopérant dans le seul sens où il compte

`internal/app/generateloop/generate.go:255`, `internal/adapter/network/csr/astar.go:151`,
`internal/adapter/osmsource/read.go:146-147`.

Un tronçon bidirectionnel devient deux arêtes dirigées, avec **deux `EdgeRef`
distincts**. `appendSegment` ne note comme consommé que le `EdgeRef` réellement
emprunté. Réemprunter le même tronçon **en sens inverse** utilise l'autre `EdgeRef`,
absent de `UsedEdges` : le multiplicateur `reuseFactor = 4` ne s'applique pas.

Or le sens inverse est précisément le cas que le §8 décrit — « l'A\* de W1→W2
réemprunte volontiers le chemin de D→W1 » se fait forcément à rebours.

Mesuré sur le graphe réel (Rennes, 5 km, `avoid_paved` 0.8), sur les cinq boucles
rendues :

| Boucle | Segments | Repassages **en sens inverse** (non pénalisés) | Repassages **dans le même sens** (pénalisés ×4) |
|---|---|---|---|
| 0 | 420 | 83 | **0** |
| 1 | 417 | 62 | **0** |
| 2 | 376 | 71 | **0** |
| 3 | 449 | 52 | **0** |
| 4 | 393 | 44 | **0** |

Les colonnes se lisent ensemble : la pénalité fonctionne parfaitement là où elle
s'applique (zéro repassage dans le même sens), et **100 % des repassages réels
tombent dans son angle mort**. Entre 11 % et 20 % de chaque boucle est un
aller-retour — 22 à 40 % des segments sont parcourus deux fois. Sur la grille
synthétique, une candidate monte à 70 %.

Le §8 dit de ce réglage qu'il est « celui qui sépare une vraie boucle d'un
aller-retour déguisé ». Il ne le fait pas.

**Origine : le plan.** `domain.PathOptions{UsedEdges map[EdgeRef]struct{}}` et le
test `TestFindPathPenaliseLaReutilisation` sont dictés mot pour mot par le plan
(lignes 881, 1026, 1151). Ce test construit `used` à partir des arêtes *aller* et
vérifie un détour — il passe, et il donne l'illusion que le mécanisme marche. Ça
n'abaisse pas la gravité : la fonctionnalité annoncée n'est pas livrée.

**Correction.** Construire au `Build()` (et reconstruire au `ReadGraph`) une table
`reverse []uint32` associant chaque arête à son inverse, et tester
`used[e] || used[rev[e]]` dans la boucle de relaxation. Coût mémoire : 50 Mo sur la
Bretagne, ou zéro en exploitant le fait que `assemble` ajoute les deux sens
consécutivement (il suffit de conserver l'appariement à travers le tri). Le port, le
domaine et le format ne bougent pas. Ajouter au passage un test de propriété : *une
boucle ne repasse pas par le même tronçon, dans un sens ou dans l'autre, au-delà
d'un seuil* — c'est le cinquième invariant qui manque au §11.

#### I2 — `ReadGraph` ne valide pas la cohérence interne de l'artefact

`internal/adapter/network/csr/codec.go:164-208`.

Les plafonds (`maxNodes`, `maxEdges`) empêchent une allocation démesurée, mais rien
ne vérifie que les offsets sont croissants, que `offsets[n] == numEdges`, ni que
chaque cible est `< numNodes`. Un artefact complet mais corrompu — transfert `rsync`
interrompu puis repris de travers, bit flip sur un disque sans ECC — passe la lecture
et fait paniquer `g.targets[e]` ou `settled[next]` **à la première requête**, dans une
goroutine d'`errgroup` : le processus meurt, pas seulement la requête.

C'est exactement la situation que le constat différé nº 4 écartait (« sans risque
tant que `csr` n'ingère pas de données externes ») — sauf que `ReadGraph` *est* le
point d'ingestion de données externes, et le §5 promet que `routed` « refuse de
démarrer plutôt que de lire des octets de travers ».

**Correction.** Une passe O(n+m) après lecture : monotonie des offsets, dernier
offset égal à `numEdges`, `max(targets) < numNodes`. Quelques dizaines de
millisecondes au démarrage, une erreur au lieu d'un `panic` en production.

#### I3 — `docs/etat-des-lieux.md` décrit une branche inachevée

`docs/etat-des-lieux.md:7-32`, pointé depuis `README.md:8`.

Le document, daté du 14 septembre 2026, annonce « 8 tâches sur 9 terminées », « il
manque le score lisible (tâche 8) et l'API HTTP (tâche 9) », donne la tâche 9 comme
non commencée dans son tableau, et sa section « Ce qui existe » s'arrête à
`csr/attrs.go`. Les tâches 8 et 9 sont livrées dans cette même branche. Le SHA cité
pour la tâche 4 (`9b954bf`) n'existe pas dans l'historique publié.

C'est le seul document versionné qui dit où en est le projet, et le README y envoie
le lecteur. Il ne peut pas partir dans cet état. La section « Constats mineurs
différés » de la fin, elle, est à jour — le document a été mis à jour par morceaux.

Au passage : la « question laissée ouverte » (§ *Le benchmark de l'A\**) devait être
tranchée à la tâche 9. Elle l'a été par la mesure, mais rien ne l'écrit. Voir M9.

#### I4 — La couche HTTP dépend de l'adaptateur de sérialisation ; deux ports du §4 n'ont jamais été écrits

`internal/adapter/httpapi/handler.go:20,120,231`.

`httpapi` importe `csr` pour `csr.Provenance`, et sérialise directement ce type dans
`/v1/regions`. Conséquence : **les noms de champs JSON du contrat public
(`built_at`, `sources`, `config_hash`) sont définis par les tags de structure du
codec binaire** (`codec.go:25-38`). Changer le format de l'artefact change l'API. Le
paquet dont le commentaire d'en-tête dit « les DTO définis ici sont volontairement
distincts des types du domaine : le contrat public doit pouvoir rester stable pendant
que le modèle interne évolue » fait exactement l'inverse pour cet endpoint.

Le §4 prévoyait `app/port/catalog.go` (« chargement / découverte des régions ») et
`app/port/exporter.go` (`TrackExporter`) : ni l'un ni l'autre n'existe, et
`internal/platform/` non plus. Le test d'architecture ne garde que `domain` et `app`
— rien ne surveille les dépendances adaptateur → adaptateur, et c'est par là que
c'est passé. L'esprit de la règle n'est donc pas tenu au-delà de sa lettre.

**Correction.** Déplacer `Provenance` / `Source` vers le domaine (ce sont des données
pures, pas une affaire de sérialisation), ou leur donner un DTO dans `httpapi` comme
pour tout le reste. L'absence de `port.TrackExporter` est défendable — un seul format
d'export, la dépendance est concrète et assumée — mais mérite d'être notée dans
l'état des lieux plutôt que découverte plus tard.

#### I5 — La limitation de débit compte des requêtes, le service dépense des secondes de CPU

`internal/adapter/httpapi/ratelimit.go:13-22`, `handler.go:31,84-85`,
`internal/app/generateloop/generate.go:96`.

Le §12 dit que « le rate limiting n'est pas optionnel » parce que « chaque requête
consomme des centaines de millisecondes de CPU ». Le limiteur livré est calibré en
requêtes (1/s, rafale de 5) alors que la ressource rare est le CPU :

- `requestTTL` vaut **5 s**, et `distance_m` est accepté jusqu'à **200 000 m** —
  une boucle de 200 km fait travailler 20 candidates × 4 branches × 5 itérations de
  dichotomie sur des A\* de 40 km chacun, jusqu'à épuisement du délai ;
- chaque requête ouvre son propre `errgroup` avec `SetLimit(runtime.NumCPU())`, et
  **rien ne borne le nombre de générations simultanées** ;
- une seule IP soutient donc en permanence ~1 cœur-seconde par seconde ; sur le VPS
  à 2 vCPU du §12, une dizaine d'IP saturent la machine.

`MaxNodes` n'est jamais renseigné par `generateloop` : le plafond effectif est le
`defaultMaxNodes = 400 000` de l'adaptateur, soit jusqu'à 32 M d'expansions par
requête avant que le délai ne coupe.

**Correction.** Un sémaphore global de générations concurrentes (≈ `NumCPU`, requête
refusée en 503 au-delà), un `MaxNodes` explicite passé par `generateloop` et
dimensionné sur la distance demandée, et un plafond de `distance_m` cohérent avec ce
qu'on veut réellement servir (50 km suffisent largement à l'étape 1). Compléter par
une métrique de requêtes refusées — voir M9.

---

### Minor

#### M1 — `NearestNode` n'est pas exacte

`nearest.go:47-72`. La boucle s'arrête un anneau après la première touche, or un nœud
d'un anneau plus lointain peut encore être plus proche : la diagonale d'une cellule
(≈ 669 m à 48°N) dépasse la largeur d'une cellule (≈ 372 m en longitude).
Contre-exemple exécuté : `NearestNode` retourne un nœud à **460 m** alors qu'il en
existe un à **376 m**. Sans conséquence visible sur un réseau dense, mais la condition
d'arrêt correcte est métrique, pas en anneaux : continuer tant que
`(ring−1) × côté_cellule_min < bestDist`.

#### M2 — Une vérification qui ne peut pas échouer, en production et dans un test

`generate.go:237` : `if loop.Nodes[len(loop.Nodes)-1] != start` — le dernier segment
est le résultat de `FindPath(current, start)`, il se termine sur `start` par
construction. Code mort. Même motif dans `generate_test.go:40`
(`TestGenerateRevientAuDepart`), déjà repéré. Le vrai invariant à tester est celui de
I1 ; celui-là ne teste rien.

#### M3 — Une troisième assertion tautologique

`osmsource/read_test.go:24` : `g.NumEdges() != 2*stats.Edges`. `stats.Edges` est
incrémenté dans la même itération que les deux `AddEdge` (`read.go:146-148`) :
l'égalité tient par construction quelles que soient les données. Elle attraperait la
suppression d'un `AddEdge`, rien d'autre — ce n'est pas ce que le commentaire annonce.

#### M4 — La chaîne d'attribution ODbL existe en deux exemplaires

`httpapi/handler.go:25` et `gpxfile/write.go:15` déclarent chacun un `attribution`
identique. C'est un engagement de licence avec deux sources de vérité, à la veille de
l'étape 2 qui doit y ajouter « IGN ». Une seule constante, dans `domain`.

#### M5 — `ConfigHash` promet plus qu'il ne tient

`osmsource/tags.go:62-72` : « deux graphes bâtis avec des règles différentes doivent
porter des empreintes différentes ». Faux — l'empreinte ne couvre que les trois
tables de classification. Modifier la règle `access` de `Classify` (`tags.go:126`) ou
la logique de `assemble` (découpage en segments, calcul de longueur) laisse
l'empreinte inchangée. La provenance ne porte par ailleurs aucune version du code
(commit, tag de build). L'engagement du §9 — « fournir les moyens de reconstruire » —
tient littéralement grâce au SHA256 de la source et au dépôt public, mais le
commentaire doit dire ce que le hash couvre vraiment, et un champ `builder_version`
fermerait la boucle pour un coût nul.

#### M6 — Identifiants en français dans du code non-test

La contrainte globale dit « commentaires et messages d'erreur en français,
identifiants en anglais ». `internal/testsupport/grille.go` est un paquet compilé, pas
un fichier `_test.go`, et tout y est en français (`Grille`, `NouvelleGrille`,
`voisins`, `arete`, `Relies`, `errPasDeChemin`). `osmsource.WayClassesPourTest`
(`tags.go:110`) est un identifiant exporté français en plein code de production. Les
noms de fonctions de test sont un cas à part et peuvent rester.

#### M7 — `Cache-Control` sur une réponse à un POST ne cache rien

`handler.go:191`. Le §10 justifie le déterminisme par la cachabilité HTTP ; aucun
intermédiaire ne réutilisera une réponse à un POST à corps variable. Soit exposer une
forme `GET /v1/loops?...` (le déterminisme rend l'URL suffisante), soit retirer
l'en-tête et reformuler l'argument : le déterminisme reste précieux pour la
régénération sans état du GPX et pour les tests, ce qui suffit à le justifier.

#### M8 — Commentaire recopié qui décrit autre chose

`domain/track.go:54-57` : les agrégats de `Loop` portent, après la phrase correcte
(« cumulés par la couche applicative »), la phrase copiée de `Path` sur le passage de
frontière. `Loop` ne traverse aucune frontière entrante. Supprimer la seconde phrase.

#### M9 — Les métriques ne peuvent pas répondre aux questions du §12

`handler.go:236-245`. `hent_errors_total` agrège 400, 404, 500 et 504 : impossible de
voir la population de C1. Les 429 ne sont pas comptés du tout (le limiteur
court-circuite avant le handler). La durée n'est qu'une somme, donc pas de p95 —
alors que c'est le p95 qui décide du §1. Enfin `hent_candidates_dropped_total` mélange
« abandon sur plafond de nœuds » et « dichotomie non convergente », alors que le §12
demande explicitement le premier. `/metrics` est par ailleurs public et exempté du
limiteur.

#### M10 à M13 — divers

- `ratelimit.go:48` : `http.Error` envoie un corps JSON sous `Content-Type:
  text/plain` — seule réponse de l'API qui ne respecte pas son propre format.
- `handler.go:275` : `fmt.Sscanf(index, "%d", &i)` accepte `"3xyz"`. `strconv.Atoi`.
- `handler.go:137` : `GET /v1/loops/{id}` sans `.gpx` rend aussi du GPX, le suffixe
  étant simplement retiré. Inoffensif, mais ce n'est pas le contrat du §10.
- `cmd/routed/main.go:72-77` : ni `ReadTimeout` ni `IdleTimeout`. Le chargement du
  graphe Bretagne prend **22 s** avant que le port n'écoute — le §12 parle de « mise
  à jour par `rsync` + redémarrage, quelques secondes ». À écrire quelque part, sans
  quoi la première mise à jour surprendra.

---

## Triage des six constats différés

| # | Constat | Décision |
|---|---|---|
| 1 | `architecture_test.go` — frontière de segment dans le préfixe d'import | **Peut rester.** Aucune couche `app2`/`domain2` n'existe ni n'est prévue. Correctif d'une ligne (`pkg == b \|\| strings.HasPrefix(pkg, b+"/")`), à faire en même temps que les autres retouches de tests plutôt que seul. |
| 2 | `geo_test.go` — `TestBBoxContains` ne couvre pas `Min`/`Max` exactement | **Peut rester.** Deux cas à ajouter, aucun risque : `Contains` est le seul usage et son inclusivité n'est ambiguë nulle part. |
| 3 | `csr/graph.go` — offsets typés `[]uint32` au lieu de références d'arête | **Peut rester.** `EdgeRange` convertit déjà en `domain.EdgeRef`, les deux types sont `uint32`, il n'y a pas de bug latent — seulement une intention moins lisible. |
| 4 | `csr/graph.go` — `AddEdge`/`AddNode` ne valident pas les bornes | **Peut rester pour le `Builder`, à corriger pour le lecteur.** La justification inscrite au dossier (« sans risque tant que `csr` n'ingère pas de données externes ») est fausse depuis la tâche 6 : `ReadGraph` ingère un fichier. C'est le constat **I2** ci-dessus, à corriger avant fusion. Le `Builder`, lui, n'est alimenté que par `osmsource` et peut attendre. |
| 5 | `graph_test.go` — coquille dans un commentaire | **À corriger avant fusion.** `graph_test.go:93` : « un débordement d'indice s'userait faire croire à une arête » — la phrase est cassée, elle rend le commentaire incompréhensible. Trente secondes. |
| 6 | `osmsource` — le chemin qui saute un segment à nœud manquant n'est pas testé | **Peut rester.** La justification tient (`osmium` produit des extraits auto-contenus), et le chemin est de trois lignes sans état. Un test unitaire sur `assemble` avec une map `coords` incomplète serait facile si l'occasion se présente — mais ce n'est pas ce qui manque le plus à cette branche. |

---

## Ce qui manquerait pour exploiter le service

Le §12 décrit « un binaire statique, un `graph.bin`, une unité systemd, Caddy
devant ». Rien dans le code ne l'empêche, mais cinq choses manquent pour le suivre
tel quel :

1. **Le critère de réussite du §1 n'est pas atteint sur la cible.** Mesuré sur le
   graphe Bretagne, départ Rennes, 18 km, `avoid_paved` 0.8, dix variantes :

   | | médiane | max |
   |---|---|---|
   | 11 cœurs (Apple Silicon) | 313 ms | 939 ms |
   | **`GOMAXPROCS=2`** (VPS 5 €/mois) | **738 ms** | **1,17 s** |

   Et ce cœur-là est plus rapide qu'un vCPU de VPS à 5 €. La cible est « moins de
   500 ms ». Les leviers sont ceux que le §6 annonce : baisser `numCandidates`
   (20 → 12), plafonner les pondérations maximales, ou renoncer au critère et le
   réécrire. Le §12 demande la métrique « temps par requête » précisément pour
   trancher ça — encore faut-il un p95 (M9).

2. **Aucun artefact de déploiement.** Ni unité systemd, ni `Caddyfile` d'exemple, ni
   cible de build reproductible (`CGO_ENABLED=0`, `GOOS=linux GOARCH=amd64`). Le
   drapeau `-trusted-proxies` doit impérativement être renseigné avec l'adresse de
   Caddy, faute de quoi toutes les requêtes partagent une seule clé de limiteur —
   ça ne se devine pas, ça doit être dans un fichier d'exemple.

3. **Aucun garde-fou de disponibilité.** `/healthz` répond `ok` inconditionnellement
   dès que le port écoute, donc ne distingue pas « prêt » de « graphe chargé mais
   incohérent ». Pas de sémaphore global (I5), pas d'`IdleTimeout`, pas de
   `recover()` autour de la génération — un `panic` dans une goroutine d'`errgroup`
   (voir I2) emporte le processus entier.

4. **Le budget mémoire n'est écrit nulle part.** L'état des lieux annonce « environ
   260 Mo en mémoire » ; mesurer le RSS réel après chargement et le consigner évite
   de découvrir un OOM sur un VPS à 4 Go partagé avec Caddy.

5. **Rien ne dit publiquement où reconstruire la base.** L'engagement ODbL du §9
   repose sur « fournir les moyens de reconstruire ». `/v1/regions` expose bien la
   provenance (nom de source, SHA256, `config_hash`) — c'est bien joué — mais sans
   lien vers le dépôt. Une ligne dans la réponse et dans le README ferme le sujet.
