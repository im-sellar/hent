import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { goto } from '$app/navigation';
import type { Carte, Geocodeur, MoteurDeBoucles, Position } from '$lib/app/ports';
import type { Coord, Lieu } from '$lib/domaine/depart';
import { DELAI_MS } from '$lib/app/depart/recherche.svelte';

const bruz: Lieu = { libelle: 'Bruz', complement: 'Commune, Ille-et-Vilaine', coord: { lat: 48.0246, lon: -1.7461 }, distanceM: 1800 };
const bretagne = { minLat: 47.2, minLon: -5.2, maxLat: 48.95, maxLon: -0.95 };

const faux = vi.hoisted(() => ({
  chercher: vi.fn(),
  nommer: vi.fn(),
  obtenir: vi.fn(),
  poserDepart: vi.fn(),
  depart: null as { coord: Coord; libelle: string } | null,
  carte: null as null | {
    centrer: ReturnType<typeof vi.fn>;
    afficherBoucles: ReturnType<typeof vi.fn>;
    marquerDepart: ReturnType<typeof vi.fn>;
    montrerZone: ReturnType<typeof vi.fn>;
    surDeplacement: ReturnType<typeof vi.fn>;
    changerStyle: ReturnType<typeof vi.fn>;
    redimensionner: ReturnType<typeof vi.fn>;
    detruire: ReturnType<typeof vi.fn>;
    deplacer: (c: Coord) => void;
  }
}));

/** Une carte doublée dont le test déclenche lui-même les fins de déplacement. */
function fausseCarte() {
  let rappel: ((c: Coord) => void) | null = null;
  return {
    centrer: vi.fn(),
    afficherBoucles: vi.fn(),
    marquerDepart: vi.fn(),
    montrerZone: vi.fn(),
    surDeplacement: vi.fn((r: (c: Coord) => void) => {
      rappel = r;
      return () => {
        rappel = null;
      };
    }),
    changerStyle: vi.fn(),
    redimensionner: vi.fn(),
    detruire: vi.fn(),
    deplacer: (c: Coord) => rappel?.(c)
  };
}

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

vi.mock('$lib/assemblage.svelte', async () => {
  const { creerResultats } = await import('$lib/app/generation/resultats.svelte');
  const { creerRecherche } = await import('$lib/app/depart/recherche.svelte');
  const geocodeur: Geocodeur = { chercher: faux.chercher, nommer: faux.nommer };
  const position: Position = { obtenir: faux.obtenir };
  const moteur: MoteurDeBoucles = {
    generer: async () => [],
    ouvrir: async () => {
      throw new Error('inattendu');
    },
    urlGPX: (id) => `/v1/loops/${id}.gpx`,
    zone: async () => bretagne
  };
  return {
    appEtat: {
      moteur,
      geocodeur,
      position,
      recherche: creerRecherche(geocodeur),
      resultats: creerResultats(moteur),
      zone: () => moteur.zone(),
      get carte() {
        return faux.carte as unknown as Carte | null;
      },
      get depart() {
        return faux.depart;
      },
      poserDepart: faux.poserDepart
    }
  };
});

const { appEtat } = await import('$lib/assemblage.svelte');
const Depart = (await import('./+page.svelte')).default;

/** Laisse retomber les promesses en vol (zone, nommage) et les effets Svelte. */
async function retomber() {
  await vi.advanceTimersByTimeAsync(0);
  await tick();
}

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] });
  faux.chercher.mockReset().mockResolvedValue([bruz]);
  faux.nommer.mockReset().mockResolvedValue('2 Rue Lesage 35000 Rennes');
  faux.obtenir.mockReset();
  faux.poserDepart.mockReset();
  faux.depart = null;
  faux.carte = fausseCarte();
  vi.mocked(goto).mockClear();
  appEtat.recherche.effacer();
  appEtat.resultats.reinitialiser();
});

afterEach(() => vi.useRealTimers());

