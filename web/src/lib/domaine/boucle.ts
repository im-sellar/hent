import type { Coord } from './depart';

/** Les mesures d'une boucle, telles que l'API les expose. */
export type Score = {
  distanceM: number;
  partNonBitume: number;
  partTrafic: number;
  partRetracee: number;
  ecartCible: number;
};

/** Une boucle : son identifiant régénérable, ses mesures, son tracé. */
export type Boucle = {
  id: string;
  score: Score;
  /** Points au format GeoJSON, `[lon, lat]` — consommable tel quel par une carte. */
  geometrie: [number, number][];
};

/** Ce qu'on demande au moteur. */
export type Demande = {
  depart: Coord;
  distanceM: number;
  eviterBitume: number;
  maxResultats: number;
  variante: number;
};

/**
 * Durée estimée à une allure donnée, en minutes. Une allure nulle ou négative
 * rend zéro plutôt qu'un infini : une valeur infinie traverserait ensuite tout
 * le formatage sans déclencher la moindre alerte.
 */
export function dureeMinutes(distanceM: number, allureKmH: number): number {
  if (!Number.isFinite(allureKmH) || allureKmH <= 0) return 0;
  if (!Number.isFinite(distanceM) || distanceM <= 0) return 0;
  return (distanceM / 1000 / allureKmH) * 60;
}
