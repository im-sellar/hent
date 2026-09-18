# Le front web de hent

L'interface publique. Le moteur existe, le design est fait, l'API est stable et
gardée par des témoins. Reste à écrire ce qui donnera au projet son premier
usage réel.

## Ce qu'on livre

Six écrans, deux thèmes, une carte. On pose un départ, on règle une distance et
une aversion au bitume, on reçoit cinq boucles, on en choisit une, on télécharge
un GPX. Pas de compte, pas d'installation, pas de traçage.

**Critère de réussite :** quelqu'un qui ouvre le site sur son téléphone un
samedi matin repart avec un GPX dans sa montre sans avoir eu à réfléchir.

## La pile

| | |
|---|---|
| Langage | TypeScript |
| Rendu | Svelte 5, via SvelteKit avec `adapter-static` |
| Build | Vite |
| Carte | MapLibre GL JS |
| Tests | Vitest |
| Service | statique, servi par Caddy, qui proxifie `/v1/*` vers `routed` |

Trois dépendances de production. Le raisonnement de ces choix est consigné dans
`docs/etat-des-lieux.md`, section « L'étape web » : il n'est pas rejoué ici.

**SvelteKit avec `adapter-static`** produit du statique pur — aucun Node en
production. Ce qu'il apporte et qui compte : la page d'accueil est **pré-rendue
en HTML**, donc indexable. C'est précisément la page dont le contenu ne dépend
de rien. Le reste part en repli SPA.

## Architecture

Les mêmes trois couches que le back, avec des tranches verticales dans `app/`.
C'est une symétrie voulue : qui comprend `internal/` comprend `web/src/lib/`.

```
web/src/lib/
├── domaine/          types et règles pures — n'importe RIEN
│   ├── boucle.ts       Boucle, Score, durée à une allure donnée
│   ├── depart.ts       Coord, Depart, validation
│   └── reglages.ts     bornes 2–50 km, valeurs par défaut
├── app/              cas d'usage et état — n'importe que domaine/ et ports
│   ├── ports.ts        MoteurDeBoucles, Geocodeur, Preferences, Carte
│   ├── depart/         poser depuis la position, chercher une adresse, poser sur la carte
│   ├── generation/     lancer, et la machine à états des résultats
│   └── boucle/         ouvrir — par la mémoire ou par l'API
├── infra/            les implémentations — peut tout importer
│   ├── hent-api.ts     MoteurDeBoucles, par fetch
│   ├── ban.ts          Geocodeur, par api-adresse.data.gouv.fr
│   ├── maplibre.ts     Carte
│   └── stockage.ts     Preferences, par localStorage
└── ui/               composants Svelte — n'importe que app/ et domaine/
```

`src/routes/` porte les écrans, et tombe sous la même règle que `ui/` : il lit
`domaine/`, `app/` et `ui/`, jamais `infra/`. Seul `lib/assemblage.svelte.ts`,
le point de câblage, connaît les implémentations et les distribue.

### Quatre ports, et la règle qui les borne

Un port **seulement devant une frontière technique réelle** — le réseau, le
stockage, une bibliothèque tierce. Jamais pour du code interne. Quatre suffisent
et ce nombre ne devrait pas beaucoup grandir.

```ts
interface MoteurDeBoucles {
  generer(demande: Demande, signal?: AbortSignal): Promise<Boucle[]>
  ouvrir(id: string, signal?: AbortSignal): Promise<{ boucle: Boucle; demande: Demande }>
  urlGPX(id: string): string
}

interface Geocodeur {
  chercher(texte: string, autour?: Coord, signal?: AbortSignal): Promise<Lieu[]>
  nommer(point: Coord, signal?: AbortSignal): Promise<string | null>
}

interface Preferences {
  lire(): Reglages | null
  ecrire(r: Reglages): void
}

interface Carte {
  centrer(point: Coord, zoom?: number): void
  afficherBoucles(boucles: Boucle[], selectionnee: string | null): void
  marquerDepart(point: Coord | null): void
  surDeplacement(rappel: (centre: Coord) => void): () => void
}
```

`urlGPX` rend une URL plutôt que des octets : le téléchargement est un lien que
le navigateur suit, pas un `fetch` qu'on relaie.

