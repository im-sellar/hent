import type { Boucle, Demande } from '$lib/domaine/boucle';
import type { Coord, Lieu } from '$lib/domaine/depart';
import type { Reglages } from '$lib/domaine/reglages';

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
}

/**
 * Le stockage des préférences. `lire` rend `null` plutôt que de lever quand le
 * stockage est indisponible — navigation privée, quota, stockage bloqué : rien
 * de tout cela ne doit empêcher l'application de démarrer.
 */
export interface Preferences {
  lire(): Reglages | null;
  ecrire(r: Reglages): void;
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
