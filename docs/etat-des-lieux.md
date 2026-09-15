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
  départ, une distance et des préférences. Chaque boucle porte un score :
  `distance_m`, `part_non_bitume`, `part_trafic`, `part_retracee`,
  `ecart_cible` ;
- `GET /v1/loops/{id}` — régénère une boucle précise et la rend en JSON, avec
  la demande d'origine et l'attribution. L'identifiant encode la demande, donc
  un lien vers une boucle se partage sans qu'aucun état ne soit conservé — c'est
  le déterminisme de la génération qui le permet ;
- `GET /v1/loops/{id}.gpx` — la même boucle, exportée en GPX. Le suffixe est ce
  qui distingue les deux représentations ;
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

## Revue finale de branche du front (15 septembre 2026)

Le socle du front a été bâti en huit tâches, chacune relue séparément. Une revue
finale a ensuite porté sur les 23 commits d'un coup, et a trouvé deux blocages
fonctionnels complets qu'aucune revue unitaire ne pouvait voir : ils naissent de
la rencontre entre deux tâches justes séparément. Le verdict complet et le
rapport de correction vivent dans
`.superpowers/sdd/2026-09-15-front-socle-plan/` (locaux, non versionnés).

**C1 (Critical) — l'écran de détail ne s'affichait jamais par lien partagé.**
L'effet de `/b/<id>` lisait l'état de la recherche en tête et l'écrivait par
`poser()` dans son `.then` : il s'invalidait lui-même, Svelte planifiait le
flush avant que le `.finally` chaîné ne soit dépilé, et le teardown posait
`annulee = true` à temps pour que la garde avale le `chargement = false`. La
ré-exécution sortait par la branche « boucle connue » sans jamais retoucher
`chargement`, et le template affiche `{#if chargement}` en premier : la boucle,
ses jauges et le bouton GPX ne s'affichaient jamais. Le chemin liste → détail
n'était pas touché, ce qui rendait le défaut invisible en usage nominal et
systématique sur tout rechargement — c'est-à-dire exactement ce que le repli
`200.html` et le `try_files` du Caddyfile servent à permettre. Corrigé en
sortant la lecture de l'état du graphe de dépendances : un effet ne dépend pas
de ce qu'il écrit.

**C2 (Critical) — « hors zone » sur l'écran de réglage était un cul-de-sac.**
Un départ hors Bretagne rend un 400, traduit en `HorsZone`. L'écran remplaçait
alors son bouton par un panneau dont la seule issue était un lien vers la page
où l'on se trouvait déjà, et l'état de la recherche étant un singleton de
module, le bouton « Tracer ma boucle » ne revenait jamais : seul un rechargement
complet débloquait. Corrigé côté écran plutôt que côté panneau — modifier une
coordonnée efface l'erreur et rend l'écran utilisable, ce qui est le modèle
mental réel. Un point hors zone n'offre désormais aucun bouton quand il n'y a
nulle part où aller : « Réessayer » y renverrait les mêmes coordonnées au même
serveur.

**I4 (Important) — la cause systémique : aucun test ne pouvait couvrir un
`$effect`.** `environment: 'node'` faisait compiler tout `.svelte.ts` en mode
serveur, où `$effect` est un no-op. Les cinquante lignes les plus denses de la
branche n'avaient aucune couverture possible, et c'est par là que C1 est passé
sous huit revues. `$state` seul fonctionnant en mode serveur, les tests de la
machine à états étaient verts et rassurants. Corrigé par deux projets Vitest,
client (jsdom) et serveur, ce qui a permis d'écrire le test de régression de C1.

**I1 et I5 — la validation du départ était décorative et muette.** `tracer()`
ne consultait pas `estCoordValide`, testée cinq fois pour un résultat que
personne ne lisait ; une latitude à 500 partait au serveur. Le message d'erreur
était un `<p>` nu, sans région live ni `aria-invalid`. Le bouton reste actif et
refuse en annonçant la raison : un bouton désactivé n'est pas focusable et
n'annonce pas pourquoi il l'est.

**I2 et I3 — des gardes que rien ne tenait.** Quatre mutations survivaient dans
la machine à états, dont les deux lignes qui étaient tout le contenu du commit
censé les poser. Et le gardien d'architecture ne couvrait pas `src/routes/` —
512 des 737 lignes d'interface — pendant qu'il gardait les 225 lignes de
`lib/ui/` ; ses imports à effet de bord et dynamiques lui échappaient aussi.
Les écrans sont désormais gardés sous la même règle que les composants.

