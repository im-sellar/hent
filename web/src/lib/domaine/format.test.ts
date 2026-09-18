import { describe, expect, it } from 'vitest';
import { formatDistance, formatDuree, formatEcartCible, formatKilometres, formatPourcent } from './format';

describe('formatDistance', () => {
  it('rend des kilomètres avec une décimale et une virgule', () => {
    expect(formatDistance(17_400)).toBe('17,4 km');
  });

  it('garde la décimale même quand elle est nulle', () => {
    // « 18 km » et « 18,0 km » ne se ressemblent pas dans une liste alignée.
    expect(formatDistance(18_000)).toBe('18,0 km');
  });

  it('rend des mètres sous le kilomètre', () => {
    expect(formatDistance(850)).toBe('850 m');
  });

  it('rend zéro sans unité fantaisiste', () => {
    expect(formatDistance(0)).toBe('0 m');
  });

  it('bascule au kilomètre pile, pas un mètre après', () => {
    expect(formatDistance(999)).toBe('999 m');
    expect(formatDistance(1000)).toBe('1,0 km');
  });
});

describe('formatDuree', () => {
  it('sépare heures et minutes', () => {
    expect(formatDuree(130)).toBe('2 h 10');
  });

  it('complète les minutes à deux chiffres', () => {
    // « 2 h 5 » se lit mal à côté de « 2 h 10 ».
    expect(formatDuree(125)).toBe('2 h 05');
  });

  it('omet les heures en dessous de soixante minutes', () => {
    expect(formatDuree(45)).toBe('45 min');
  });

  it('bascule à l’heure pile, pas une minute après', () => {
    expect(formatDuree(59)).toBe('59 min');
    expect(formatDuree(60)).toBe('1 h 00');
  });
});

describe('formatPourcent', () => {
  it('rend une part en pourcentage entier', () => {
    expect(formatPourcent(0.55)).toBe('55 %');
  });

  it('arrondit au plus proche', () => {
    expect(formatPourcent(0.238)).toBe('24 %');
  });

  it('distingue un vrai zéro d’une valeur infime', () => {
    // 0,4 % de tracé refait n'est pas la même chose que zéro : arrondir à
    // l'entier effacerait l'information que l'écran de détail affiche.
    expect(formatPourcent(0)).toBe('0 %');
    expect(formatPourcent(0.004)).toBe('0,4 %');
  });
});

describe('formatEcartCible', () => {
  it('dit la distance demandée quand elle diffère', () => {
    expect(formatEcartCible(18_000, 17_400)).toBe('tu en demandais 18');
  });

  it('ne dit rien quand la distance obtenue est celle demandée', () => {
    // Afficher « tu en demandais 18 » à côté de « 18,0 km » serait du bruit.
    expect(formatEcartCible(18_000, 18_000)).toBe('');
  });

  it('arrondit la demande au kilomètre', () => {
    // 18 600 arrondit à 19, pas à 18 : un test qui s'arrêterait à 18 400
    // (que l'arrondi et la troncature rendent identiquement) ne prouverait
    // rien de l'arrondi.
    expect(formatEcartCible(18_600, 17_000)).toBe('tu en demandais 19');
  });

  it('ne dit rien quand la demande est inconnue', () => {
    expect(formatEcartCible(0, 17_400)).toBe('');
  });

  it('ne dit rien quand la demande n’est pas un nombre', () => {
    expect(formatEcartCible(Number.NaN, 17_400)).toBe('');
  });

  it('tait un écart réellement dans le seuil des 500 m', () => {
    expect(formatEcartCible(18_000, 18_400)).toBe('');
  });

  it('ne tait pas un écart de 500 m ou plus, même arrondi au même kilomètre', () => {
    // 17 600 et 18 400 arrondissent tous deux à 18 km, mais les séparent
    // 800 m : le seuil ne doit pas confondre un même arrondi avec un même
    // point.
    expect(formatEcartCible(17_600, 18_400)).toBe('tu en demandais 18');
  });

  it('place le seuil des 500 m sur la borne, pas à côté', () => {
    // Les deux paires arrondissent au même kilomètre : seul l'écart les
    // sépare, et il vaut 499 m d'un côté, 500 m de l'autre.
    expect(formatEcartCible(18_400, 17_901)).toBe('');
    expect(formatEcartCible(18_400, 17_900)).toBe('tu en demandais 18');
  });

  it('arrondit aussi la distance obtenue, pas seulement la demande', () => {
    // 17 600 arrondit à 18, pas à 17 : un test qui tronquerait la distance
    // obtenue au lieu de l'arrondir la confondrait avec la demande à tort.
    expect(formatEcartCible(18_000, 17_600)).toBe('');
  });
});

describe('formatKilometres', () => {
  it('arrondit au kilomètre entier', () => {
    expect(formatKilometres(18_000)).toBe('18 km');
    expect(formatKilometres(17_400)).toBe('17 km');
    expect(formatKilometres(17_500)).toBe('18 km');
  });

  it('rend un tiret pour une valeur non finie plutôt qu’un « NaN km »', () => {
    expect(formatKilometres(NaN)).toBe('— km');
  });
});
