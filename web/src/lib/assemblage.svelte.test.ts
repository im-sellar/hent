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

vi.mock('$lib/infra/styles-carte', () => ({
  urlStyle: (t: ThemeEffectif) => `/carte/hent-${t}.json`,
  webglDisponible: () => faux.webgl
}));

vi.mock('$lib/infra/maplibre', () => ({ creerCarte: faux.creerCarte }));

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

/**
 * Un conteneur attaché au document : `monterCarte` charge l'adaptateur à la
 * demande et abandonne si le conteneur a quitté la page entre-temps.
 */
function conteneur(): HTMLElement {
  return document.body.appendChild(document.createElement('div'));
}

beforeEach(() => {
  faux.webgl = true;
  faux.themeStocke = null;
  faux.creerCarte.mockReset();
  faux.creerCarte.mockReturnValue(faux.carte);
  faux.zone.mockReset();
  faux.carte.changerStyle.mockClear();
  faux.carte.detruire.mockClear();
  document.documentElement.removeAttribute('data-theme');
  document.body.replaceChildren();
});

describe('assemblage', () => {
  it('ne crée qu’une carte, même monté deux fois', async () => {
    const appEtat = await chargerAssemblage();

    await appEtat.monterCarte(conteneur());
    await appEtat.monterCarte(conteneur());

    expect(faux.creerCarte).toHaveBeenCalledOnce();
    expect(appEtat.carte?.changerStyle).toBe(faux.carte.changerStyle);
  });

  it('ne crée qu’une carte même si deux montages se croisent', async () => {
    const appEtat = await chargerAssemblage();

    await Promise.all([appEtat.monterCarte(conteneur()), appEtat.monterCarte(conteneur())]);

    expect(faux.creerCarte).toHaveBeenCalledOnce();
  });

  it('abandonne le montage si le conteneur a quitté la page', async () => {
    const appEtat = await chargerAssemblage();
    const cible = conteneur();
    const montage = appEtat.monterCarte(cible);
    cible.remove();

    await montage;

    expect(faux.creerCarte).not.toHaveBeenCalled();
    expect(appEtat.carte).toBeNull();
  });

  it('se passe de carte quand WebGL manque', async () => {
    faux.webgl = false;
    const appEtat = await chargerAssemblage();

    await appEtat.monterCarte(conteneur());

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
    await appEtat.monterCarte(conteneur());

    appEtat.changerTheme('clair');
    appEtat.appliquerTheme();

    expect(faux.carte.changerStyle).toHaveBeenCalledOnce();
    expect(faux.carte.changerStyle).toHaveBeenCalledWith('/carte/hent-clair.json');

    appEtat.changerTheme('sombre');

    expect(faux.carte.changerStyle).toHaveBeenCalledTimes(2);
    expect(faux.carte.changerStyle).toHaveBeenLastCalledWith('/carte/hent-sombre.json');
  });
});
