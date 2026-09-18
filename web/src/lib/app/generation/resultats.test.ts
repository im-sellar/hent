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
    urlGPX: (id) => `/v1/loops/${id}.gpx`,
    zone: async () => ({ minLat: 47.2, minLon: -5.2, maxLat: 48.95, maxLon: -0.95 })
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

  it('abandonne l’appel en cours quand on pose des résultats connus', () => {
    // L'écran de détail pose la boucle qu'il vient d'obtenir alors qu'une
    // recherche peut encore être en vol : la laisser courir la ferait aboutir
    // dans le vide, et consommerait le service pour rien.
    let signalVu: AbortSignal | undefined;
    const r = creerResultats(
      moteurQui((_demande, signal) => {
        signalVu = signal;
        return new Promise<Boucle[]>(() => {});
      })
    );

    void r.lancer(demande);
    r.poser([uneBoucle], demande);

    expect(signalVu?.aborted).toBe(true);
  });

  it('ignore une recherche déjà partie quand des résultats connus sont posés', async () => {
    // Le moteur ici ne regarde pas le signal : seule la génération peut écarter
    // sa réponse, qui arrive après la pose.
    const tardive = { ...uneBoucle, id: 'tardive' };
    const connue = { ...uneBoucle, id: 'connue' };
    let debloquer!: (b: Boucle[]) => void;
    const attente = new Promise<Boucle[]>((res) => (debloquer = res));
    const r = creerResultats(moteurQui(() => attente));

    const fini = r.lancer(demande);
    r.poser([connue], demande);
    debloquer([tardive]);
    await fini;

    const etat = r.etat();
    if (etat.statut !== 'ok') throw new Error('état inattendu');
    expect(etat.boucles.map((b) => b.id)).toEqual(['connue']);
  });

  it('n’annule plus rien une fois la recherche terminée', async () => {
    let signalVu: AbortSignal | undefined;
    const r = creerResultats(
      moteurQui(async (_demande, signal) => {
        signalVu = signal;
        return [uneBoucle];
      })
    );

    await r.lancer(demande);
    r.annuler();

    expect(signalVu?.aborted).toBe(false);
  });

  it('abandonne l’appel en cours quand on réinitialise', () => {
    // Corriger le départ réinitialise l'état pendant qu'une recherche peut
    // encore être en vol : la laisser courir consommerait le service pour rien.
    let signalVu: AbortSignal | undefined;
    const r = creerResultats(
      moteurQui((_demande, signal) => {
        signalVu = signal;
        return new Promise<Boucle[]>(() => {});
      })
    );

    void r.lancer(demande);
    r.reinitialiser();

    expect(signalVu?.aborted).toBe(true);
  });

  it('ignore une recherche déjà partie quand on réinitialise', async () => {
    // Le moteur ici ne regarde pas le signal : seule la génération peut écarter
    // sa réponse, qui arrive après la réinitialisation.
    let debloquer!: (b: Boucle[]) => void;
    const attente = new Promise<Boucle[]>((res) => (debloquer = res));
    const r = creerResultats(moteurQui(() => attente));

    const fini = r.lancer(demande);
    r.reinitialiser();
    debloquer([uneBoucle]);
    await fini;

    expect(r.etat().statut).toBe('vide');
  });

  it('range sans toucher à la liste reçue', () => {
    // L'appelant garde sa liste : `poser` la range pour son propre état, il ne
    // réordonne pas celle d'en face.
    const moins = { ...uneBoucle, id: 'moins', score: { ...uneBoucle.score, partNonBitume: 0.4 } };
    const plus = { ...uneBoucle, id: 'plus', score: { ...uneBoucle.score, partNonBitume: 0.9 } };
    const recue = [moins, plus];
    const r = creerResultats(moteurQui(async () => []));

    r.poser(recue, demande);

    expect(recue.map((b) => b.id)).toEqual(['moins', 'plus']);
    const etat = r.etat();
    if (etat.statut !== 'ok') throw new Error('état inattendu');
    expect(etat.boucles.map((b) => b.id)).toEqual(['plus', 'moins']);
  });
});
