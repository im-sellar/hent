import { describe, expect, it } from 'vitest';
import { ErreurAPI, creerMoteurHTTP } from './hent-api';
import type { Demande } from '$lib/domaine/boucle';

const demande: Demande = {
  depart: { lat: 48.135, lon: -1.628 },
  distanceM: 4000,
  eviterBitume: 0.5,
  maxResultats: 3,
  variante: 0
};

/** Construit un `fetch` doublé qui rend la réponse donnée, sans réseau. */
function fauxFetch(statut: number, corps: unknown, entetes: Record<string, string> = {}) {
  const appels: { url: string; init?: RequestInit }[] = [];
  const impl = (async (url: string, init?: RequestInit) => {
    appels.push({ url, init });
    return new Response(JSON.stringify(corps), {
      status: statut,
      headers: { 'Content-Type': 'application/json', ...entetes }
    });
  }) as unknown as typeof fetch;
  return { impl, appels };
}

describe('generer', () => {
  it('rend les boucles de la réponse', async () => {
    const { impl } = fauxFetch(200, {
      loops: [
        {
          id: 'abc',
          score: {
            distance_m: 3924.32,
            part_non_bitume: 1,
            part_trafic: 0,
            part_retracee: 0,
            ecart_cible: -0.0189
          },
          geometry: [
            [-1.628, 48.135],
            [-1.629, 48.136]
          ]
        }
      ],
      attribution: 'Données © les contributeurs OpenStreetMap'
    });

    const boucles = await creerMoteurHTTP(impl).generer(demande);

    expect(boucles).toHaveLength(1);
    expect(boucles[0]!.id).toBe('abc');
    expect(boucles[0]!.score.distanceM).toBe(3924.32);
    expect(boucles[0]!.score.ecartCible).toBeCloseTo(-0.0189, 4);
    expect(boucles[0]!.geometrie[0]).toEqual([-1.628, 48.135]);
  });

  it('envoie la demande dans le corps, au format de l’API', async () => {
    const { impl, appels } = fauxFetch(200, { loops: [], attribution: '' });

    await creerMoteurHTTP(impl).generer(demande);

    expect(appels).toHaveLength(1);
    const envoye = JSON.parse(String(appels[0]!.init!.body));
    expect(envoye).toEqual({
      start: { lat: 48.135, lon: -1.628 },
      distance_m: 4000,
      preferences: { avoid_paved: 0.5 },
      max_results: 3,
      variant: 0
    });
  });

  it('traduit un 400 en HorsZone', async () => {
    const { impl } = fauxFetch(400, { error: 'le point de départ est hors de la zone couverte' });
    await expect(creerMoteurHTTP(impl).generer(demande)).rejects.toMatchObject({ genre: 'HorsZone' });
  });

  it('traduit un 404 en AucuneBoucle', async () => {
    const { impl } = fauxFetch(404, { error: 'aucune boucle trouvée pour ces critères' });
    await expect(creerMoteurHTTP(impl).generer(demande)).rejects.toMatchObject({ genre: 'AucuneBoucle' });
  });

  it('traduit un 429 et retient le délai annoncé', async () => {
    const { impl } = fauxFetch(429, { error: 'trop de requêtes' }, { 'Retry-After': '30' });
    await expect(creerMoteurHTTP(impl).generer(demande)).rejects.toMatchObject({
      genre: 'TropDeDemandes',
      reessayerDansS: 30
    });
  });

  it('traduit un 504 en DelaiDepasse', async () => {
    const { impl } = fauxFetch(504, { error: 'délai dépassé' });
    await expect(creerMoteurHTTP(impl).generer(demande)).rejects.toMatchObject({ genre: 'DelaiDepasse' });
  });

  it('traduit un échec de transport en Reseau', async () => {
    const impl = (async () => {
      throw new TypeError('Failed to fetch');
    }) as unknown as typeof fetch;
    await expect(creerMoteurHTTP(impl).generer(demande)).rejects.toMatchObject({ genre: 'Reseau' });
  });

  it('laisse passer une annulation sans la déguiser en erreur réseau', async () => {
    // Une annulation volontaire n'est pas une panne : la confondre avec un
    // échec ferait afficher un écran d'erreur à quelqu'un qui vient de cliquer
    // sur « Annuler ».
    const impl = (async () => {
      throw new DOMException('aborted', 'AbortError');
    }) as unknown as typeof fetch;

    await expect(creerMoteurHTTP(impl).generer(demande)).rejects.toSatisfy(
      (e: unknown) => e instanceof DOMException && e.name === 'AbortError'
    );
  });
});

describe('ouvrir', () => {
  it('rend la boucle et la demande d’origine', async () => {
    const { impl, appels } = fauxFetch(200, {
      loop: {
        id: '0.abc',
        score: {
          distance_m: 17_400,
          part_non_bitume: 0.55,
          part_trafic: 0.023,
          part_retracee: 0.004,
          ecart_cible: -0.032
        },
        geometry: [[-1.628, 48.135]]
      },
      request: {
        start: { lat: 48.135, lon: -1.628 },
        distance_m: 18_000,
        tolerance: 0.15,
        preferences: { avoid_paved: 0.8 },
        max_results: 5,
        variant: 2
      },
      attribution: 'Données © les contributeurs OpenStreetMap'
    });

    const { boucle, demande: dem } = await creerMoteurHTTP(impl).ouvrir('0.abc');

    expect(appels[0]!.url).toContain('/v1/loops/0.abc');
    expect(boucle.score.partRetracee).toBeCloseTo(0.004, 4);
    // La demande d'origine est ce qui permet d'afficher « tu en demandais 18 » :
    // elle ne se déduit pas de la boucle.
    expect(dem.distanceM).toBe(18_000);
    expect(dem.eviterBitume).toBe(0.8);
    expect(dem.variante).toBe(2);
  });

  it('encode l’identifiant dans l’URL', async () => {
    const { impl, appels } = fauxFetch(200, {
      loop: { id: 'x', score: { distance_m: 0, part_non_bitume: 0, part_trafic: 0, part_retracee: 0, ecart_cible: 0 }, geometry: [] },
      request: { start: { lat: 0, lon: 0 }, distance_m: 0, preferences: { avoid_paved: 0 }, max_results: 0, variant: 0 },
      attribution: ''
    });

    await creerMoteurHTTP(impl).ouvrir('0.a+b/c');

    expect(appels[0]!.url).toContain(encodeURIComponent('0.a+b/c'));
  });
});

describe('urlGPX', () => {
  it('rend une URL téléchargeable, avec le suffixe qui distingue l’export', () => {
    expect(creerMoteurHTTP().urlGPX('0.abc')).toBe('/v1/loops/0.abc.gpx');
  });

  it('encode l’identifiant', () => {
    expect(creerMoteurHTTP().urlGPX('0.a+b')).toBe(`/v1/loops/${encodeURIComponent('0.a+b')}.gpx`);
  });
});
