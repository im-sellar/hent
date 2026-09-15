# Front de hent, plan 1 — le socle et la chaîne sans carte

> **Pour les agents exécutants :** SOUS-COMPÉTENCE REQUISE — utiliser
> `superpowers:subagent-driven-development` (recommandé) ou
> `superpowers:executing-plans` pour dérouler ce plan tâche par tâche. Les
> étapes utilisent des cases à cocher (`- [ ]`).

**But :** livrer une application utilisable de bout en bout — régler une
distance, obtenir des boucles, en ouvrir une, télécharger son GPX — sans carte
ni géocodage, qui font l'objet du plan 2.

**Architecture :** SvelteKit en sortie statique, trois couches comme le back
(`domaine` → `app` → `infra`/`ui`), quatre ports devant les frontières
techniques, et un test qui vérifie les imports. L'état vit hors du DOM, ce qui
rend les cas d'usage testables sans navigateur.

**Pile :** TypeScript, Svelte 5, SvelteKit avec `adapter-static`, Vite, Vitest.
Node 24, npm 11.

**Spec :** `docs/front-web.md`

**Ce plan est le premier de deux.** Le plan 2 apportera MapLibre, l'écran de
départ définitif et le géocodage BAN. Ici, le départ est saisi en coordonnées
brutes par un champ provisoire, explicitement marqué comme tel.

## Contraintes globales

- **Répertoire de travail : `web/`** à la racine du dépôt `hent`. Rien hors de
  `web/` ne bouge, sauf mention explicite.
- **Demander avant d'installer quoi que ce soit.** Le propriétaire du dépôt
  exige d'être consulté avant tout `npm install` de dépendance nouvelle. Les
  dépendances prévues ici sont : `svelte`, `@sveltejs/kit`,
  `@sveltejs/adapter-static`, `vite`, `typescript`, `vitest`. Aucune autre.
- **Aucune dépendance de production hors de cette liste.** Pas de bibliothèque
  de dates, de validation, de composants ou d'icônes.
- **Le français est la langue du produit et du code.** Noms de fonctions, de
  types, de variables et de tests en français, comme dans `internal/`. Les mots
  du domaine HTTP et TypeScript restent en anglais (`fetch`, `signal`, `Promise`).
- **Messages de commit :** `type: description` à l'impératif, minuscule, une
  ligne, sans numéro de ticket. **Aucune ligne `Co-Authored-By`, aucune mention
  « Generated with »** — règle explicite du propriétaire.
- **Toute assertion doit pouvoir échouer.** Ce dépôt a corrigé onze assertions
  creuses côté Go, en quatre formes : comparaison qu'un `NaN` traverse, valeur
  attendue égale au zéro du type, itération sur une collection jamais vérifiée
  non vide, comparaison de valeurs identiques par construction. Chaque test doit
  être **vu échouer** avant d'être rendu vert.
- **Ne pas toucher au dossier `internal/adapter/httpapi/web/`** ni aux
  modifications non versionnées de `handler.go` et `handler_test.go` : c'est un
  prototype local que le propriétaire ne veut pas voir commité.

---

### Tâche 1 : Échafauder le projet et poser le gardien d'architecture

Le gardien arrive avec l'échafaudage, pas après : c'est lui qui rend la règle
des couches réelle plutôt qu'affichée.

**Fichiers :**
- Créer : `web/package.json`, `web/svelte.config.js`, `web/vite.config.ts`,
  `web/tsconfig.json`, `web/.gitignore`
- Créer : `web/src/routes/+layout.svelte`, `web/src/routes/+page.svelte`
- Créer : `web/src/lib/architecture.test.ts`
- Créer : `web/src/lib/domaine/.gitkeep`, `web/src/lib/app/.gitkeep`,
  `web/src/lib/infra/.gitkeep`, `web/src/lib/ui/.gitkeep`
- Modifier : `.gitignore` à la racine du dépôt

**Interfaces :**
- Produit : l'arborescence `web/src/lib/{domaine,app,infra,ui}` sur laquelle
  toutes les tâches suivantes s'appuient.
- Produit : `npm test` (Vitest) et `npm run build` (sortie statique dans
  `web/build/`).

- [ ] **Étape 1 : demander l'autorisation d'installer**

Ce projet interdit d'installer sans accord. Poser la question, attendre la
réponse, et ne rien lancer avant. Les paquets sont :

```
svelte ^5.57  @sveltejs/kit ^2.70  @sveltejs/adapter-static ^3.0
vite ^8.3  typescript ^7.0  vitest ^5.0
```

Si l'accord n'est pas donné, **s'arrêter et le signaler** — aucune tâche de ce
plan n'est réalisable sans.

- [ ] **Étape 2 : créer les fichiers de configuration**

`web/package.json` :

```json
{
  "name": "hent-web",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite dev",
    "build": "vite build",
    "preview": "vite preview",
    "test": "vitest run",
    "check": "svelte-kit sync && tsc --noEmit"
  }
}
```

`web/svelte.config.js` :

```js
import adapter from '@sveltejs/adapter-static';

export default {
  kit: {
    // Sortie 100 % statique : Caddy la sert telle quelle, aucun Node en
    // production. `fallback` renvoie les routes non prérendues vers une page
    // unique, ce qui permet à /b/<id> de fonctionner sans serveur applicatif.
    adapter: adapter({ fallback: 'index.html', strict: false })
  }
};
```

`web/vite.config.ts` :

```ts
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [sveltekit()],
  server: {
    // En développement, Vite sert le front et relaie l'API : même origine,
    // donc aucune configuration de CORS, comme en production derrière Caddy.
    proxy: { '/v1': 'http://localhost:8080' }
  },
  test: { environment: 'node', include: ['src/**/*.test.ts'] }
});
```

`web/tsconfig.json` :

```json
{
  "extends": "./.svelte-kit/tsconfig.json",
  "compilerOptions": {
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "verbatimModuleSyntax": true
  }
}
```

`web/.gitignore` :

```
node_modules/
build/
.svelte-kit/
```

Ajouter à la fin du `.gitignore` **à la racine du dépôt** :

```
# Dépendances et sorties de build du front
web/node_modules/
web/build/
web/.svelte-kit/
```

- [ ] **Étape 3 : créer les deux fichiers de route minimaux**

`web/src/routes/+layout.svelte` :

```svelte
<script lang="ts">
  let { children } = $props();
</script>

{@render children()}
```

`web/src/routes/+page.svelte` :

```svelte
<h1>hent</h1>
```

Créer les quatre répertoires de `src/lib/` avec un `.gitkeep` dans chacun, pour
qu'ils existent dans git avant que les tâches suivantes ne les remplissent.

- [ ] **Étape 4 : écrire le gardien d'architecture, qui doit échouer**

`web/src/lib/architecture.test.ts` :

```ts
import { describe, expect, it } from 'vitest';
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join } from 'node:path';

/**
 * Règles de dépendance entre couches, calquées sur `internal/architecture_test.go`
 * du back. Une couche ne doit jamais importer celles listées en face d'elle.
 */
const interdits: Record<string, string[]> = {
  domaine: ['app', 'infra', 'ui'],
  app: ['infra', 'ui'],
  ui: ['infra']
};

const racine = new URL('.', import.meta.url).pathname;

function fichiersSources(dossier: string): string[] {
  let trouves: string[] = [];
  for (const entree of readdirSync(dossier)) {
    const chemin = join(dossier, entree);
    if (statSync(chemin).isDirectory()) {
      trouves = trouves.concat(fichiersSources(chemin));
    } else if (/\.(ts|svelte)$/.test(entree) && !entree.endsWith('.test.ts')) {
      trouves.push(chemin);
    }
  }
  return trouves;
}

/** Chemins importés par un fichier, qu'ils soient relatifs ou en alias $lib. */
function importsDe(chemin: string): string[] {
  const source = readFileSync(chemin, 'utf8');
  return [...source.matchAll(/from\s+['"]([^'"]+)['"]/g)].map((m) => m[1]!);
}

/** Dit si un import vise la couche donnée, en respectant la frontière de segment. */
export function viseLaCouche(specifieur: string, couche: string): boolean {
  const segments = specifieur.split('/').filter((s) => s !== '.' && s !== '..');
  const depart = segments[0] === '$lib' ? 1 : 0;
  return segments[depart] === couche;
}

describe('règles de dépendance entre couches', () => {
  for (const [couche, bannies] of Object.entries(interdits)) {
    it(`${couche} n'importe pas ${bannies.join(', ')}`, () => {
      const dossier = join(racine, couche);
      const fichiers = fichiersSources(dossier);

      // Sans cette garde, une couche vide ou un chemin faux rendrait le test
      // vert sans avoir rien inspecté.
      expect(fichiers.length, `aucun fichier inspecté dans ${couche}`).toBeGreaterThan(0);

      const fautes: string[] = [];
      for (const fichier of fichiers) {
        for (const specifieur of importsDe(fichier)) {
          for (const bannie of bannies) {
            if (viseLaCouche(specifieur, bannie)) {
              fautes.push(`${fichier.replace(racine, '')} importe ${specifieur}`);
            }
          }
        }
      }
      expect(fautes, `la couche ${couche} ne doit pas dépendre de ${bannies.join(', ')}`).toEqual([]);
    });
  }
});

describe('viseLaCouche', () => {
  it('respecte la frontière de segment', () => {
    // Une comparaison de préfixe nue confondrait `infra` et `infrastructure` :
    // le test qui garantit l'architecture est le dernier endroit où tolérer
    // une approximation.
    expect(viseLaCouche('$lib/infra/hent-api', 'infra')).toBe(true);
    expect(viseLaCouche('../infra/hent-api', 'infra')).toBe(true);
    expect(viseLaCouche('$lib/infrastructure/x', 'infra')).toBe(false);
    expect(viseLaCouche('$lib/domaine/boucle', 'infra')).toBe(false);
    expect(viseLaCouche('vitest', 'infra')).toBe(false);
  });
});
```

- [ ] **Étape 5 : lancer le gardien et constater l'échec attendu**

```
cd web && npm test
```

Attendu : les trois cas de `règles de dépendance` **échouent** sur « aucun
fichier inspecté » — les couches ne contiennent que des `.gitkeep`. C'est la
garde anti-test-creux qui parle, et c'est la preuve qu'elle fonctionne.

Le bloc `viseLaCouche` doit, lui, **passer**.

- [ ] **Étape 6 : peupler chaque couche d'un fichier trivial**

Pour que le gardien ait quelque chose à inspecter, créer dans chaque couche un
module minimal qui sera remplacé par les tâches suivantes :

`web/src/lib/domaine/version.ts` :

```ts
/** Version du contrat de données attendue de l'API. */
export const versionContrat = 'v1';
```

`web/src/lib/app/version.ts` :

```ts
import { versionContrat } from '$lib/domaine/version';

export function contratAttendu(): string {
  return versionContrat;
}
```

`web/src/lib/infra/version.ts` :

```ts
import { contratAttendu } from '$lib/app/version';

export function baseAPI(): string {
  return `/${contratAttendu()}`;
}
```

`web/src/lib/ui/version.ts` :

```ts
import { contratAttendu } from '$lib/app/version';

export function libelleContrat(): string {
  return `API ${contratAttendu()}`;
}
```

Supprimer les quatre `.gitkeep`.

- [ ] **Étape 7 : vérifier que le gardien passe, puis qu'il mord**

```
cd web && npm test
```

Attendu : tout vert.

Puis, temporairement, ajouter en tête de `web/src/lib/domaine/version.ts` :

```ts
import { baseAPI } from '$lib/infra/version';
```

Relancer. Attendu : **ÉCHEC** sur `domaine n'importe pas app, infra, ui`, avec
le nom du fichier fautif. **Retirer la ligne**, relancer, constater le vert, et
confirmer avec `git diff` que rien ne subsiste.

Sans cette vérification, rien ne prouve que le gardien attrape quoi que ce soit.

- [ ] **Étape 8 : vérifier que le build produit bien du statique**

```
cd web && npm run build && ls build/
```

Attendu : un répertoire `build/` contenant au moins `index.html`. Il n'est pas
versionné.

