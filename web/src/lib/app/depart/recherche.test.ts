import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { DELAI_MS, creerRecherche } from './recherche.svelte';
import type { Geocodeur } from '$lib/app/ports';
import type { Lieu } from '$lib/domaine/depart';

const bruz: Lieu = { libelle: 'Bruz', complement: 'Commune, Ille-et-Vilaine', coord: { lat: 48.02, lon: -1.75 } };
const rennes: Lieu = { libelle: 'Rennes', complement: 'Commune, Ille-et-Vilaine', coord: { lat: 48.11, lon: -1.68 } };

type Appel = { texte: string; signal?: AbortSignal; repondre: (l: Lieu[]) => void; echouer: (e: unknown) => void };

/** Un géocodeur qui note chaque appel et laisse le test décider quand et comment répondre. */
function geocodeurPilote() {
  const appels: Appel[] = [];
  const geocodeur: Geocodeur = {
    chercher: (texte, _autour, signal) =>
      new Promise<Lieu[]>((repondre, echouer) => {
        appels.push({ texte, signal, repondre, echouer });
      }),
    nommer: async () => null
  };
  return { geocodeur, appels };
}

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

describe('creerRecherche', () => {
  it('reste vide sous trois caractères et n’appelle pas le géocodeur', () => {
    const { geocodeur, appels } = geocodeurPilote();
    const r = creerRecherche(geocodeur);

    r.saisir('br');
    vi.advanceTimersByTime(DELAI_MS);

    expect(r.etat()).toEqual({ statut: 'vide' });
    expect(appels).toHaveLength(0);
  });

  it('attend le délai avant d’appeler, puis rend les lieux', async () => {
    const { geocodeur, appels } = geocodeurPilote();
    const r = creerRecherche(geocodeur);

    r.saisir('bruz');
    expect(r.etat()).toEqual({ statut: 'attente' });
    vi.advanceTimersByTime(DELAI_MS - 1);
    expect(appels).toHaveLength(0);
    vi.advanceTimersByTime(1);
    expect(appels).toHaveLength(1);
    expect(appels[0]!.texte).toBe('bruz');

    appels[0]!.repondre([bruz]);
    await vi.advanceTimersByTimeAsync(0);

    expect(r.etat()).toEqual({ statut: 'ok', lieux: [bruz] });
  });

  it('ne fait qu’un appel pour plusieurs frappes rapprochées', () => {
    const { geocodeur, appels } = geocodeurPilote();
    const r = creerRecherche(geocodeur);

    r.saisir('bru');
    vi.advanceTimersByTime(100);
    r.saisir('bruz');
    vi.advanceTimersByTime(100);
    r.saisir('bruz v');
    vi.advanceTimersByTime(DELAI_MS);

    expect(appels.map((a) => a.texte)).toEqual(['bruz v']);
  });

  it('annule l’appel précédent quand on continue de taper', () => {
    const { geocodeur, appels } = geocodeurPilote();
    const r = creerRecherche(geocodeur);

    r.saisir('bruz');
    vi.advanceTimersByTime(DELAI_MS);
    r.saisir('bruz vert');
    vi.advanceTimersByTime(DELAI_MS);

    expect(appels).toHaveLength(2);
    expect(appels[0]!.signal?.aborted).toBe(true);
    expect(appels[1]!.signal?.aborted).toBe(false);
  });

  it('ignore une réponse lente arrivée après une réponse rapide, même sans annulation', async () => {
    // Le géocodeur pilote ne regarde pas le signal : seule la génération protège ici.
    const { geocodeur, appels } = geocodeurPilote();
    const r = creerRecherche(geocodeur);

    r.saisir('bruz');
    vi.advanceTimersByTime(DELAI_MS);
    r.saisir('rennes');
    vi.advanceTimersByTime(DELAI_MS);

    appels[1]!.repondre([rennes]);
    await vi.advanceTimersByTimeAsync(0);
    appels[0]!.repondre([bruz]);
    await vi.advanceTimersByTimeAsync(0);

    expect(r.etat()).toEqual({ statut: 'ok', lieux: [rennes] });
  });

  it('garde les lieux affichés pendant qu’on affine, plutôt que de clignoter', () => {
    const { geocodeur, appels } = geocodeurPilote();
    const r = creerRecherche(geocodeur);

    r.saisir('bruz');
    vi.advanceTimersByTime(DELAI_MS);
    appels[0]!.repondre([bruz]);

    return vi.advanceTimersByTimeAsync(0).then(() => {
      r.saisir('bruz v');
      expect(r.etat()).toEqual({ statut: 'ok', lieux: [bruz] });
    });
  });

  it('passe en erreur quand le géocodeur échoue, pas quand il est annulé', async () => {
    const { geocodeur, appels } = geocodeurPilote();
    const r = creerRecherche(geocodeur);

    r.saisir('bruz');
    vi.advanceTimersByTime(DELAI_MS);
    appels[0]!.echouer(new Error('BAN : statut 503'));
    await vi.advanceTimersByTimeAsync(0);
    expect(r.etat()).toEqual({ statut: 'erreur' });

    r.saisir('rennes');
    vi.advanceTimersByTime(DELAI_MS);
    appels[1]!.echouer(new DOMException('annulé', 'AbortError'));
    await vi.advanceTimersByTimeAsync(0);
    expect(r.etat()).toEqual({ statut: 'attente' });
  });

  it('efface : revient à vide, annule l’appel en vol et n’en lance plus', async () => {
    const { geocodeur, appels } = geocodeurPilote();
    const r = creerRecherche(geocodeur);

    r.saisir('bruz');
    vi.advanceTimersByTime(DELAI_MS);
    r.effacer();

    expect(appels[0]!.signal?.aborted).toBe(true);
    expect(r.etat()).toEqual({ statut: 'vide' });

    r.saisir('rennes');
    r.effacer();
    vi.advanceTimersByTime(DELAI_MS);
    expect(appels).toHaveLength(1);

    appels[0]!.repondre([bruz]);
    await vi.advanceTimersByTimeAsync(0);
    expect(r.etat()).toEqual({ statut: 'vide' });
  });

  it('transmet la position pour pondérer', () => {
    let recu: unknown;
    const geocodeur: Geocodeur = {
      chercher: async (_t, autour) => {
        recu = autour;
        return [];
      },
      nommer: async () => null
    };
    creerRecherche(geocodeur).saisir('bruz', { lat: 48.02, lon: -1.75 });
    vi.advanceTimersByTime(DELAI_MS);

    expect(recu).toEqual({ lat: 48.02, lon: -1.75 });
  });
});
