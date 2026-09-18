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
import type { Carte } from '$lib/app/ports';
import type { Depart } from '$lib/domaine/depart';
import { REGLAGES_PAR_DEFAUT, borner, type Reglages } from '$lib/domaine/reglages';
import { themeEffectif, type Theme } from '$lib/domaine/theme';
import { CENTRE_BRETAGNE, type Zone } from '$lib/domaine/zone';
import { creerResultats } from './app/generation/resultats.svelte';
import { creerRecherche } from './app/depart/recherche.svelte';
import { creerMoteurHTTP } from '$lib/infra/hent-api';
import { creerPreferences } from '$lib/infra/stockage';
import { creerGeocodeurBAN } from '$lib/infra/ban';
import { creerPosition } from '$lib/infra/geolocalisation';
import { urlStyle, webglDisponible } from '$lib/infra/styles-carte';

const prefs = creerPreferences();
const moteur = creerMoteurHTTP();
const geocodeur = creerGeocodeurBAN();
const position = creerPosition();

function systemePrefereClair(): boolean {
  return typeof matchMedia === 'function' && matchMedia('(prefers-color-scheme: light)').matches;
}

/**
 * L'état de l'application, assemblé une fois.
 *
 * `depart` et `resultats` ne sont pas persistés : un départ vieux de trois jours
 * n'a aucun sens, et des résultats périmés encore moins. Seuls les réglages et
 * le thème survivent à la session.
 *
 * La carte n'existe qu'après `monterCarte`, appelé par le layout une fois dans
 * le navigateur, et seulement si WebGL est disponible : sans lui `carte` reste
 * `null` et les écrans se passent d'elle — les chiffres portent l'information.
 */
function creerEtat() {
  let depart = $state<Depart | null>(null);
  let reglages = $state<Reglages>(prefs.lire() ?? REGLAGES_PAR_DEFAUT);
  let theme = $state<Theme>(prefs.lireTheme() ?? 'auto');
  let carte = $state<Carte | null>(null);
  let montageEnCours = false;
  let styleActuel: string | null = null;
  let zoneConnue: Promise<Zone> | null = null;
  const resultats = creerResultats(moteur);
  const recherche = creerRecherche(geocodeur);

  function appliquerTheme() {
    if (typeof document === 'undefined') return;
    if (theme === 'auto') document.documentElement.removeAttribute('data-theme');
    else document.documentElement.setAttribute('data-theme', theme);
    const url = urlStyle(themeEffectif(theme, systemePrefereClair()));
    if (carte && url !== styleActuel) carte.changerStyle(url);
    styleActuel = url;
  }

  return {
    moteur,
    geocodeur,
    position,
    resultats,
    recherche,

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
      prefs.ecrireTheme(t);
      appliquerTheme();
    },
    appliquerTheme,

    get carte() {
      return carte;
    },
    async monterCarte(conteneur: HTMLElement): Promise<void> {
      if (carte || montageEnCours || !webglDisponible()) return;
      montageEnCours = true;
      try {
        const { creerCarte } = await import('$lib/infra/maplibre');
        if (!conteneur.isConnected) return;
        styleActuel = urlStyle(themeEffectif(theme, systemePrefereClair()));
        carte = creerCarte(conteneur, styleActuel, CENTRE_BRETAGNE, 7);
      } finally {
        montageEnCours = false;
      }
    },
    demonterCarte() {
      carte?.detruire();
      carte = null;
    },

    /**
     * L'emprise couverte, demandée une fois et gardée. Un échec n'est pas
     * gardé : le prochain appel réessaie.
     */
    zone(): Promise<Zone> {
      zoneConnue ??= moteur.zone().catch((e: unknown) => {
        zoneConnue = null;
        throw e;
      });
      return zoneConnue;
    }
  };
}

export const appEtat = creerEtat();
