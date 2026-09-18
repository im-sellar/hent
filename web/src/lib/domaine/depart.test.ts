import { describe, expect, it } from 'vitest';
import { estCoordValide, estLatValide, estLonValide, libelleParDefaut, memePoint } from './depart';

describe('estLatValide et estLonValide', () => {
  it('n’ont pas les mêmes bornes', () => {
    // 150 est une longitude valide et une latitude impossible : c'est ce qui
    // permet au formulaire de désigner le champ fautif plutôt que la paire.
    expect(estLatValide(150)).toBe(false);
    expect(estLonValide(150)).toBe(true);
  });

  it('acceptent leurs bornes', () => {
    expect(estLatValide(90)).toBe(true);
    expect(estLatValide(-90)).toBe(true);
    expect(estLonValide(180)).toBe(true);
    expect(estLonValide(-180)).toBe(true);
  });

  it('refusent juste au-delà', () => {
    expect(estLatValide(90.0001)).toBe(false);
    expect(estLonValide(180.0001)).toBe(false);
  });
});

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

describe('libelleParDefaut', () => {
  it('écrit les coordonnées à quatre décimales, virgule décimale', () => {
    expect(libelleParDefaut({ lat: 48.117, lon: -1.677 })).toBe('48,1170, -1,6770');
  });

  it('arrondit plutôt que tronquer', () => {
    expect(libelleParDefaut({ lat: 48.11706, lon: 0 })).toBe('48,1171, 0,0000');
  });
});

describe('memePoint', () => {
  it('confond deux points à moins d’un mètre, pas au-delà', () => {
    expect(memePoint({ lat: 48.1, lon: -1.7 }, { lat: 48.100004, lon: -1.700004 })).toBe(true);
    expect(memePoint({ lat: 48.1, lon: -1.7 }, { lat: 48.1001, lon: -1.7 })).toBe(false);
    expect(memePoint({ lat: 48.1, lon: -1.7 }, { lat: 48.1, lon: -1.7001 })).toBe(false);
  });

  it('ne confond jamais un point non fini avec un autre', () => {
    expect(memePoint({ lat: NaN, lon: -1.7 }, { lat: NaN, lon: -1.7 })).toBe(false);
  });
});
