import { describe, expect, it } from 'vitest';
import { creerPreferences } from './stockage';
import { REGLAGES_PAR_DEFAUT } from '$lib/domaine/reglages';

/** Stockage en mémoire, suffisant pour ce que le port promet. */
function stockageMemoire(initial: Record<string, string> = {}): Storage {
  const donnees = new Map(Object.entries(initial));
  return {
    getItem: (c: string) => donnees.get(c) ?? null,
    setItem: (c: string, v: string) => void donnees.set(c, v),
    removeItem: (c: string) => void donnees.delete(c),
    clear: () => donnees.clear(),
    key: (i: number) => [...donnees.keys()][i] ?? null,
    get length() {
      return donnees.size;
    }
  } as Storage;
}

/** Stockage qui refuse tout, comme en navigation privée ou quota dépassé. */
function stockageQuiLeve(): Storage {
  const lever = () => {
    throw new DOMException('refusé', 'SecurityError');
    };
  return { getItem: lever, setItem: lever, removeItem: lever, clear: lever, key: lever, length: 0 } as unknown as Storage;
}

describe('creerPreferences', () => {
  it('relit ce qui a été écrit', () => {
    const prefs = creerPreferences(stockageMemoire());
    prefs.ecrire({ distanceM: 18_000, eviterBitume: 0.8 });
    expect(prefs.lire()).toEqual({ distanceM: 18_000, eviterBitume: 0.8 });
  });

  it("rend null quand rien n'a été écrit", () => {
    expect(creerPreferences(stockageMemoire()).lire()).toBeNull();
  });

  it("rend null sur un contenu illisible plutôt que de lever", () => {
    const prefs = creerPreferences(stockageMemoire({ 'hent.reglages': 'pas du json' }));
    expect(prefs.lire()).toBeNull();
  });

  it("borne ce qu'il relit", () => {
    // Une valeur écrite par une version antérieure, ou trafiquée à la main, ne
    // doit pas traverser l'application telle quelle.
    const prefs = creerPreferences(stockageMemoire({
      'hent.reglages': JSON.stringify({ distanceM: 900_000, eviterBitume: 12 })
    }));
    const lu = prefs.lire();
    expect(lu!.distanceM).toBeLessThanOrEqual(50_000);
    expect(lu!.eviterBitume).toBeLessThanOrEqual(1);
  });

  it('rend null quand le stockage refuse de lire', () => {
    // Navigation privée, stockage bloqué, quota : l'application doit démarrer
    // sur ses valeurs par défaut, pas planter.
    expect(creerPreferences(stockageQuiLeve()).lire()).toBeNull();
  });

  it('avale une écriture refusée sans lever', () => {
    const prefs = creerPreferences(stockageQuiLeve());
    expect(() => prefs.ecrire(REGLAGES_PAR_DEFAUT)).not.toThrow();
  });

  it("rend null quand il n'y a aucun stockage du tout", () => {
    // Le rendu côté serveur n'a pas de localStorage : `lire` doit y répondre
    // comme à un stockage vide.
    expect(creerPreferences(null).lire()).toBeNull();
  });
});

describe('thème', () => {
  it('relit le thème écrit', () => {
    const prefs = creerPreferences(stockageMemoire());
    prefs.ecrireTheme('clair');
    expect(prefs.lireTheme()).toBe('clair');
  });

  it('rend null sans thème, sur une valeur inconnue et sur un stockage qui lève', () => {
    expect(creerPreferences(stockageMemoire()).lireTheme()).toBeNull();
    expect(creerPreferences(stockageMemoire({ 'hent.theme': 'nuit' })).lireTheme()).toBeNull();
    expect(creerPreferences(stockageQuiLeve()).lireTheme()).toBeNull();
  });

  it('range le thème à part des réglages', () => {
    const stockage = stockageMemoire();
    const prefs = creerPreferences(stockage);
    prefs.ecrireTheme('sombre');
    prefs.ecrire(REGLAGES_PAR_DEFAUT);
    expect(prefs.lireTheme()).toBe('sombre');
    expect(prefs.lire()).toEqual(REGLAGES_PAR_DEFAUT);
  });

  it('n’échoue pas quand l’écriture est refusée', () => {
    expect(() => creerPreferences(stockageQuiLeve()).ecrireTheme('clair')).not.toThrow();
  });
});