`Carte` est une **façade à ordres**, pas un état que `app/` posséderait. MapLibre
est impératif ; essayer de le rendre déclaratif reviendrait à se battre contre
lui. `surDeplacement` rend sa fonction de désabonnement — un abonnement qu'on ne
peut pas rompre fuit à chaque navigation.

### Le test d'architecture, porté depuis le Go

`internal/architecture_test.go` parcourt les imports avec `go/parser`.
L'équivalent, en Vitest, lit les `import` des fichiers de `web/src/lib/` et
échoue si :

- `domaine/` importe quoi que ce soit du projet,
- `app/` importe `infra/` ou `ui/`,
- `ui/` importe `infra/`.

Sans ce test, la règle tient trois semaines. Avec, elle tient. Il doit échouer
avant d'être vert, comme tous les gardiens de ce dépôt : on introduit
temporairement une violation, on constate, on la retire.

## L'état

Quatre stores. Le dernier est une machine à états explicite, et c'est ce qui
évite que les écrans d'attente et d'erreur soient bricolés au cas par cas.

```ts
depart:    { coord: Coord; libelle: string } | null
reglages:  { distanceM: number; eviterBitume: number }   // retenus en localStorage
theme:     'auto' | 'sombre' | 'clair'
resultats: { statut: 'vide' } 
         | { statut: 'calcul' } 
         | { statut: 'ok'; boucles: Boucle[]; demande: Demande }
         | { statut: 'erreur'; erreur: ErreurMoteur }
```

`reglages` et `theme` sont persistés ; `depart` et `resultats` ne le sont pas.
Un départ vieux de trois jours n'a aucun sens, et des résultats périmés encore
moins.

**`localStorage` peut échouer** — navigation privée, stockage bloqué, quota. Le
port `Preferences` avale l'erreur et rend `null` : l'application démarre sur les
valeurs par défaut plutôt que de planter. Un test le couvre avec un double qui
lève.

## Les écrans

`/` accueil · `/depart` poser · `/reglage` régler · `/boucles` choisir ·
`/b/<id>` une boucle

Les maquettes font foi pour l'apparence — canevas publié, page « Parcours ».
Cette spec ne les redécrit pas ; elle dit ce que le code doit faire.

### Poser un départ, trois moyens

**La géolocalisation est un raccourci, jamais un passage obligé.** Un refus de
permission ne bloque rien : l'écran reste utilisable par la recherche et par la
carte. C'est le cinquième panneau de la planche États.

1. **Recherche d'adresse** — champ flottant, anti-rebond de 300 ms, et
   **annulation de la requête précédente à chaque frappe**. Sans annulation, une
   réponse lente écrase une réponse rapide et la liste affiche les résultats
   d'une requête périmée. Un test le couvre.
2. **Géolocalisation** — `navigator.geolocation`, qui exige HTTPS. Trois issues à
   traiter : accordée, refusée, indisponible. Le géocodage inverse nomme ensuite
   le point.
3. **Déplacement de la carte** — un réticule fixe au centre ; on fait glisser la
   carte sous le point. C'est la façon mobile de cliquer. Le géocodage inverse
   est appelé à la fin du déplacement, jamais pendant.

### Choisir

Les cinq boucles sont sur la carte, la sélectionnée en accent, les autres en
`--voie`. La liste les trie **par part hors bitume décroissante** — le critère
du produit, pas la distance.

Passer de la liste au détail est instantané : les boucles sont en mémoire.

### Ouvrir une boucle, deux chemins

`/b/<id>` doit fonctionner dans les deux cas, et c'est le point qu'on découvre
trop tard si on ne l'écrit pas :

- **à chaud** — on vient de `/boucles`, la boucle est dans `resultats` ;
- **à froid** — lien partagé ou rechargement de page : appel à
  `GET /v1/loops/{id}`, qui rend la boucle **et la demande d'origine**.

La demande sert à afficher « tu en demandais 18 » à côté de « 17,4 km ». Sans
elle, il faudrait la recalculer à rebours depuis `ecart_cible`.

**Le partage prévient.** L'identifiant encode le point de départ : partager une
boucle partie de chez soi diffuse son adresse. Une phrase au moment de copier le
lien, pas un article.

