import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { goto } from '$app/navigation';
import type { MoteurDeBoucles } from '$lib/app/ports';
import { ErreurAPI } from '$lib/infra/hent-api';

const faux = vi.hoisted(() => ({
  generer: vi.fn(),
  changerTheme: vi.fn(),
  depart: { coord: { lat: 48.117, lon: -1.677 }, libelle: '2 Rue Lesage 35000 Rennes' } as { coord: { lat: number; lon: number }; libelle: string } | null
}));

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

vi.mock('$lib/assemblage.svelte', async () => {
  const { creerResultats } = await import('$lib/app/generation/resultats.svelte');
  const moteur: MoteurDeBoucles = {
    generer: faux.generer,
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
      reglages: { distanceM: 12_000, eviterBitume: 0.7 },
      poserDepart: vi.fn(),
      regler: vi.fn(),
      get depart() {
        return faux.depart;
      },
      theme: 'auto',
      changerTheme: faux.changerTheme
    }
  };
});

const { appEtat } = await import('$lib/assemblage.svelte');
const Reglage = (await import('./+page.svelte')).default;

beforeEach(() => {
  faux.generer.mockReset();
  faux.changerTheme.mockClear();
  faux.depart = { coord: { lat: 48.117, lon: -1.677 }, libelle: '2 Rue Lesage 35000 Rennes' };
  vi.mocked(goto).mockClear();
  appEtat.resultats.reinitialiser();
});

describe('écran de réglage', () => {
  it('mène aux boucles quand la recherche aboutit', async () => {
    faux.generer.mockResolvedValue([]);

    render(Reglage);
    screen.getByRole('button', { name: 'Tracer ma boucle' }).click();

    await vi.waitFor(() => expect(goto).toHaveBeenCalledWith('/boucles'));
  });

  it('reprend le focus quand l’état remplace le bouton activé', async () => {
    faux.generer.mockRejectedValue(new ErreurAPI('Serveur', 'boum'));

    render(Reglage);
    screen.getByRole('button', { name: 'Tracer ma boucle' }).click();
    await screen.findByText('Le service a rencontré un problème.');

    // Comparer le seul textContent laisserait passer `document.body`, qui
    // contient le texte de toute la page.
    expect(document.activeElement).not.toBe(document.body);
    expect(document.activeElement?.getAttribute('role')).toBe('alert');
    expect(document.activeElement?.textContent).toContain('Le service a rencontré un problème.');
  });

  it('offre un autre départ quand le point est hors zone, et rien d’autre', async () => {
    faux.generer.mockRejectedValue(new ErreurAPI('HorsZone', 'hors zone'));
    render(Reglage);

    await fireEvent.click(screen.getByRole('button', { name: 'Tracer ma boucle' }));
    const panneau = await screen.findByRole('alert');

    const sorties = panneau.querySelectorAll('button, a');
    expect(sorties).toHaveLength(1);
    expect(sorties[0]!.getAttribute('href')).toBe('/depart');
    expect(sorties[0]!.textContent).toContain('Choisir un autre départ');
    expect(screen.queryByRole('button', { name: 'Tracer ma boucle' })).toBeNull();
    expect(goto).not.toHaveBeenCalled();
  });

  it('affiche le départ posé et le lien pour le changer', () => {
    render(Reglage);
    expect(screen.getByText('2 Rue Lesage 35000 Rennes')).toBeDefined();
    expect(screen.getByRole('link', { name: 'Changer' }).getAttribute('href')).toBe('/depart');
  });

  it('renvoie vers le départ quand aucun n’est posé', async () => {
    faux.depart = null;
    render(Reglage);
    await tick();
    expect(goto).toHaveBeenCalledWith('/depart');
  });

  it('lance la recherche depuis le départ posé', async () => {
    faux.generer.mockResolvedValue([]);
    render(Reglage);
    await fireEvent.click(screen.getByRole('button', { name: 'Tracer ma boucle' }));
    expect(faux.generer).toHaveBeenCalledWith(
      expect.objectContaining({ depart: { lat: 48.117, lon: -1.677 } }),
      expect.anything()
    );
  });

  it('change le thème depuis l’écran', async () => {
    render(Reglage);
    await fireEvent.click(screen.getByRole('radio', { name: 'Clair' }));
    expect(faux.changerTheme).toHaveBeenCalledWith('clair');
  });
});
