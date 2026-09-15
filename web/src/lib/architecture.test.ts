import { describe, expect, it } from 'vitest';
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join } from 'node:path';

const lib = new URL('.', import.meta.url).pathname;
const src = new URL('..', import.meta.url).pathname;

/**
 * Règles de dépendance entre couches, calquées sur `internal/architecture_test.go`
 * du back. Une couche ne doit jamais importer celles listées en face d'elle.
 *
 * `routes/` y figure au même titre que `ui/` : ce sont les écrans, la plus
 * grosse part de l'interface. Seul `assemblage.svelte.ts` reste hors de toute
 * couche gardée, à la racine de `lib/` — c'est le point d'assemblage, il doit
 * pouvoir connaître `infra`.
 */
const couches = [
  { nom: 'domaine', dossier: join(lib, 'domaine'), bannies: ['app', 'infra', 'ui'] },
  { nom: 'app', dossier: join(lib, 'app'), bannies: ['infra', 'ui'] },
  { nom: 'ui', dossier: join(lib, 'ui'), bannies: ['infra'] },
  { nom: 'routes', dossier: join(src, 'routes'), bannies: ['infra'] }
];

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

/**
 * Chemins importés par une source, sous les trois formes qu'accepte le langage :
 * `from '…'`, l'import à effet de bord `import '…'`, et l'import dynamique
 * `import('…')`. Ne reconnaître que la première laisserait passer les deux
 * formes par lesquelles on contournerait volontairement la règle.
 */
export function extraireImports(source: string): string[] {
  return [...source.matchAll(/(?:from|import)\s*\(?\s*['"]([^'"]+)['"]/g)].map((m) => m[1]!);
}

/** Dit si un import vise la couche donnée, en respectant la frontière de segment. */
export function viseLaCouche(specifieur: string, couche: string): boolean {
  const segments = specifieur.split('/').filter((s) => s !== '.' && s !== '..');
  const depart = segments[0] === '$lib' ? 1 : 0;
  return segments[depart] === couche;
}

describe('règles de dépendance entre couches', () => {
  for (const { nom, dossier, bannies } of couches) {
    it(`${nom} n'importe pas ${bannies.join(', ')}`, () => {
      const fichiers = fichiersSources(dossier);

      // Sans cette garde, une couche vide ou un chemin faux rendrait le test
      // vert sans avoir rien inspecté.
      expect(fichiers.length, `aucun fichier inspecté dans ${nom}`).toBeGreaterThan(0);

      const fautes: string[] = [];
      for (const fichier of fichiers) {
        for (const specifieur of extraireImports(readFileSync(fichier, 'utf8'))) {
          for (const bannie of bannies) {
            if (viseLaCouche(specifieur, bannie)) {
              fautes.push(`${fichier.replace(src, '')} importe ${specifieur}`);
            }
          }
        }
      }
      expect(fautes, `la couche ${nom} ne doit pas dépendre de ${bannies.join(', ')}`).toEqual([]);
    });
  }
});

describe('extraireImports', () => {
  it('voit les trois formes d’import', () => {
    expect(extraireImports("import { a } from '$lib/domaine/boucle';")).toEqual(['$lib/domaine/boucle']);
    expect(extraireImports("import '$lib/infra/hent-api';")).toEqual(['$lib/infra/hent-api']);
    expect(extraireImports("const m = await import('$lib/infra/stockage');")).toEqual([
      '$lib/infra/stockage'
    ]);
  });

  it('ne prend pas un import de type pour un chemin', () => {
    expect(extraireImports("import type { Boucle } from './boucle';")).toEqual(['./boucle']);
  });
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
