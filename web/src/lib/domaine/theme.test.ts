import { describe, expect, it } from 'vitest';
import { THEMES, estTheme, themeEffectif } from './theme';

describe('estTheme', () => {
  it('reconnaît les trois valeurs et rien d’autre', () => {
    for (const t of THEMES) expect(estTheme(t)).toBe(true);
    expect(estTheme('nuit')).toBe(false);
    expect(estTheme(null)).toBe(false);
    expect(estTheme(1)).toBe(false);
  });
});

describe('themeEffectif', () => {
  it('suit le système en automatique', () => {
    expect(themeEffectif('auto', true)).toBe('clair');
    expect(themeEffectif('auto', false)).toBe('sombre');
  });

  it('ignore le système quand un thème est choisi', () => {
    expect(themeEffectif('sombre', true)).toBe('sombre');
    expect(themeEffectif('clair', false)).toBe('clair');
  });
});
