import { describe, expect, it } from 'vitest';
import { CENTRE_BRETAGNE, centreDe, contient, type Zone } from './zone';

const bretagne: Zone = { minLat: 47.2, minLon: -5.2, maxLat: 48.95, maxLon: -0.95 };

describe('contient', () => {
  it('accepte un point dedans et ses bornes', () => {
    expect(contient(bretagne, { lat: 48.1, lon: -1.7 })).toBe(true);
    expect(contient(bretagne, { lat: 47.2, lon: -5.2 })).toBe(true);
    expect(contient(bretagne, { lat: 48.95, lon: -0.95 })).toBe(true);
  });

  it('refuse un point hors bornes sur chaque axe', () => {
    expect(contient(bretagne, { lat: 49.0, lon: -1.7 })).toBe(false);
    expect(contient(bretagne, { lat: 48.1, lon: -0.9 })).toBe(false);
  });

  it('refuse une coordonnée non finie', () => {
    expect(contient(bretagne, { lat: NaN, lon: -1.7 })).toBe(false);
    expect(contient(bretagne, { lat: 48.1, lon: Infinity })).toBe(false);
  });
});

describe('centreDe', () => {
  it('rend le milieu de chaque axe', () => {
    expect(centreDe({ minLat: 47, minLon: -5, maxLat: 49, maxLon: -1 })).toEqual({ lat: 48, lon: -3 });
  });
});

describe('CENTRE_BRETAGNE', () => {
  it('est dans la zone couverte', () => {
    expect(contient(bretagne, CENTRE_BRETAGNE)).toBe(true);
  });
});
