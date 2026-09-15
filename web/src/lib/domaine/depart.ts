/** Un point sur la Terre, en degrés décimaux. */
export type Coord = { lat: number; lon: number };

/** Un point de départ, avec le nom qu'on lui donne à l'écran. */
export type Depart = { coord: Coord; libelle: string };

/**
 * Valide une coordonnée. Les valeurs non finies sont écartées en premier :
 * `NaN` traverse toute comparaison de bornes sans jamais la faire échouer.
 */
export function estCoordValide(c: Coord): boolean {
  if (!Number.isFinite(c.lat) || !Number.isFinite(c.lon)) return false;
  return c.lat >= -90 && c.lat <= 90 && c.lon >= -180 && c.lon <= 180;
}
