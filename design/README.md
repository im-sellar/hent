# Le design de hent

Un canevas multi-planches, publié comme Artifact Claude. Trois pages :
**Parcours** (les six écrans, en sombre et en clair, plus les états),
**Système** (symbole, fondations, composants, thèmes), **Explorations**
(les pistes écartées, gardées pour mémoire).

## Source et dérivé

Une couleur n'est écrite qu'à un seul endroit : `_themes.json`. Les maquettes
n'emploient que des `var(--jeton)`, jamais une valeur. Tout le reste en découle.

| Fichier | Rôle |
|---|---|
| `_themes.json` | **La source.** Vingt jetons, deux jeux de valeurs. |
| `Main`, `Depart`, `DepartRecherche`, `Generateur`, `Boucles`, `Detail` `.dc.html` | Les six écrans, en jetons. Le bloc `:root` y est posé par le script. |
| `Symbole`, `Fondations`, `Composants`, `Themes`, `Etats` `.dc.html` | Les planches de documentation. |
| `*Clair.dc.html`, `Themes.dc.html`, `carte/*.json` | **Générés.** Ne pas les modifier à la main : la prochaine génération les écrase. |
| `canvas.json` | Disposition, pages, vue de lancement. |
| `hent-directions.html` | Le canevas assemblé. Non versionné — il pèse 2,7 Mo et se reconstruit. |

## Régénérer

Après toute modification d'une maquette ou de `_themes.json`, dans cet ordre :

```sh
python3 appliquer-theme.py        # pose les jetons, dérive les variantes claires
python3 generer-planche-themes.py # réécrit la planche Thèmes, ratios recalculés
python3 generer-style-carte.py    # réécrit les styles MapLibre des deux thèmes
```

Sauter la première étape fait diverger les variantes claires en silence.

`generer-planche-themes.py` **calcule** les ratios de contraste qu'il affiche :
la planche ne peut donc pas mentir sur ce qu'elle documente. `generer-style-carte.py`
vérifie que chaque `source-layer` du style existe vraiment dans les tuiles de
l'IGN, et échoue sinon.

## Réassembler et publier

Le canevas s'assemble avec le `seed-canvas.mjs` de la compétence `design`, en
listant les artboards dans l'ordre d'affichage, puis se republie à la même
adresse. Version en ligne :
<https://claude.ai/code/artifact/a997c10e-db7f-49df-9854-a4ae3630ff05>

## Le fond de carte

`carte/hent-sombre.json` et `carte/hent-clair.json` sont des styles MapLibre GL
prêts à l'emploi. Ils s'appuient sur les tuiles vectorielles `PLAN.IGN` de la
Géoplateforme — sans clé ni quota — et posent le chemin **au-dessus** de la
route, plus épais qu'elle : c'est l'inverse d'un fond routier, et c'est ce que
trie hent. Le tracé de la boucle n'y figure pas : l'application l'ajoute en
source GeoJSON par-dessus. Attribution `© IGN` obligatoire.
