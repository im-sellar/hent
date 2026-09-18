/**
 * Le point d'assemblage de l'application : il relie les implémentations
 * concrètes d'`infra` aux ports que `domaine` et `app` consomment. C'est le
 * rôle que joue `cmd/routed/main.go` côté back, délibérément situé hors de
 * `internal/` pour la même raison — un point d'assemblage doit pouvoir
 * connaître toutes les couches, ce qu'aucune couche gardée ne peut se
 * permettre. Ce fichier vit donc à la racine de `lib/`, hors de `domaine/`,
 * `app/`, `infra/` et `ui/` : `architecture.test.ts` ne le couvre pas, comme
 * `architecture_test.go` ne couvre pas `cmd/`.
 */
import type { Depart } from '$lib/domaine/depart';
import { REGLAGES_PAR_DEFAUT, borner, type Reglages } from '$lib/domaine/reglages';
import { creerResultats } from './app/generation/resultats.svelte';
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
