import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { Theme, ThemeEffectif } from '$lib/domaine/theme';
import type { Zone } from '$lib/domaine/zone';

const faux = vi.hoisted(() => ({
  webgl: true,
  themeStocke: null as string | null,
  creerCarte: vi.fn(),
  zone: vi.fn(),
  carte: {
    centrer: vi.fn(),
    afficherBoucles: vi.fn(),
    marquerDepart: vi.fn(),
    montrerZone: vi.fn(),
    surDeplacement: vi.fn(() => () => {}),
    changerStyle: vi.fn(),
    redimensionner: vi.fn(),
    detruire: vi.fn()
  }
}));

vi.mock('$lib/infra/maplibre', () => ({
  urlStyle: (t: ThemeEffectif) => `/carte/hent-${t}.json`,
  webglDisponible: () => faux.webgl,
  creerCarte: faux.creerCarte
}));

vi.mock('$lib/infra/stockage', () => ({
  creerPreferences: () => ({
    lire: () => null,
    ecrire: () => {},
    lireTheme: () => faux.themeStocke,
    ecrireTheme: (t: Theme) => void (faux.themeStocke = t)
  })
}));

vi.mock('$lib/infra/hent-api', () => ({
  creerMoteurHTTP: () => ({
    generer: vi.fn(),
    ouvrir: vi.fn(),
    urlGPX: (id: string) => `/v1/loops/${id}.gpx`,
    zone: faux.zone
  })
}));

/**
 * Rend un assemblage neuf. `appEtat` est un singleton créé à l'import du
 * module : sans réinitialisation, le deuxième test hériterait de la carte, du
 * thème et de la zone du premier.
 */
async function chargerAssemblage() {
  vi.resetModules();
  return (await import('$lib/assemblage.svelte')).appEtat;
}

const emprise: Zone = { minLat: 47.2, minLon: -5.2, maxLat: 48.95, maxLon: -0.95 };

beforeEach(() => {
  faux.webgl = true;
  faux.themeStocke = null;
  faux.creerCarte.mockReset();
  faux.creerCarte.mockReturnValue(faux.carte);
  faux.zone.mockReset();
  faux.carte.changerStyle.mockClear();
  faux.carte.detruire.mockClear();
  document.documentElement.removeAttribute('data-theme');
});

describe('assemblage', () => {
  it('ne crée qu’une carte, même monté deux fois', async () => {
    const appEtat = await chargerAssemblage();

    appEtat.monterCarte(document.createElement('div'));
    appEtat.monterCarte(document.createElement('div'));

    expect(faux.creerCarte).toHaveBeenCalledOnce();
    expect(appEtat.carte?.changerStyle).toBe(faux.carte.changerStyle);
  });

  it('se passe de carte quand WebGL manque', async () => {
    faux.webgl = false;
    const appEtat = await chargerAssemblage();

    appEtat.monterCarte(document.createElement('div'));

    expect(faux.creerCarte).not.toHaveBeenCalled();
    expect(appEtat.carte).toBeNull();
  });

  it('ne demande la zone qu’une fois, et la redemande après un échec', async () => {
    faux.zone.mockResolvedValue(emprise);
    const memorise = await chargerAssemblage();

    expect(await memorise.zone()).toEqual(emprise);
    expect(await memorise.zone()).toEqual(emprise);
    expect(faux.zone).toHaveBeenCalledOnce();

    faux.zone.mockReset();
    faux.zone.mockRejectedValueOnce(new Error('panne'));
    faux.zone.mockResolvedValue(emprise);
    const apresEchec = await chargerAssemblage();

    await expect(apresEchec.zone()).rejects.toThrow('panne');
    expect(await apresEchec.zone()).toEqual(emprise);
    expect(faux.zone).toHaveBeenCalledTimes(2);
  });

  it('ne recharge le style que lorsque le thème effectif change', async () => {
    const appEtat = await chargerAssemblage();
    appEtat.monterCarte(document.createElement('div'));

    appEtat.changerTheme('clair');
    appEtat.appliquerTheme();

    expect(faux.carte.changerStyle).toHaveBeenCalledOnce();
    expect(faux.carte.changerStyle).toHaveBeenCalledWith('/carte/hent-clair.json');

    appEtat.changerTheme('sombre');

    expect(faux.carte.changerStyle).toHaveBeenCalledTimes(2);
    expect(faux.carte.changerStyle).toHaveBeenLastCalledWith('/carte/hent-sombre.json');
  });
});
