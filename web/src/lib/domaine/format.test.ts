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
