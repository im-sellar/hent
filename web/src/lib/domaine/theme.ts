/** Ce que la personne choisit. `auto` suit le réglage du système. */
export type Theme = 'auto' | 'sombre' | 'clair';

/** Ce qui est réellement affiché : le style de carte et les jetons CSS n'en connaissent que deux. */
export type ThemeEffectif = 'sombre' | 'clair';

export const THEMES: readonly Theme[] = ['auto', 'sombre', 'clair'];

export function estTheme(v: unknown): v is Theme {
  return typeof v === 'string' && (THEMES as readonly string[]).includes(v);
}

export function themeEffectif(t: Theme, systemePrefereClair: boolean): ThemeEffectif {
  if (t === 'auto') return systemePrefereClair ? 'clair' : 'sombre';
  return t;
}
