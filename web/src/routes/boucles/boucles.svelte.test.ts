import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import type { Boucle, Demande } from '$lib/domaine/boucle';
import type { MoteurDeBoucles } from '$lib/app/ports';

const score = { distanceM: 17_400, partNonBitume: 0.55, partTrafic: 0.02, partRetracee: 0.004, ecartCible: -0.03 };
const verte: Boucle = { id: 'verte', score, geometrie: [[-1.7, 48.1], [-1.6, 48.2]] };
const grise: Boucle = { id: 'grise', score: { ...score, distanceM: 18_100, partNonBitume: 0.51 }, geometrie: [[-1.75, 48.12]] };
const demande: Demande = { depart: { lat: 48.0246, lon: -1.7461 }, distanceM: 18_000, eviterBitume: 0.7, maxResultats: 5, variante: 0 };

const faux = vi.hoisted(() => ({
  carte: { afficherBoucles: vi.fn(), marquerDepart: vi.fn(), montrerZone: vi.fn() } as Record<string, ReturnType<typeof vi.fn>> | null,
  depart: { coord: { lat: 48.0246, lon: -1.7461 }, libelle: 'Bruz' } as { coord: { lat: number; lon: number }; libelle: string } | null
}));

vi.mock('$lib/assemblage.svelte', async () => {
  const { creerResultats } = await import('$lib/app/generation/resultats.svelte');
  const moteur: MoteurDeBoucles = {
    generer: async () => [],
    ouvrir: async () => {
      throw new Error('inattendu');
    },
    urlGPX: (id) => `/v1/loops/${id}.gpx`,
    zone: async () => ({ minLat: 47.2, minLon: -5.2, maxLat: 48.95, maxLon: -0.95 })
  };
  return {
    appEtat: {
      moteur,
      resultats: creerResultats(moteur),
      get carte() {
        return faux.carte;
      },
      get depart() {
        return faux.depart;
      }
    }
  };
});

const { appEtat } = await import('$lib/assemblage.svelte');
const Boucles = (await import('./+page.svelte')).default;

beforeEach(() => {
  faux.carte = { afficherBoucles: vi.fn(), marquerDepart: vi.fn(), montrerZone: vi.fn() };
  faux.depart = { coord: { lat: 48.0246, lon: -1.7461 }, libelle: 'Bruz' };
  appEtat.resultats.reinitialiser();
});

describe('écran des boucles', () => {
  it('titre par la demande et le départ, puis liste la plus verte d’abord', () => {
    appEtat.resultats.poser([grise, verte], demande);
    render(Boucles);

    expect(screen.getByRole('heading', { level: 1 }).textContent).toBe('18 km au départ de Bruz');
    const liens = screen.getAllByRole('link').filter((l) => l.getAttribute('href')?.startsWith('/b/'));
    expect(liens.map((l) => l.getAttribute('href'))).toEqual(['/b/verte', '/b/grise']);
  });

  it('se passe du départ quand il n’est pas connu', () => {
    faux.depart = null;
    appEtat.resultats.poser([verte], demande);
    render(Boucles);
    expect(screen.getByRole('heading', { level: 1 }).textContent).toBe('18 km');
  });

  it('montre les boucles sur la carte, la première sélectionnée, et marque le départ', async () => {
    appEtat.resultats.poser([grise, verte], demande);
    render(Boucles);
    await tick();

    expect(faux.carte!.afficherBoucles).toHaveBeenLastCalledWith([verte, grise], 'verte');
    expect(faux.carte!.marquerDepart).toHaveBeenLastCalledWith({ lat: 48.0246, lon: -1.7461 });
    expect(faux.carte!.montrerZone).toHaveBeenCalledWith(null);
  });

  it('la sélection suit le survol et le focus', async () => {
    appEtat.resultats.poser([grise, verte], demande);
    render(Boucles);
    await tick();
    const lienGrise = screen.getByRole('link', { name: /18,1 km/ });

    await fireEvent.mouseEnter(lienGrise);
    await tick();
    expect(faux.carte!.afficherBoucles).toHaveBeenLastCalledWith([verte, grise], 'grise');

    await fireEvent.focus(screen.getByRole('link', { name: /17,4 km/ }));
    await tick();
    expect(faux.carte!.afficherBoucles).toHaveBeenLastCalledWith([verte, grise], 'verte');
  });

  it('vide la carte quand il n’y a pas de résultats', async () => {
    render(Boucles);
    await tick();
    expect(faux.carte!.afficherBoucles).toHaveBeenLastCalledWith([], null);
    expect(screen.getByRole('link', { name: 'Régler une boucle' })).toBeDefined();
  });

  it('reste lisible sans carte', () => {
    faux.carte = null;
    appEtat.resultats.poser([verte], demande);
    render(Boucles);
    expect(screen.getByRole('link', { name: /17,4 km/ })).toBeDefined();
  });
});
