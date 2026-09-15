import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/svelte';
import type { MoteurDeBoucles } from '$lib/app/ports';
import { ErreurAPI } from '$lib/infra/hent-api';

const faux = vi.hoisted(() => ({ generer: vi.fn() }));

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

vi.mock('$lib/assemblage.svelte', async () => {
  const { creerResultats } = await import('$lib/app/generation/resultats.svelte');
  const moteur: MoteurDeBoucles = {
    generer: faux.generer,
    ouvrir: async () => {
      throw new Error('inattendu');
    },
    urlGPX: (id) => `/v1/loops/${id}.gpx`
  };
  return {
    appEtat: {
      moteur,
      resultats: creerResultats(moteur),
      reglages: { distanceM: 12_000, eviterBitume: 0.7 },
      poserDepart: vi.fn(),
      regler: vi.fn()
    }
  };
});

const { appEtat } = await import('$lib/assemblage.svelte');
const Reglage = (await import('./+page.svelte')).default;

beforeEach(() => {
  faux.generer.mockReset();
  appEtat.resultats.reinitialiser();
});

describe('écran de réglage', () => {
  it('redevient utilisable quand on corrige un départ hors zone', async () => {
    // Le 400 « hors zone » est le chemin nominal pour un départ hors Bretagne.
    // L'état des résultats étant partagé, il restait en erreur indéfiniment et
    // seul un rechargement complet ramenait le bouton.
    faux.generer.mockRejectedValue(new ErreurAPI('HorsZone', 'hors du graphe'));

    render(Reglage);
    screen.getByRole('button', { name: 'Tracer ma boucle' }).click();
    await screen.findByText('Ce point est en dehors de la Bretagne.');

    await fireEvent.input(screen.getByLabelText('Latitude'), { target: { value: '48.2' } });

    expect(await screen.findByRole('button', { name: 'Tracer ma boucle' })).toBeDefined();
  });

  it('n’offre pas de lien vers l’écran qui l’affiche', async () => {
    faux.generer.mockRejectedValue(new ErreurAPI('HorsZone', 'hors du graphe'));

    render(Reglage);
    screen.getByRole('button', { name: 'Tracer ma boucle' }).click();
    await screen.findByText('Ce point est en dehors de la Bretagne.');

    expect(screen.queryByRole('link', { name: 'Choisir un autre départ' })).toBeNull();
  });
});
