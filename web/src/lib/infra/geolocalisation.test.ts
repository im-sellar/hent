import { describe, expect, it } from 'vitest';
import { creerPosition } from './geolocalisation';

/** Un `navigator.geolocation` doublé qui répond comme on le lui dit. */
function geoQui(reponse: { coords: { latitude: number; longitude: number } } | { code: number }): Geolocation {
  return {
    getCurrentPosition(succes: PositionCallback, echec?: PositionErrorCallback) {
      if ('coords' in reponse) succes(reponse as GeolocationPosition);
      else echec?.(reponse as GeolocationPositionError);
    }
  } as unknown as Geolocation;
}

describe('obtenir', () => {
  it('rend la position accordée, latitude et longitude à leur place', async () => {
    const position = creerPosition(geoQui({ coords: { latitude: 48.117, longitude: -1.677 } }));

    expect(await position.obtenir()).toEqual({ statut: 'ok', coord: { lat: 48.117, lon: -1.677 } });
  });

  it('distingue le refus de permission', async () => {
    expect(await creerPosition(geoQui({ code: 1 })).obtenir()).toEqual({ statut: 'refusee' });
  });

  it('range les autres échecs en indisponible', async () => {
    expect(await creerPosition(geoQui({ code: 2 })).obtenir()).toEqual({ statut: 'indisponible' });
    expect(await creerPosition(geoQui({ code: 3 })).obtenir()).toEqual({ statut: 'indisponible' });
  });

  it('est indisponible sans API de géolocalisation', async () => {
    expect(await creerPosition(undefined).obtenir()).toEqual({ statut: 'indisponible' });
  });

  it('demande une précision modeste et n’attend pas plus de dix secondes', async () => {
    let options: PositionOptions | undefined;
    const geo = {
      getCurrentPosition(_s: unknown, _e: unknown, o?: PositionOptions) {
        options = o;
      }
    } as unknown as Geolocation;

    void creerPosition(geo).obtenir();

    expect(options?.enableHighAccuracy).toBe(false);
    expect(options?.timeout).toBe(10_000);
  });
});
