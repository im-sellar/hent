/**
 * Ce que la carte demande au navigateur sans avoir besoin de MapLibre : l'URL
 * du style servi, la disponibilité de WebGL, et les couleurs des tracés lues
 * dans les jetons CSS. Séparé de `maplibre.ts` pour que l'assemblage puisse le
 * consulter sans tirer le moteur de carte dans le morceau du layout — l'accueil
 * pré-rendu ne montre jamais de carte.
 */
import type { ThemeEffectif } from '$lib/domaine/theme';

/** Couleurs des tracés, lues dans les jetons CSS au moment de peindre : elles suivent le thème. */
export type Couleurs = { trace: string; ecartee: string; depart: string; contourDepart: string; zone: string };

export function urlStyle(t: ThemeEffectif): string {
  return `/carte/hent-${t}.json`;
}

/** Vrai si un contexte WebGL peut être créé. Sans lui la carte ne s'affiche pas, et les chiffres suffisent. */
export function webglDisponible(doc: Document = document): boolean {
  try {
    const toile = doc.createElement('canvas');
    return Boolean(toile.getContext('webgl2') ?? toile.getContext('webgl'));
  } catch {
    return false;
  }
}

export function couleursDepuisJetons(racine: Element = document.documentElement): Couleurs {
  const styles = getComputedStyle(racine);
  const jeton = (nom: string) => styles.getPropertyValue(nom).trim();
  return {
    trace: jeton('--accent'),
    ecartee: jeton('--voie'),
    depart: jeton('--accent-vif'),
    contourDepart: jeton('--fond'),
    zone: jeton('--alerte')
  };
}
