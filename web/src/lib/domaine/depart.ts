/** Un point sur la Terre, en degrés décimaux. */
export type Coord = { lat: number; lon: number };

/** Un point de départ, avec le nom qu'on lui donne à l'écran. */
export type Depart = { coord: Coord; libelle: string };

/**
 * Valide une coordonnée. Aucune garde explicite sur les valeurs non finies
 * n'est nécessaire : toute comparaison impliquant `NaN` rend `false`, et un
 * infini sort toujours des bornes — les comparaisons ci-dessous les écartent
 * donc d'elles-mêmes.
 */
export function estCoordValide(c: Coord): boolean {
  return c.lat >= -90 && c.lat <= 90 && c.lon >= -180 && c.lon <= 180;
}