## Le contrat de l'API

Relevé sur le code, pas supposé.

```ts
POST /v1/loops
  → { loops: [{ id, score, geometry }], attribution }
GET  /v1/loops/{id}
  → { loop: { id, score, geometry }, request, attribution }
GET  /v1/loops/{id}.gpx   → GPX, Content-Disposition: attachment
GET  /v1/regions          → { bbox, data, attribution }

score    : { distance_m, part_non_bitume, part_trafic, part_retracee, ecart_cible }
geometry : [[lon, lat], …]   ordre GeoJSON, consommable par MapLibre sans conversion
erreurs  : { error: string }
```

**Chaque échec devient une variante typée** — quatre viennent du serveur, la
cinquième de ce qui ne l'atteint jamais. C'est ce qui permet aux écrans de
proposer des sorties différentes, comme la planche États les dessine :

| Code | Variante | Ce que l'écran propose |
|---|---|---|
| 400 | `HorsZone` | déplacer le départ, en montrant la zone couverte |
| 404 | `AucuneBoucle` | essayer une distance plus courte, ou accepter plus de bitume |
| 429 | `TropDeDemandes` | attendre ; la réponse porte `Retry-After` |
| 504 | `DelaiDepasse` | réessayer |
| — | `Reseau` | ce qui n'arrive jamais au serveur |

Pas de `catch` générique. Le message technique va dans la console, jamais à
l'écran, et jamais le code HTTP.

**Une attente est prévue, pas subie.** 738 ms en médiane, 1,17 s au pire sur la
cible de déploiement : l'écran d'attente n'est pas une précaution, c'est le cas
courant. Il s'annule.

## Le géocodage

`api-adresse.data.gouv.fr`, sans clé ni quota dur. Réponses en GeoJSON
`FeatureCollection` :

```
search/?q=<texte>&limit=5&lat=&lon=      → features[].properties.{label,name,postcode,city,type}
reverse/?lon=<lon>&lat=<lat>             → features[0].properties.label
```

Passer `lat`/`lon` à la recherche pondère les résultats par proximité — utile,
puisqu'on cherche presque toujours autour de soi.

**Deux points à traiter :**

- La réponse **ne porte aucune attribution** : elle est à afficher en dur sous la
  liste de résultats. La BAN est sous **Licence Ouverte 2.0** (vérifié sur
  `data.gouv.fr`), qui demande de mentionner la paternité — « source : Base
  Adresse Nationale » suffit.
- Le texte saisi part vers un service tiers. C'est inhérent à la recherche
  d'adresse, mais cela doit être dit dans la page de confidentialité, et c'est
  une raison de plus pour que la carte et la géolocalisation restent des moyens
  suffisants à eux seuls.

## Le fond de carte

Les styles existent : `design/carte/hent-sombre.json` et `hent-clair.json`,
générés depuis `design/_themes.json`. Tuiles vectorielles `PLAN.IGN` de la
Géoplateforme, sans clé. **Attribution « © IGN » obligatoire et visible.**

Le style pose le chemin **au-dessus** de la route et plus épais qu'elle. C'est
l'inverse d'un fond routier, et c'est ce que trie hent.

**Une seule instance de carte, jamais démontée.** Elle vit dans le layout,
au-dessus du routeur ; les écrans lui donnent des ordres. Si elle vivait dans
les routes, chaque navigation repaierait le chargement du style et des tuiles —
et sur téléphone en 4G médiocre, ça se verrait.

**Changer de thème échange le style**, ce qui recharge les couches. Les sources
GeoJSON des tracés doivent être reposées après `styledata` : c'est le piège
classique de MapLibre, et il vaut la peine d'être écrit ici.

## Les thèmes

`design/_themes.json` reste la source. Un cinquième générateur,
`design/generer-css.py`, rejoint les quatre existants et produit
`web/src/styles/jetons.css` : deux blocs de
variables, sous `:root` et sous `[data-theme="clair"]`, plus un
`prefers-color-scheme` pour le mode automatique.

Sans ce lien, le système de design et le front divergent au premier ajustement —
ce que les scripts de `design/` existent précisément pour empêcher.

