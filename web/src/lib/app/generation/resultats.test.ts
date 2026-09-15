import { describe, expect, it, vi } from 'vitest';
import { creerResultats } from './resultats.svelte';
import { ErreurAPI } from '$lib/infra/hent-api';
import type { Boucle, Demande } from '$lib/domaine/boucle';
import type { MoteurDeBoucles } from '$lib/app/ports';

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

function moteurQui(resultat: (demande: Demande, signal?: AbortSignal) => Promise<Boucle[]>): MoteurDeBoucles {
  return {
    generer: resultat,
    ouvrir: async () => ({ boucle: uneBoucle, demande }),
    urlGPX: (id) => `/v1/loops/${id}.gpx`
  };
}

describe('creerResultats', () => {
  it('part de l’état vide', () => {
    expect(creerResultats(moteurQui(async () => [])).etat().statut).toBe('vide');
  });

  it('passe par calcul avant de rendre ok', async () => {
    let debloquer!: (b: Boucle[]) => void;
    const attente = new Promise<Boucle[]>((r) => (debloquer = r));
    const r = creerResultats(moteurQui(() => attente));

    const fini = r.lancer(demande);
    expect(r.etat().statut).toBe('calcul');

    debloquer([uneBoucle]);
    await fini;

    const etat = r.etat();
    expect(etat.statut).toBe('ok');
    if (etat.statut !== 'ok') throw new Error('état inattendu');
    expect(etat.boucles).toHaveLength(1);
    expect(etat.demande.distanceM).toBe(4000);
  });

  it('range les boucles par part hors bitume décroissante', async () => {
    // C'est le critère du produit : la plus verte d'abord, pas la plus proche
    // de la distance demandée.
    const moins = { ...uneBoucle, id: 'moins', score: { ...uneBoucle.score, partNonBitume: 0.4 } };
    const plus = { ...uneBoucle, id: 'plus', score: { ...uneBoucle.score, partNonBitume: 0.9 } };
    const r = creerResultats(moteurQui(async () => [moins, plus]));

    await r.lancer(demande);

    const etat = r.etat();
    if (etat.statut !== 'ok') throw new Error('état inattendu');
    expect(etat.boucles.map((b) => b.id)).toEqual(['plus', 'moins']);
  });

  it('passe en erreur en gardant le genre', async () => {
    const r = creerResultats(moteurQui(async () => {
      throw new ErreurAPI('AucuneBoucle', 'aucune boucle trouvée pour ces critères');
    }));

    await r.lancer(demande);

    const etat = r.etat();
    expect(etat.statut).toBe('erreur');
    if (etat.statut !== 'erreur') throw new Error('état inattendu');
    expect(etat.erreur.genre).toBe('AucuneBoucle');
  });

  it('revient à vide après une annulation, sans afficher d’erreur', async () => {
    // Annuler est un geste volontaire : le traiter comme une panne montrerait
    // un écran d'erreur à quelqu'un qui vient de cliquer sur « Annuler ».
    const r = creerResultats(
      moteurQui(
        (_demande, signal) =>
          new Promise<Boucle[]>((_, rejeter) => {
            signal?.addEventListener('abort', () => rejeter(new DOMException('aborted', 'AbortError')));
          })
      )
    );

    const fini = r.lancer(demande);
    r.annuler();
    await fini;

    expect(r.etat().statut).toBe('vide');
  });

  it('ignore la réponse d’un lancement remplacé par un autre', async () => {
    // Deux recherches enchaînées : la première, plus lente, ne doit pas écraser
    // le résultat de la seconde.
    const lente = { ...uneBoucle, id: 'lente' };
    const rapide = { ...uneBoucle, id: 'rapide' };
    let numero = 0;
    const r = creerResultats(
      moteurQui(async () => {
        numero += 1;
        if (numero === 1) {
          await new Promise((res) => setTimeout(res, 20));
          return [lente];
        }
        return [rapide];
      })
    );

    const premier = r.lancer(demande);
    const second = r.lancer(demande);
    await Promise.all([premier, second]);

    const etat = r.etat();
    if (etat.statut !== 'ok') throw new Error('état inattendu');
    expect(etat.boucles.map((b) => b.id)).toEqual(['rapide']);
  });

  it('ignore l’échec d’un lancement remplacé par un autre déjà abouti', async () => {
    // Le premier lancement traîne puis échoue, après que le second, plus
    // rapide, a déjà écrit son résultat : l'échec tardif ne doit pas écraser
    // un état « ok » valide par un écran d'erreur.
    const rapide = { ...uneBoucle, id: 'rapide' };
    let numero = 0;
    const r = creerResultats(
      moteurQui(async () => {
        numero += 1;
        if (numero === 1) {
          await new Promise((res) => setTimeout(res, 20));
          throw new ErreurAPI('Serveur', 'erreur tardive');
        }
        return [rapide];
      })
    );

    const premier = r.lancer(demande);
    const second = r.lancer(demande);
    await Promise.all([premier, second]);

    const etat = r.etat();
    expect(etat.statut).toBe('ok');
    if (etat.statut !== 'ok') throw new Error('état inattendu');
    expect(etat.boucles.map((b) => b.id)).toEqual(['rapide']);
  });

  it('pose des résultats déjà connus sans appeler le moteur', () => {
    const generer = vi.fn();
    const r = creerResultats({ ...moteurQui(async () => []), generer } as MoteurDeBoucles);

    r.poser([uneBoucle], demande);

    expect(generer).not.toHaveBeenCalled();
    expect(r.etat().statut).toBe('ok');
  });
});
