import type { Boucle, Demande } from '$lib/domaine/boucle';
import type { Coord, Lieu } from '$lib/domaine/depart';
import type { Reglages } from '$lib/domaine/reglages';
import type { Theme } from '$lib/domaine/theme';
import type { Zone } from '$lib/domaine/zone';

/**
 * La carte, vue par l'application : une façade à ordres, pas un état.
 * MapLibre est impératif ; le rendre déclaratif reviendrait à se battre
 * contre lui.
 *
 * `surDeplacement` rend sa fonction de désabonnement : un abonnement qu'on ne
 * peut pas rompre fuit à chaque navigation. `changerStyle` recharge toutes les
 * couches, l'adaptateur repose ce qu'il affichait. `redimensionner` est à
 * appeler quand le conteneur redevient visible.
 */
export interface Carte {
  centrer(point: Coord, zoom?: number): void;
  afficherBoucles(boucles: Boucle[], selectionnee: string | null): void;
  marquerDepart(point: Coord | null): void;
  montrerZone(zone: Zone | null): void;
  surDeplacement(rappel: (centre: Coord) => void): () => void;
  changerStyle(url: string): void;
  redimensionner(): void;
  detruire(): void;
}

/**
 * Les six façons dont une recherche peut échouer. Cinq viennent du serveur,
 * la sixième de ce qui ne l'atteint jamais.
 *
 * Elles sont nommées plutôt que réduites à un message, parce que chaque écran
 * d'erreur propose une sortie différente : déplacer le départ, assouplir un
 * réglage, ou seulement attendre. `Serveur` couvre les statuts d'erreur
 * serveur (5xx) non répertoriés individuellement — la requête a été reçue et
 * a échoué côté serveur, ce n'est donc pas une panne réseau.
 */
export type GenreErreur =
  | 'HorsZone'
  | 'AucuneBoucle'
  | 'TropDeDemandes'
  | 'DelaiDepasse'
  | 'Serveur'
  | 'Reseau';

export type ErreurMoteur = {
  genre: GenreErreur;
  /** Message du serveur, destiné au journal — jamais affiché tel quel. */
  message: string;
  /** Délai annoncé par l'en-tête Retry-After, en secondes, quand il existe. */
  reessayerDansS?: number;
};

/** Le moteur de boucles, vu par l'application. */
export interface MoteurDeBoucles {
  generer(demande: Demande, signal?: AbortSignal): Promise<Boucle[]>;
  ouvrir(id: string, signal?: AbortSignal): Promise<{ boucle: Boucle; demande: Demande }>;
  /** URL d'export : un lien que le navigateur suit, pas un corps qu'on relaie. */
  urlGPX(id: string): string;
  /** L'emprise couverte par le moteur : tout départ hors de cette zone sera refusé. */
  zone(signal?: AbortSignal): Promise<Zone>;
}

/**
 * Le stockage des préférences. `lire` rend `null` plutôt que de lever quand le
 * stockage est indisponible — navigation privée, quota, stockage bloqué : rien
 * de tout cela ne doit empêcher l'application de démarrer.
 */
export interface Preferences {
  lire(): Reglages | null;
  ecrire(r: Reglages): void;
  lireTheme(): Theme | null;
  ecrireTheme(t: Theme): void;
}

/**
 * Le géocodage, vu par l'application : trouver des lieux depuis un texte, et
 * nommer un point.
 *
 * `chercher` pondère par `autour` quand on connaît la position — on cherche
 * presque toujours près de soi. `nommer` rend `null` plutôt que de lever : un
 * point sans adresse reste un point de départ valable, et l'écran lui donnera
 * ses coordonnées pour nom.
 */
export interface Geocodeur {
  chercher(texte: string, autour?: Coord, signal?: AbortSignal): Promise<Lieu[]>;
  nommer(point: Coord, signal?: AbortSignal): Promise<string | null>;
}

export type ResultatPosition =
  | { statut: 'ok'; coord: Coord }
  | { statut: 'refusee' }
  | { statut: 'indisponible' };

/**
 * La position de la personne, vue par l'application. Trois issues et jamais
 * de rejet : un refus de permission est une réponse, pas une panne, et l'écran
 * propose une sortie différente pour chacune.
 */
export interface Position {
  obtenir(): Promise<ResultatPosition>;
}