- [ ] **Étape 9 : commit**

```bash
git add web/ .gitignore
git commit -m "feat: échafaude le front SvelteKit et son gardien de couches"
```

---

### Tâche 2 : Générer les jetons CSS depuis la source de design

Sans ce lien, le système de design et le front divergent au premier ajustement.
C'est ce que les scripts de `design/` existent pour empêcher.

**Fichiers :**
- Créer : `design/generer-css.py`
- Créer : `web/src/styles/jetons.css` (généré)
- Modifier : `web/src/routes/+layout.svelte`
- Modifier : `design/README.md`

**Interfaces :**
- Consomme : `design/_themes.json`, qui porte deux jeux de vingt jetons sous les
  clés `sombre` et `clair`. Les noms de jetons y sont déjà préfixés par `--`.
- Produit : `web/src/styles/jetons.css`, importé par le layout, définissant les
  variables sous `:root`, sous `[data-theme='clair']` et sous
  `@media (prefers-color-scheme: light)`.

- [ ] **Étape 1 : écrire le générateur**

`design/generer-css.py` :

```python
#!/usr/bin/env python3
"""Écrit les variables CSS du front à partir de `_themes.json`.

Le front ne connaît que des noms de jetons ; leurs valeurs vivent ici, dans le
même fichier que celui qui alimente les maquettes et les styles de carte. Un
ajustement de palette se propage donc au code sans recopie.

Trois blocs sont produits : le thème sombre par défaut, le thème clair quand le
système le demande, et le thème clair forcé par un attribut sur la racine — la
bascule manuelle doit l'emporter sur la préférence système.
"""
from __future__ import annotations

import json
import pathlib

SORTIE = pathlib.Path(__file__).parent.parent / "web" / "src" / "styles" / "jetons.css"
SOURCE = pathlib.Path(__file__).parent / "_themes.json"


def bloc(jetons: dict[str, str], retrait: str = "  ") -> str:
    return "\n".join(f"{retrait}{nom}: {valeur};" for nom, valeur in jetons.items())


def main() -> int:
    themes = json.loads(SOURCE.read_text())
    sombre, clair = themes["sombre"], themes["clair"]

    css = f"""/* Généré par design/generer-css.py — ne pas modifier à la main.
   Les valeurs vivent dans design/_themes.json, avec les maquettes. */

:root {{
  color-scheme: dark light;
{bloc(sombre)}
}}

@media (prefers-color-scheme: light) {{
  :root:not([data-theme='sombre']) {{
{bloc(clair, "    ")}
  }}
}}

:root[data-theme='clair'] {{
{bloc(clair)}
}}
"""
    SORTIE.parent.mkdir(parents=True, exist_ok=True)
    SORTIE.write_text(css)
    print(f"  {SORTIE} — {len(sombre)} jetons par thème")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
```

- [ ] **Étape 2 : lancer et vérifier la sortie**

```
cd design && python3 generer-css.py
```

Attendu : le fichier annonce vingt jetons par thème. L'ouvrir et vérifier que
`--fond`, `--accent` et `--texte` y figurent dans les trois blocs, avec des
valeurs **différentes** entre sombre et clair pour `--fond` et `--accent`.

Si les deux thèmes portaient les mêmes valeurs, la bascule ne se verrait pas :
c'est le seul défaut que cette étape peut produire silencieusement.

- [ ] **Étape 3 : importer les jetons dans le layout**

`web/src/routes/+layout.svelte` :

```svelte
<script lang="ts">
  import '../styles/jetons.css';

  let { children } = $props();
</script>

{@render children()}
```

- [ ] **Étape 4 : documenter le générateur**

Dans `design/README.md`, section « Régénérer », ajouter la quatrième commande à
la suite des trois existantes :

```sh
python3 generer-css.py          # réécrit les variables CSS du front
```

Et ajouter `web/src/styles/jetons.css` à la ligne « Générés » du tableau des
fichiers, à côté des `*Clair.dc.html` et des `carte/*.json`.

- [ ] **Étape 5 : vérifier que tout tient**

```
cd web && npm test && npm run build
```

Attendu : vert, et `build/` produit.

- [ ] **Étape 6 : commit**

```bash
git add design/generer-css.py design/README.md web/src/styles/jetons.css web/src/routes/+layout.svelte
git commit -m "feat: génère les variables CSS du front depuis la source de design"
```

---

### Tâche 3 : Le domaine — types, règles et formatage

Aucune dépendance, aucun effet de bord. C'est ce qui rend cette couche testable
sans rien monter.

**Fichiers :**
- Créer : `web/src/lib/domaine/boucle.ts`, `web/src/lib/domaine/boucle.test.ts`
- Créer : `web/src/lib/domaine/depart.ts`, `web/src/lib/domaine/depart.test.ts`
- Créer : `web/src/lib/domaine/reglages.ts`, `web/src/lib/domaine/reglages.test.ts`
- Créer : `web/src/lib/domaine/format.ts`, `web/src/lib/domaine/format.test.ts`
- Supprimer : `web/src/lib/domaine/version.ts`

**Interfaces :**
- Produit : `Coord = { lat: number; lon: number }`
- Produit : `Score = { distanceM, partNonBitume, partTrafic, partRetracee, ecartCible }`, tous `number`
- Produit : `Boucle = { id: string; score: Score; geometrie: [number, number][] }` — `[lon, lat]`, ordre GeoJSON
- Produit : `Demande = { depart: Coord; distanceM: number; eviterBitume: number; maxResultats: number; variante: number }`
- Produit : `Reglages = { distanceM: number; eviterBitume: number }`
- Produit : `DISTANCE_MIN_M = 2000`, `DISTANCE_MAX_M = 50000`, `REGLAGES_PAR_DEFAUT: Reglages`
- Produit : `estCoordValide(c: Coord): boolean`
- Produit : `borner(r: Reglages): Reglages`
- Produit : `dureeMinutes(distanceM: number, allureKmH: number): number`
- Produit : `formatDistance(m: number): string`, `formatDuree(minutes: number): string`, `formatPourcent(part: number): string`

- [ ] **Étape 1 : écrire les tests du formatage**

Le formatage est ce qui casse en silence : une virgule devenue point, une unité
perdue, un arrondi qui dérape. `web/src/lib/domaine/format.test.ts` :

```ts
import { describe, expect, it } from 'vitest';
import { formatDistance, formatDuree, formatPourcent } from './format';

describe('formatDistance', () => {
  it('rend des kilomètres avec une décimale et une virgule', () => {
    expect(formatDistance(17_400)).toBe('17,4 km');
  });

  it('garde la décimale même quand elle est nulle', () => {
    // « 18 km » et « 18,0 km » ne se ressemblent pas dans une liste alignée.
    expect(formatDistance(18_000)).toBe('18,0 km');
  });

  it('rend des mètres sous le kilomètre', () => {
    expect(formatDistance(850)).toBe('850 m');
  });

  it('rend zéro sans unité fantaisiste', () => {
    expect(formatDistance(0)).toBe('0 m');
  });
});

describe('formatDuree', () => {
  it('sépare heures et minutes', () => {
    expect(formatDuree(130)).toBe('2 h 10');
  });

  it('complète les minutes à deux chiffres', () => {
    // « 2 h 5 » se lit mal à côté de « 2 h 10 ».
    expect(formatDuree(125)).toBe('2 h 05');
  });

  it('omet les heures en dessous de soixante minutes', () => {
    expect(formatDuree(45)).toBe('45 min');
  });
});

describe('formatPourcent', () => {
  it('rend une part en pourcentage entier', () => {
    expect(formatPourcent(0.55)).toBe('55 %');
  });

  it('arrondit au plus proche', () => {
    expect(formatPourcent(0.238)).toBe('24 %');
  });

  it('distingue un vrai zéro d’une valeur infime', () => {
    // 0,4 % de tracé refait n'est pas la même chose que zéro : arrondir à
    // l'entier effacerait l'information que l'écran de détail affiche.
    expect(formatPourcent(0)).toBe('0 %');
    expect(formatPourcent(0.004)).toBe('0,4 %');
  });
});
```

- [ ] **Étape 2 : lancer et constater l'échec**

```
cd web && npx vitest run src/lib/domaine/format.test.ts
```

Attendu : ÉCHEC — le module `./format` n'existe pas.

- [ ] **Étape 3 : écrire le formatage**

`web/src/lib/domaine/format.ts` :

```ts
/**
 * Mise en forme des mesures, en français.
 *
 * Les séparateurs sont posés explicitement plutôt que délégués à
 * `toLocaleString` : le rendu doit être identique quelle que soit la locale du
 * navigateur, et les témoins de test ne doivent pas dépendre de l'ICU installée.
 */

/** Distance lisible : mètres sous le kilomètre, kilomètres à une décimale au-delà. */
export function formatDistance(metres: number): string {
  if (metres < 1000) return `${Math.round(metres)} m`;
  return `${(metres / 1000).toFixed(1).replace('.', ',')} km`;
}

/** Durée lisible : « 45 min » en dessous d'une heure, « 2 h 05 » au-delà. */
export function formatDuree(minutes: number): string {
  const total = Math.round(minutes);
  if (total < 60) return `${total} min`;
  const heures = Math.floor(total / 60);
  return `${heures} h ${String(total % 60).padStart(2, '0')}`;
}

/**
 * Part en pourcentage. Une valeur non nulle mais inférieure à un demi-point
 * garde une décimale : « 0,4 % de tracé refait » est une information que
 * l'arrondi à l'entier effacerait.
 */
export function formatPourcent(part: number): string {
  const pourcent = part * 100;
  if (pourcent > 0 && pourcent < 1) return `${pourcent.toFixed(1).replace('.', ',')} %`;
  return `${Math.round(pourcent)} %`;
}
```

- [ ] **Étape 4 : vérifier le vert, puis le mordant**

```
cd web && npx vitest run src/lib/domaine/format.test.ts
```

Attendu : PASS.

Puis remplacer temporairement `.replace('.', ',')` par `.replace('.', '.')` dans
`formatDistance`, relancer, constater l'échec, **restaurer**, confirmer avec
`git diff`.

- [ ] **Étape 5 : écrire les tests des types et règles**

`web/src/lib/domaine/reglages.test.ts` :

```ts
import { describe, expect, it } from 'vitest';
import { DISTANCE_MAX_M, DISTANCE_MIN_M, REGLAGES_PAR_DEFAUT, borner } from './reglages';

describe('borner', () => {
  it('laisse passer des réglages valides sans les modifier', () => {
    const r = { distanceM: 18_000, eviterBitume: 0.8 };
    expect(borner(r)).toEqual(r);
  });

  it('remonte une distance sous le plancher', () => {
    expect(borner({ distanceM: 500, eviterBitume: 0.5 }).distanceM).toBe(DISTANCE_MIN_M);
  });

  it('abaisse une distance au-dessus du plafond', () => {
    expect(borner({ distanceM: 90_000, eviterBitume: 0.5 }).distanceM).toBe(DISTANCE_MAX_M);
  });

  it('borne l’aversion au bitume dans [0, 1]', () => {
    expect(borner({ distanceM: 10_000, eviterBitume: -3 }).eviterBitume).toBe(0);
    expect(borner({ distanceM: 10_000, eviterBitume: 4 }).eviterBitume).toBe(1);
  });

  it('remplace une valeur non finie par le défaut', () => {
    // NaN traverse toutes les comparaisons sans en faire échouer aucune : le
    // borner par des `<` et des `>` ne suffit pas, il faut le tester d'abord.
    const r = borner({ distanceM: Number.NaN, eviterBitume: Number.NaN });
    expect(Number.isFinite(r.distanceM)).toBe(true);
    expect(Number.isFinite(r.eviterBitume)).toBe(true);
    expect(r).toEqual(REGLAGES_PAR_DEFAUT);
  });
});

describe('REGLAGES_PAR_DEFAUT', () => {
  it('est lui-même dans les bornes', () => {
    expect(borner(REGLAGES_PAR_DEFAUT)).toEqual(REGLAGES_PAR_DEFAUT);
  });
});
```