**Ce que la revue a cherché sans le trouver**, et qui vaut d'être noté : aucune
incohérence de contrat entre le front et l'API Go — noms de champs, formes
imbriquées, ordre `[lon, lat]`, bornes — aucune interversion de champs de même
type dans le client HTTP, où les six mutations injectées sont toutes attrapées,
aucun `outline: none` nulle part, et la palette tient le RGAA AA sur douze
paires mesurées sur treize.

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

Le budget mémoire du §12, jusqu'ici estimé, est désormais mesuré : `routed`
répond **15,2 s après son démarrage**, le temps de charger les 294 Mo de
`graph.bin`, pour une empreinte de **593 Mo de RSS** une fois le graphe en
mémoire — ce qui impose un VPS d'au moins 2 Go.

Ce premier chiffre a une conséquence de conception : embarquer l'interface web
dans le binaire ferait payer quinze secondes d'indisponibilité à chaque
retouche de feuille de style. C'est ce qui a fait retenir un service statique
séparé, servi par Caddy, plutôt qu'un `//go:embed` dans `routed`.

## L'étape web — décisions prises

Le moteur est complet ; ce qui suit concerne l'interface publique, conçue avant
d'être codée. Le canevas de design vit dans `design/`, son README dit ce qui est
source et ce qui est dérivé.

**Le système de design tient dans un seul fichier.** `design/_themes.json` porte
vingt jetons en deux jeux de valeurs ; les maquettes n'emploient que des
`var(--jeton)`. Le thème clair est *dérivé* des écrans sombres par script, pas
redessiné — et les planches de documentation sont générées depuis la même
source, ratios de contraste compris.

**Conformité RGAA AA, mesurée et non estimée.** Un premier passage a trouvé six
non-conformités réelles : un gris à 3,62:1, quatre contours de composants entre
1,29 et 2,48:1 là où il en faut 3, et un lien identifié par la seule couleur
(1,68:1 contre le texte environnant). S'y ajoutaient deux manques structurels —
aucun état de focus dans tout le système, et des cibles tactiles à 34 px. Le
thème clair n'hérite d'aucun de ces contrastes : les vingt jetons ont été
recalculés, et un seul change de caractère — l'accent, car `#a8c98a` tombe à
1,6:1 sur du papier et ne peut y être ni lien, ni tracé, ni aplat.

**Fond de carte : MapLibre GL et les tuiles vectorielles `PLAN.IGN`** de la
Géoplateforme, servies sans clé ni quota. L'argument décisif est que ce jeu
porte `routier_chemin` comme couche distincte de `routier_route` : le style peut
dessiner le chemin *au-dessus* de la route et plus épais qu'elle, ce qui est
l'inverse d'un fond routier et exactement ce que trie hent. Les deux styles sont
générés depuis les mêmes jetons ; le générateur vérifie à chaque exécution que
les `source-layer` employées existent réellement. Attribution `© IGN` obligatoire.
Le style officiel publié par l'IGN n'est pas réutilisable tel quel : son JSON est
mal formé.

**Géocodage : la Base Adresse Nationale** (`api-adresse.data.gouv.fr`), pour la
recherche comme pour le géocodage inverse — ce dernier étant nécessaire dès
qu'on pose un départ par la géolocalisation ou en déplaçant la carte. Nominatim
a été écarté : sa limite d'une requête par seconde interdit l'autocomplétion, et
les maquettes de l'écran de recherche en dépendent. La géolocalisation est un
raccourci et jamais un passage obligé — un refus de permission ne doit rien
bloquer.

**La pile du front est arrêtée** — TypeScript, Svelte, Vite et MapLibre GL,
testés au Vitest. Trois dépendances en tout. Svelte plutôt que rien : six écrans,
trois états d'erreur, une autocomplétion et deux thèmes ne se tiennent pas à la
main sans que l'état finisse dispersé dans le DOM. Svelte plutôt que Preact :
compilé, sa syntaxe est du HTML augmenté — les maquettes s'y transposent presque
littéralement — et il intègre proprement un composant impératif comme MapLibre,
via ses actions et `onMount`. Le poids n'a pas départagé : MapLibre pèse
vingt fois un framework, il écrase ce budget à lui seul.

