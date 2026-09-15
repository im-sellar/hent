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