`web/src/lib/domaine/depart.test.ts` :

```ts
import { describe, expect, it } from 'vitest';
import { estCoordValide } from './depart';

describe('estCoordValide', () => {
  it('accepte un point en Bretagne', () => {
    expect(estCoordValide({ lat: 48.117, lon: -1.677 })).toBe(true);
  });

  it('refuse une latitude hors bornes', () => {
    expect(estCoordValide({ lat: 91, lon: 0 })).toBe(false);
    expect(estCoordValide({ lat: -91, lon: 0 })).toBe(false);
  });

  it('refuse une longitude hors bornes', () => {
    expect(estCoordValide({ lat: 0, lon: 181 })).toBe(false);
  });

  it('refuse une valeur non finie', () => {
    // Même motif que pour les réglages : sans test explicite, NaN passerait
    // les comparaisons de bornes sans en faire échouer aucune.
    expect(estCoordValide({ lat: Number.NaN, lon: 0 })).toBe(false);
    expect(estCoordValide({ lat: 0, lon: Number.POSITIVE_INFINITY })).toBe(false);
  });

  it('accepte le point zéro', () => {
    // Zéro est une coordonnée valide ; la refuser par un test de véracité
    // serait le bogue classique.
    expect(estCoordValide({ lat: 0, lon: 0 })).toBe(true);
  });
});
```

`web/src/lib/domaine/boucle.test.ts` :

```ts
import { describe, expect, it } from 'vitest';
import { dureeMinutes } from './boucle';

describe('dureeMinutes', () => {
  it('convertit une distance en minutes à l’allure donnée', () => {
    expect(dureeMinutes(8000, 8)).toBe(60);
  });

  it('rend une durée proportionnelle à la distance', () => {
    expect(dureeMinutes(17_400, 8)).toBeCloseTo(130.5, 1);
  });

  it('rend zéro pour une allure nulle plutôt qu’un infini', () => {
    // Une division par zéro donnerait Infinity, qui traverserait ensuite tout
    // le formatage sans qu'aucune assertion ne bronche.
    expect(dureeMinutes(10_000, 0)).toBe(0);
  });
});
```

- [ ] **Étape 6 : lancer et constater les échecs**

```
cd web && npx vitest run src/lib/domaine/
```

Attendu : trois fichiers en ÉCHEC, modules introuvables. `format.test.ts` reste
vert.

- [ ] **Étape 7 : écrire les modules**

`web/src/lib/domaine/depart.ts` :

```ts
/** Un point sur la Terre, en degrés décimaux. */
export type Coord = { lat: number; lon: number };

/** Un point de départ, avec le nom qu'on lui donne à l'écran. */
export type Depart = { coord: Coord; libelle: string };

/**
 * Valide une coordonnée. Les valeurs non finies sont écartées en premier :
 * `NaN` traverse toute comparaison de bornes sans jamais la faire échouer.
 */
export function estCoordValide(c: Coord): boolean {
  if (!Number.isFinite(c.lat) || !Number.isFinite(c.lon)) return false;
  return c.lat >= -90 && c.lat <= 90 && c.lon >= -180 && c.lon <= 180;
}
```

`web/src/lib/domaine/reglages.ts` :

```ts
/** Ce que la personne règle avant de lancer une recherche. */
export type Reglages = { distanceM: number; eviterBitume: number };

export const DISTANCE_MIN_M = 2000;
export const DISTANCE_MAX_M = 50_000;

export const REGLAGES_PAR_DEFAUT: Reglages = { distanceM: 12_000, eviterBitume: 0.7 };

/**
 * Ramène des réglages dans leurs bornes. Une valeur non finie ne peut pas être
 * bornée — la comparer ne produirait ni vrai ni faux — elle est donc remplacée
 * par le défaut plutôt que propagée.
 */
export function borner(r: Reglages): Reglages {
  const distanceM = Number.isFinite(r.distanceM)
    ? Math.min(DISTANCE_MAX_M, Math.max(DISTANCE_MIN_M, r.distanceM))
    : REGLAGES_PAR_DEFAUT.distanceM;

  const eviterBitume = Number.isFinite(r.eviterBitume)
    ? Math.min(1, Math.max(0, r.eviterBitume))
    : REGLAGES_PAR_DEFAUT.eviterBitume;

  return { distanceM, eviterBitume };
}
```

`web/src/lib/domaine/boucle.ts` :

```ts
import type { Coord } from './depart';

/** Les mesures d'une boucle, telles que l'API les expose. */
export type Score = {
  distanceM: number;
  partNonBitume: number;
  partTrafic: number;
  partRetracee: number;
  ecartCible: number;
};

/** Une boucle : son identifiant régénérable, ses mesures, son tracé. */
export type Boucle = {
  id: string;
  score: Score;
  /** Points au format GeoJSON, `[lon, lat]` — consommable tel quel par une carte. */
  geometrie: [number, number][];
};

/** Ce qu'on demande au moteur. */
export type Demande = {
  depart: Coord;
  distanceM: number;
  eviterBitume: number;
  maxResultats: number;
  variante: number;
};

/**
 * Durée estimée à une allure donnée, en minutes. Une allure nulle ou négative
 * rend zéro plutôt qu'un infini : une valeur infinie traverserait ensuite tout
 * le formatage sans déclencher la moindre alerte.
 */
export function dureeMinutes(distanceM: number, allureKmH: number): number {
  if (!Number.isFinite(allureKmH) || allureKmH <= 0) return 0;
  if (!Number.isFinite(distanceM) || distanceM <= 0) return 0;
  return (distanceM / 1000 / allureKmH) * 60;
}
```

Supprimer `web/src/lib/domaine/version.ts` et corriger `app/version.ts` pour
qu'il n'en dépende plus :

```ts
export function contratAttendu(): string {
  return 'v1';
}
```

- [ ] **Étape 8 : vérifier le vert et le mordant**

```
cd web && npm test
```

Attendu : tout vert, gardien d'architecture compris.

Puis, dans `borner`, remplacer temporairement `Number.isFinite(r.distanceM)` par
`true`. Relancer. Attendu : ÉCHEC sur « remplace une valeur non finie par le
défaut ». **Restaurer**, relancer, confirmer avec `git diff`.

- [ ] **Étape 9 : commit**

```bash
git add web/src/lib/domaine/ web/src/lib/app/version.ts
git commit -m "feat: types, règles et formatage du domaine du front"
```

---

### Tâche 4 : Les ports et le client de l'API

Le client traduit les échecs plutôt que de les propager. Sans cela, les écrans
d'erreur dessinés ne peuvent pas proposer des sorties différentes.

**Fichiers :**
- Créer : `web/src/lib/app/ports.ts`
- Créer : `web/src/lib/infra/hent-api.ts`, `web/src/lib/infra/hent-api.test.ts`
- Supprimer : `web/src/lib/infra/version.ts`, `web/src/lib/app/version.ts`,
  `web/src/lib/ui/version.ts`

**Interfaces :**
- Consomme : `Boucle`, `Demande`, `Coord`, `Score` de `$lib/domaine/boucle` et `$lib/domaine/depart`
- Produit : `type ErreurMoteur = { genre: 'HorsZone' | 'AucuneBoucle' | 'TropDeDemandes' | 'DelaiDepasse' | 'Reseau'; message: string; reessayerDansS?: number }`
- Produit : `class ErreurAPI extends Error { readonly genre: ErreurMoteur['genre']; readonly reessayerDansS?: number }`
- Produit : `interface MoteurDeBoucles { generer(d: Demande, signal?: AbortSignal): Promise<Boucle[]>; ouvrir(id: string, signal?: AbortSignal): Promise<{ boucle: Boucle; demande: Demande }>; urlGPX(id: string): string }`
- Produit : `interface Preferences { lire(): Reglages | null; ecrire(r: Reglages): void }`
- Produit : `function creerMoteurHTTP(fetchImpl?: typeof fetch): MoteurDeBoucles`

- [ ] **Étape 1 : écrire les tests du client**

`web/src/lib/infra/hent-api.test.ts` :

```ts
import { describe, expect, it } from 'vitest';
import { ErreurAPI, creerMoteurHTTP } from './hent-api';
import type { Demande } from '$lib/domaine/boucle';

const demande: Demande = {
  depart: { lat: 48.135, lon: -1.628 },
  distanceM: 4000,
  eviterBitume: 0.5,
  maxResultats: 3,
  variante: 0
};

/** Construit un `fetch` doublé qui rend la réponse donnée, sans réseau. */
function fauxFetch(statut: number, corps: unknown, entetes: Record<string, string> = {}) {
  const appels: { url: string; init?: RequestInit }[] = [];
  const impl = (async (url: string, init?: RequestInit) => {
    appels.push({ url, init });
    return new Response(JSON.stringify(corps), {
      status: statut,
      headers: { 'Content-Type': 'application/json', ...entetes }
    });
  }) as unknown as typeof fetch;
  return { impl, appels };
}

describe('generer', () => {
  it('rend les boucles de la réponse', async () => {
    const { impl } = fauxFetch(200, {
      loops: [
        {
          id: 'abc',
          score: {
            distance_m: 3924.32,
            part_non_bitume: 1,
            part_trafic: 0,
            part_retracee: 0,
            ecart_cible: -0.0189
          },
          geometry: [
            [-1.628, 48.135],
            [-1.629, 48.136]
          ]
        }
      ],
      attribution: 'Données © les contributeurs OpenStreetMap'
    });

    const boucles = await creerMoteurHTTP(impl).generer(demande);

    expect(boucles).toHaveLength(1);
    expect(boucles[0]!.id).toBe('abc');
    expect(boucles[0]!.score.distanceM).toBe(3924.32);
    expect(boucles[0]!.score.ecartCible).toBeCloseTo(-0.0189, 4);
    expect(boucles[0]!.geometrie[0]).toEqual([-1.628, 48.135]);
  });

  it('envoie la demande dans le corps, au format de l’API', async () => {
    const { impl, appels } = fauxFetch(200, { loops: [], attribution: '' });

    await creerMoteurHTTP(impl).generer(demande);

    expect(appels).toHaveLength(1);
    const envoye = JSON.parse(String(appels[0]!.init!.body));
    expect(envoye).toEqual({
      start: { lat: 48.135, lon: -1.628 },
      distance_m: 4000,
      preferences: { avoid_paved: 0.5 },
      max_results: 3,
      variant: 0
    });
  });

  it('traduit un 400 en HorsZone', async () => {
    const { impl } = fauxFetch(400, { error: 'le point de départ est hors de la zone couverte' });
    await expect(creerMoteurHTTP(impl).generer(demande)).rejects.toMatchObject({ genre: 'HorsZone' });
  });

  it('traduit un 404 en AucuneBoucle', async () => {
    const { impl } = fauxFetch(404, { error: 'aucune boucle trouvée pour ces critères' });
    await expect(creerMoteurHTTP(impl).generer(demande)).rejects.toMatchObject({ genre: 'AucuneBoucle' });
  });

  it('traduit un 429 et retient le délai annoncé', async () => {
    const { impl } = fauxFetch(429, { error: 'trop de requêtes' }, { 'Retry-After': '30' });
    await expect(creerMoteurHTTP(impl).generer(demande)).rejects.toMatchObject({
      genre: 'TropDeDemandes',
      reessayerDansS: 30
    });
  });

  it('traduit un 504 en DelaiDepasse', async () => {
    const { impl } = fauxFetch(504, { error: 'délai dépassé' });
    await expect(creerMoteurHTTP(impl).generer(demande)).rejects.toMatchObject({ genre: 'DelaiDepasse' });
  });

  it('traduit un échec de transport en Reseau', async () => {
    const impl = (async () => {
      throw new TypeError('Failed to fetch');
    }) as unknown as typeof fetch;
    await expect(creerMoteurHTTP(impl).generer(demande)).rejects.toMatchObject({ genre: 'Reseau' });
  });

  it('laisse passer une annulation sans la déguiser en erreur réseau', async () => {
    // Une annulation volontaire n'est pas une panne : la confondre avec un
    // échec ferait afficher un écran d'erreur à quelqu'un qui vient de cliquer
    // sur « Annuler ».
    const impl = (async () => {
      throw new DOMException('aborted', 'AbortError');
    }) as unknown as typeof fetch;

    await expect(creerMoteurHTTP(impl).generer(demande)).rejects.toSatisfy(
      (e: unknown) => e instanceof DOMException && e.name === 'AbortError'
    );
  });
});

describe('ouvrir', () => {
  it('rend la boucle et la demande d’origine', async () => {
    const { impl, appels } = fauxFetch(200, {
      loop: {
        id: '0.abc',
        score: {
          distance_m: 17_400,
          part_non_bitume: 0.55,
          part_trafic: 0.023,
          part_retracee: 0.004,
          ecart_cible: -0.032
        },
        geometry: [[-1.628, 48.135]]
      },
      request: {
        start: { lat: 48.135, lon: -1.628 },
        distance_m: 18_000,
        tolerance: 0.15,
        preferences: { avoid_paved: 0.8 },
        max_results: 5,
        variant: 2
      },
      attribution: 'Données © les contributeurs OpenStreetMap'
    });

    const { boucle, demande: dem } = await creerMoteurHTTP(impl).ouvrir('0.abc');

    expect(appels[0]!.url).toContain('/v1/loops/0.abc');
    expect(boucle.score.partRetracee).toBeCloseTo(0.004, 4);
    // La demande d'origine est ce qui permet d'afficher « tu en demandais 18 » :
    // elle ne se déduit pas de la boucle.
    expect(dem.distanceM).toBe(18_000);
    expect(dem.eviterBitume).toBe(0.8);
    expect(dem.variante).toBe(2);
  });

  it('encode l’identifiant dans l’URL', async () => {
    const { impl, appels } = fauxFetch(200, {
      loop: { id: 'x', score: { distance_m: 0, part_non_bitume: 0, part_trafic: 0, part_retracee: 0, ecart_cible: 0 }, geometry: [] },
      request: { start: { lat: 0, lon: 0 }, distance_m: 0, preferences: { avoid_paved: 0 }, max_results: 0, variant: 0 },
      attribution: ''
    });

    await creerMoteurHTTP(impl).ouvrir('0.a+b/c');

    expect(appels[0]!.url).toContain(encodeURIComponent('0.a+b/c'));
  });
});

describe('urlGPX', () => {
  it('rend une URL téléchargeable, avec le suffixe qui distingue l’export', () => {
    expect(creerMoteurHTTP().urlGPX('0.abc')).toBe('/v1/loops/0.abc.gpx');
  });

  it('encode l’identifiant', () => {
    expect(creerMoteurHTTP().urlGPX('0.a+b')).toBe(`/v1/loops/${encodeURIComponent('0.a+b')}.gpx`);
  });
});
```