htmx a été écarté pour une raison précise : il remplace des fragments de DOM
venus du serveur, or une carte est l'inverse — un composant impératif qui tient
son viewport et ses gestes, et qu'il ne faut jamais remplacer. S'y ajoutent la
latence (chaque geste deviendrait un aller-retour, sur un service sans état donc
une régénération complète) et le terrain, où le réseau est souvent mauvais.

**Le front est servi en statique par Caddy**, qui proxifie `/v1/*` vers `routed`.
Ce qui a tranché : `routed` met 15,2 s à répondre après un redémarrage, donc
embarquer l'interface dans le binaire ferait payer quinze secondes
d'indisponibilité à chaque retouche de feuille de style. En développement, le
serveur Vite proxifie vers `routed`. Effet de bord heureux : le piège du
`//go:embed`, qui casse `go build ./...` sur un clone frais tant que le front
n'est pas bâti, disparaît entièrement.

**Vraies URLs et lien partageable.** `/b/<id>` affiche une boucle ; recharger ne
perd rien, le bouton Précédent fonctionne. Un avertissement accompagne le partage :
l'identifiant encode le point de départ, donc partager une boucle partie de chez
soi diffuse son adresse.

**Restent à trancher :** rien dans le design. Côté service, la latence (plus
bas) et les artefacts de déploiement du §12.

## Architecture — ce qui est garanti par un test

Le durcissement de l'architecture (`docs/clean-archi-back.md`) a ajouté des
gardiens à `go test ./...`, en plus de la règle de dépendance qui existait déjà
depuis l'étape 1. Sept règles, vérifiées à chaque exécution :

| Test | Ce qu'il empêche |
|---|---|
| `TestRegleDeDependance` | qu'une couche importe une couche interdite |
| `TestDependanceInterditeRespecteLesSegments` | que la règle ci-dessus confonde `internal/app2` avec `internal/app` |
| `TestAdaptateursCloisonnes` | qu'un adaptateur en importe un autre |
| `TestDomaineIgnoreLaSerialisation` | qu'un tag `json:` réapparaisse dans le domaine |
| Assertion de compilation (`internal/app/port/generator_test.go`) | que le moteur s'écarte du contrat entrant — au moment du build, avant même l'exécution des tests |
| `TestAdaptateurNImportePasLeMoteurConcret` | qu'un adaptateur importe le moteur concret plutôt que `port.LoopGenerator` |
| `TestNewConsommeLePort` | qu'un type concret remplace le port dans la signature de `httpapi.New` sans qu'aucun test ne bronche |

Et six témoins, qui figent ce qui ne doit pas bouger :

| Témoin | Ce qu'il fige |
|---|---|
| `TestContratJSONInchange` | les réponses HTTP publiques, octet pour octet |
| `TestFormatArtefactStable` | l'en-tête binaire de `graph.bin`, relu depuis un artefact versionné |
| `TestScoreDTOApparieLesChamps` | l'appariement des cinq champs du score entre domaine et DTO |
| `TestLoopRequestApparieLesChamps` | l'appariement des sept champs de la requête entrante entre DTO et domaine |
| `TestCodecAllerRetourProvenanceChampParChamp` | que l'aller-retour `Write`/`ReadGraph` perde un champ de `domain.Provenance`, ou l'intervertisse de façon asymétrique entre écriture et lecture |
| `TestVersEnTeteDistingueSourcesNilEtVide` | qu'un `Sources` nil et un `Sources` vide non-nil convergent vers la même valeur JSON dans l'en-tête écrit |

`TestFormatArtefactStable` ne couvre que la relecture d'un artefact déjà
produit : il ne peut rien dire du sens écriture. C'est exactement le trou
trouvé en revue de la tâche 4 — retirer un champ de la conversion aurait
laissé toute la suite verte, et un futur artefact se serait écrit avec ce
champ vide, en silence. `TestCodecAllerRetourProvenanceChampParChamp` ferme ce
trou en vérifiant que chaque champ de `Provenance` survit à l'aller-retour
`Write`/`ReadGraph`, avec des valeurs toutes distinctes pour qu'une perte ou
une interversion se voie.

