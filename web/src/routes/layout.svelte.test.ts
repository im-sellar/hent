import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render } from '@testing-library/svelte';
import { tick } from 'svelte';

const faux = vi.hoisted(() => ({
  routeId: '/' as string | null,
  carte: { redimensionner: vi.fn(), detruire: vi.fn() },
  appliquerTheme: vi.fn(),
  monterCarte: vi.fn(),
  demonterCarte: vi.fn(),
  theme: 'auto' as string
}));

vi.mock('$app/state', () => ({
  page: {
    get route() {
      return { id: faux.routeId };
    }
  }
}));

vi.mock('$lib/assemblage.svelte', () => ({
  appEtat: {
    get carte() {
      return faux.carte;
    },
    get theme() {
      return faux.theme;
    },
    appliquerTheme: faux.appliquerTheme,
    monterCarte: faux.monterCarte,
    demonterCarte: faux.demonterCarte
  }
}));

const Layout = (await import('./+layout.svelte')).default;

beforeEach(() => {
  faux.routeId = '/';
  faux.theme = 'auto';
  faux.carte.redimensionner.mockClear();
  faux.appliquerTheme.mockClear();
  faux.monterCarte.mockClear();
  faux.demonterCarte.mockClear();
});

describe('layout', () => {
  it('applique le thème puis monte la carte dans son conteneur, une fois', async () => {
    const { container } = render(Layout);
    await tick();

    expect(faux.appliquerTheme).toHaveBeenCalledOnce();
    expect(faux.monterCarte).toHaveBeenCalledOnce();
    const conteneur = faux.monterCarte.mock.calls[0]![0] as HTMLElement;
    expect(container.contains(conteneur)).toBe(true);
    expect(faux.appliquerTheme.mock.invocationCallOrder[0]).toBeLessThan(faux.monterCarte.mock.invocationCallOrder[0]!);
  });

  it('cache la carte sur l’accueil et le réglage, la montre sur le départ, les boucles et le détail', async () => {
    for (const [route, visible] of [['/', false], ['/reglage', false], ['/depart', true], ['/boucles', true], ['/b/[id]', true]] as const) {
      faux.routeId = route;
      const { container, unmount } = render(Layout);
      await tick();
      const carte = container.querySelector('.carte') as HTMLElement;
      expect(carte, route).not.toBeNull();
      expect(carte.hidden, route).toBe(!visible);
      expect(carte.getAttribute('aria-hidden'), route).toBe('true');
      unmount();
    }
  });

  it('redimensionne la carte quand un écran la montre', async () => {
    faux.routeId = '/depart';
    render(Layout);
    await tick();
    await tick();

    expect(faux.carte.redimensionner).toHaveBeenCalled();
  });

  it('ne redimensionne pas une carte cachée', async () => {
    faux.routeId = '/';
    render(Layout);
    await tick();
    await tick();

    expect(faux.carte.redimensionner).not.toHaveBeenCalled();
  });

  it('montre le réticule sur l’écran de départ seulement', async () => {
    faux.routeId = '/depart';
    const avec = render(Layout);
    expect(avec.container.querySelector('.reticule')).not.toBeNull();
    avec.unmount();

    faux.routeId = '/boucles';
    const sans = render(Layout);
    expect(sans.container.querySelector('.reticule')).toBeNull();
  });

  it('suit le système en automatique, et seulement en automatique', async () => {
    const ecouteurs: ((e: unknown) => void)[] = [];
    vi.spyOn(window, 'matchMedia').mockImplementation(
      (requete: string) =>
        ({
          matches: false,
          media: requete,
          addEventListener: (_: string, fn: (e: unknown) => void) => void ecouteurs.push(fn),
          removeEventListener: vi.fn()
        }) as unknown as MediaQueryList
    );
    render(Layout);
    await tick();
    faux.appliquerTheme.mockClear();

    ecouteurs.forEach((fn) => fn({}));
    expect(faux.appliquerTheme).toHaveBeenCalledOnce();

    faux.theme = 'sombre';
    ecouteurs.forEach((fn) => fn({}));
    expect(faux.appliquerTheme).toHaveBeenCalledOnce();
    vi.restoreAllMocks();
  });

  it('détruit la carte au démontage', async () => {
    const { unmount } = render(Layout);
    await tick();
    unmount();
    expect(faux.demonterCarte).toHaveBeenCalledOnce();
  });
});
