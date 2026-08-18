
# hent — générateur de boucles trail « nature-aware »

*`hent` = « chemin » en breton.*

## 1. Objectif

Répondre à une demande de la forme « une boucle de 18 km au départ de chez moi,
un maximum de chemins et d'ombre, environ 500 m de dénivelé » par un tracé
GPX exploitable sur une montre.

Ce n'est pas du calcul d'itinéraire point à point. Générer une **boucle** de
longueur cible en maximisant un score est un problème d'optimisation
(proche de l'*orienteering problem*), que les acteurs existants — Komoot,
Strava — traitent mal.

### Critères de réussite

- Une boucle demandée à 18 km tombe entre 16,2 et 19,8 km (tolérance ±10 %).
- La réponse arrive en moins de 500 ms sur un VPS à 5 €/mois.
- Les boucles proposées sont réellement courues par l'auteur, sans retouche.
- Le service tourne publiquement pour quelques euros par mois.

### Non-objectifs

- Couverture nationale (voir §12, ce n'est pas une simple montée en charge).
- Navigation temps réel, suivi GPS, enregistrement d'activité.
- Compte utilisateur, historique, social.
- Interface soignée en V1 : l'API est le produit, l'UI viendra après.

## 2. Périmètre initial et feuille de route

Chaque étape produit quelque chose d'utilisable et tient en un à deux week-ends
de 4-6 h. On peut s'arrêter à n'importe quel étage.

| # | Contenu | Source à intégrer |
|---|---|---|
| 1 | Trail à pied, anti-bitume, longueur cible | OSM seul |
| 2 | Dénivelé (D+ cible, profil) | RGE ALTI |
| 3 | Paysage et points d'intérêt | OSM (score spatial) |
| 4 | Ombre et couvert forestier | BD Forêt V2 |
| 5 | Profils gravel puis vélo route | — (modèle de coût) |

Zone de départ : **Ille-et-Vilaine**. Geofabrik ne découpe pas la France par
département : on part de `bretagne-latest.osm.pbf` et on recoupe sur la bbox du
département avec `osmium extract` pour accélérer les itérations de build.

## 3. Architecture d'ensemble

Deux binaires, un artefact entre les deux.

```
        .osm.pbf  +  RGE ALTI  +  BD Forêt  +  POI OSM
                        │
                ┌───────▼────────┐
                │  graphbuild    │  hors ligne, poste de dev
                │                │  parse, filtre, enrichit, sérialise
                └───────┬────────┘
                        │
                   graph.bin        artefact versionné, ~qq centaines de Mo
                        │
                ┌───────▼────────┐
                │    routed      │  VPS, charge au démarrage
                │  serveur HTTP  │  ne fait plus que router
                └────────────────┘
```

Toutes les jointures géospatiales coûteuses se paient **une fois**, au build.
Le serveur ne lit qu'un graphe déjà digéré : pas de base de données, pas de
calcul raster en ligne. C'est ce qui rend l'hébergement quasi gratuit réaliste.
Mettre à jour les données = pousser un nouvel artefact.

### Décision structurante : attributs et coût sont séparés

Le graphe stocke des **attributs bruts et objectifs**. Il ne stocke **aucun
coût**. Le coût est calculé à la volée par un profil modulé par les préférences
de la requête.

Conséquences :

- Ajouter un profil vélo ne demande pas de reconstruire le graphe.
- Les préférences de l'utilisateur agissent en temps réel — c'est là que vit
  la valeur du produit.
- **En contrepartie**, on renonce aux accélérations type *contraction
  hierarchies*, qui exigent des poids figés au précalcul. Sans importance à
  l'échelle d'un département (A* en dizaines de ms). Bloquant à l'échelle
  nationale : il faudrait alors une technique à poids personnalisables
  (famille *customizable route planning*). Voir §12.

## 4. Structure des paquets

Clean architecture, avec une règle explicite : **les ports sont à granularité
grossière, et ce qui traverse la frontière est de la donnée, jamais du
comportement.** Un A* parcourt des millions d'arêtes par requête ; une
abstraction dans cette boucle coûte un facteur 3 à 5 irrécupérable.

```
cmd/graphbuild            pipeline de construction
cmd/routed                serveur HTTP

internal/
  domain/                 types purs, aucun import hors stdlib
    geo.go                  Coord, BBox, distances
    track.go                Path, Loop, Segment
    profile.go              Activity, Profile, Preferences → Weights
    score.go                Score et sa décomposition

  app/
    port/
      network.go            RouteNetwork : FindPath, NearestNode, BBox
      catalog.go            chargement / découverte des régions
      exporter.go           TrackExporter
    generateloop/           stratégie de génération — métier, pas infra
    exporttrack/

  adapter/
    network/csr/            graphe CSR + A* → implémente port.RouteNetwork
    osmsource/              lecture .pbf              (graphbuild)
    elevation/              RGE ALTI                  (graphbuild)
    landcover/              BD Forêt                  (graphbuild)
    httpapi/                handlers, DTO, validation
    gpxfile/                export

  platform/                 config, logs, serveur, arrêt propre
```

Deux applications de la règle de granularité :

- **`RouteNetwork` expose `FindPath`, pas `Neighbors`.** Une boucle = 3-4
  appels de chemin. L'A* et le parcours du CSR restent entiers dans
  l'adaptateur : quelques dizaines d'appels d'interface par requête au lieu de
  millions.
- **Le profil traverse la frontière en coefficients, pas en fonction.** Pas de
  `func(Edge) float64` (indirection par arête), mais un `domain.Weights`,
  structure de scalaires dont l'adaptateur fait le produit scalaire avec les
  attributs stockés.

`generateloop` est en `app/` et non dans l'adaptateur : c'est la valeur du
produit, elle doit être testable sur un graphe jouet de vingt nœuds et
remplaçable (waypoints d'abord, recherche locale ensuite) sans qu'aucun
adaptateur ne bouge.

**Convergence avec Go :** l'idiome « définir l'interface chez le consommateur »
*est* l'inversion de dépendance. Les ports vivent dans `app/port` ; le paquet
CSR ignore leur existence et expose un type concret. C'est `main` qui câble.

On ne reprend pas le nommage littéral `entities/ usecases/ adapters/` : en Go
on nomme par domaine. La règle de dépendance, elle, est vérifiée par un test
(§11), pas par la discipline.

## 5. Modèle du graphe

Représentation **CSR** (*compressed sparse row*) : un tableau d'offsets indexé
par nœud, un tableau d'arêtes contiguës. Compact, contigu en mémoire donc
rapide à parcourir, sérialisable quasiment tel quel.

Attributs par arête :

| Attribut | Origine | Étape |
|---|---|---|
| longueur | géométrie OSM | 1 |
| classe de revêtement | `surface`, `highway`, `tracktype` | 1 |
| classe de voie | `highway` | 1 |
| exposition au trafic | `highway` + `maxspeed` | 1 |
| pente, **signée selon le sens** | RGE ALTI | 2 |
| densité de canopée (0-255) | BD Forêt V2 | 4 |
| score de proximité POI (0-255) | POI OSM | 3 |

La pente signée impose que **les arêtes soient dirigées** : un tronçon
bidirectionnel devient deux arêtes, montée et descente. Le tableau d'arêtes
double — sans importance à l'échelle d'un département, mais à décider
maintenant plutôt qu'à l'étape 2.

Le fichier `graph.bin` porte un numéro de version de format ; `routed` refuse
de démarrer sur une version qu'il ne comprend pas, plutôt que de lire des
octets de travers. Son en-tête porte aussi la provenance des données (§9).

## 6. Modèle de coût

```
coût(arête) = longueur × ( 1 + Σ wᵢ × pénalitéᵢ )    pénalitéᵢ ∈ [0,1], wᵢ ≥ 0
```

Le coût reste homogène à une **distance ressentie** : une départementale de
1 km « coûte » 4 km de ressenti. Lisible, débogable, et surtout le facteur est
toujours ≥ 1 — donc le coût minimal par mètre vaut exactement 1, donc
l'heuristique de l'A* est la distance à vol d'oiseau brute, admissible par
construction et sans calibrage fragile.

### Règle invariante : tout critère est une pénalité, jamais une récompense

Favoriser l'ombre par un bonus reviendrait à introduire des coûts négatifs.
**Dijkstra et A* deviennent alors faux**, silencieusement : des chemins absurdes
sans la moindre erreur. La formulation correcte inverse la question — on ne
bonifie pas l'ombre, on pénalise le soleil :

```
pénalité_exposition = 1 − canopée
pénalité_fadeur     = 1 − score_POI
```

Même classement des chemins à une constante près, mais tout reste positif.
Cette règle doit figurer en commentaire au-dessus de la fonction de coût :
chaque nouveau critère invitera à la violer.

### Arbitrage coût / performance

Plus l'écart entre meilleur et pire terrain est large (w = 8 sur le bitume),
plus l'heuristique devient molle relativement aux coûts réels, et plus A*
re-dérive vers Dijkstra. **Les pondérations sont aussi un curseur de
performance.** À instrumenter (compteur de nœuds explorés, §11), pas à deviner.

## 7. A* et heuristique

A* explore en priorité le nœud minimisant `f(n) = g(n) + h(n)`, où `g` est le
coût réel depuis le départ et `h` une estimation **optimiste** du coût restant.
Contrairement à Dijkstra, qui explore un disque autour du départ, A* explore une
ellipse orientée vers la cible : même résultat, fraction du travail.

`h` doit être *admissible* — ne jamais surestimer — sous peine de perdre la
garantie d'optimalité. Ici : distance à vol d'oiseau × coût minimal par mètre,
soit la distance à vol d'oiseau brute grâce à la formulation du §6.

## 8. Génération de boucles

Le chemin de coût minimal d'un point à lui-même est de ne pas bouger : un
algorithme de plus court chemin ne peut pas, seul, produire une boucle. Il faut
lui imposer une forme.

```
     Pour une boucle de L, viser un cercle de rayon
     r ≈ L / (2π × détour)      détour initial ≈ 1.3
                   W2
                ╱      ╲            1. tirer un angle θ
             ╱            ╲         2. poser W1,W2,W3 à θ, θ+120°, θ+240°
           │       ·D      │        3. accrocher au nœud le plus proche
             ╲            ╱         4. router D→W1→W2→W3→D en A*
                ╲      ╱            5. mesurer, corriger r, recommencer
               W3    W1
```

Trois difficultés, et c'est là qu'est le vrai travail :

**Empêcher les allers-retours.** Sans précaution, l'A* de W1→W2 réemprunte
volontiers le chemin de D→W1 s'il est bon. Remède : passer à l'A* l'ensemble
des arêtes déjà consommées, avec un multiplicateur de coût. Un facteur modéré
autorise la réutilisation là où il n'y a pas d'alternative — un pont, un col —
tout en la décourageant. C'est le réglage qui sépare une vraie boucle d'un
aller-retour déguisé.

**Atteindre la longueur demandée.** Le détour de 1,3 est une devinette : en
relief les chemins serpentent, en plaine ils vont droit. On mesure la longueur
obtenue et on corrige `r` par dichotomie ; 3-4 itérations suffisent pour entrer
dans la tolérance de ±10 %.

**Ne pas proposer cinq fois la même boucle.** Deux angles voisins convergent
souvent vers le même itinéraire. Comparaison par indice de Jaccard sur les
ensembles d'arêtes ; au-delà de 70 % de recouvrement, doublon.

### Coût et score sont deux choses distinctes

Le **coût** est interne et guide l'A* ; il n'a aucun sens pour un humain.
Le **score** est calculé après coup sur la boucle finie et il est fait pour
être lu — c'est lui qui trie les candidates et qui est exposé dans l'API :

```json
{ "distance_m": 17840, "denivele_positif_m": 512,
  "part_non_bitume": 0.87, "part_ombragee": 0.61,
  "points_interet": 4, "ecart_cible": -0.03 }
```

L'utilisateur voit *pourquoi* une boucle lui est proposée, et sur quel critère
elle est moins bonne que la suivante.

### Budget de temps

20 candidates × 4 segments = 80 A* par requête. À 20 ms pièce, 1,6 s en
séquentiel — inacceptable. Les candidates sont indépendantes : une goroutine
par candidate, `errgroup` avec limite de parallélisme, et on retombe autour de
200 ms sur un petit VPS.

Deux garde-fous dès le départ : un plafond de nœuds explorés par A* (au-delà,
la candidate est abandonnée plutôt que de faire souffrir le serveur), et un
`context` avec deadline propagé jusque dans la boucle d'exploration, pour qu'un
client qui raccroche libère immédiatement le CPU.

## 9. Sources de données et contrainte juridique

| Étape | Donnée | Source | Licence |
|---|---|---|---|
| 1 | réseau | extraits `.osm.pbf` Geofabrik | ODbL |
| 2 | altitude | RGE ALTI (IGN), maille 5 m | Licence Ouverte Etalab 2.0 |
| 3 | POI | OSM (`tourism=viewpoint`, `natural=water`, `historic=*`…) | ODbL |
| 4 | canopée | BD Forêt V2 (IGN), vectoriel | Licence Ouverte Etalab 2.0 |

Depuis le 1er janvier 2021 les données publiques de l'IGN sont gratuites sous
Licence Ouverte Etalab 2.0, qui autorise adaptation, transformation et création
de produits dérivés, y compris commerciaux, moyennant attribution.
Téléchargement sur cartes.gouv.fr.

**Pourquoi BD Forêt et non Corine Land Cover :** l'unité minimale de collecte de
Corine est de 25 ha, beaucoup trop grossière pour qualifier l'ombre d'un
sentier. BD Forêt V2 est vectorielle, bien plus fine, et distingue feuillus et
conifères — un feuillu ne donne pas d'ombre en février.

### ODbL : le partage à l'identique se déclenche à l'usage public

`graph.bin` fusionne du réseau OSM avec des données IGN : c'est une
**Derivative Database** au sens de l'ODbL (la FAQ de l'OSMF cite explicitement
l'enrichissement d'un graphe de routage par des données tierces). Or l'ODbL
déclenche le partage à l'identique sur l'**usage public**, pas seulement sur la
distribution : exposer l'API publiquement suffit, même sans jamais mettre le
fichier en téléchargement.

La licence offre une alternative : fournir sur demande *soit* la base dérivée,
*soit* **les moyens de la reconstruire**. `graphbuild` étant open source et les
sources publiques, on est couvert par la seconde branche — sans héberger
300 Mo.

**Conséquence sur le design : `graphbuild` doit être reproductible.** L'en-tête
de `graph.bin` enregistre la version exacte de chaque source (date de l'extrait
Geofabrik, millésime des dalles IGN) et le hash de la configuration. Sans cela,
« les moyens de reconstruire » sont une fiction. Bénéfice collatéral : quand une
boucle sera absurde, on saura sur quelles données.

L'attribution — « © les contributeurs OpenStreetMap » et « IGN » — figure dans
**chaque réponse de l'API**, pas seulement en pied de page d'une UI future.

## 10. API

```http
POST /v1/loops
{
  "start":       {"lat": 48.117, "lon": -1.677},
  "distance_m":  18000,
  "tolerance":   0.10,
  "activity":    "trail",
  "preferences": { "avoid_paved": 0.8, "elevation": 0.5,
                   "shade": 0.3, "scenery": 0.4 },
  "max_results": 5
}
```

```
GET  /v1/loops/{id}.gpx      export
GET  /v1/regions             couverture et millésime des données
GET  /healthz  /metrics
```

Deux décisions non évidentes :

**Les préférences sont des curseurs 0-1, pas les poids internes.** Le client
exprime une intention, le serveur la traduit en `domain.Weights`. Le modèle de
coût peut être entièrement révisé sans casser un client, et sans exposer des
nombres qui n'ont de sens qu'en interne.

**La même requête rend le même résultat.** La génération tire des angles au
hasard ; un hasard réellement aléatoire rendrait l'API non cachable, non
testable, et ferait perdre à l'utilisateur la boucle qu'il aimait en
rechargeant. Le PRNG est donc initialisé par un hash de la requête :
déterministe, cachable en HTTP, reproductible en test. Un champ `variant`
optionnel permet de demander explicitement autre chose.

## 11. Tests

L'enjeu, à 4-6 h le week-end, est de pouvoir reprendre le fil sans relire le
projet entier. La stratégie en découle.

- **`domain/` et `app/` sur graphes jouets** : une vingtaine de nœuds dessinés à
  la main dans le test. Aucun fichier, exécution instantanée. C'est là que
  vivent les tests qu'on relit.
- **Test d'architecture** : parcourt les imports, échoue si `domain/` ou `app/`
  importe `adapter/`. Vingt lignes, et la frontière ne dérive jamais.
- **CSR contre un micro-extrait `.pbf` commité** (une commune, quelques Mo),
  golden files.
- **Tests de propriété sur la génération** — le filet de sécurité principal.
  Pour N requêtes tirées au sort, la boucle produite : revient à son point de
  départ, est connexe, respecte la tolérance de distance, n'emprunte que des
  arêtes autorisées par le profil. Ces quatre invariants attrapent l'essentiel
  des régressions.
- **Benchmark sur l'A\*** comptant les nœuds explorés : garde-fou contre
  l'arbitrage du §6. Le jour où une pondération monte et où l'exploration
  double, le benchmark le dit — au lieu d'un timeout découvert en production.

## 12. Déploiement et exploitation

Un binaire statique, un `graph.bin`, une unité systemd. Ni base de données ni
conteneur obligatoire. VPS 5 €/mois avec 4 Go de RAM : largement suffisant pour
un département. Caddy devant pour le TLS automatique. Mise à jour des données
par `rsync` + redémarrage, quelques secondes.

- **Le rate limiting n'est pas optionnel** : chaque requête consomme des
  centaines de millisecondes de CPU, c'est un vecteur de déni de service
  trivial. Limiteur en mémoire par IP au début.
- **Métriques dès le premier jour** : temps par requête, nœuds explorés par A*,
  taux de candidates abandonnées sur plafond. Sans cela les pondérations se
  règlent à l'aveugle.

### Ce qui casse à l'échelle nationale

À noter pour ne pas se croire à une simple montée en charge : le graphe de la
France ne tient pas confortablement en RAM sur une petite machine, et surtout
le choix des poids dynamiques (§3) interdit les accélérations classiques. Passer
à l'échelle nationale demanderait une technique à poids personnalisables, ce
qui est un projet en soi — pas une optimisation.

## 13. Risques et points ouverts

| Risque | Traitement |
|---|---|
| Le détour de 1,3 converge mal en zone peu dense | dichotomie sur `r`, plafond d'itérations, échec explicite plutôt que boucle hors tolérance |
| Boucles « bêtes » malgré la pénalisation de réutilisation | c'est le motif attendu de bascule vers une recherche locale (§14) |
| Coût du plaquage BD Forêt sur des millions d'arêtes | index spatial au build ; le build est hors ligne, on peut se permettre des minutes |
| Qualité inégale des tags `surface` dans OSM | valeurs par défaut par `highway`, et contribuer à OSM sur sa zone |
| Nom `hent` : validation du breton | faire relire par un bretonnant avant de communiquer |

## 14. Suite naturelle

La génération par waypoints produira des boucles correctes mais parfois naïves.
La suite n'est pas un concurrent de ce design mais son prolongement : remplacer
la couche `generateloop` par une vraie recherche locale (recuit simulé ou GRASP)
sur la formulation *orienteering*. Le refactor est contenu — la représentation
du graphe, les ports et l'API ne bougent pas.

---

## Références

- [OSMF — Licence and Legal FAQ](https://osmfoundation.org/wiki/Licence/Licence_and_Legal_FAQ)
- [OSM Wiki — Open Database License](https://wiki.openstreetmap.org/wiki/Open_Database_License)
- [Référentiel à grande échelle (RGE) — data.gouv.fr](https://www.data.gouv.fr/datasets/referentiel-a-grande-echelle-rge)
- [Les données ouvertes de l'IGN](https://ideo.ternum-bfc.fr/les-donnees-ouvertes-de-lign)
- [IGN — ouverture des données publiques](https://www.batiactu.com/edito/ign-rend-libres-et-gratuites-toutes-ses-donnees-publiques-60916.php)
