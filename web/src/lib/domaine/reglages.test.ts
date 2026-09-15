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