Cet aller-retour a sa propre limite : une interversion symétrique entre deux
champs, appliquée à la fois dans `versEnTete` et dans `depuisEnTete`, écrit un
en-tête faux tout en relisant la valeur d'origine — le test reste vert. C'est
`TestFormatArtefactStable` qui l'attrape : il relit `testdata/artefact-v2.bin`,
écrit par un autre code, dont les champs sont donc à leur place d'origine, et
compare le résultat à `provenanceTemoin()`. Une conversion qui échange deux
champs des deux côtés les restitue croisés à cette relecture, et la
comparaison tombe. La garantie vient précisément de là : relire un fichier
qu'on n'a pas écrit soi-même, ce qu'un aller-retour par le même code ne peut
pas offrir.

`TestVersEnTeteDistingueSourcesNilEtVide` couvre un cas que la reconstruction
du type ne peut pas voir : `depuisEnTete` renvoie `nil` aussi bien pour un
`Sources` nil que pour un `Sources` vide non-nil, si bien que la distinction
ne survit que dans les octets bruts de l'en-tête écrit — c'est là, et pas
après relecture, qu'il faut la vérifier.

**Deux couplages entre adaptateurs restent tolérés**, déclarés explicitement dans
`couplagesToleres` :

- `osmsource → network` — construire le graphe CSR en flux est l'unique raison
  d'être de `osmsource` : la dépendance est son produit. La rompre demanderait un
  port `GraphBuilder` et une refonte du chemin d'ingestion de six millions de
  nœuds, qu'aucun témoin de performance n'encadre.
- `httpapi → gpxfile` — l'adaptateur HTTP compose l'exportateur GPX. Un port
  `LoopExporter` serait plus propre et peu coûteux ; c'est le candidat naturel si
  ce point est repris.

Ce sont des dettes identifiées, pas des oublis : tout couplage **nouveau** est
refusé, et une exception devenue inutile fait échouer le test.

## Les documents

- **[`design/README.md`](../design/README.md)** — le canevas de design : ce qui est source, ce qui est généré, comment régénérer.
- **`docs/design.md`** — la spec. Autorité sur toutes les décisions techniques : modèle de coût, invariant des pénalités, contrainte ODbL, feuille de route en 5 étapes.
- **[`docs/plan-etape-1.md`](plan-etape-1.md)** — le plan d'implémentation : neuf tâches, tout le code à écrire, en TDD. C'est lui qui a été déroulé, et il porte les corrections apportées en cours de route.
- **[`docs/journal-execution.md`](journal-execution.md)** — le journal de bord : chaque décision prise pendant l'exécution, avec sa justification et son coût si elle s'avérait fausse.
- **[`docs/revue-finale.md`](revue-finale.md)** — la revue de l'ensemble de la branche, celle qui a trouvé les deux défauts que les revues tâche par tâche ne pouvaient pas voir.
- **[`docs/clean-archi-back.md`](clean-archi-back.md)** — la conception du durcissement de l'architecture : l'audit des quatre failles, la cible, l'ordre d'exécution en six étapes.

Les briefs, rapports et revues détaillés de chaque tâche restent dans `.superpowers/sdd/`, ignoré par git et local à la machine. Le journal en contient la substance.

## Décisions prises pendant la mise en route

Cinq points que le plan laissait ouverts ont été tranchés pour ne pas bloquer. Les deux premiers ont depuis été résolus pour de bon ; les trois autres restent des choix réversibles.

1. ~~**Identité git**~~ — **résolu.** Les commits ont été réécrits sous `aurelien.morice@ik.me` avant toute publication, et le dépôt porte cette identité en local.
2. ~~**Chemin du module**~~ — **résolu.** La supposition initiale (`github.com/amorice/hent`) était fausse : le dépôt est `github.com/im-sellar/hent`. Le chemin a été corrigé partout, y compris là où il est codé en dur dans `internal/architecture_test.go`, et dans le plan d'implémentation.
3. ~~**Tests HTTP de la Task 9**~~ — **appliqué au plan.** Le point de départ des tests partait de 160 m du bord de la grille synthétique ; les waypoints d'une boucle de 4 km en seraient sortis et le test aurait échoué par intermittence. Il passe à `48.135 / -1.628`, au centre.
4. ~~**`TestGenerateVariantDonneAutreChose` (Task 7)**~~ — **appliqué au plan.** Il comparait deux boucles par leur longueur et leur nombre de nœuds ; sur une grille régulière, deux boucles distinctes ont très souvent ces deux valeurs identiques. Il compare désormais les ensembles d'arêtes.
5. **Sérialisation (Task 6)** — l'écriture élément par élément via `binary.Write` est conservée telle que le plan la spécifie. `graphbuild` tourne hors ligne, sa lenteur ne touche jamais le service. Si le build de la Bretagne dépasse deux minutes, passer à un `bufio.Writer` avec encodage en tampon.

