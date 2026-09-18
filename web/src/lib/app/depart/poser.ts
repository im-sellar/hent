import type { Geocodeur } from '$lib/app/ports';
import { libelleParDefaut, type Coord, type Depart } from '$lib/domaine/depart';

/**
 * Fait d'une coordonnée un départ nommé. Le géocodage inverse peut échouer ou
 * ne rien trouver : le point reste valable et prend ses coordonnées pour nom.
 * Une annulation, elle, est relancée — l'appelant a changé d'avis, ce départ
 * ne doit pas être posé.
 */
export async function nommerPoint(geocodeur: Geocodeur, coord: Coord, signal?: AbortSignal): Promise<Depart> {
  let libelle: string | null = null;
  try {
    libelle = await geocodeur.nommer(coord, signal);
  } catch (e) {
    if (e instanceof DOMException && e.name === 'AbortError') throw e;
  }
  return { coord, libelle: libelle ?? libelleParDefaut(coord) };
}
