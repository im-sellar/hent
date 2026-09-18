import type { Position, ResultatPosition } from '$lib/app/ports';

/** Code de `GeolocationPositionError` pour un refus de permission. */
const PERMISSION_REFUSEE = 1;

/**
 * Position adossée à `navigator.geolocation`. Sans l'API — rendu hors
 * navigateur, contexte non sécurisé — la position est indisponible, pas en
 * erreur : l'écran propose alors la recherche et la carte.
 *
 * La précision grossière suffit à centrer une carte, et elle répond bien plus
 * vite qu'un GPS à froid. Une position d'une minute est acceptée pour la même
 * raison.
 */
export function creerPosition(
  geo: Geolocation | undefined = typeof navigator === 'undefined' ? undefined : navigator.geolocation
): Position {
  return {
    obtenir(): Promise<ResultatPosition> {
      return new Promise((resoudre) => {
        if (!geo) return resoudre({ statut: 'indisponible' });
        geo.getCurrentPosition(
          (p) => resoudre({ statut: 'ok', coord: { lat: p.coords.latitude, lon: p.coords.longitude } }),
          (e) => resoudre({ statut: e.code === PERMISSION_REFUSEE ? 'refusee' : 'indisponible' }),
          { enableHighAccuracy: false, timeout: 10_000, maximumAge: 60_000 }
        );
      });
    }
  };
}