Les points 3 et 4 étaient de vrais défauts du plan, trouvés avant exécution ; ils y ont été corrigés le 19 août, il n'y a plus rien à reporter au moment d'attaquer les Tasks 7 et 9.

## Dettes assumées du durcissement d'architecture

Trois points relevés pendant la revue de branche, jugés à corriger plus tard ou
jamais. Ils sont ici pour ne pas être redécouverts comme des défauts.

- `internal/adapter/network/csr/codec.go` — `depuisEnTete` perd la distinction
  entre une liste de sources vide et nulle, que `versEnTete` prend soin de
  préserver dans l'autre sens. Un cycle relecture puis réécriture transformerait
  donc `"sources":[]` en `"sources":null`. Aucun chemin d'appel ne fait ce cycle
  aujourd'hui : `graphbuild` écrit, `routed` lit. À traiter si un outil de
  relecture-réécriture apparaît.
- `internal/architecture_test.go` — `internal/testsupport` n'apparaît dans aucune
  règle de couches, et la vérification ne voit que les imports directs. Un chemin
  `app → testsupport → adapter` passerait donc sans alerte. Sans risque
  actuellement : `testsupport` ne dépend que du domaine.
- `internal/adapter/network/csr/format_test.go` — le `SHA256` de la provenance
  témoin fait quarante caractères hexadécimaux, c'est-à-dire un SHA-1. Cosmétique
  et confiné à un test, sans effet sur ce qu'il prouve.

## Constats mineurs laissés de côté

- `internal/domain/geo_test.go` — `TestBBoxContains` ne couvre pas les points situés exactement sur `Min` ou `Max`, alors que `Contains` utilise des comparaisons inclusives.
- `internal/adapter/network/csr/graph.go` — les offsets sont typés `[]uint32` plutôt que `domain.EdgeRef` ; `EdgeRange` convertit déjà, aucun bug latent, seulement une intention moins lisible.
- `internal/adapter/network/csr/graph.go` — `AddEdge`/`AddNode` ne valident toujours pas les bornes côté `Builder` (alimenté uniquement par `osmsource`, sans risque). `ReadGraph`, en revanche, valide désormais la cohérence de ce qu'il lit (voir I2 ci-dessus) : c'est lui qui ingère des données externes.
- `internal/adapter/osmsource` — le chemin qui saute un segment à nœud manquant (extrait hors bbox) n'est pas testé unitairement ; trois lignes sans état, `osmium` produit des extraits auto-contenus.

## Reprendre — y compris depuis une autre machine

### Par où commencer

L'étape 1 est **complète et revue**, et l'architecture a été durcie depuis (voir
plus haut). Deux chantiers restent ouverts.

**Le front, premier jet.** Le socle est livré : trois couches gardées par un
test d'imports, le client de l'API avec ses six variantes d'erreur, les
préférences persistées, et la machine à états de la recherche. Quatre écrans
fonctionnent — accueil, réglage, résultats, détail — et la chaîne va jusqu'au
téléchargement du GPX. Le point de départ se saisit encore en coordonnées
brutes : la carte, la géolocalisation et la recherche d'adresse font l'objet du
plan suivant, et le champ provisoire le dit à l'écran.

La branche a été revue dans son ensemble et les deux blocages trouvés sont
corrigés (voir la revue finale du front, plus haut). **Une re-revue de ces
corrections reste à faire** : elle doit refaire les mutations, vérifier que le
test de régression de C1 tombe bien sur le code d'avant, et chercher ce que les
corrections auraient pu introduire. La branche `worktree-front-socle` n'est donc
pas fusionnée.

Se bâtit par `make web`, se sert en copiant `web/build/` vers `/srv/hent/web`.
`deploy/Caddyfile` donne la configuration : l'API en proxy sur `/v1/*`, le reste
en repli vers `200.html` — sans quoi recharger `/b/<id>` donnerait un 404.

