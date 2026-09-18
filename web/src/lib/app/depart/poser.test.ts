import { describe, expect, it } from 'vitest';
import { nommerPoint } from './poser';
import type { Geocodeur } from '$lib/app/ports';

function geocodeurQuiNomme(nommer: Geocodeur['nommer']): Geocodeur {
  return { chercher: async () => [], nommer };
}

const point = { lat: 48.117, lon: -1.677 };

describe('nommerPoint', () => {
  it('donne au point le nom que rend le géocodeur', async () => {
    const depart = await nommerPoint(geocodeurQuiNomme(async () => '2 Rue Lesage 35000 Rennes'), point);
    expect(depart).toEqual({ coord: point, libelle: '2 Rue Lesage 35000 Rennes' });
  });

  it('se replie sur les coordonnées sans nom ou sur une panne', async () => {
    expect((await nommerPoint(geocodeurQuiNomme(async () => null), point)).libelle).toBe('48,1170, -1,6770');
    const panne = geocodeurQuiNomme(async () => {
      throw new Error('réseau');
    });
    expect((await nommerPoint(panne, point)).libelle).toBe('48,1170, -1,6770');
  });

  it('relance une annulation : un départ annulé ne doit pas être posé', async () => {
    const annule = geocodeurQuiNomme(async () => {
      throw new DOMException('annulé', 'AbortError');
    });
    await expect(nommerPoint(annule, point)).rejects.toMatchObject({ name: 'AbortError' });
  });

  it('transmet le signal', async () => {
    let recu: AbortSignal | undefined;
    const controleur = new AbortController();
    await nommerPoint(geocodeurQuiNomme(async (_p, signal) => {
      recu = signal;
      return null;
    }), point, controleur.signal);
    expect(recu).toBe(controleur.signal);
  });
});