- [ ] **Étape 2 : lancer et constater l'échec**

```
cd web && npx vitest run src/lib/infra/hent-api.test.ts
```

Attendu : ÉCHEC, module introuvable.

- [ ] **Étape 3 : écrire les ports**

`web/src/lib/app/ports.ts` :

```ts
import type { Boucle, Demande } from '$lib/domaine/boucle';
import type { Reglages } from '$lib/domaine/reglages';

/**
 * Les cinq façons dont une recherche peut échouer. Quatre viennent du serveur,
 * la cinquième de ce qui ne l'atteint jamais.
 *
 * Elles sont nommées plutôt que réduites à un message, parce que chaque écran
 * d'erreur propose une sortie différente : déplacer le départ, assouplir un
 * réglage, ou seulement attendre.
 */
export type GenreErreur = 'HorsZone' | 'AucuneBoucle' | 'TropDeDemandes' | 'DelaiDepasse' | 'Reseau';

export type ErreurMoteur = {
  genre: GenreErreur;
  /** Message du serveur, destiné au journal — jamais affiché tel quel. */
  message: string;
  /** Délai annoncé par l'en-tête Retry-After, en secondes, quand il existe. */
  reessayerDansS?: number;
};

/** Le moteur de boucles, vu par l'application. */
export interface MoteurDeBoucles {
  generer(demande: Demande, signal?: AbortSignal): Promise<Boucle[]>;
  ouvrir(id: string, signal?: AbortSignal): Promise<{ boucle: Boucle; demande: Demande }>;
  /** URL d'export : un lien que le navigateur suit, pas un corps qu'on relaie. */
  urlGPX(id: string): string;
}

/**
 * Le stockage des préférences. `lire` rend `null` plutôt que de lever quand le
 * stockage est indisponible — navigation privée, quota, stockage bloqué : rien
 * de tout cela ne doit empêcher l'application de démarrer.
 */
export interface Preferences {
  lire(): Reglages | null;
  ecrire(r: Reglages): void;
}
```

- [ ] **Étape 4 : écrire le client**

`web/src/lib/infra/hent-api.ts` :

```ts
import type { Boucle, Demande, Score } from '$lib/domaine/boucle';
import type { GenreErreur, MoteurDeBoucles } from '$lib/app/ports';

/** Erreur du moteur, portant le genre qui décide de l'écran à montrer. */
export class ErreurAPI extends Error {
  readonly genre: GenreErreur;
  readonly reessayerDansS?: number;

  constructor(genre: GenreErreur, message: string, reessayerDansS?: number) {
    super(message);
    this.name = 'ErreurAPI';
    this.genre = genre;
    this.reessayerDansS = reessayerDansS;
  }
}

type ScoreJSON = {
  distance_m: number;
  part_non_bitume: number;
  part_trafic: number;
  part_retracee: number;
  ecart_cible: number;
};

type BoucleJSON = { id: string; score: ScoreJSON; geometry: [number, number][] };

type DemandeJSON = {
  start: { lat: number; lon: number };
  distance_m: number;
  preferences: { avoid_paved: number };
  max_results: number;
  variant: number;
};

function versScore(s: ScoreJSON): Score {
  return {
    distanceM: s.distance_m,
    partNonBitume: s.part_non_bitume,
    partTrafic: s.part_trafic,
    partRetracee: s.part_retracee,
    ecartCible: s.ecart_cible
  };
}

function versBoucle(b: BoucleJSON): Boucle {
  return { id: b.id, score: versScore(b.score), geometrie: b.geometry };
}

function versDemande(d: DemandeJSON): Demande {
  return {
    depart: { lat: d.start.lat, lon: d.start.lon },
    distanceM: d.distance_m,
    eviterBitume: d.preferences.avoid_paved,
    maxResultats: d.max_results,
    variante: d.variant
  };
}

function corpsDeDemande(d: Demande): DemandeJSON {
  return {
    start: { lat: d.depart.lat, lon: d.depart.lon },
    distance_m: d.distanceM,
    preferences: { avoid_paved: d.eviterBitume },
    max_results: d.maxResultats,
    variant: d.variante
  };
}

const parStatut: Record<number, GenreErreur> = {
  400: 'HorsZone',
  404: 'AucuneBoucle',
  429: 'TropDeDemandes',
  504: 'DelaiDepasse'
};

async function erreurDe(reponse: Response): Promise<ErreurAPI> {
  const genre = parStatut[reponse.status] ?? 'Reseau';
  let message = `statut ${reponse.status}`;
  try {
    const corps = (await reponse.json()) as { error?: string };
    if (corps.error) message = corps.error;
  } catch {
    // Un corps illisible ne change pas le genre de l'erreur : le statut suffit
    // à décider de l'écran, le message ne sert qu'au journal.
  }
  const delai = Number(reponse.headers.get('Retry-After'));
  return new ErreurAPI(genre, message, Number.isFinite(delai) && delai > 0 ? delai : undefined);
}

/**
 * Enveloppe un appel réseau : traduit les statuts en genres d'erreur, et les
 * pannes de transport en `Reseau`. Une annulation volontaire est relancée telle
 * quelle — la confondre avec une panne ferait afficher un écran d'erreur à
 * quelqu'un qui vient de cliquer sur « Annuler ».
 */
async function appeler(
  fetchImpl: typeof fetch,
  url: string,
  init?: RequestInit
): Promise<Response> {
  let reponse: Response;
  try {
    reponse = await fetchImpl(url, init);
  } catch (e) {
    if (e instanceof DOMException && e.name === 'AbortError') throw e;
    throw new ErreurAPI('Reseau', e instanceof Error ? e.message : String(e));
  }
  if (!reponse.ok) throw await erreurDe(reponse);
  return reponse;
}

/**
 * Crée un moteur adossé à l'API HTTP. `fetchImpl` est injectable pour que les
 * cas d'usage se testent sans réseau.
 */
export function creerMoteurHTTP(fetchImpl: typeof fetch = globalThis.fetch): MoteurDeBoucles {
  return {
    async generer(demande, signal) {
      const reponse = await appeler(fetchImpl, '/v1/loops', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(corpsDeDemande(demande)),
        signal
      });
      const corps = (await reponse.json()) as { loops: BoucleJSON[] | null };
      return (corps.loops ?? []).map(versBoucle);
    },

    async ouvrir(id, signal) {
      const reponse = await appeler(fetchImpl, `/v1/loops/${encodeURIComponent(id)}`, { signal });
      const corps = (await reponse.json()) as { loop: BoucleJSON; request: DemandeJSON };
      return { boucle: versBoucle(corps.loop), demande: versDemande(corps.request) };
    },

    urlGPX(id) {
      return `/v1/loops/${encodeURIComponent(id)}.gpx`;
    }
  };
}
```

Supprimer les trois modules provisoires de la tâche 1 : `web/src/lib/infra/version.ts`,
`web/src/lib/app/version.ts` et **`web/src/lib/ui/version.ts`**. Ce dernier
importe `app/version.ts` : l'oublier casserait la compilation.

La couche `ui/` se retrouve donc vide jusqu'à la tâche 7, ce qui fera échouer la
garde anti-test-creux du gardien d'architecture. C'est attendu — créer
`web/src/lib/ui/Bouton.svelte` (tâche 7, étape 5) dès maintenant, avec son
contenu final, pour que la couche ne soit jamais vide. Il n'a aucune dépendance
et se teste à la compilation.

- [ ] **Étape 5 : vérifier le vert**

```
cd web && npm test
```

Attendu : tout vert, gardien d'architecture compris — `infra/` importe `app/` et
`domaine/`, ce qui est autorisé.

- [ ] **Étape 6 : vérifier que la traduction des erreurs mord**

Remplacer temporairement, dans `parStatut`, `404: 'AucuneBoucle'` par
`404: 'HorsZone'`. Relancer. Attendu : ÉCHEC sur « traduit un 404 en
AucuneBoucle ». **Restaurer**, relancer, confirmer avec `git diff`.

Puis retirer temporairement le `if (e instanceof DOMException …)` de `appeler`.
Attendu : ÉCHEC sur « laisse passer une annulation ». **Restaurer**, confirmer.

- [ ] **Étape 7 : commit**

```bash
git add web/src/lib/app/ports.ts web/src/lib/infra/hent-api.ts \
        web/src/lib/infra/hent-api.test.ts web/src/lib/ui/Bouton.svelte
git rm web/src/lib/infra/version.ts web/src/lib/app/version.ts web/src/lib/ui/version.ts
git commit -m "feat: ports du front et client de l'API avec ses cinq variantes d'erreur"
```

---

### Tâche 5 : Le stockage des préférences

**Fichiers :**
- Créer : `web/src/lib/infra/stockage.ts`, `web/src/lib/infra/stockage.test.ts`

**Interfaces :**
- Consomme : `Preferences` de `$lib/app/ports`, `Reglages`, `borner`,
  `REGLAGES_PAR_DEFAUT` de `$lib/domaine/reglages`
- Produit : `function creerPreferences(stockage?: Storage | null): Preferences`

- [ ] **Étape 1 : écrire les tests**

`web/src/lib/infra/stockage.test.ts` :

