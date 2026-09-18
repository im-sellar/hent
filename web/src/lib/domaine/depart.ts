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

/** Un résultat de recherche d'adresse : ce qu'on affiche, où c'est, et à quelle distance de la personne. */
export type Lieu = {
  libelle: string;
  complement: string;
  coord: Coord;
  distanceM?: number;
};

/**
 * Nom d'un point dont on n'a pas d'adresse : ses coordonnées, lisibles.
 * Sert de repli quand le géocodage inverse ne rend rien.
 */
export function libelleParDefaut(c: Coord): string {
  const virgule = (n: number) => n.toFixed(4).replace('.', ',');
  return `${virgule(c.lat)}, ${virgule(c.lon)}`;
}

/**
 * Vrai si deux points sont à moins de 1e-5 degré l'un de l'autre sur chaque
 * axe — environ un mètre. Sert à reconnaître un déplacement de carte que
 * l'écran a lui-même commandé. Un `NaN` échoue à la comparaison : jamais égal.
 */
export function memePoint(a: Coord, b: Coord): boolean {
  return Math.abs(a.lat - b.lat) < 1e-5 && Math.abs(a.lon - b.lon) < 1e-5;
}
