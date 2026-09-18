import { describe, expect, it } from 'vitest';
import { bornesDe, dureeMinutes, type Boucle } from './boucle';

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

  it('rend zéro pour une distance absente ou absurde', () => {
    // Sans garde, un NaN ressortirait NaN et une distance négative une durée
    // négative — « -12 min » est un affichage que rien n'arrêterait ensuite.
    expect(dureeMinutes(Number.NaN, 8)).toBe(0);
    expect(dureeMinutes(Number.POSITIVE_INFINITY, 8)).toBe(0);
    expect(dureeMinutes(-10_000, 8)).toBe(0);
  });
});

const score = { distanceM: 1, partNonBitume: 0, partTrafic: 0, partRetracee: 0, ecartCible: 0 };

describe('bornesDe', () => {
  it('englobe tous les points de toutes les boucles', () => {
    const a: Boucle = { id: 'a', score, geometrie: [[-1.7, 48.1], [-1.6, 48.2]] };
    const b: Boucle = { id: 'b', score, geometrie: [[-1.8, 48.15]] };
    expect(bornesDe([a, b])).toEqual([[-1.8, 48.1], [-1.6, 48.2]]);
  });

  it('rend null sans aucun point', () => {
    expect(bornesDe([])).toBeNull();
    expect(bornesDe([{ id: 'a', score, geometrie: [] }])).toBeNull();
  });
});
