import { describe, expect, it } from 'vitest';
import { formatEcartCible } from './etat.svelte';

describe('formatEcartCible', () => {
  it('dit la distance demandée quand elle diffère', () => {
    expect(formatEcartCible(18_000, 17_400)).toBe('tu en demandais 18');
  });

  it('ne dit rien quand la distance obtenue est celle demandée', () => {
    // Afficher « tu en demandais 18 » à côté de « 18,0 km » serait du bruit.
    expect(formatEcartCible(18_000, 18_000)).toBe('');
  });

  it('arrondit la demande au kilomètre', () => {
    expect(formatEcartCible(18_400, 17_000)).toBe('tu en demandais 18');
  });

  it('ne dit rien quand la demande est inconnue', () => {
    expect(formatEcartCible(0, 17_400)).toBe('');
  });
});
