import type { Coord } from './depart';

/** L'emprise couverte par le moteur, telle que `GET /v1/regions` la rend. */
export type Zone = { minLat: number; minLon: number; maxLat: number; maxLon: number };

/**
 * Centre de la Bretagne, pour cadrer la carte avant que la zone réelle soit
 * connue ou quand le service ne répond pas.
 */
export const CENTRE_BRETAGNE: Coord = { lat: 48.2, lon: -2.9 };

/** Appartenance, bornes comprises. Un `NaN` échoue à toute comparaison et sort donc faux. */
export function contient(z: Zone, c: Coord): boolean {
  return c.lat >= z.minLat && c.lat <= z.maxLat && c.lon >= z.minLon && c.lon <= z.maxLon;
}

export function centreDe(z: Zone): Coord {
  return { lat: (z.minLat + z.maxLat) / 2, lon: (z.minLon + z.maxLon) / 2 };
}
