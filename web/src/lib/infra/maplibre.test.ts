import { describe, expect, it, vi, type Mock } from 'vitest';
import { adapterCarte, urlStyle, versCollection, type MapLike } from './maplibre';
import type { Boucle } from '$lib/domaine/boucle';

const score = { distanceM: 1, partNonBitume: 0, partTrafic: 0, partRetracee: 0, ecartCible: 0 };
const a: Boucle = { id: 'a', score, geometrie: [[-1.7, 48.1], [-1.6, 48.2]] };
const b: Boucle = { id: 'b', score, geometrie: [[-1.8, 48.15], [-1.75, 48.16]] };
const couleurs = () => ({ trace: '#t', ecartee: '#e', depart: '#d', contourDepart: '#c', zone: '#z' });

/** Un double de `maplibregl.Map` qui enregistre les ordres et laisse le test déclencher les événements. */
function fausseMap() {
  const ecouteurs = new Map<string, Set<(...a: unknown[]) => void>>();
  const sources = new Map<string, { setData: Mock }>();
  const ordres: string[] = [];
  const map: MapLike = {
    on: (ev, fn) => {
      if (!ecouteurs.has(ev)) ecouteurs.set(ev, new Set());
      ecouteurs.get(ev)!.add(fn);
    },
    off: (ev, fn) => void ecouteurs.get(ev)?.delete(fn),
    getSource: (id) => sources.get(id),
    addSource: vi.fn((id: string) => {
      sources.set(id, { setData: vi.fn() });
      ordres.push(`source:${id}`);
    }),
    addLayer: vi.fn((spec: { id: string }) => void ordres.push(`layer:${spec.id}`)),
    easeTo: vi.fn(),
    jumpTo: vi.fn(),
    fitBounds: vi.fn(),
    setStyle: vi.fn(),
    resize: vi.fn(),
    remove: vi.fn(),
    getCenter: () => ({ lng: -1.677, lat: 48.117 })
  };
  const declencher = (ev: string) => ecouteurs.get(ev)?.forEach((fn) => fn());
  return { map, sources, ordres, declencher, ecouteurs };
}

describe('versCollection', () => {
  it('fait un trait par boucle et marque la sélectionnée, coordonnées inchangées', () => {
    const c = versCollection([a, b], 'b');
    expect(c.features).toHaveLength(2);
    expect(c.features[0]).toEqual({
      type: 'Feature',
      properties: { id: 'a', selectionnee: false },
      geometry: { type: 'LineString', coordinates: [[-1.7, 48.1], [-1.6, 48.2]] }
    });
    expect(c.features[1]!.properties.selectionnee).toBe(true);
  });

  it('ne sélectionne rien quand l’identifiant est inconnu ou nul', () => {
    expect(versCollection([a], 'zzz').features[0]!.properties.selectionnee).toBe(false);
    expect(versCollection([a], null).features[0]!.properties.selectionnee).toBe(false);
  });
});

describe('urlStyle', () => {
  it('pointe sur les styles servis en statique', () => {
    expect(urlStyle('sombre')).toBe('/carte/hent-sombre.json');
    expect(urlStyle('clair')).toBe('/carte/hent-clair.json');
  });
});

