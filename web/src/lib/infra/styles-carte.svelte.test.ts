import { afterEach, describe, expect, it } from 'vitest';
import { couleursDepuisJetons, webglDisponible } from './styles-carte';

afterEach(() => document.documentElement.removeAttribute('style'));

describe('couleursDepuisJetons', () => {
  it('apparie chaque jeton à son usage sur la carte', () => {
    const racine = document.documentElement;
    racine.style.setProperty('--accent', '#111111');
    racine.style.setProperty('--voie', '#222222');
    racine.style.setProperty('--accent-vif', '#333333');
    racine.style.setProperty('--fond', '#444444');
    racine.style.setProperty('--alerte', '#555555');

    expect(couleursDepuisJetons(racine)).toEqual({
      trace: '#111111',
      ecartee: '#222222',
      depart: '#333333',
      contourDepart: '#444444',
      zone: '#555555'
    });
  });
});

describe('webglDisponible', () => {
  it('dit non quand le canevas ne rend aucun contexte', () => {
    expect(webglDisponible()).toBe(false);
  });

  it('dit oui quand un contexte est rendu', () => {
    const doc = { createElement: () => ({ getContext: () => ({}) }) } as unknown as Document;

    expect(webglDisponible(doc)).toBe(true);
  });

  it('dit non quand la création du canevas lève', () => {
    const doc = {
      createElement: () => {
        throw new Error('pas de canevas');
      }
    } as unknown as Document;

    expect(webglDisponible(doc)).toBe(false);
  });
});