```ts
import { describe, expect, it } from 'vitest';
import { creerPreferences } from './stockage';
import { REGLAGES_PAR_DEFAUT } from '$lib/domaine/reglages';

/** Stockage en mémoire, suffisant pour ce que le port promet. */
function stockageMemoire(initial: Record<string, string> = {}): Storage {
  const donnees = new Map(Object.entries(initial));
  return {
    getItem: (c: string) => donnees.get(c) ?? null,
    setItem: (c: string, v: string) => void donnees.set(c, v),
    removeItem: (c: string) => void donnees.delete(c),
    clear: () => donnees.clear(),
    key: (i: number) => [...donnees.keys()][i] ?? null,
    get length() {
      return donnees.size;
    }
  } as Storage;
}

/** Stockage qui refuse tout, comme en navigation privée ou quota dépassé. */
function stockageQuiLeve(): Storage {
  const lever = () => {
    throw new DOMException('refusé', 'SecurityError');
    };
  return { getItem: lever, setItem: lever, removeItem: lever, clear: lever, key: lever, length: 0 } as unknown as Storage;
}

describe('creerPreferences', () => {
  it('relit ce qui a été écrit', () => {
    const prefs = creerPreferences(stockageMemoire());
    prefs.ecrire({ distanceM: 18_000, eviterBitume: 0.8 });
    expect(prefs.lire()).toEqual({ distanceM: 18_000, eviterBitume: 0.8 });
  });

  it('rend null quand rien n’a été écrit', () => {
    expect(creerPreferences(stockageMemoire()).lire()).toBeNull();
  });

  it('rend null sur un contenu illisible plutôt que de lever', () => {
    const prefs = creerPreferences(stockageMemoire({ 'hent.reglages': 'pas du json' }));
    expect(prefs.lire()).toBeNull();
  });

  it('borne ce qu’il relit', () => {
    // Une valeur écrite par une version antérieure, ou trafiquée à la main, ne
    // doit pas traverser l'application telle quelle.
    const prefs = creerPreferences(stockageMemoire({
      'hent.reglages': JSON.stringify({ distanceM: 900_000, eviterBitume: 12 })
    }));
    const lu = prefs.lire();
    expect(lu!.distanceM).toBeLessThanOrEqual(50_000);
    expect(lu!.eviterBitume).toBeLessThanOrEqual(1);
  });

  it('rend null quand le stockage refuse de lire', () => {
    // Navigation privée, stockage bloqué, quota : l'application doit démarrer
    // sur ses valeurs par défaut, pas planter.
    expect(creerPreferences(stockageQuiLeve()).lire()).toBeNull();
  });

  it('avale une écriture refusée sans lever', () => {
    const prefs = creerPreferences(stockageQuiLeve());
    expect(() => prefs.ecrire(REGLAGES_PAR_DEFAUT)).not.toThrow();
  });

  it('rend null quand il n’y a aucun stockage du tout', () => {
    // Le rendu côté serveur n'a pas de localStorage : `lire` doit y répondre
    // comme à un stockage vide.
    expect(creerPreferences(null).lire()).toBeNull();
  });
});
```

- [ ] **Étape 2 : lancer et constater l'échec**

```
cd web && npx vitest run src/lib/infra/stockage.test.ts
```

Attendu : ÉCHEC, module introuvable.

- [ ] **Étape 3 : écrire le module**

`web/src/lib/infra/stockage.ts`:

```ts
import type { Preferences } from '$lib/app/ports';
import { borner, type Reglages } from '$lib/domaine/reglages';

const CLE = 'hent.reglages';

/**
 * Préférences adossées au stockage du navigateur.
 *
 * Toute opération est protégée : le stockage peut être absent (rendu hors
 * navigateur), refusé (navigation privée, politique de site) ou plein. Aucun de
 * ces cas ne doit empêcher l'application de démarrer, donc `lire` rend `null` et
 * `ecrire` ne fait rien plutôt que de lever.
 *
 * Ce qui est relu est borné : une valeur écrite par une version antérieure, ou
 * modifiée à la main, ne doit pas traverser l'application telle quelle.
 */
export function creerPreferences(
  stockage: Storage | null = typeof localStorage === 'undefined' ? null : localStorage
): Preferences {
  return {
    lire(): Reglages | null {
      if (!stockage) return null;
      let brut: string | null;
      try {
        brut = stockage.getItem(CLE);
      } catch {
        return null;
      }
      if (!brut) return null;
      try {
        const lu = JSON.parse(brut) as Partial<Reglages>;
        if (typeof lu?.distanceM !== 'number' || typeof lu?.eviterBitume !== 'number') return null;
        return borner({ distanceM: lu.distanceM, eviterBitume: lu.eviterBitume });
      } catch {
        return null;
      }
    },

    ecrire(r: Reglages): void {
      if (!stockage) return;
      try {
        stockage.setItem(CLE, JSON.stringify(borner(r)));
      } catch {
        // Quota atteint ou écriture refusée : la préférence ne survivra pas à la
        // session, ce qui est préférable à une application qui s'arrête.
      }
    }
  };
}
```

- [ ] **Étape 4 : vérifier le vert et le mordant**

```
cd web && npm test
```

Attendu : tout vert.

Puis retirer temporairement le `try`/`catch` autour de `stockage.getItem`.
Attendu : ÉCHEC sur « rend null quand le stockage refuse de lire ».
**Restaurer**, confirmer avec `git diff`.

- [ ] **Étape 5 : commit**

```bash
git add web/src/lib/infra/stockage.ts web/src/lib/infra/stockage.test.ts
git commit -m "feat: préférences persistées, tolérantes à un stockage indisponible"
```

---

### Tâche 6 : La machine à états des résultats

C'est ce qui évite que les écrans d'attente et d'erreur soient bricolés au cas
par cas : chaque écran lit un statut explicite.

**Fichiers :**
- Créer : `web/src/lib/app/generation/resultats.svelte.ts`
- Créer : `web/src/lib/app/generation/resultats.test.ts`

**Interfaces :**
- Consomme : `MoteurDeBoucles`, `ErreurMoteur` de `$lib/app/ports` ; `Boucle`,
  `Demande` de `$lib/domaine/boucle`
- Produit : `type EtatResultats = { statut: 'vide' } | { statut: 'calcul' } | { statut: 'ok'; boucles: Boucle[]; demande: Demande } | { statut: 'erreur'; erreur: ErreurMoteur }`
- Produit : `function creerResultats(moteur: MoteurDeBoucles): { etat(): EtatResultats; lancer(d: Demande): Promise<void>; annuler(): void; poser(boucles: Boucle[], demande: Demande): void; reinitialiser(): void }`

- [ ] **Étape 1 : écrire les tests**

`web/src/lib/app/generation/resultats.test.ts` :

```ts
import { describe, expect, it, vi } from 'vitest';
import { creerResultats } from './resultats.svelte';
import { ErreurAPI } from '$lib/infra/hent-api';
import type { Boucle, Demande } from '$lib/domaine/boucle';
import type { MoteurDeBoucles } from '$lib/app/ports';

const demande: Demande = {
  depart: { lat: 48.135, lon: -1.628 },
  distanceM: 4000,
  eviterBitume: 0.5,
  maxResultats: 3,
  variante: 0
};

const uneBoucle: Boucle = {
  id: 'abc',
  score: { distanceM: 3924, partNonBitume: 0.8, partTrafic: 0.02, partRetracee: 0.01, ecartCible: -0.02 },
  geometrie: [[-1.628, 48.135]]
};

function moteurQui(resultat: (signal?: AbortSignal) => Promise<Boucle[]>): MoteurDeBoucles {
  return {
    generer: resultat,
    ouvrir: async () => ({ boucle: uneBoucle, demande }),
    urlGPX: (id) => `/v1/loops/${id}.gpx`
  };
}

describe('creerResultats', () => {
  it('part de l’état vide', () => {
    expect(creerResultats(moteurQui(async () => [])).etat().statut).toBe('vide');
  });

  it('passe par calcul avant de rendre ok', async () => {
    let debloquer!: (b: Boucle[]) => void;
    const attente = new Promise<Boucle[]>((r) => (debloquer = r));
    const r = creerResultats(moteurQui(() => attente));

    const fini = r.lancer(demande);
    expect(r.etat().statut).toBe('calcul');

    debloquer([uneBoucle]);
    await fini;

    const etat = r.etat();
    expect(etat.statut).toBe('ok');
    if (etat.statut !== 'ok') throw new Error('état inattendu');
    expect(etat.boucles).toHaveLength(1);
    expect(etat.demande.distanceM).toBe(4000);
  });

  it('range les boucles par part hors bitume décroissante', async () => {
    // C'est le critère du produit : la plus verte d'abord, pas la plus proche
    // de la distance demandée.
    const moins = { ...uneBoucle, id: 'moins', score: { ...uneBoucle.score, partNonBitume: 0.4 } };
    const plus = { ...uneBoucle, id: 'plus', score: { ...uneBoucle.score, partNonBitume: 0.9 } };
    const r = creerResultats(moteurQui(async () => [moins, plus]));

    await r.lancer(demande);

    const etat = r.etat();
    if (etat.statut !== 'ok') throw new Error('état inattendu');
    expect(etat.boucles.map((b) => b.id)).toEqual(['plus', 'moins']);
  });

  it('passe en erreur en gardant le genre', async () => {
    const r = creerResultats(moteurQui(async () => {
      throw new ErreurAPI('AucuneBoucle', 'aucune boucle trouvée pour ces critères');
    }));

    await r.lancer(demande);

    const etat = r.etat();
    expect(etat.statut).toBe('erreur');
    if (etat.statut !== 'erreur') throw new Error('état inattendu');
    expect(etat.erreur.genre).toBe('AucuneBoucle');
  });

  it('revient à vide après une annulation, sans afficher d’erreur', async () => {
    // Annuler est un geste volontaire : le traiter comme une panne montrerait
    // un écran d'erreur à quelqu'un qui vient de cliquer sur « Annuler ».
    const r = creerResultats(
      moteurQui(
        (signal) =>
          new Promise<Boucle[]>((_, rejeter) => {
            signal?.addEventListener('abort', () => rejeter(new DOMException('aborted', 'AbortError')));
          })
      )
    );

    const fini = r.lancer(demande);
    r.annuler();
    await fini;

    expect(r.etat().statut).toBe('vide');
  });

  it('ignore la réponse d’un lancement remplacé par un autre', async () => {
    // Deux recherches enchaînées : la première, plus lente, ne doit pas écraser
    // le résultat de la seconde.
    const lente = { ...uneBoucle, id: 'lente' };
    const rapide = { ...uneBoucle, id: 'rapide' };
    let numero = 0;
    const r = creerResultats(
      moteurQui(async () => {
        numero += 1;
        if (numero === 1) {
          await new Promise((res) => setTimeout(res, 20));
          return [lente];
        }
        return [rapide];
      })
    );

    const premier = r.lancer(demande);
    const second = r.lancer(demande);
    await Promise.all([premier, second]);

    const etat = r.etat();
    if (etat.statut !== 'ok') throw new Error('état inattendu');
    expect(etat.boucles.map((b) => b.id)).toEqual(['rapide']);
  });

  it('pose des résultats déjà connus sans appeler le moteur', () => {
    const generer = vi.fn();
    const r = creerResultats({ ...moteurQui(async () => []), generer } as MoteurDeBoucles);

    r.poser([uneBoucle], demande);

    expect(generer).not.toHaveBeenCalled();
    expect(r.etat().statut).toBe('ok');
  });
});
```

- [ ] **Étape 2 : lancer et constater l'échec**

```
cd web && npx vitest run src/lib/app/generation/
```

Attendu : ÉCHEC, module introuvable.

- [ ] **Étape 3 : écrire la machine à états**

`web/src/lib/app/generation/resultats.svelte.ts` :

