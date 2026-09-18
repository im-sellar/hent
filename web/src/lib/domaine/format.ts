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

/**
 * Libellé de l'écart à la demande, tel que l'écran de détail l'affiche à côté de
 * la distance obtenue. Rendu vide quand il n'apprendrait rien : demande inconnue,
 * ou distance obtenue à la fois arrondie au même kilomètre que la demande et
 * distante d'elle de moins de 500 m. Les deux conditions sont nécessaires : deux
 * distances qui partagent leur arrondi peuvent être séparées de presque un
 * kilomètre.
 */
export function formatEcartCible(demandeM: number, obtenueM: number): string {
  if (!Number.isFinite(demandeM) || demandeM <= 0) return '';
  const demandeKm = Math.round(demandeM / 1000);
  if (demandeKm === Math.round(obtenueM / 1000) && Math.abs(demandeM - obtenueM) < 500) return '';
  return `tu en demandais ${demandeKm}`;
}

/**
 * Distance demandee, au kilometre entier : « 18 km au depart de Bruz ». Une
 * demande se fait en kilometres ronds ; la decimale de `formatDistance` n'y
 * apporterait rien.
 */
export function formatKilometres(metres: number): string {
  if (!Number.isFinite(metres)) return '— km';
  return `${Math.round(metres / 1000)} km`;
}
