/** Un point sur la Terre, en degrés décimaux. */
export type Coord = { lat: number; lon: number };

/** Un point de départ, avec le nom qu'on lui donne à l'écran. */
export type Depart = { coord: Coord; libelle: string };

/**
 * Valide une latitude, bornes comprises. Aucune garde explicite sur les valeurs
 * non finies n'est nécessaire : toute comparaison impliquant `NaN` rend `false`,
 * et un infini sort toujours des bornes — les comparaisons ci-dessous les
 * écartent donc d'elles-mêmes.
 */
export function estLatValide(lat: number): boolean {
  return lat >= -90 && lat <= 90;
}

/** Valide une longitude, bornes comprises. Même raisonnement que pour la latitude. */
export function estLonValide(lon: number): boolean {
  return lon >= -180 && lon <= 180;
}

/**
 * Valide une coordonnée. Les deux axes sont validés séparément parce qu'un
 * formulaire doit pouvoir désigner celui qui est fautif.
 */
export function estCoordValide(c: Coord): boolean {
  return estLatValide(c.lat) && estLonValide(c.lon);
}