```ts
import type { Boucle, Demande } from '$lib/domaine/boucle';
import type { ErreurMoteur, MoteurDeBoucles } from '$lib/app/ports';

/**
 * Les quatre états d'une recherche. Ils sont exhaustifs et mutuellement
 * exclusifs : un écran qui les lit ne peut pas se retrouver à deviner s'il doit
 * montrer une liste, une attente ou une erreur.
 */
export type EtatResultats =
  | { statut: 'vide' }
  | { statut: 'calcul' }
  | { statut: 'ok'; boucles: Boucle[]; demande: Demande }
  | { statut: 'erreur'; erreur: ErreurMoteur };

function estAnnulation(e: unknown): boolean {
  return e instanceof DOMException && e.name === 'AbortError';
}

function versErreurMoteur(e: unknown): ErreurMoteur {
  if (e && typeof e === 'object' && 'genre' in e) {
    const erreur = e as { genre: ErreurMoteur['genre']; message?: string; reessayerDansS?: number };
    return {
      genre: erreur.genre,
      message: erreur.message ?? '',
      reessayerDansS: erreur.reessayerDansS
    };
  }
  return { genre: 'Reseau', message: e instanceof Error ? e.message : String(e) };
}

/**
 * Pilote une recherche et expose son état.
 *
 * Deux protections méritent d'être dites : une annulation ramène à l'état vide
 * plutôt qu'en erreur — c'est un geste volontaire, pas une panne — et seul le
 * dernier lancement peut écrire dans l'état, sans quoi une première recherche
 * lente écraserait le résultat d'une seconde plus rapide.
 */
export function creerResultats(moteur: MoteurDeBoucles) {
  let etat = $state<EtatResultats>({ statut: 'vide' });
  let enCours: AbortController | null = null;
  let generation = 0;

  function trier(boucles: Boucle[]): Boucle[] {
    return [...boucles].sort((a, b) => b.score.partNonBitume - a.score.partNonBitume);
  }

  return {
    etat: () => etat,

    async lancer(demande: Demande): Promise<void> {
      enCours?.abort();
      const controleur = new AbortController();
      enCours = controleur;
      const mienne = ++generation;

      etat = { statut: 'calcul' };
      try {
        const boucles = await moteur.generer(demande, controleur.signal);
        if (mienne !== generation) return;
        etat = { statut: 'ok', boucles: trier(boucles), demande };
      } catch (e) {
        if (mienne !== generation) return;
        if (estAnnulation(e)) {
          etat = { statut: 'vide' };
          return;
        }
        etat = { statut: 'erreur', erreur: versErreurMoteur(e) };
      } finally {
        if (enCours === controleur) enCours = null;
      }
    },

    annuler(): void {
      enCours?.abort();
    },

    /** Pose des résultats déjà connus — au retour d'un détail, par exemple. */
    poser(boucles: Boucle[], demande: Demande): void {
      generation += 1;
      etat = { statut: 'ok', boucles: trier(boucles), demande };
    },

    reinitialiser(): void {
      enCours?.abort();
      generation += 1;
      etat = { statut: 'vide' };
    }
  };
}
```

- [ ] **Étape 4 : vérifier le vert**

```
cd web && npm test
```

Attendu : tout vert. Si Vitest refuse les runes `$state` hors composant,
vérifier que le fichier porte bien l'extension `.svelte.ts` — c'est elle qui les
active.

- [ ] **Étape 5 : vérifier que les deux protections mordent**

Retirer temporairement `if (mienne !== generation) return;` du bloc de succès.
Attendu : ÉCHEC sur « ignore la réponse d'un lancement remplacé ». **Restaurer.**

Puis remplacer le traitement de l'annulation par un passage en erreur. Attendu :
ÉCHEC sur « revient à vide après une annulation ». **Restaurer**, confirmer avec
`git diff`.

- [ ] **Étape 6 : commit**

```bash
git add web/src/lib/app/generation/
git commit -m "feat: machine à états de la recherche, annulable et à l'abri des réponses périmées"
```

---

### Tâche 7 : Les écrans — accueil, réglage, résultats, détail

Une seule tâche pour les quatre écrans : ils partagent la même navigation et le
même état, et les séparer ferait relire le même contexte quatre fois. Les
maquettes font foi pour l'apparence — canevas publié, page « Parcours ».

**Fichiers :**
- Créer : `web/src/lib/app/etat.svelte.ts`
- Créer : `web/src/lib/ui/Bouton.svelte`, `web/src/lib/ui/Curseur.svelte`, `web/src/lib/ui/Jauge.svelte`, `web/src/lib/ui/EtatEcran.svelte`
- Créer : `web/src/routes/+page.svelte` (remplace le contenu provisoire)
- Créer : `web/src/routes/reglage/+page.svelte`
- Créer : `web/src/routes/boucles/+page.svelte`
- Créer : `web/src/routes/b/[id]/+page.svelte`
- Créer : `web/src/routes/+page.ts`
- Créer : `web/src/lib/app/etat.test.ts`

**Interfaces :**
- Consomme : tout ce que les tâches 3 à 6 produisent
- Produit : `web/src/lib/app/etat.svelte.ts` exportant un singleton
  `appEtat = { depart, reglages, theme, resultats, moteur }` — les stores
  assemblés une fois, avec les implémentations réelles des ports
- Produit : `formatEcartCible(demandeM: number, obtenueM: number): string`

- [ ] **Étape 1 : écrire le test de l'assemblage et du libellé d'écart**

`web/src/lib/app/etat.test.ts` :

```ts
import { describe, expect, it } from 'vitest';
import { formatEcartCible } from './etat.svelte';

describe('formatEcartCible', () => {
  it('dit la distance demandée quand elle diffère', () => {
    expect(formatEcartCible(18_000, 17_400)).toBe('tu en demandais 18');
  });

  it('ne dit rien quand la distance obtenue est celle demandée', () => {
    // Afficher « tu en demandais 18 » à côté de « 18,0 km » serait du bruit.
    expect(formatEcartCible(18_000, 18_000)).toBe('');
  });

  it('arrondit la demande au kilomètre', () => {
    expect(formatEcartCible(18_400, 17_000)).toBe('tu en demandais 18');
  });

  it('ne dit rien quand la demande est inconnue', () => {
    expect(formatEcartCible(0, 17_400)).toBe('');
  });
});
```

- [ ] **Étape 2 : lancer et constater l'échec**

```
cd web && npx vitest run src/lib/app/etat.test.ts
```

Attendu : ÉCHEC, module introuvable.

- [ ] **Étape 3 : écrire l'assemblage**

`web/src/lib/app/etat.svelte.ts` :

```ts
import type { Depart } from '$lib/domaine/depart';
import { REGLAGES_PAR_DEFAUT, borner, type Reglages } from '$lib/domaine/reglages';
import { creerResultats } from './generation/resultats.svelte';
import { creerMoteurHTTP } from '$lib/infra/hent-api';
import { creerPreferences } from '$lib/infra/stockage';

export type Theme = 'auto' | 'sombre' | 'clair';

const prefs = creerPreferences();
const moteur = creerMoteurHTTP();

/**
 * L'état de l'application, assemblé une fois.
 *
 * `depart` et `resultats` ne sont pas persistés : un départ vieux de trois jours
 * n'a aucun sens, et des résultats périmés encore moins. Seuls les réglages et
 * le thème survivent à la session.
 */
function creerEtat() {
  let depart = $state<Depart | null>(null);
  let reglages = $state<Reglages>(prefs.lire() ?? REGLAGES_PAR_DEFAUT);
  let theme = $state<Theme>('auto');
  const resultats = creerResultats(moteur);

  return {
    moteur,
    resultats,
    get depart() {
      return depart;
    },
    poserDepart(d: Depart | null) {
      depart = d;
    },
    get reglages() {
      return reglages;
    },
    regler(r: Reglages) {
      reglages = borner(r);
      prefs.ecrire(reglages);
    },
    get theme() {
      return theme;
    },
    changerTheme(t: Theme) {
      theme = t;
      if (typeof document !== 'undefined') {
        if (t === 'auto') document.documentElement.removeAttribute('data-theme');
        else document.documentElement.setAttribute('data-theme', t);
      }
    }
  };
}

export const appEtat = creerEtat();

/**
 * Libellé de l'écart à la demande, tel que l'écran de détail l'affiche à côté de
 * la distance obtenue. Rendu vide quand il n'apprendrait rien : demande inconnue,
 * ou distance obtenue égale à la demande arrondie.
 */
export function formatEcartCible(demandeM: number, obtenueM: number): string {
  if (!Number.isFinite(demandeM) || demandeM <= 0) return '';
  const demandeKm = Math.round(demandeM / 1000);
  if (demandeKm === Math.round(obtenueM / 1000) && Math.abs(demandeM - obtenueM) < 500) return '';
  return `tu en demandais ${demandeKm}`;
}
```

- [ ] **Étape 4 : vérifier le vert**

```
cd web && npx vitest run src/lib/app/etat.test.ts
```

Attendu : PASS.

- [ ] **Étape 5 : écrire les composants d'interface**

Les quatre composants reprennent les mesures du système de design : hauteur
tactile de 52 px, rayon de 4 px, anneau de focus de 2 px posé hors du composant.
Toutes les couleurs passent par les jetons — **aucune valeur en dur**.

`web/src/lib/ui/Bouton.svelte` :

```svelte
<script lang="ts">
  type Props = {
    variante?: 'primaire' | 'secondaire';
    href?: string;
    onclick?: () => void;
    children: import('svelte').Snippet;
  };
  let { variante = 'primaire', href, onclick, children }: Props = $props();
</script>

{#if href}
  <a class="bouton {variante}" {href}>{@render children()}</a>
{:else}
  <button class="bouton {variante}" type="button" {onclick}>{@render children()}</button>
{/if}

<style>
  .bouton {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 52px;
    padding: 0 22px;
    border-radius: 4px;
    border: none;
    font: inherit;
    font-weight: 700;
    font-size: 1.0625rem;
    text-decoration: none;
    cursor: pointer;
  }
  .primaire {
    background: var(--accent);
    color: var(--sur-accent);
  }
  .secondaire {
    background: transparent;
    color: var(--accent);
    border: 1px solid var(--trait-vif);
  }
  .bouton:focus-visible {
    outline: 2px solid var(--accent-vif);
    outline-offset: 2px;
  }
</style>
```

`web/src/lib/ui/Curseur.svelte` :

```svelte
<script lang="ts">
  type Props = {
    etiquette: string;
    valeur: number;
    min: number;
    max: number;
    pas?: number;
    /** Valeur lue à voix haute, quand un nombre nu ne veut rien dire. */
    texteValeur: string;
    onchange: (v: number) => void;
  };
  let { etiquette, valeur, min, max, pas = 1, texteValeur, onchange }: Props = $props();
</script>

<div class="reglage">
  <div class="ligne">
    <label for={etiquette}>{etiquette}</label>
    <span class="valeur">{texteValeur}</span>
  </div>
  <input
    id={etiquette}
    type="range"
    {min}
    {max}
    step={pas}
    value={valeur}
    aria-valuetext={texteValeur}
    oninput={(e) => onchange(Number(e.currentTarget.value))}
  />
</div>

<style>
  .ligne {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    margin-bottom: 11px;
  }
  label {
    font-size: 0.875rem;
    color: var(--texte-gris);
    letter-spacing: 0.04em;
  }
  .valeur {
    font-family: Spectral, Georgia, serif;
    font-size: 1.5625rem;
  }
  input[type='range'] {
    width: 100%;
    accent-color: var(--accent-vif);
  }
  input:focus-visible {
    outline: 2px solid var(--accent-vif);
    outline-offset: 2px;
  }
</style>
```

`web/src/lib/ui/Jauge.svelte` :

```svelte
<script lang="ts">
  type Props = { etiquette: string; part: number; texte: string; alerte?: boolean };
  let { etiquette, part, texte, alerte = false }: Props = $props();
</script>

<div>
  <div class="ligne">
    <span class="etiquette">{etiquette}</span>
    <span class="valeur">{texte}</span>
  </div>
  <div class="piste">
    <div class="remplissage" class:alerte style="width: {Math.min(100, Math.max(0, part * 100))}%"></div>
  </div>
</div>

<style>
  .ligne {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    margin-bottom: 6px;
  }
  .etiquette {
    font-size: 0.875rem;
    color: var(--texte-doux);
  }
  .valeur {
    font-family: Spectral, Georgia, serif;
    font-size: 1.0625rem;
  }
  .piste {
    height: 6px;
    background: var(--trait);
    border-radius: 3px;
    overflow: hidden;
  }
  .remplissage {
    height: 100%;
    min-width: 4px;
    background: var(--accent);
  }
  .alerte {
    background: var(--alerte);
  }
</style>
```

