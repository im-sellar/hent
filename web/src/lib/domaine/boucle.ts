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

/**
 * Emprise de plusieurs boucles, au format que MapLibre attend pour cadrer :
 * `[[minLon, minLat], [maxLon, maxLat]]`. `null` s'il n'y a aucun point — un
 * cadrage sur rien n'a pas de sens, et la carte ne doit pas bouger.
 */
export function bornesDe(boucles: Boucle[]): [[number, number], [number, number]] | null {
  let minLon = Infinity, minLat = Infinity, maxLon = -Infinity, maxLat = -Infinity;
  for (const b of boucles) {
    for (const [lon, lat] of b.geometrie) {
      if (lon < minLon) minLon = lon;
      if (lat < minLat) minLat = lat;
      if (lon > maxLon) maxLon = lon;
      if (lat > maxLat) maxLat = lat;
    }
  }
  if (minLon === Infinity) return null;
  return [[minLon, minLat], [maxLon, maxLat]];
}
