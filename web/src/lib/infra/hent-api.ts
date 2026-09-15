import type { Boucle, Demande, Score } from '$lib/domaine/boucle';
import type { GenreErreur, MoteurDeBoucles } from '$lib/app/ports';

/** Erreur du moteur, portant le genre qui décide de l'écran à montrer. */
export class ErreurAPI extends Error {
  readonly genre: GenreErreur;
  readonly reessayerDansS?: number;

  constructor(genre: GenreErreur, message: string, reessayerDansS?: number) {
    super(message);
    this.name = 'ErreurAPI';
    this.genre = genre;
    this.reessayerDansS = reessayerDansS;
  }
}

type ScoreJSON = {
  distance_m: number;
  part_non_bitume: number;
  part_trafic: number;
  part_retracee: number;
  ecart_cible: number;
};

type BoucleJSON = { id: string; score: ScoreJSON; geometry: [number, number][] };

type DemandeJSON = {
  start: { lat: number; lon: number };
  distance_m: number;
  preferences: { avoid_paved: number };
  max_results: number;
  variant: number;
};

function versScore(s: ScoreJSON): Score {
  return {
    distanceM: s.distance_m,
    partNonBitume: s.part_non_bitume,
    partTrafic: s.part_trafic,
    partRetracee: s.part_retracee,
    ecartCible: s.ecart_cible
  };
}

function versBoucle(b: BoucleJSON): Boucle {
  return { id: b.id, score: versScore(b.score), geometrie: b.geometry };
}

function versDemande(d: DemandeJSON): Demande {
  return {
    depart: { lat: d.start.lat, lon: d.start.lon },
    distanceM: d.distance_m,
    eviterBitume: d.preferences.avoid_paved,
    maxResultats: d.max_results,
    variante: d.variant
  };
}

function corpsDeDemande(d: Demande): DemandeJSON {
  return {
    start: { lat: d.depart.lat, lon: d.depart.lon },
    distance_m: d.distanceM,
    preferences: { avoid_paved: d.eviterBitume },
    max_results: d.maxResultats,
    variant: d.variante
  };
}

const parStatut: Record<number, GenreErreur> = {
  400: 'HorsZone',
  404: 'AucuneBoucle',
  429: 'TropDeDemandes',
  504: 'DelaiDepasse'
};

async function erreurDe(reponse: Response): Promise<ErreurAPI> {
  const genre = parStatut[reponse.status] ?? 'Reseau';
  let message = `statut ${reponse.status}`;
  try {
    const corps = (await reponse.json()) as { error?: string };
    if (corps.error) message = corps.error;
  } catch {
    // Un corps illisible ne change pas le genre de l'erreur : le statut suffit
    // à décider de l'écran, le message ne sert qu'au journal.
  }
  const delai = Number(reponse.headers.get('Retry-After'));
  return new ErreurAPI(genre, message, Number.isFinite(delai) && delai > 0 ? delai : undefined);
}

/**
 * Enveloppe un appel réseau : traduit les statuts en genres d'erreur, et les
 * pannes de transport en `Reseau`. Une annulation volontaire est relancée telle
 * quelle — la confondre avec une panne ferait afficher un écran d'erreur à
 * quelqu'un qui vient de cliquer sur « Annuler ».
 */
async function appeler(
  fetchImpl: typeof fetch,
  url: string,
  init?: RequestInit
): Promise<Response> {
  let reponse: Response;
  try {
    reponse = await fetchImpl(url, init);
  } catch (e) {
    if (e instanceof DOMException && e.name === 'AbortError') throw e;
    throw new ErreurAPI('Reseau', e instanceof Error ? e.message : String(e));
  }
  if (!reponse.ok) throw await erreurDe(reponse);
  return reponse;
}

/**
 * Crée un moteur adossé à l'API HTTP. `fetchImpl` est injectable pour que les
 * cas d'usage se testent sans réseau.
 */
export function creerMoteurHTTP(fetchImpl: typeof fetch = globalThis.fetch): MoteurDeBoucles {
  return {
    async generer(demande, signal) {
      const reponse = await appeler(fetchImpl, '/v1/loops', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(corpsDeDemande(demande)),
        signal
      });
      const corps = (await reponse.json()) as { loops: BoucleJSON[] | null };
      return (corps.loops ?? []).map(versBoucle);
    },

    async ouvrir(id, signal) {
      const reponse = await appeler(fetchImpl, `/v1/loops/${encodeURIComponent(id)}`, { signal });
      const corps = (await reponse.json()) as { loop: BoucleJSON; request: DemandeJSON };
      return { boucle: versBoucle(corps.loop), demande: versDemande(corps.request) };
    },

    urlGPX(id) {
      return `/v1/loops/${encodeURIComponent(id)}.gpx`;
    }
  };
}