describe('adapterCarte', () => {
  it('ne pose rien avant que le style soit chargé, puis pose sources et couches dans l’ordre', () => {
    const { map, ordres, declencher } = fausseMap();
    adapterCarte(map, couleurs, () => true);

    expect(ordres).toEqual([]);
    declencher('style.load');

    expect(ordres).toEqual([
      'source:zone', 'layer:zone',
      'source:boucles', 'layer:boucles-ecartees', 'layer:boucle-selectionnee',
      'source:depart', 'layer:depart'
    ]);
  });

  it('repose ce qu’il affichait après un changement de style', () => {
    const { map, sources, declencher } = fausseMap();
    const carte = adapterCarte(map, couleurs, () => true);
    declencher('style.load');
    carte.afficherBoucles([a], 'a');
    carte.marquerDepart({ lat: 48.1, lon: -1.7 });

    carte.changerStyle('/carte/hent-clair.json');
    expect(map.setStyle).toHaveBeenCalledWith('/carte/hent-clair.json');
    (map.addSource as ReturnType<typeof vi.fn>).mockClear();
    declencher('style.load');

    const appels = (map.addSource as ReturnType<typeof vi.fn>).mock.calls as [string, { data: unknown }][];
    const boucles = appels.find(([id]) => id === 'boucles')![1].data as ReturnType<typeof versCollection>;
    expect(boucles.features.map((f) => f.properties.id)).toEqual(['a']);
    const depart = appels.find(([id]) => id === 'depart')![1].data as { features: { geometry: { coordinates: number[] } }[] };
    expect(depart.features[0]!.geometry.coordinates).toEqual([-1.7, 48.1]);
    expect(sources.size).toBeGreaterThan(0);
  });

  it('met les boucles à jour par setData et cadre sur elles quand l’ensemble change', () => {
    const { map, sources, declencher } = fausseMap();
    const carte = adapterCarte(map, couleurs, () => true);
    declencher('style.load');

    carte.afficherBoucles([a, b], 'a');
    expect(sources.get('boucles')!.setData).toHaveBeenCalledTimes(1);
    expect(map.fitBounds).toHaveBeenCalledTimes(1);
    expect(map.fitBounds).toHaveBeenCalledWith([[-1.8, 48.1], [-1.6, 48.2]], expect.objectContaining({ duration: 600 }));

    carte.afficherBoucles([a, b], 'b');
    expect(sources.get('boucles')!.setData).toHaveBeenCalledTimes(2);
    expect(map.fitBounds).toHaveBeenCalledTimes(1);

    carte.afficherBoucles([], null);
    expect(map.fitBounds).toHaveBeenCalledTimes(1);
  });

  it('n’anime pas quand le mouvement réduit est demandé', () => {
    const { map, declencher } = fausseMap();
    const carte = adapterCarte(map, couleurs, () => false);
    declencher('style.load');

    carte.centrer({ lat: 48.1, lon: -1.7 }, 14);
    carte.afficherBoucles([a], 'a');

    expect(map.jumpTo).toHaveBeenCalledWith({ center: [-1.7, 48.1], zoom: 14 });
    expect(map.easeTo).not.toHaveBeenCalled();
    expect(map.fitBounds).toHaveBeenCalledWith(expect.anything(), expect.objectContaining({ duration: 0 }));
  });

  it('anime sinon', () => {
    const { map, declencher } = fausseMap();
    adapterCarte(map, couleurs, () => true).centrer({ lat: 48.1, lon: -1.7 });
    declencher('style.load');
    expect(map.easeTo).toHaveBeenCalledWith({ center: [-1.7, 48.1], zoom: undefined });
  });

  it('marque et retire le départ', () => {
    const { map, sources, declencher } = fausseMap();
    const carte = adapterCarte(map, couleurs, () => true);
    declencher('style.load');

    carte.marquerDepart({ lat: 48.1, lon: -1.7 });
    carte.marquerDepart(null);

    const [avec, sans] = sources.get('depart')!.setData.mock.calls as [{ features: unknown[] }][];
    expect(avec![0].features).toHaveLength(1);
    expect(sans![0].features).toHaveLength(0);
  });

  it('dessine la zone comme un contour fermé, et la retire', () => {
    const { map, sources, declencher } = fausseMap();
    const carte = adapterCarte(map, couleurs, () => true);
    declencher('style.load');

    carte.montrerZone({ minLat: 47, minLon: -5, maxLat: 49, maxLon: -1 });
    const [[avec]] = sources.get('zone')!.setData.mock.calls as [[{ features: { geometry: { coordinates: number[][][] } }[] }]];
    const anneau = avec.features[0]!.geometry.coordinates[0]!;
    expect(anneau).toHaveLength(5);
    expect(anneau[0]).toEqual([-5, 47]);
    expect(anneau[2]).toEqual([-1, 49]);
    expect(anneau[4]).toEqual(anneau[0]);

    carte.montrerZone(null);
    expect(sources.get('zone')!.setData).toHaveBeenLastCalledWith({ type: 'FeatureCollection', features: [] });
  });

  it('rend le centre à chaque fin de déplacement, et se désabonne', () => {
    const { map, declencher, ecouteurs } = fausseMap();
    const carte = adapterCarte(map, couleurs, () => true);
    const rappel = vi.fn();

    const arreter = carte.surDeplacement(rappel);
    declencher('moveend');
    expect(rappel).toHaveBeenCalledWith({ lat: 48.117, lon: -1.677 });

    arreter();
    declencher('moveend');
    expect(rappel).toHaveBeenCalledTimes(1);
    expect(ecouteurs.get('moveend')?.size ?? 0).toBe(0);
  });

  it('relaie redimensionner et détruire', () => {
    const { map } = fausseMap();
    const carte = adapterCarte(map, couleurs, () => true);
    carte.redimensionner();
    carte.detruire();
    expect(map.resize).toHaveBeenCalledOnce();
    expect(map.remove).toHaveBeenCalledOnce();
  });

  it('peint avec les couleurs du moment à chaque pose', () => {
    const { map, declencher } = fausseMap();
    let trace = '#avant';
    adapterCarte(map, () => ({ ...couleurs(), trace }), () => true);
    declencher('style.load');
    trace = '#apres';
    declencher('style.load');

    const peintures = (map.addLayer as ReturnType<typeof vi.fn>).mock.calls
      .map(([spec]) => spec as { id: string; paint: Record<string, unknown> })
      .filter((s) => s.id === 'boucle-selectionnee')
      .map((s) => s.paint['line-color']);
    expect(peintures).toEqual(['#avant', '#apres']);
  });
});

describe('chargement du module', () => {
  it('donne à MapLibre l’URL de son worker, que le bundler ne sait pas résoudre seul', async () => {
    const { getWorkerUrl } = await import('maplibre-gl');
    expect(getWorkerUrl()).toMatch(/maplibre-gl-worker/);
  });
});
