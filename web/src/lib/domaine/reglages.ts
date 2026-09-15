/** Ce que la personne règle avant de lancer une recherche. */
export type Reglages = { distanceM: number; eviterBitume: number };

export const DISTANCE_MIN_M = 2000;
export const DISTANCE_MAX_M = 50_000;

export const REGLAGES_PAR_DEFAUT: Reglages = { distanceM: 12_000, eviterBitume: 0.7 };

/**
 * Ramène des réglages dans leurs bornes. Une valeur non finie ne peut pas être
 * bornée — la comparer ne produirait ni vrai ni faux — elle est donc remplacée
 * par le défaut plutôt que propagée.
 */
export function borner(r: Reglages): Reglages {
  const distanceM = Number.isFinite(r.distanceM)
    ? Math.min(DISTANCE_MAX_M, Math.max(DISTANCE_MIN_M, r.distanceM))
    : REGLAGES_PAR_DEFAUT.distanceM;

  const eviterBitume = Number.isFinite(r.eviterBitume)
    ? Math.min(1, Math.max(0, r.eviterBitume))
    : REGLAGES_PAR_DEFAUT.eviterBitume;

  return { distanceM, eviterBitume };
}