`web/src/lib/ui/EtatEcran.svelte` — l'attente et les quatre échecs, chacun avec
sa sortie :

```svelte
<script lang="ts">
  import type { ErreurMoteur } from '$lib/app/ports';
  import Bouton from './Bouton.svelte';

  type Props = {
    erreur?: ErreurMoteur;
    enAttente?: boolean;
    onannuler?: () => void;
    onreessayer?: () => void;
    onassouplir?: () => void;
  };
  let { erreur, enAttente = false, onannuler, onreessayer, onassouplir }: Props = $props();

  const titres: Record<ErreurMoteur['genre'], string> = {
    HorsZone: 'Ce point est en dehors de la Bretagne.',
    AucuneBoucle: 'Pas de boucle à cette distance par ici.',
    TropDeDemandes: 'Une minute, le temps de souffler.',
    DelaiDepasse: 'La recherche a pris trop de temps.',
    Reseau: 'Impossible de joindre le service.'
  };

  const details: Record<ErreurMoteur['genre'], string> = {
    HorsZone: "C'est la seule région cartographiée pour l'instant. Le reste viendra.",
    AucuneBoucle: 'Les chemins autour de ce point ne se referment pas sur cette distance.',
    TropDeDemandes: 'Trop de demandes d’un coup. Le service est petit et gratuit ; il tient debout comme ça.',
    DelaiDepasse: 'Elle aboutit parfois au second essai.',
    Reseau: 'Vérifie ta connexion, puis réessaie.'
  };
</script>

{#if enAttente}
  <div class="bloc" role="status">
    <h2>Je parcours les chemins.</h2>
    <p>Une poignée de secondes, le temps de comparer les boucles.</p>
    {#if onannuler}
      <Bouton variante="secondaire" onclick={onannuler}>Annuler</Bouton>
    {/if}
  </div>
{:else if erreur}
  <div class="bloc" role="alert">
    <h2>{titres[erreur.genre]}</h2>
    <p>{details[erreur.genre]}</p>
    {#if erreur.genre === 'AucuneBoucle' && onassouplir}
      <Bouton onclick={onassouplir}>Accepter plus de bitume</Bouton>
    {:else if erreur.genre === 'HorsZone'}
      <Bouton href="/reglage">Choisir un autre départ</Bouton>
    {:else if onreessayer}
      <Bouton variante="secondaire" onclick={onreessayer}>
        {erreur.reessayerDansS ? `Réessayer dans ${erreur.reessayerDansS} s` : 'Réessayer'}
      </Bouton>
    {/if}
  </div>
{/if}

<style>
  .bloc {
    display: flex;
    flex-direction: column;
    gap: 18px;
    padding: 22px 20px;
  }
  h2 {
    font-family: Spectral, Georgia, serif;
    font-size: 1.1875rem;
    font-weight: 400;
    margin: 0;
  }
  p {
    margin: 0;
    color: var(--texte-doux);
    font-size: 0.9375rem;
    line-height: 1.55;
  }
</style>
```

- [ ] **Étape 6 : écrire les quatre écrans**

`web/src/routes/+page.ts` — la page d'accueil est pré-rendue, c'est ce qui la
rend indexable :

```ts
export const prerender = true;
```

`web/src/routes/+page.svelte` — l'accueil, texte des maquettes :

```svelte
<script lang="ts">
  import Bouton from '$lib/ui/Bouton.svelte';
</script>

<svelte:head>
  <title>hent — une boucle, jamais un aller-retour</title>
  <meta
    name="description"
    content="Dis où tu pars et combien de kilomètres. hent te rend une boucle qui fuit le bitume et revient à ton point de départ. Bretagne, sans compte."
  />
</svelte:head>

<main>
  <h1>Une boucle.<br />Jamais un aller‑retour.</h1>
  <p>
    Dis où tu pars et combien de kilomètres. <em>hent</em> te rend une boucle qui fuit le bitume et
    revient à ton point de départ.
  </p>
  <Bouton href="/reglage">Tracer ma boucle</Bouton>

  <section>
    <h2>Ce que fait hent</h2>
    <dl>
      <dt>Des vraies boucles</dt>
      <dd>Le tracé ne repasse pas sur ses pas. C’est plus dur à calculer qu’un simple trajet, et c’est tout l’intérêt.</dd>
      <dt>Le bitume en dernier recours</dt>
      <dd>Sentiers, chemins creux, pistes forestières. Tu règles le curseur, le tracé s’écarte des routes.</dd>
      <dt>Un GPX, et tu pars</dt>
      <dd>Le fichier se charge dans ta montre ou ton téléphone. Rien à installer, pas de compte.</dd>
    </dl>
  </section>

  <footer>
    Données <a href="https://www.openstreetmap.org/copyright">© les contributeurs OpenStreetMap</a>,
    sous licence ODbL.<br />
    <em>hent</em> — « chemin », en breton.
  </footer>
</main>

<style>
  main {
    max-width: 390px;
    margin: 0 auto;
    padding: 56px 24px 44px;
    display: flex;
    flex-direction: column;
    gap: 22px;
    background: var(--fond);
    color: var(--texte);
    min-height: 100vh;
  }
  h1 {
    font-family: Spectral, Georgia, serif;
    font-size: 2.5rem;
    line-height: 1.1;
    font-weight: 600;
    margin: 0;
  }
  h2 {
    font-family: Spectral, Georgia, serif;
    font-size: 0.8125rem;
    font-weight: 400;
    letter-spacing: 0.16em;
    text-transform: uppercase;
    color: var(--texte-gris);
  }
  p {
    margin: 0;
    color: var(--texte-doux);
    font-size: 1.0625rem;
  }
  dt {
    font-family: Spectral, Georgia, serif;
    font-size: 1.1875rem;
    margin-top: 22px;
  }
  dd {
    margin: 3px 0 0;
    color: var(--texte-doux);
    font-size: 0.9375rem;
  }
  footer {
    font-size: 0.75rem;
    color: var(--texte-gris);
    line-height: 1.6;
  }
  a {
    color: var(--accent);
    text-underline-offset: 3px;
  }
</style>
```

`web/src/routes/reglage/+page.svelte` — les réglages et le départ provisoire :

```svelte
<script lang="ts">
  import { goto } from '$app/navigation';
  import { appEtat } from '$lib/app/etat.svelte';
  import { DISTANCE_MAX_M, DISTANCE_MIN_M } from '$lib/domaine/reglages';
  import { estCoordValide } from '$lib/domaine/depart';
  import { formatDistance } from '$lib/domaine/format';
  import Bouton from '$lib/ui/Bouton.svelte';
  import Curseur from '$lib/ui/Curseur.svelte';
  import EtatEcran from '$lib/ui/EtatEcran.svelte';

  // Saisie provisoire du départ : le plan 2 la remplace par la carte et la
  // recherche d'adresse. Elle existe pour que la chaîne soit utilisable de bout
  // en bout dès maintenant.
  let lat = $state(48.117);
  let lon = $state(-1.677);

  const etat = $derived(appEtat.resultats.etat());
  const coordValide = $derived(estCoordValide({ lat, lon }));

  const motsBitume = ['jamais', 'un peu', 'moyennement', 'beaucoup', 'autant que possible'];
  const motBitume = $derived(
    motsBitume[Math.min(motsBitume.length - 1, Math.floor(appEtat.reglages.eviterBitume * motsBitume.length))]!
  );

  async function tracer() {
    appEtat.poserDepart({ coord: { lat, lon }, libelle: `${lat.toFixed(4)}, ${lon.toFixed(4)}` });
    await appEtat.resultats.lancer({
      depart: { lat, lon },
      distanceM: appEtat.reglages.distanceM,
      eviterBitume: appEtat.reglages.eviterBitume,
      maxResultats: 5,
      variante: 0
    });
    if (appEtat.resultats.etat().statut === 'ok') await goto('/boucles');
  }

  function assouplir() {
    appEtat.regler({ ...appEtat.reglages, eviterBitume: Math.max(0, appEtat.reglages.eviterBitume - 0.3) });
    void tracer();
  }
</script>

<svelte:head><title>Régler — hent</title></svelte:head>

<main>
  <h1>Ta boucle</h1>

  <Curseur
    etiquette="Distance"
    valeur={appEtat.reglages.distanceM}
    min={DISTANCE_MIN_M}
    max={DISTANCE_MAX_M}
    pas={500}
    texteValeur={formatDistance(appEtat.reglages.distanceM)}
    onchange={(v) => appEtat.regler({ ...appEtat.reglages, distanceM: v })}
  />

  <Curseur
    etiquette="Éviter le bitume"
    valeur={appEtat.reglages.eviterBitume}
    min={0}
    max={1}
    pas={0.05}
    texteValeur={motBitume}
    onchange={(v) => appEtat.regler({ ...appEtat.reglages, eviterBitume: v })}
  />

  <fieldset>
    <legend>Départ</legend>
    <p class="provisoire">Saisie temporaire : la carte et la recherche d’adresse arrivent ensuite.</p>
    <label>Latitude <input type="number" step="0.0001" bind:value={lat} /></label>
    <label>Longitude <input type="number" step="0.0001" bind:value={lon} /></label>
    {#if !coordValide}
      <p class="invalide">Ces coordonnées ne sont pas valides.</p>
    {/if}
  </fieldset>

  {#if etat.statut === 'calcul'}
    <EtatEcran enAttente onannuler={() => appEtat.resultats.annuler()} />
  {:else if etat.statut === 'erreur'}
    <EtatEcran erreur={etat.erreur} onreessayer={tracer} onassouplir={assouplir} />
  {:else}
    <Bouton onclick={tracer}>Tracer ma boucle</Bouton>
  {/if}
</main>

<style>
  main {
    max-width: 390px;
    margin: 0 auto;
    padding: 24px 20px 40px;
    display: flex;
    flex-direction: column;
    gap: 22px;
    background: var(--fond);
    color: var(--texte);
    min-height: 100vh;
  }
  h1 {
    font-family: Spectral, Georgia, serif;
    font-size: 1.5625rem;
    font-weight: 600;
    margin: 0;
  }
  fieldset {
    border: 1px solid var(--trait);
    border-radius: 4px;
    padding: 13px;
  }
  legend {
    font-size: 0.875rem;
    color: var(--texte-gris);
  }
  label {
    display: block;
    margin-top: 9px;
    font-size: 0.875rem;
    color: var(--texte-doux);
  }
  input[type='number'] {
    width: 100%;
    min-height: 44px;
    padding: 0 11px;
    background: var(--tuile);
    color: var(--texte);
    border: 1px solid var(--trait-vif);
    border-radius: 4px;
    font: inherit;
  }
  input:focus-visible {
    outline: 2px solid var(--accent-vif);
    outline-offset: 2px;
  }
  .provisoire,
  .invalide {
    margin: 0;
    font-size: 0.8125rem;
    color: var(--texte-gris);
  }
  .invalide {
    color: var(--alerte);
  }
</style>
```

`web/src/routes/boucles/+page.svelte` — la liste :

