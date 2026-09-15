/**
 * Mise en forme des mesures, en français.
 *
 * Les séparateurs sont posés explicitement plutôt que délégués à
 * `toLocaleString` : le rendu doit être identique quelle que soit la locale du
 * navigateur, et les témoins de test ne doivent pas dépendre de l'ICU installée.
 */

/** Distance lisible : mètres sous le kilomètre, kilomètres à une décimale au-delà. */
export function formatDistance(metres: number): string {
  if (metres < 1000) return `${Math.round(metres)} m`;
  return `${(metres / 1000).toFixed(1).replace('.', ',')} km`;
}

/** Durée lisible : « 45 min » en dessous d'une heure, « 2 h 05 » au-delà. */
export function formatDuree(minutes: number): string {
  const total = Math.round(minutes);
  if (total < 60) return `${total} min`;
  const heures = Math.floor(total / 60);
  return `${heures} h ${String(total % 60).padStart(2, '0')}`;
}

/**
 * Part en pourcentage. Une valeur non nulle mais inférieure à un demi-point
 * garde une décimale : « 0,4 % de tracé refait » est une information que
 * l'arrondi à l'entier effacerait.
 */
export function formatPourcent(part: number): string {
  const pourcent = part * 100;
  if (pourcent > 0 && pourcent < 1) return `${pourcent.toFixed(1).replace('.', ',')} %`;
  return `${Math.round(pourcent)} %`;
}
