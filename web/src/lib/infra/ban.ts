import type { Geocodeur } from '$lib/app/ports';
import type { Coord, Lieu } from '$lib/domaine/depart';

const BASE = 'https://api-adresse.data.gouv.fr';

type ProprietesBAN = {
  label: string;
  name: string;
  postcode?: string;
  city?: string;
  type: 'housenumber' | 'street' | 'locality' | 'municipality';
  /** « 35, Ille-et-Vilaine, Bretagne » : numéro, nom du département, région. */
  context?: string;
  /** Mètres depuis `lat`/`lon` quand la recherche les a reçus. */
  distance?: number;
};

type TraitBAN = { geometry: { coordinates: [number, number] }; properties: ProprietesBAN };
type CollectionBAN = { features?: TraitBAN[] };

function estAnnulation(e: unknown): boolean {
  return e instanceof DOMException && e.name === 'AbortError';
}

/**
 * Ce que l'écran affiche sous le nom : pour une commune, sa nature et son
 * département ; pour tout le reste, code postal et commune.
 */
export function complementDe(p: ProprietesBAN): string {
  if (p.type === 'municipality') {
    const departement = p.context?.split(',')[1]?.trim();
    return departement ? `Commune, ${departement}` : 'Commune';
  }
  return [p.postcode, p.city].filter(Boolean).join(' ');
}

export function versLieu(t: TraitBAN): Lieu {
  const [lon, lat] = t.geometry.coordinates;
  const lieu: Lieu = { libelle: t.properties.name, complement: complementDe(t.properties), coord: { lat, lon } };
  if (typeof t.properties.distance === 'number') lieu.distanceM = t.properties.distance;
  return lieu;
}

async function lire(fetchImpl: typeof fetch, url: URL, signal?: AbortSignal): Promise<CollectionBAN> {
  const reponse = await fetchImpl(url.toString(), { signal });
  if (!reponse.ok) throw new Error(`BAN : statut ${reponse.status}`);
  return (await reponse.json()) as CollectionBAN;
}

/**
 * Géocodeur adossé à la Base Adresse Nationale. `fetchImpl` est injectable pour
 * tester sans réseau.
 *
 * La réponse ne porte aucune attribution : l'écran qui affiche les résultats
 * mentionne la source lui-même (Licence Ouverte 2.0).
 */
export function creerGeocodeurBAN(fetchImpl: typeof fetch = globalThis.fetch): Geocodeur {
  return {
    async chercher(texte, autour, signal) {
      const url = new URL('/search/', BASE);
      url.searchParams.set('q', texte);
      url.searchParams.set('limit', '5');
      if (autour) {
        url.searchParams.set('lat', String(autour.lat));
        url.searchParams.set('lon', String(autour.lon));
      }
      const corps = await lire(fetchImpl, url, signal);
      return (corps.features ?? []).map(versLieu);
    },

    async nommer(point, signal) {
      const url = new URL('/reverse/', BASE);
      url.searchParams.set('lat', String(point.lat));
      url.searchParams.set('lon', String(point.lon));
      try {
        const corps = await lire(fetchImpl, url, signal);
        return corps.features?.[0]?.properties.label ?? null;
      } catch (e) {
        if (estAnnulation(e)) throw e;
        return null;
      }
    }
  };
}
