import { describe, expect, it } from 'vitest';
import { creerGeocodeurBAN } from './ban';

/** Un `fetch` doublé qui rend le corps donné et note ce qu'on lui demande. */
function fauxFetch(statut: number, corps: unknown) {
  const appels: { url: string; init?: RequestInit }[] = [];
  const impl = (async (url: string, init?: RequestInit) => {
    appels.push({ url, init });
    return new Response(JSON.stringify(corps), {
      status: statut,
      headers: { 'Content-Type': 'application/json' }
    });
  }) as unknown as typeof fetch;
  return { impl, appels };
}

const reponseRecherche = {
  type: 'FeatureCollection',
  features: [
    {
      geometry: { type: 'Point', coordinates: [-1.757305, 48.031122] },
      properties: {
        label: 'Place du Vert Buisson 35170 Bruz',
        name: 'Place du Vert Buisson',
        postcode: '35170',
        city: 'Bruz',
        type: 'street',
        context: '35, Ille-et-Vilaine, Bretagne',
        distance: 1349
      }
    },
    {
      geometry: { type: 'Point', coordinates: [-1.7461, 48.0246] },
      properties: {
        label: 'Bruz',
        name: 'Bruz',
        postcode: '35170',
        city: 'Bruz',
        type: 'municipality',
        context: '35, Ille-et-Vilaine, Bretagne'
      }
    }
  ]
};

describe('chercher', () => {
  it('traduit chaque résultat en lieu, coordonnées dans le bon ordre', async () => {
    const { impl } = fauxFetch(200, reponseRecherche);

    const lieux = await creerGeocodeurBAN(impl).chercher('bruz vert');

    expect(lieux).toHaveLength(2);
    // Valeurs toutes distinctes : une interversion lat/lon ou libellé/complément se verrait.
    expect(lieux[0]).toEqual({
      libelle: 'Place du Vert Buisson',
      complement: '35170 Bruz',
      coord: { lat: 48.031122, lon: -1.757305 },
      distanceM: 1349
    });
  });

  it('présente une commune par son département, sans distance quand la BAN n’en donne pas', async () => {
    const { impl } = fauxFetch(200, reponseRecherche);

    const [, commune] = await creerGeocodeurBAN(impl).chercher('bruz');

    expect(commune?.libelle).toBe('Bruz');
    expect(commune?.complement).toBe('Commune, Ille-et-Vilaine');
    expect(commune?.distanceM).toBeUndefined();
  });

  it('encode le texte, limite à cinq et pondère par la position', async () => {
    const { impl, appels } = fauxFetch(200, { features: [] });

    await creerGeocodeurBAN(impl).chercher('rue du carré vert', { lat: 48.02, lon: -1.75 });

    const url = new URL(appels[0]!.url);
    expect(url.origin).toBe('https://api-adresse.data.gouv.fr');
    expect(url.pathname).toBe('/search/');
    expect(url.searchParams.get('q')).toBe('rue du carré vert');
    expect(url.searchParams.get('limit')).toBe('5');
    expect(url.searchParams.get('lat')).toBe('48.02');
    expect(url.searchParams.get('lon')).toBe('-1.75');
  });

  it('n’envoie pas de position quand on ne la connaît pas', async () => {
    const { impl, appels } = fauxFetch(200, { features: [] });

    await creerGeocodeurBAN(impl).chercher('bruz');

    const url = new URL(appels[0]!.url);
    expect(url.searchParams.has('lat')).toBe(false);
    expect(url.searchParams.has('lon')).toBe(false);
  });

  it('transmet le signal d’annulation', async () => {
    const { impl, appels } = fauxFetch(200, { features: [] });
    const controleur = new AbortController();

    await creerGeocodeurBAN(impl).chercher('bruz', undefined, controleur.signal);

    expect(appels[0]!.init?.signal).toBe(controleur.signal);
  });

  it('lève sur un statut d’erreur', async () => {
    const { impl } = fauxFetch(503, { message: 'indisponible' });

    await expect(creerGeocodeurBAN(impl).chercher('bruz')).rejects.toThrow(/503/);
  });

  it('relance une annulation telle quelle, sans la déguiser en panne', async () => {
    const impl = (async () => {
      throw new DOMException('annulé', 'AbortError');
    }) as unknown as typeof fetch;

    await expect(creerGeocodeurBAN(impl).chercher('bruz')).rejects.toMatchObject({ name: 'AbortError' });
  });
});

describe('nommer', () => {
  it('rend le libellé du premier résultat', async () => {
    const { impl, appels } = fauxFetch(200, {
      features: [{ geometry: { coordinates: [-1.677, 48.117] }, properties: { label: '2 Rue Lesage 35000 Rennes', name: '2 Rue Lesage', type: 'housenumber' } }]
    });

    const nom = await creerGeocodeurBAN(impl).nommer({ lat: 48.117, lon: -1.677 });

    expect(nom).toBe('2 Rue Lesage 35000 Rennes');
    const url = new URL(appels[0]!.url);
    expect(url.pathname).toBe('/reverse/');
    expect(url.searchParams.get('lat')).toBe('48.117');
    expect(url.searchParams.get('lon')).toBe('-1.677');
  });

  it('rend null sans résultat, sur un statut d’erreur et sur une panne réseau', async () => {
    expect(await creerGeocodeurBAN(fauxFetch(200, { features: [] }).impl).nommer({ lat: 48, lon: -2 })).toBeNull();
    expect(await creerGeocodeurBAN(fauxFetch(500, {}).impl).nommer({ lat: 48, lon: -2 })).toBeNull();
    const panne = (async () => {
      throw new TypeError('Failed to fetch');
    }) as unknown as typeof fetch;
    expect(await creerGeocodeurBAN(panne).nommer({ lat: 48, lon: -2 })).toBeNull();
  });

  it('relance une annulation plutôt que de rendre null', async () => {
    const impl = (async () => {
      throw new DOMException('annulé', 'AbortError');
    }) as unknown as typeof fetch;

    await expect(creerGeocodeurBAN(impl).nommer({ lat: 48, lon: -2 })).rejects.toMatchObject({ name: 'AbortError' });
  });
});
