import type { Preferences } from '$lib/app/ports';
import { borner, type Reglages } from '$lib/domaine/reglages';

const CLE = 'hent.reglages';

/**
 * Préférences adossées au stockage du navigateur.
 *
 * Toute opération est protégée : le stockage peut être absent (rendu hors
 * navigateur), refusé (navigation privée, politique de site) ou plein. Aucun de
 * ces cas ne doit empêcher l'application de démarrer, donc `lire` rend `null` et
 * `ecrire` ne fait rien plutôt que de lever.
 *
 * Ce qui est relu est borné : une valeur écrite par une version antérieure de
 * l'application, ou modifiée à la main dans les outils du navigateur, ne traverse
 * l'application telle quelle. Elle subit un contrôle de plage.
 */
export function creerPreferences(
  stockage: Storage | null = typeof localStorage === 'undefined' ? null : localStorage
): Preferences {
  return {
    lire(): Reglages | null {
      if (!stockage) return null;
      let brut: string | null;
      try {
        brut = stockage.getItem(CLE);
      } catch {
        // Stockage refusé (navigation privée, politique de site, quota) : démarrer
        // sur les valeurs par défaut plutôt que de planter.
        return null;
      }
      if (!brut) return null;
      try {
        const lu = JSON.parse(brut) as Partial<Reglages>;
        if (typeof lu?.distanceM !== 'number' || typeof lu?.eviterBitume !== 'number') return null;
        return borner({ distanceM: lu.distanceM, eviterBitume: lu.eviterBitume });
      } catch {
        // JSON malformé ou structure invalide : valeur corrompue dans le stockage,
        // on repart sur les valeurs par défaut.
        return null;
      }
    },

    ecrire(r: Reglages): void {
      if (!stockage) return;
      try {
        stockage.setItem(CLE, JSON.stringify(borner(r)));
      } catch {
        // Quota atteint ou écriture refusée : la préférence ne survivra pas à la
        // session, ce qui est préférable à une application qui s'arrête.
      }
    }
  };
}
