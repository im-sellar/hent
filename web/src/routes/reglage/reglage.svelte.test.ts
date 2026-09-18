import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/svelte';
import { goto } from '$app/navigation';
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
  vi.mocked(goto).mockClear();
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

    // Le bouton ne doit pas seulement revenir : il doit repartir, avec le
    // départ corrigé.
    (await screen.findByRole('button', { name: 'Tracer ma boucle' })).click();

    expect(faux.generer).toHaveBeenLastCalledWith(
      expect.objectContaining({ depart: { lat: 48.2, lon: -1.677 } }),
      expect.anything()
    );
  });

  it('n’offre aucune action quand le départ est hors zone', async () => {
    faux.generer.mockRejectedValue(new ErreurAPI('HorsZone', 'hors du graphe'));

    render(Reglage);
    screen.getByRole('button', { name: 'Tracer ma boucle' }).click();
    const panneau = (await screen.findByText('Ce point est en dehors de la Bretagne.')).closest(
      '[role="alert"]'
    );

    expect(panneau?.querySelectorAll('button, a')).toHaveLength(0);
    // Hors du panneau non plus : « Tracer ma boucle » renverrait les mêmes
    // coordonnées au même refus.
    expect(screen.queryByRole('button', { name: 'Tracer ma boucle' })).toBeNull();
    expect(goto).not.toHaveBeenCalled();
  });

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

  it('refuse de partir sur un départ invalide', async () => {
    render(Reglage);
    await fireEvent.input(screen.getByLabelText('Latitude'), { target: { value: '500' } });

    screen.getByRole('button', { name: 'Tracer ma boucle' }).click();

    expect(faux.generer).not.toHaveBeenCalled();
    expect(screen.getByRole('button', { name: 'Tracer ma boucle' })).toBeDefined();
  });

  it('désigne le champ fautif et le décrit', async () => {
    render(Reglage);
    const champLat = screen.getByLabelText('Latitude');
    const champLon = screen.getByLabelText('Longitude');

    await fireEvent.input(champLat, { target: { value: '500' } });

    expect(champLat.getAttribute('aria-invalid')).toBe('true');
    expect(champLon.getAttribute('aria-invalid')).toBe('false');
    const decrivant = champLat.getAttribute('aria-describedby');
    expect(decrivant).toBeTruthy();
    expect(document.getElementById(decrivant!)?.textContent).toContain('ne sont pas valides');
  });

  it('donne le focus au champ fautif quand il refuse', async () => {
    render(Reglage);
    const champLon = screen.getByLabelText('Longitude');
    await fireEvent.input(champLon, { target: { value: '500' } });

    screen.getByRole('button', { name: 'Tracer ma boucle' }).click();

    expect(document.activeElement).toBe(champLon);
  });

  it('tient la région du message de départ prête avant l’erreur', () => {
    // Une région live créée en même temps que son texte n'est pas annoncée :
    // elle doit préexister à la mutation.
    render(Reglage);

    const champLat = screen.getByLabelText('Latitude');
    const region = document.getElementById(champLat.getAttribute('aria-describedby')!);
    expect(region?.getAttribute('role')).toBe('alert');
    expect(region?.textContent).toBe('');
  });

  it('n’offre pas de lien vers l’écran qui l’affiche', async () => {
    faux.generer.mockRejectedValue(new ErreurAPI('HorsZone', 'hors du graphe'));

    render(Reglage);
    screen.getByRole('button', { name: 'Tracer ma boucle' }).click();
    await screen.findByText('Ce point est en dehors de la Bretagne.');

    expect(screen.queryByRole('link', { name: 'Choisir un autre départ' })).toBeNull();
  });
});
