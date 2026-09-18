import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';

const racine = new URL('../../../..', import.meta.url).pathname;
const themes = ['sombre', 'clair'] as const;

type Style = {
  version: number;
  sources: Record<string, { type: string; attribution?: string }>;
  layers: { id: string; type: string; paint?: Record<string, unknown> }[];
};

function lire(chemin: string): Style {
  return JSON.parse(readFileSync(chemin, 'utf8')) as Style;
}

describe('styles de carte', () => {
  for (const theme of themes) {
    it(`sert hent-${theme} tel que le design l'a généré`, () => {
      const source = readFileSync(join(racine, 'design', 'carte', `hent-${theme}.json`), 'utf8');
      const servi = readFileSync(join(racine, 'web', 'static', 'carte', `hent-${theme}.json`), 'utf8');
      expect(servi).toBe(source);
    });

    it(`hent-${theme} est un style MapLibre attribué à l'IGN`, () => {
      const style = lire(join(racine, 'web', 'static', 'carte', `hent-${theme}.json`));
      expect(style.version).toBe(8);
      expect(style.sources['plan-ign']?.attribution).toContain('© IGN');
      expect(style.layers.length).toBeGreaterThan(5);
    });
  }

  it('pose le chemin au-dessus de la route', () => {
    const ids = lire(join(racine, 'web', 'static', 'carte', 'hent-sombre.json')).layers.map((l) => l.id);
    expect(ids.indexOf('chemins')).toBeGreaterThan(ids.indexOf('routes'));
    expect(ids.indexOf('routes')).toBeGreaterThan(-1);
  });

  it('distingue les deux thèmes', () => {
    const fond = (theme: string) =>
      lire(join(racine, 'web', 'static', 'carte', `hent-${theme}.json`)).layers.find((l) => l.id === 'fond')
        ?.paint?.['background-color'];
    expect(fond('sombre')).toBeDefined();
    expect(fond('sombre')).not.toBe(fond('clair'));
  });
});