describe('écran de départ', () => {
  it('centre la carte sur la Bretagne sans départ connu, sur le départ sinon', async () => {
    render(Depart);
    await retomber();
    expect(faux.carte!.centrer).toHaveBeenCalledWith({ lat: 48.2, lon: -2.9 }, 7);

    faux.carte = fausseCarte();
    faux.depart = { coord: { lat: 48.117, lon: -1.677 }, libelle: 'Rennes' };
    render(Depart);
    await retomber();
    expect(faux.carte!.centrer).toHaveBeenCalledWith({ lat: 48.117, lon: -1.677 }, 14);
    expect(screen.getByText('Rennes')).toBeDefined();
  });

  it('nomme le centre à la fin d’un déplacement, jamais pendant la frappe', async () => {
    render(Depart);
    await retomber();
    faux.nommer.mockClear();

    faux.carte!.deplacer({ lat: 48.117, lon: -1.677 });
    await retomber();

    expect(faux.nommer).toHaveBeenCalledWith({ lat: 48.117, lon: -1.677 }, expect.anything());
    expect(screen.getByText('2 Rue Lesage 35000 Rennes')).toBeDefined();
  });

  it('ne garde que le dernier nommage quand on enchaîne deux déplacements', async () => {
    let repondreLent!: (n: string) => void;
    faux.nommer
      .mockImplementationOnce(() => new Promise<string>((res) => (repondreLent = res)))
      .mockResolvedValueOnce('Place du Vert Buisson 35170 Bruz');
    render(Depart);
    await retomber();

    faux.carte!.deplacer({ lat: 48.1, lon: -1.7 });
    faux.carte!.deplacer({ lat: 48.03, lon: -1.75 });
    await retomber();
    repondreLent('Ailleurs');
    await retomber();

    expect(screen.queryByText('Ailleurs')).toBeNull();
    expect(screen.getByText('Place du Vert Buisson 35170 Bruz')).toBeDefined();
  });

  it('propose des lieux après le délai, avec leur complément et leur distance', async () => {
    render(Depart);
    await retomber();

    await fireEvent.input(screen.getByRole('searchbox'), { target: { value: 'bruz' } });
    expect(faux.chercher).not.toHaveBeenCalled();
    vi.advanceTimersByTime(DELAI_MS);
    await retomber();

    expect(faux.chercher).toHaveBeenCalledWith('bruz', { lat: 48.2, lon: -2.9 }, expect.anything());
    const choix = screen.getByRole('button', { name: /Bruz/ });
    expect(choix.textContent).toContain('Commune, Ille-et-Vilaine');
    expect(choix.textContent).toContain('1,8 km');
    expect(screen.getByText(/Base Adresse Nationale/)).toBeDefined();
  });

  it('choisir un lieu centre la carte dessus, le nomme, et ne le renomme pas au déplacement qui suit', async () => {
    render(Depart);
    await retomber();
    faux.nommer.mockClear();

    await fireEvent.input(screen.getByRole('searchbox'), { target: { value: 'bruz' } });
    vi.advanceTimersByTime(DELAI_MS);
    await retomber();
    await fireEvent.click(screen.getByRole('button', { name: /Bruz/ }));
    await retomber();

    expect(faux.carte!.centrer).toHaveBeenLastCalledWith(bruz.coord, 14);
    expect(screen.getByRole('status').textContent).toContain('Bruz');
    expect(screen.queryByRole('button', { name: /Bruz/ })).toBeNull();

    faux.carte!.deplacer(bruz.coord);
    await retomber();
    expect(faux.nommer).not.toHaveBeenCalled();
  });

  it('ferme la liste à Échap', async () => {
    render(Depart);
    await retomber();
    const champ = screen.getByRole('searchbox');
    await fireEvent.input(champ, { target: { value: 'bruz' } });
    vi.advanceTimersByTime(DELAI_MS);
    await retomber();
    expect(screen.getByRole('button', { name: /Bruz/ })).toBeDefined();

    await fireEvent.keyDown(champ, { key: 'Escape' });
    await retomber();

    expect(screen.queryByRole('button', { name: /Bruz/ })).toBeNull();
    expect((champ as HTMLInputElement).value).toBe('');
  });

  it('« Autour de moi » centre sur la position accordée', async () => {
    faux.obtenir.mockResolvedValue({ statut: 'ok', coord: { lat: 48.117, lon: -1.677 } });
    render(Depart);
    await retomber();

    await fireEvent.click(screen.getByRole('button', { name: 'Autour de moi' }));
    await retomber();

    expect(faux.carte!.centrer).toHaveBeenLastCalledWith({ lat: 48.117, lon: -1.677 }, 14);
    expect(faux.nommer).toHaveBeenCalledWith({ lat: 48.117, lon: -1.677 }, expect.anything());
  });

  it('un refus de position ne bloque rien : un panneau prend le focus et renvoie vers la recherche', async () => {
    faux.obtenir.mockResolvedValue({ statut: 'refusee' });
    render(Depart);
    await retomber();

    await fireEvent.click(screen.getByRole('button', { name: 'Autour de moi' }));
    await retomber();

    const panneau = screen.getByRole('alert', { name: 'Je n’ai pas accès à ta position.' });
    expect(panneau.textContent).toContain('Je n’ai pas accès à ta position.');
    expect(document.activeElement).toBe(panneau);
    expect(document.activeElement).not.toBe(document.body);

    await fireEvent.click(screen.getByRole('button', { name: 'Chercher une adresse' }));
    expect(document.activeElement).toBe(screen.getByRole('searchbox'));
  });

  it('une position indisponible le dit autrement', async () => {
    faux.obtenir.mockResolvedValue({ statut: 'indisponible' });
    render(Depart);
    await retomber();

    await fireEvent.click(screen.getByRole('button', { name: 'Autour de moi' }));
    await retomber();

    expect(screen.getByRole('alert', { name: /Ta position n’est pas disponible/ }).textContent).toContain('Ta position n’est pas disponible');
  });

  it('« Partir d’ici » pose le départ nommé, oublie les anciens résultats et mène au réglage', async () => {
    render(Depart);
    await retomber();
    appEtat.resultats.poser([], { depart: { lat: 48, lon: -2 }, distanceM: 1, eviterBitume: 0, maxResultats: 1, variante: 0 });
    faux.carte!.deplacer({ lat: 48.117, lon: -1.677 });
    await retomber();

    await fireEvent.click(screen.getByRole('button', { name: 'Partir d’ici' }));

    expect(faux.poserDepart).toHaveBeenCalledWith({ coord: { lat: 48.117, lon: -1.677 }, libelle: '2 Rue Lesage 35000 Rennes' });
    expect(appEtat.resultats.etat().statut).toBe('vide');
    expect(goto).toHaveBeenCalledWith('/reglage');
  });

  it('hors de la zone couverte : la carte montre la zone, le bouton refuse et le dit, rien n’est posé', async () => {
    render(Depart);
    await retomber();

    faux.carte!.deplacer({ lat: 49.5, lon: 2.3 });
    await retomber();

    expect(faux.carte!.montrerZone).toHaveBeenLastCalledWith(bretagne);
    await fireEvent.click(screen.getByRole('button', { name: 'Partir d’ici' }));
    expect(faux.poserDepart).not.toHaveBeenCalled();
    expect(goto).not.toHaveBeenCalled();
    const avertissement = screen.getByText(/en dehors de la Bretagne/);
    expect(avertissement.getAttribute('role')).toBe('alert');
    expect(document.activeElement).toBe(avertissement);

    faux.carte!.deplacer({ lat: 48.117, lon: -1.677 });
    await retomber();
    expect(faux.carte!.montrerZone).toHaveBeenLastCalledWith(null);
  });

  it('nomme le centre par ses coordonnées quand le géocodage inverse ne rend rien', async () => {
    faux.nommer.mockResolvedValue(null);
    render(Depart);
    await retomber();

    faux.carte!.deplacer({ lat: 48.117, lon: -1.677 });
    await retomber();

    expect(screen.getByRole('status').textContent).toContain('48,1170, -1,6770');
  });

  it('sans carte, l’écran reste utilisable par la recherche et la position', async () => {
    faux.carte = null;
    faux.obtenir.mockResolvedValue({ statut: 'ok', coord: { lat: 48.117, lon: -1.677 } });
    render(Depart);
    await retomber();

    expect(screen.getByText(/carte n’est pas disponible/)).toBeDefined();
    await fireEvent.click(screen.getByRole('button', { name: 'Autour de moi' }));
    await retomber();
    await fireEvent.click(screen.getByRole('button', { name: 'Partir d’ici' }));

    expect(faux.poserDepart).toHaveBeenCalledWith({ coord: { lat: 48.117, lon: -1.677 }, libelle: '2 Rue Lesage 35000 Rennes' });
  });

  it('se désabonne de la carte et abandonne le nommage en vol au démontage', async () => {
    faux.nommer.mockImplementation((_c: Coord, signal?: AbortSignal) => new Promise<string>((_r, rej) => signal?.addEventListener('abort', () => rej(new DOMException('annulé', 'AbortError')))));
    const { unmount } = render(Depart);
    await retomber();
    faux.carte!.deplacer({ lat: 48.117, lon: -1.677 });
    const signal = faux.nommer.mock.calls[0]![1] as AbortSignal;

    unmount();

    expect(signal.aborted).toBe(true);
    faux.carte!.deplacer({ lat: 48.2, lon: -1.6 });
    expect(faux.nommer).toHaveBeenCalledTimes(1);
  });
});
