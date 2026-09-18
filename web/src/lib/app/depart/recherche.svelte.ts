import type { Geocodeur } from '$lib/app/ports';
import type { Coord, Lieu } from '$lib/domaine/depart';

export type EtatRecherche =
  | { statut: 'vide' }
  | { statut: 'attente' }
  | { statut: 'ok'; lieux: Lieu[] }
  | { statut: 'erreur' };

export const DELAI_MS = 300;
export const MIN_CARACTERES = 3;

function estAnnulation(e: unknown): boolean {
  return e instanceof DOMException && e.name === 'AbortError';
}

/**
 * Pilote la recherche d'adresse et expose son état.
 *
 * Trois protections : l'appel part `DELAI_MS` après la dernière frappe, pas à
 * chacune ; toute frappe annule l'appel en vol ; et seul le dernier appel peut
 * écrire dans l'état, même si le géocodeur ignore le signal — sans quoi une
 * réponse lente écraserait une réponse rapide et la liste montrerait les
 * résultats d'un texte qu'on n'a plus sous les yeux.
 *
 * Les lieux déjà affichés restent pendant qu'on affine : passer par « attente »
 * à chaque lettre ferait clignoter la liste.
 */
export function creerRecherche(geocodeur: Geocodeur) {
  let etat = $state<EtatRecherche>({ statut: 'vide' });
  let minuterie: ReturnType<typeof setTimeout> | null = null;
  let enCours: AbortController | null = null;
  let generation = 0;

  function arreter() {
    if (minuterie !== null) clearTimeout(minuterie);
    minuterie = null;
    enCours?.abort();
    enCours = null;
    generation += 1;
  }

  return {
    etat: () => etat,

    saisir(texte: string, autour?: Coord): void {
      arreter();
      const propre = texte.trim();
      if (propre.length < MIN_CARACTERES) {
        etat = { statut: 'vide' };
        return;
      }
      if (etat.statut !== 'ok') etat = { statut: 'attente' };
      const mienne = generation;

      minuterie = setTimeout(() => {
        minuterie = null;
        const controleur = new AbortController();
        enCours = controleur;
        geocodeur
          .chercher(propre, autour, controleur.signal)
          .then((lieux) => {
            if (mienne !== generation) return;
            etat = { statut: 'ok', lieux };
          })
          .catch((e: unknown) => {
            if (mienne !== generation || estAnnulation(e)) return;
            etat = { statut: 'erreur' };
          })
          .finally(() => {
            if (enCours === controleur) enCours = null;
          });
      }, DELAI_MS);
    },

    effacer(): void {
      arreter();
      etat = { statut: 'vide' };
    }
  };
}
