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

/**
 * `etat.svelte.ts` est le point d'assemblage de l'application : il relie les
 * implémentations concrètes d'`infra` aux ports que le reste de la couche
 * `app` consomme. C'est le même rôle que joue `cmd/routed/main.go` côté back,
 * délibérément situé hors de `internal/` pour la même raison. Rien de tel
 * n'existe ici hors de `lib/` : le lui interdire forcerait un point
 * d'assemblage séparé, pour un bénéfice nul — cette règle continue de garantir
 * que rien d'autre dans `app` ne dépend d'`infra`.
 */
const exceptions: Record<string, string[]> = {
  app: ['etat.svelte.ts']
};

const racine = new URL('.', import.meta.url).pathname;

function fichiersSources(dossier: string, exclus: string[] = []): string[] {
  let trouves: string[] = [];
  for (const entree of readdirSync(dossier)) {
    const chemin = join(dossier, entree);
    if (statSync(chemin).isDirectory()) {
      trouves = trouves.concat(fichiersSources(chemin, exclus));
    } else if (/\.(ts|svelte)$/.test(entree) && !entree.endsWith('.test.ts') && !exclus.includes(entree)) {
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
      const fichiers = fichiersSources(dossier, exceptions[couche] ?? []);

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
