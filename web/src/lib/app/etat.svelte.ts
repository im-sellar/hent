import type { Depart } from '$lib/domaine/depart';
import { REGLAGES_PAR_DEFAUT, borner, type Reglages } from '$lib/domaine/reglages';
import { creerResultats } from './generation/resultats.svelte';
import { creerMoteurHTTP } from '$lib/infra/hent-api';
import { creerPreferences } from '$lib/infra/stockage';

export type Theme = 'auto' | 'sombre' | 'clair';

const prefs = creerPreferences();
const moteur = creerMoteurHTTP();

/**
 * L'état de l'application, assemblé une fois.
 *
 * `depart` et `resultats` ne sont pas persistés : un départ vieux de trois jours
 * n'a aucun sens, et des résultats périmés encore moins. Seuls les réglages et
 * le thème survivent à la session.
 */
function creerEtat() {
  let depart = $state<Depart | null>(null);
  let reglages = $state<Reglages>(prefs.lire() ?? REGLAGES_PAR_DEFAUT);
  let theme = $state<Theme>('auto');
  const resultats = creerResultats(moteur);

  return {
    moteur,
    resultats,
    get depart() {
      return depart;
    },
    poserDepart(d: Depart | null) {
      depart = d;
    },
    get reglages() {
      return reglages;
    },
    regler(r: Reglages) {
      reglages = borner(r);
      prefs.ecrire(reglages);
    },
    get theme() {
      return theme;
    },
    changerTheme(t: Theme) {
      theme = t;
      if (typeof document !== 'undefined') {
        if (t === 'auto') document.documentElement.removeAttribute('data-theme');
        else document.documentElement.setAttribute('data-theme', t);
      }
    }
  };
}

export const appEtat = creerEtat();

/**
 * Libellé de l'écart à la demande, tel que l'écran de détail l'affiche à côté de
 * la distance obtenue. Rendu vide quand il n'apprendrait rien : demande inconnue,
 * ou distance obtenue égale à la demande arrondie.
 */
export function formatEcartCible(demandeM: number, obtenueM: number): string {
  if (!Number.isFinite(demandeM) || demandeM <= 0) return '';
  const demandeKm = Math.round(demandeM / 1000);
  if (demandeKm === Math.round(obtenueM / 1000) && Math.abs(demandeM - obtenueM) < 500) return '';
  return `tu en demandais ${demandeKm}`;
}
