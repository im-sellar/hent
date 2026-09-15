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
    // Contrairement aux réglages, aucune garde n'est requise ici : NaN rend
    // toute comparaison fausse, et un infini sort des bornes par construction.
    expect(estCoordValide({ lat: Number.NaN, lon: 0 })).toBe(false);
    expect(estCoordValide({ lat: 0, lon: Number.POSITIVE_INFINITY })).toBe(false);
  });

  it('accepte le point zéro', () => {
    // Zéro est une coordonnée valide ; la refuser par un test de véracité
    // serait le bogue classique.
    expect(estCoordValide({ lat: 0, lon: 0 })).toBe(true);
  });
});