```svelte
<script lang="ts">
  import { appEtat } from '$lib/app/etat.svelte';
  import { dureeMinutes } from '$lib/domaine/boucle';
  import { formatDistance, formatDuree, formatPourcent } from '$lib/domaine/format';

  const etat = $derived(appEtat.resultats.etat());
</script>

<svelte:head><title>Les boucles — hent</title></svelte:head>

<main>
  <a class="retour" href="/reglage">← Changer les réglages</a>

  {#if etat.statut === 'ok'}
    <h1>{etat.boucles.length === 1 ? 'Une boucle' : `${etat.boucles.length} boucles`}</h1>
    <p class="tri">la plus verte d’abord</p>
    <ul>
      {#each etat.boucles as boucle (boucle.id)}
        <li>
          <a href="/b/{encodeURIComponent(boucle.id)}">
            <span class="distance">{formatDistance(boucle.score.distanceM)}</span>
            <span class="mesures">
              {formatPourcent(boucle.score.partNonBitume)} hors bitume ·
              {formatDuree(dureeMinutes(boucle.score.distanceM, 8))}
            </span>
          </a>
        </li>
      {/each}
    </ul>
  {:else}
    <p class="vide">Aucune recherche en cours. <a href="/reglage">Régler une boucle</a></p>
  {/if}
</main>

<style>
  main {
    max-width: 390px;
    margin: 0 auto;
    padding: 24px 20px 40px;
    background: var(--fond);
    color: var(--texte);
    min-height: 100vh;
  }
  h1 {
    font-family: Spectral, Georgia, serif;
    font-size: 1.5625rem;
    font-weight: 600;
    margin: 22px 0 0;
  }
  .tri,
  .vide {
    color: var(--texte-gris);
    font-size: 0.8125rem;
  }
  ul {
    list-style: none;
    padding: 0;
    margin: 22px 0 0;
  }
  li a {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-height: 52px;
    padding: 11px 13px;
    border-bottom: 1px solid var(--trait);
    color: inherit;
    text-decoration: none;
  }
  li a:focus-visible {
    outline: 2px solid var(--accent-vif);
    outline-offset: 2px;
  }
  .distance {
    font-family: Spectral, Georgia, serif;
    font-size: 1.5625rem;
  }
  .mesures {
    font-size: 0.875rem;
    color: var(--texte-gris);
  }
  .retour {
    color: var(--accent);
    font-size: 0.875rem;
    text-underline-offset: 3px;
  }
</style>
```

`web/src/routes/b/[id]/+page.svelte` — le détail, avec ses **deux chemins
d'entrée** :

```svelte
<script lang="ts">
  import { page } from '$app/state';
  import { appEtat, formatEcartCible } from '$lib/app/etat.svelte';
  import { dureeMinutes, type Boucle, type Demande } from '$lib/domaine/boucle';
  import { formatDistance, formatDuree, formatPourcent } from '$lib/domaine/format';
  import type { ErreurMoteur } from '$lib/app/ports';
  import Bouton from '$lib/ui/Bouton.svelte';
  import EtatEcran from '$lib/ui/EtatEcran.svelte';
  import Jauge from '$lib/ui/Jauge.svelte';

  const id = $derived(page.params.id ?? '');

  let boucle = $state<Boucle | null>(null);
  let demande = $state<Demande | null>(null);
  let erreur = $state<ErreurMoteur | null>(null);
  let chargement = $state(false);

  // Deux chemins d'entrée : on arrive de la liste, ou par un lien partagé. Le
  // second impose un appel, puisque rien n'est en mémoire.
  $effect(() => {
    const etat = appEtat.resultats.etat();
    if (etat.statut === 'ok') {
      const connue = etat.boucles.find((b) => b.id === id);
      if (connue) {
        boucle = connue;
        demande = etat.demande;
        erreur = null;
        return;
      }
    }
    if (!id || boucle?.id === id) return;

    chargement = true;
    erreur = null;
    appEtat.moteur
      .ouvrir(id)
      .then((r) => {
        boucle = r.boucle;
        demande = r.demande;
      })
      .catch((e: unknown) => {
        const g = e && typeof e === 'object' && 'genre' in e ? e : null;
        erreur = g
          ? { genre: (g as ErreurMoteur).genre, message: (g as ErreurMoteur).message }
          : { genre: 'Reseau', message: String(e) };
      })
      .finally(() => {
        chargement = false;
      });
  });

  const ecart = $derived(boucle && demande ? formatEcartCible(demande.distanceM, boucle.score.distanceM) : '');
</script>

<svelte:head><title>Une boucle — hent</title></svelte:head>

<main>
  <a class="retour" href="/boucles">← Les boucles</a>

  {#if chargement}
    <EtatEcran enAttente />
  {:else if erreur}
    <EtatEcran erreur={erreur} />
  {:else if boucle}
    <div class="titre">
      <span class="distance">{formatDistance(boucle.score.distanceM)}</span>
      {#if ecart}<span class="ecart">{ecart}</span>{/if}
    </div>

    <Jauge
      etiquette="Hors bitume"
      part={boucle.score.partNonBitume}
      texte={formatPourcent(boucle.score.partNonBitume)}
    />
    <Jauge
      etiquette="Exposition au trafic"
      part={boucle.score.partTrafic}
      texte={formatPourcent(boucle.score.partTrafic)}
      alerte
    />

    <dl class="tuiles">
      <div><dt>{formatDuree(dureeMinutes(boucle.score.distanceM, 8))}</dt><dd>à 8 km/h</dd></div>
      <div><dt>{formatPourcent(boucle.score.partRetracee)}</dt><dd>de chemin refait</dd></div>
    </dl>

    <Bouton href={appEtat.moteur.urlGPX(boucle.id)}>Télécharger le GPX</Bouton>
    <p class="partage">
      Le lien de cette page contient ton point de départ : ne le partage qu’en connaissance de cause.
    </p>
  {/if}
</main>

<style>
  main {
    max-width: 390px;
    margin: 0 auto;
    padding: 24px 20px 40px;
    display: flex;
    flex-direction: column;
    gap: 20px;
    background: var(--fond);
    color: var(--texte);
    min-height: 100vh;
  }
  .titre {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
  }
  .distance {
    font-family: Spectral, Georgia, serif;
    font-size: 2.25rem;
    line-height: 1;
  }
  .ecart,
  .partage {
    font-size: 0.8125rem;
    color: var(--texte-gris);
  }
  .tuiles {
    display: flex;
    gap: 9px;
    margin: 0;
  }
  .tuiles > div {
    flex-grow: 1;
    padding: 11px 13px;
    background: var(--tuile);
    border: 1px solid var(--trait);
    border-radius: 4px;
  }
  dt {
    font-family: Spectral, Georgia, serif;
    font-size: 1.1875rem;
  }
  dd {
    margin: 2px 0 0;
    font-size: 0.75rem;
    color: var(--texte-gris);
    letter-spacing: 0.04em;
  }
  .retour {
    color: var(--accent);
    font-size: 0.875rem;
    text-underline-offset: 3px;
  }
</style>
```

- [ ] **Étape 7 : vérifier que tout compile et tourne**

```
cd web && npm test && npm run check && npm run build
```

Attendu : tests verts, aucune erreur de type, `build/` produit.

`npm run check` est ce qui attrape les erreurs de type dans les composants
Svelte — Vitest ne les voit pas, puisqu'il ne teste pas le rendu.

- [ ] **Étape 8 : vérifier à l'œil, avec le vrai serveur**

Lancer le back dans un terminal :

```
CGO_ENABLED=0 go build -o routed ./cmd/routed && ./routed -graph graph.bin
```

Le chargement du graphe prend une quinzaine de secondes. Puis, dans un autre
terminal :

```
cd web && npm run dev
```

Ouvrir la page, régler une distance, tracer, choisir une boucle, télécharger le
GPX. Vérifier aussi **le chemin à froid** : recharger la page de détail (F5) —
elle doit se réafficher, en passant cette fois par l'appel à l'API.

Si `graph.bin` n'est pas présent, le construire prend près d'une heure : dans ce
cas, signaler que la vérification visuelle n'a pas pu être faite plutôt que de
la déclarer faite.

- [ ] **Étape 9 : commit**

```bash
git add web/src/
git commit -m "feat: écrans d'accueil, de réglage, de résultats et de détail"
```

---

### Tâche 8 : Le déploiement statique et la documentation

**Fichiers :**
- Créer : `deploy/Caddyfile`
- Créer : `Makefile` à la racine du dépôt
- Modifier : `docs/etat-des-lieux.md`
- Modifier : `README.md`

**Interfaces :**
- Consomme : la sortie de `npm run build`, dans `web/build/`

- [ ] **Étape 1 : écrire le Caddyfile**

`deploy/Caddyfile` :

```
# hent — le front est du statique, l'API un service local.
#
# Séparer les deux permet de déployer une correction d'interface sans toucher
# au service : routed met une quinzaine de secondes à répondre après un
# redémarrage, le temps de charger les 294 Mo de graphe.

hent.example.org {
	encode zstd gzip

	# L'API d'abord : ce préfixe ne doit jamais tomber dans le repli SPA.
	handle /v1/* {
		reverse_proxy localhost:8080
	}

	handle /healthz {
		reverse_proxy localhost:8080
	}

	handle {
		root * /srv/hent/web
		# Les routes du front sont rendues côté navigateur : tout ce qui n'est
		# pas un fichier réel retombe sur index.html, sans quoi /b/<id> donnerait
		# un 404 au rechargement.
		try_files {path} /index.html
		file_server
	}
}
```

- [ ] **Étape 2 : écrire le Makefile**

`Makefile` à la racine :

```make
# Le front doit être bâti avant le back quand on produit un artefact complet :
# rien ne l'embarque, mais la cible `build` les veut tous deux à jour.

CGO := CGO_ENABLED=0

.PHONY: aide web serve dev test build

aide:
	@echo "web    — bâtit le front dans web/build/"
	@echo "serve  — lance routed sur le graphe local"
	@echo "dev    — front et API en parallèle, pour développer"
	@echo "test   — go test ./... puis vitest"
	@echo "build  — front et binaires"

web:
	cd web && npm ci && npm run build

serve:
	$(CGO) go run ./cmd/routed -graph graph.bin

dev:
	@echo "Lancer « make serve » dans un autre terminal, puis :"
	cd web && npm run dev

test:
	$(CGO) go test ./...
	cd web && npm test

build: web
	$(CGO) go build -o routed ./cmd/routed
	$(CGO) go build -o graphbuild ./cmd/graphbuild
```

- [ ] **Étape 3 : vérifier que les cibles marchent**

```
make test
```

Attendu : la suite Go puis la suite Vitest, toutes deux vertes.

- [ ] **Étape 4 : documenter**

Dans `docs/etat-des-lieux.md`, section « Par où commencer », remplacer le
paragraphe qui commence par « **Le front**, dont la conception est faite » par :

> **Le front, premier jet.** Le socle est livré : trois couches gardées par un
> test d'imports, le client de l'API avec ses cinq variantes d'erreur, les
> préférences persistées, et la machine à états de la recherche. Quatre écrans
> fonctionnent — accueil, réglage, résultats, détail — et la chaîne va jusqu'au
> téléchargement du GPX. Le point de départ se saisit encore en coordonnées
> brutes : la carte, la géolocalisation et la recherche d'adresse font l'objet du
> plan suivant, et le champ provisoire le dit à l'écran.
>
> Se bâtit par `make web`, se sert en copiant `web/build/` vers `/srv/hent/web`.
> `deploy/Caddyfile` donne la configuration : l'API en proxy sur `/v1/*`, le reste
> en repli vers `index.html` — sans quoi recharger `/b/<id>` donnerait un 404.

Dans `README.md`, ajouter avant la section des licences :

> ## Développer
>
> ```sh
> make serve   # l'API, sur le graphe local (une quinzaine de secondes au démarrage)
> make dev     # le front, qui relaie /v1 vers l'API — dans un second terminal
> make test    # la suite Go, puis celle du front
> ```

- [ ] **Étape 5 : commit**

```bash
git add deploy/ Makefile docs/etat-des-lieux.md README.md
git commit -m "feat: service statique par Caddy, cibles de build et documentation"
```

---

## Ce que ce plan ne fait pas

Tout cela appartient au plan 2 :

- **MapLibre et le fond de carte.** Les styles existent dans `design/carte/`,
  rien ne les consomme encore.
- **L'écran de départ définitif** — géolocalisation, recherche d'adresse,
  réticule sur la carte. Le champ de coordonnées de la tâche 7 est une béquille,
  et il est marqué comme telle à l'écran.
- **Le géocodage BAN**, avec son anti-rebond et l'annulation des requêtes
  périmées.
- **La bascule de thème à l'écran.** `appEtat.changerTheme` existe et pose
  l'attribut, mais aucun bouton ne l'appelle : la carte doit exister d'abord,
  puisque changer de thème échange son style.
- **L'alternative textuelle de la carte** — elle n'a pas lieu d'être tant qu'il
  n'y a pas de carte. Les chiffres de l'écran de détail la portent déjà.
