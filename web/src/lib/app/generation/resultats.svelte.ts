import type { Boucle, Demande } from '$lib/domaine/boucle';
import type { ErreurMoteur, MoteurDeBoucles } from '$lib/app/ports';

/**
 * Les quatre états d'une recherche. Ils sont exhaustifs et mutuellement
 * exclusifs : un écran qui les lit ne peut pas se retrouver à deviner s'il doit
 * montrer une liste, une attente ou une erreur.
 */
export type EtatResultats =
  | { statut: 'vide' }
  | { statut: 'calcul' }
  | { statut: 'ok'; boucles: Boucle[]; demande: Demande }
  | { statut: 'erreur'; erreur: ErreurMoteur };

function estAnnulation(e: unknown): boolean {
  return e instanceof DOMException && e.name === 'AbortError';
}

function versErreurMoteur(e: unknown): ErreurMoteur {
  if (e && typeof e === 'object' && 'genre' in e) {
    const erreur = e as { genre: ErreurMoteur['genre']; message?: string; reessayerDansS?: number };
    return {
      genre: erreur.genre,
      message: erreur.message ?? '',
      reessayerDansS: erreur.reessayerDansS
    };
  }
  return { genre: 'Reseau', message: e instanceof Error ? e.message : String(e) };
}

/**
 * Pilote une recherche et expose son état.
 *
 * Deux protections méritent d'être dites : une annulation ramène à l'état vide
 * plutôt qu'en erreur — c'est un geste volontaire, pas une panne — et seul le
 * dernier lancement peut écrire dans l'état, sans quoi une première recherche
 * lente écraserait le résultat d'une seconde plus rapide.
 */
export function creerResultats(moteur: MoteurDeBoucles) {
  let etat = $state<EtatResultats>({ statut: 'vide' });
  let enCours: AbortController | null = null;
  let generation = 0;

  function trier(boucles: Boucle[]): Boucle[] {
    return [...boucles].sort((a, b) => b.score.partNonBitume - a.score.partNonBitume);
  }

  return {
    etat: () => etat,

    async lancer(demande: Demande): Promise<void> {
      enCours?.abort();
      const controleur = new AbortController();
      enCours = controleur;
      const mienne = ++generation;

      etat = { statut: 'calcul' };
      try {
        const boucles = await moteur.generer(demande, controleur.signal);
        if (mienne !== generation) return;
        etat = { statut: 'ok', boucles: trier(boucles), demande };
      } catch (e) {
        if (mienne !== generation) return;
        if (estAnnulation(e)) {
          etat = { statut: 'vide' };
          return;
        }
        etat = { statut: 'erreur', erreur: versErreurMoteur(e) };
      } finally {
        if (enCours === controleur) enCours = null;
      }
    },

    annuler(): void {
      enCours?.abort();
    },

    /** Pose des résultats déjà connus — au retour d'un détail, par exemple. */
    poser(boucles: Boucle[], demande: Demande): void {
      generation += 1;
      etat = { statut: 'ok', boucles: trier(boucles), demande };
    },

    reinitialiser(): void {
      enCours?.abort();
      generation += 1;
      etat = { statut: 'vide' };
    }
  };
}