**Le déploiement**, §12 de `docs/design.md` : unité systemd, `/healthz`
distinguant « prêt » de « graphe chargé mais incohérent », sémaphore de
générations concurrentes, `recover()` autour de la génération, `IdleTimeout`,
et un lien vers le dépôt dans `/v1/regions` et le README pour clore
l'engagement ODbL. Le budget mémoire qui figurait ici est désormais mesuré :
593 Mo de RSS, 15,2 s de démarrage, donc un VPS de 2 Go au minimum. Reste la
latence — 738 ms médian contre 500 visés.

### Où vivent les documents

| Quoi | Où | Suit-il la machine ? |
|---|---|---|
| Conception du moteur | `docs/design.md` | oui, dans le dépôt |
| Conception du front | `docs/front-web.md` | oui, dans le dépôt |
| Cet état des lieux | `docs/etat-des-lieux.md` | oui, dans le dépôt |
| Plan d'implémentation, étape 1 | `docs/plan-etape-1.md` | oui, dans le dépôt |
| Plan du socle front | `docs/plan-front-socle.md` | oui, dans le dépôt |
| Durcissement de l'architecture | `docs/clean-archi-back.md` | oui, dans le dépôt |
| Journal d'exécution et décisions | `docs/journal-execution.md` | oui, dans le dépôt |
| Revue finale de branche | `docs/revue-finale.md` | oui, dans le dépôt |
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

- `internal/domain/geo_test.go` — `TestBBoxContains` ne couvre pas les points situés exactement sur `Min` ou `Max`, alors que `Contains` est inclusive.
- `internal/adapter/network/csr/graph.go` — offsets typés `[]uint32` au lieu de `domain.EdgeRef` ; aucun bug latent.
- `internal/adapter/httpapi/handler.go` — la garde de finitude ajoutée à la
  validation n'est couverte par aucun test HTTP : le décodeur JSON de la
  bibliothèque standard ne peut pas produire de `NaN`. Elle protège les
  appelants non-HTTP. Signalé comme tel dans le commentaire du code plutôt
  que déguisé en sécurité vérifiée.
- `internal/adapter/httpapi/handler.go` — `/metrics` agrège toutes les
  erreurs sous un seul compteur (pas de p95, 429 non comptés) : pas assez fin
  pour répondre aux questions de suivi du §12.
- `cmd/routed/main.go` — ni `ReadTimeout` ni `IdleTimeout` ; le chargement du
  graphe Bretagne prend une vingtaine de secondes avant que le port n'écoute,
  à documenter pour la procédure de mise à jour par `rsync` + redémarrage.
- `internal/adapter/httpapi/handler.go` — le 400 est rendu pour six causes
  distinctes, que le front ne peut pas distinguer et traduit toutes en « hors
  zone ». Un identifiant de boucle tronqué affiche donc « ce point est en dehors
  de la Bretagne ». Le défaut est dans la taxonomie d'erreurs de l'API, pas dans
  le front.
- `web/src/lib/ui/Jauge.svelte` — la piste de la jauge est à 1,54:1 sur le fond
  en sombre et 1,28:1 en clair, pour un seuil RGAA de 3:1. La valeur est aussi
  donnée en texte juste au-dessus, ce qui rend l'application stricte du critère
  discutable : c'est un arbitrage de design à trancher, pas une correction.
- `web/src/lib/assemblage.svelte.ts` — `changerTheme` et `theme` n'ont aucun
  appelant : le sélecteur de thème est du plan suivant, alors que `jetons.css`
  gère déjà les trois états.
- `web/src/routes/reglage/+page.svelte` — la reprise de focus après une erreur
  vise une région live, ce qui peut provoquer une double annonce chez certains
  lecteurs d'écran. Ne se mesure qu'avec un vrai lecteur d'écran.
- Le `<h1>` des états d'attente et d'erreur de `/b/<id>` n'a jamais été vu dans
  un navigateur : il dépend d'un `$effect` qui ne s'exécute qu'après hydratation,
  invisible en rendu serveur. Et au tout premier rendu client, avant l'exécution
  de l'effet initial, aucune branche ne correspond : fenêtre sans `<h1>`.
- `deploy/Caddyfile` n'a jamais été validé par Caddy lui-même — il n'est pas
  installé sur la machine de développement.
