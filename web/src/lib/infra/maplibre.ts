/// <reference types="geojson" />
// Les déclarations de maplibre-gl utilisent le namespace global GeoJSON sans
// l'importer : ce paquet le fournit déjà (dépendance de maplibre-gl) mais
// n'est pas inclus automatiquement par ce tsconfig.
import { Map as MaplibreMap } from 'maplibre-gl';
import 'maplibre-gl/dist/maplibre-gl.css';
import type { Carte } from '$lib/app/ports';
import { bornesDe, type Boucle } from '$lib/domaine/boucle';
import type { Coord } from '$lib/domaine/depart';
import type { Zone } from '$lib/domaine/zone';
import { couleursDepuisJetons, type Couleurs } from './styles-carte';

export { couleursDepuisJetons, urlStyle, webglDisponible } from './styles-carte';
export type { Couleurs } from './styles-carte';

type Position2D = [number, number];
type Trait<G, P> = { type: 'Feature'; properties: P; geometry: G };
type Ligne = { type: 'LineString'; coordinates: Position2D[] };
type Point = { type: 'Point'; coordinates: Position2D };
type Polygone = { type: 'Polygon'; coordinates: Position2D[][] };
export type Collection<F = Trait<Ligne | Point | Polygone, Record<string, unknown>>> = {
  type: 'FeatureCollection';
  features: F[];
};

const VIDE: Collection = { type: 'FeatureCollection', features: [] };

/**
 * Ce que l'adaptateur emploie de `maplibregl.Map`, et rien de plus. Nommer ce
 * sous-ensemble permet de brancher un double dans les tests : jsdom n'a pas de
 * WebGL, la vraie carte n'y démarre pas.
 */
export type MapLike = {
  on(evenement: string, ecouteur: (...args: unknown[]) => void): unknown;
  off(evenement: string, ecouteur: (...args: unknown[]) => void): unknown;
  getSource(id: string): { setData(donnees: Collection): unknown } | undefined;
  addSource(id: string, spec: { type: 'geojson'; data: Collection }): unknown;
  addLayer(spec: Record<string, unknown> & { id: string }): unknown;
  easeTo(options: { center: Position2D; zoom?: number }): unknown;
  jumpTo(options: { center: Position2D; zoom?: number }): unknown;
  fitBounds(bornes: [Position2D, Position2D], options?: { padding?: number; duration?: number }): unknown;
  setStyle(url: string): unknown;
  resize(): unknown;
  remove(): unknown;
  getCenter(): { lng: number; lat: number };
};

/**
 * Traduit des boucles en tracés GeoJSON, en marquant celle qui est choisie.
 *
 * `selectionnee` est un booléen strict : les filtres de couches
 * `['get', 'selectionnee']` et `['!', ['get', 'selectionnee']]` en dépendent,
 * et une valeur non booléenne ferait disparaître le trait des deux côtés.
 */
export function versCollection(
  boucles: Boucle[],
  selectionnee: string | null
): Collection<Trait<Ligne, { id: string; selectionnee: boolean }>> {
  return {
    type: 'FeatureCollection',
    features: boucles.map((b) => ({
      type: 'Feature',
      properties: { id: b.id, selectionnee: b.id === selectionnee },
      geometry: { type: 'LineString', coordinates: b.geometrie }
    }))
  };
}

function pointDe(c: Coord | null): Collection {
  if (!c) return VIDE;
  return {
    type: 'FeatureCollection',
    features: [{ type: 'Feature', properties: {}, geometry: { type: 'Point', coordinates: [c.lon, c.lat] } }]
  };
}

function contourDe(z: Zone | null): Collection {
  if (!z) return VIDE;
  const anneau: Position2D[] = [
    [z.minLon, z.minLat],
    [z.maxLon, z.minLat],
    [z.maxLon, z.maxLat],
    [z.minLon, z.maxLat],
    [z.minLon, z.minLat]
  ];
  return { type: 'FeatureCollection', features: [{ type: 'Feature', properties: {}, geometry: { type: 'Polygon', coordinates: [anneau] } }] };
}

const ARRONDI = { 'line-cap': 'round', 'line-join': 'round' };