## Accessibilité

Le système de design est conforme RGAA AA, contrastes mesurés. Ce qui reste à
tenir dans le code :

- **La carte a une alternative textuelle.** Les chiffres de l'écran détail *sont*
  cette alternative : une boucle doit rester compréhensible sans jamais voir le
  tracé. Cela vaut pour les lecteurs d'écran comme pour un WebGL indisponible.
- **Les curseurs annoncent leur valeur** — `role="slider"`, et `aria-valuetext`
  quand la valeur est un mot (« beaucoup », pas « 80 »).
- **Le focus n'est jamais supprimé** — anneau de 2 px en `--accent-vif`, posé
  hors du composant par un décalage.
- **L'ordre de tabulation suit l'ordre visuel**, la feuille basse se ferme à Échap.
- **Les tailles en `rem`**, pour survivre à un zoom de 200 %.
- **Le tracé ne s'anime pas** sans respecter `prefers-reduced-motion`.
- **Un lien dans du texte est souligné** : le vert seul ne suffit pas à le
  distinguer.

## Les tests

**Vitest sur la logique pure, pas sur le rendu.** C'est pour rendre cela possible
que l'état vit hors du DOM et que les frontières techniques sont derrière des
ports : chaque cas d'usage se teste avec un double de trois lignes.

Ce qui est couvert :

- la machine à états des résultats — toutes les transitions, y compris l'annulation
- le client API et ses cinq variantes d'erreur, avec un `fetch` doublé
- le décodage et l'encodage des URLs
- le formatage — `17,4 km`, `2 h 10`, `55 %`, virgule décimale, espaces insécables
- l'anti-rebond et l'annulation du géocodage : **une réponse lente ne doit jamais
  écraser une réponse rapide**
- le port `Preferences` quand `localStorage` lève
- le test d'architecture sur les imports

Le rendu se vérifie à l'œil, sur le canevas de design qui fait foi.

**La règle du dépôt s'applique** : toute assertion doit pouvoir échouer. Onze
assertions creuses ont été corrigées côté Go, en quatre formes — comparaison
qu'un `NaN` traverse, valeur attendue égale au zéro du type, itération sur une
collection jamais vérifiée non vide, comparaison de valeurs identiques par
construction. Chaque test écrit ici doit être vu échouer avant d'être vert.

## Ce qui n'est pas dans ce chantier

- **Aucun compte, aucune synchronisation.** Le serveur reste sans état. Si cela
  change un jour, c'est un second sous-système, avec sa propre conception.
- **Aucun favori, aucun historique.** Seuls les réglages et le thème sont
  retenus. Le reste attend d'avoir été réclamé.
- **Aucune internationalisation.** Français, et la Bretagne.
- **Aucun travail hors ligne.** Ni service worker, ni cache de tuiles.
- **Aucune analyse d'audience.** « Pas de compte, pas de traçage » n'est pas un
  slogan.

## Risques

**MapLibre exige WebGL.** Appareil ancien, WebGL désactivé, certains
environnements d'entreprise : la carte ne s'affichera pas. Le repli existe par
conception — les chiffres portent l'information — mais il doit être testé, pas
supposé.

**Le premier chargement est lourd.** MapLibre pèse environ 250 Ko compressé, et
les tuiles s'ajoutent. Sur une application de rando consultée en 4G médiocre,
cela se sent. À mesurer sur un vrai téléphone avant de tenir le sujet pour clos.

**La latence du moteur est au-dessus de sa cible.** 738 ms médian contre 500
visés. Le front ne peut pas la corriger ; il peut seulement la rendre supportable
— d'où l'écran d'attente annulable, et les résultats gardés en mémoire pour que
liste et détail soient instantanés.

**Le chantier est le plus gros du projet.** Six écrans, deux thèmes, une carte,
un géocodeur, une machine à états. Il se découpe en tâches indépendantes — le
domaine et les ports d'abord, puis chaque tranche verticale, la carte en
dernier — mais il ne se fait pas en une session. Si le plan qui en découle
dépasse une quinzaine de tâches, c'est le signe qu'il faut le scinder : les
écrans de réglage et de résultats d'un côté, la carte et le géocodage de
l'autre.
