import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
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

const faux = vi.hoisted(() => ({ ouvrir: vi.fn() }));

let idAffiche = $state('abc');

vi.mock('$app/state', () => ({ page: { get params() { return { id: idAffiche }; } } }));

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

/**
 * Rend la main jusqu'au prochain tour de la boucle d'événements, ce qui draine
 * les microtâches en attente. `tick()` n'en dépile qu'un cran : trop peu pour
 * voir le bout d'une chaîne `.then` → `.catch` → `.finally`.
 */
const prochainTour = () => new Promise((res) => setTimeout(res, 0));

beforeEach(() => {
  vi.restoreAllMocks();
  idAffiche = 'abc';
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

    // La boucle obtenue rejoint l'état partagé : c'est ce que la liste montrera
    // au retour, sans relancer de recherche.
    const etat = appEtat.resultats.etat();
    if (etat.statut !== 'ok') throw new Error('état inattendu');
    expect(etat.boucles.map((b) => b.id)).toEqual(['abc']);
    expect(etat.demande.distanceM).toBe(4000);
  });

  it('ne relit pas l’état des résultats qu’il vient d’y poser', async () => {
    // L'effet écrit dans l'état partagé. S'il le lit en dépendance, il
    // s'invalide lui-même dès que la réponse arrive et repart pour un tour.
    // Seul compte le delta autour de la pose : ce que lit le rendu initial le
    // regarde.
    let repondre!: (r: { boucle: Boucle; demande: Demande }) => void;
    faux.ouvrir.mockImplementationOnce(() => new Promise((res) => (repondre = res)));
    const lectures = vi.spyOn(appEtat.resultats, 'etat');

    render(Detail);
    await tick();
    const avantLaPose = lectures.mock.calls.length;

    repondre({ boucle: uneBoucle, demande });
    await screen.findByRole('heading', { level: 1, name: '3,9 km' });
    await tick();

    expect(lectures.mock.calls.length).toBe(avantLaPose);
  });

  it('abandonne l’appel du détail que l’on quitte', async () => {
    const signaux: (AbortSignal | undefined)[] = [];
    faux.ouvrir.mockImplementation((_id: string, signal?: AbortSignal) => {
      signaux.push(signal);
      return new Promise(() => {});
    });

    render(Detail);
    await tick();
    idAffiche = 'def';
    await tick();

    expect(signaux).toHaveLength(2);
    expect(signaux[0]?.aborted).toBe(true);
    expect(signaux[1]?.aborted).toBe(false);
  });

  it('n’écrase pas le détail suivant par la réponse du précédent', async () => {
    const autre: Boucle = { ...uneBoucle, id: 'def', score: { ...uneBoucle.score, distanceM: 8420 } };
    let repondrePremier!: (r: { boucle: Boucle; demande: Demande }) => void;
    faux.ouvrir.mockImplementationOnce(() => new Promise((res) => (repondrePremier = res)));
    faux.ouvrir.mockResolvedValue({ boucle: autre, demande });

    render(Detail);
    await tick();
    idAffiche = 'def';
    expect(await screen.findByRole('heading', { level: 1, name: '8,4 km' })).toBeDefined();

    repondrePremier({ boucle: uneBoucle, demande });
    await tick();

    expect(screen.getByRole('heading', { level: 1 }).textContent).toBe('8,4 km');
  });

  it('n’affiche pas sur le détail suivant l’échec du précédent', async () => {
    const autre: Boucle = { ...uneBoucle, id: 'def', score: { ...uneBoucle.score, distanceM: 8420 } };
    let echouerPremier!: (e: unknown) => void;
    faux.ouvrir.mockImplementationOnce(() => new Promise((_res, rej) => (echouerPremier = rej)));
    faux.ouvrir.mockResolvedValue({ boucle: autre, demande });

    render(Detail);
    await tick();
    idAffiche = 'def';
    expect(await screen.findByRole('heading', { level: 1, name: '8,4 km' })).toBeDefined();

    echouerPremier(new ErreurAPI('Serveur', 'boum'));
    await tick();

    expect(screen.queryByText('Le service a rencontré un problème.')).toBeNull();
    expect(screen.getByRole('heading', { level: 1 }).textContent).toBe('8,4 km');
  });

  it('ne lève pas l’attente du détail suivant quand le précédent retombe', async () => {
    let echouerPremier!: (e: unknown) => void;
    faux.ouvrir.mockImplementationOnce(() => new Promise((_res, rej) => (echouerPremier = rej)));
    faux.ouvrir.mockImplementationOnce(() => new Promise(() => {}));

    render(Detail);
    await tick();
    idAffiche = 'def';
    await tick();

    echouerPremier(new ErreurAPI('Serveur', 'boum'));
    await prochainTour();
    await tick();

    expect(screen.getByText('Je parcours les chemins.')).toBeDefined();
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

  it('offre l’écran de départ quand la boucle est hors zone', async () => {
    // Ici, contrairement à l'écran de réglage, il y a bien où aller : sans
    // cette destination le panneau n'offrirait plus rien du tout.
    faux.ouvrir.mockRejectedValue(new ErreurAPI('HorsZone', 'hors du graphe'));

    render(Detail);

    const lien = await screen.findByRole('link', { name: 'Choisir un autre départ' });
    expect(lien.getAttribute('href')).toBe('/reglage');
  });

  it('reprend le focus quand le réessai détruit le bouton activé', async () => {
    faux.ouvrir.mockRejectedValueOnce(new ErreurAPI('Serveur', 'boum'));
    faux.ouvrir.mockReturnValue(new Promise(() => {}));

    render(Detail);
    (await screen.findByRole('button', { name: 'Réessayer' })).click();

    expect(await screen.findByText('Je parcours les chemins.')).toBeDefined();
    // Comparer le seul textContent laisserait passer `document.body`, qui
    // contient le texte de toute la page.
    expect(document.activeElement).not.toBe(document.body);
    expect(document.activeElement?.getAttribute('role')).toBe('status');
    expect(document.activeElement?.textContent).toContain('Je parcours les chemins.');
  });

  it('transporte le délai de réessai annoncé par le serveur', async () => {
    // Le 429 porte un Retry-After ; l'écran de détail convertissait l'erreur à
    // la main et perdait ce délai en route.
    faux.ouvrir.mockRejectedValue(new ErreurAPI('TropDeDemandes', 'trop vite', 3));

    render(Detail);

    expect(await screen.findByText('Réessayer dans 3 s')).toBeDefined();
  });
});