/**
 * Adapte une carte MapLibre au port `Carte`.
 *
 * Tout ce qui est affiché est gardé en mémoire et reposé à chaque
 * `style.load` : changer de style efface sources et couches, et le premier
 * chargement passe par le même événement. `couleurs` est relu à chaque pose,
 * parce que les jetons CSS changent avec le thème. `animer` dit si les
 * mouvements de caméra peuvent être animés (`prefers-reduced-motion`).
 */
export function adapterCarte(map: MapLike, couleurs: () => Couleurs, animer: () => boolean): Carte {
  let boucles: Boucle[] = [];
  let selectionnee: string | null = null;
  let depart: Coord | null = null;
  let zone: Zone | null = null;
  let idsCadres = '';

  function poser() {
    const c = couleurs();
    map.addSource('zone', { type: 'geojson', data: contourDe(zone) });
    map.addLayer({
      id: 'zone', type: 'line', source: 'zone', layout: ARRONDI,
      paint: { 'line-color': c.zone, 'line-width': 2, 'line-dasharray': [3, 3] }
    });
    map.addSource('boucles', { type: 'geojson', data: versCollection(boucles, selectionnee) });
    map.addLayer({
      id: 'boucles-ecartees', type: 'line', source: 'boucles', layout: ARRONDI,
      filter: ['!', ['get', 'selectionnee']],
      paint: { 'line-color': c.ecartee, 'line-width': 3, 'line-dasharray': [2, 2] }
    });
    map.addLayer({
      id: 'boucle-selectionnee', type: 'line', source: 'boucles', layout: ARRONDI,
      filter: ['get', 'selectionnee'],
      paint: { 'line-color': c.trace, 'line-width': 4 }
    });
    map.addSource('depart', { type: 'geojson', data: pointDe(depart) });
    map.addLayer({
      id: 'depart', type: 'circle', source: 'depart',
      paint: { 'circle-radius': 7, 'circle-color': c.depart, 'circle-stroke-width': 2, 'circle-stroke-color': c.contourDepart }
    });
  }

  map.on('style.load', poser);

  function donnees(id: string, d: Collection) {
    map.getSource(id)?.setData(d);
  }

  return {
    centrer(point, zoom) {
      const options = { center: [point.lon, point.lat] as Position2D, zoom };
      if (animer()) map.easeTo(options);
      else map.jumpTo(options);
    },

    afficherBoucles(nouvelles, sel) {
      boucles = nouvelles;
      selectionnee = sel;
      donnees('boucles', versCollection(boucles, selectionnee));
      const ids = boucles.map((b) => b.id).join('|');
      if (ids === idsCadres) return;
      idsCadres = ids;
      const bornes = bornesDe(boucles);
      if (bornes) map.fitBounds(bornes, { padding: 48, duration: animer() ? 600 : 0 });
    },

    marquerDepart(point) {
      depart = point;
      donnees('depart', pointDe(depart));
    },

    montrerZone(z) {
      zone = z;
      donnees('zone', contourDe(zone));
    },

    surDeplacement(rappel) {
      const ecouteur = () => {
        const c = map.getCenter();
        rappel({ lat: c.lat, lon: c.lng });
      };
      map.on('moveend', ecouteur);
      return () => map.off('moveend', ecouteur);
    },

    changerStyle(url) {
      map.setStyle(url);
    },

    redimensionner() {
      map.resize();
    },

    detruire() {
      map.remove();
    }
  };
}

/**
 * Crée la vraie carte dans `conteneur`. L'attribution reste dépliée : « © IGN »
 * doit être visible, pas caché derrière un bouton.
 */
export function creerCarte(conteneur: HTMLElement, styleUrl: string, centre: Coord, zoom: number): Carte {
  const map = new MaplibreMap({
    container: conteneur,
    style: styleUrl,
    center: [centre.lon, centre.lat],
    zoom,
    attributionControl: { compact: false }
  });
  const animer = () => !matchMedia('(prefers-reduced-motion: reduce)').matches;
  return adapterCarte(map as unknown as MapLike, couleursDepuisJetons, animer);
}
