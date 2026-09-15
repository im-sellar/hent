import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import type { Boucle, Demande } from '$lib/domaine/boucle';
import type { MoteurDeBoucles } from '$lib/app/ports';
import { ErreurAPI } from '$lib/infra/hent-api';

const demande: Demande = {
  depart: { lat: 48.135, lon: -1.628 },
  distanceM: 4000,
  eviterBitume: 0.5,
  maxResultats: 3,
  variante: 0
};

const uneBoucle: Boucle = {
  id: 'abc',
  score: { distanceM: 3924, partNonBitume: 0.8, partTrafic: 0.02, partRetracee: 0.01, ecartCible: -0.02 },
  geometrie: [[-1.628, 48.135]]
};

const faux = vi.hoisted(() => ({
  id: 'abc',
  ouvrir: vi.fn()
}));

vi.mock('$app/state', () => ({ page: { get params() { return { id: faux.id }; } } }));

vi.mock('$lib/assemblage.svelte', async () => {
  const { creerResultats } = await import('$lib/app/generation/resultats.svelte');
  const moteur: MoteurDeBoucles = {
    generer: async () => [],
    ouvrir: faux.ouvrir,
    urlGPX: (id) => `/v1/loops/${id}.gpx`
  };
  return { appEtat: { moteur, resultats: creerResultats(moteur) } };
});

const { appEtat } = await import('$lib/assemblage.svelte');
const Detail = (await import('./+page.svelte')).default;

beforeEach(() => {
  faux.id = 'abc';
  faux.ouvrir.mockReset();
  appEtat.resultats.reinitialiser();
});

describe('écran de détail', () => {
  it('affiche la boucle obtenue par lien partagé, sans rester en attente', async () => {
    // Le cas du rechargement et du lien partagé : rien n'est en mémoire, la
    // boucle vient d'un appel. C'est le seul chemin où l'effet écrit dans
    // l'état qu'il lit, et où il s'invalidait donc lui-même.
    faux.ouvrir.mockResolvedValue({ boucle: uneBoucle, demande });

    render(Detail);

    expect(await screen.findByRole('heading', { level: 1, name: '3,9 km' })).toBeDefined();
    expect(screen.queryByText('Je parcours les chemins.')).toBeNull();
    expect(screen.getByRole('link', { name: 'Télécharger le GPX' })).toBeDefined();
  });

  it('affiche la boucle déjà connue sans rappeler le moteur', async () => {
    appEtat.resultats.poser([uneBoucle], demande);

    render(Detail);

    expect(await screen.findByRole('heading', { level: 1, name: '3,9 km' })).toBeDefined();
    expect(faux.ouvrir).not.toHaveBeenCalled();
  });

  it('sort de l’erreur au réessai quand l’appel aboutit', async () => {
    faux.ouvrir.mockRejectedValueOnce(new ErreurAPI('Serveur', 'boum'));
    faux.ouvrir.mockResolvedValue({ boucle: uneBoucle, demande });

    render(Detail);

    const reessayer = await screen.findByRole('button', { name: 'Réessayer' });
    reessayer.click();

    expect(await screen.findByRole('heading', { level: 1, name: '3,9 km' })).toBeDefined();
  });

  it('transporte le délai de réessai annoncé par le serveur', async () => {
    // Le 429 porte un Retry-After ; l'écran de détail convertissait l'erreur à
    // la main et perdait ce délai en route.
    faux.ouvrir.mockRejectedValue(new ErreurAPI('TropDeDemandes', 'trop vite', 3));

    render(Detail);

    expect(await screen.findByRole('button', { name: 'Réessayer dans 3 s' })).toBeDefined();
  });
});
