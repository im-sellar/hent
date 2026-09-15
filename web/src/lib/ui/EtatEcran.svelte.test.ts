import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import EtatEcran from './EtatEcran.svelte';

afterEach(() => {
  vi.useRealTimers();
});

describe('EtatEcran', () => {
  it('tient ses régions prêtes avant d’avoir quelque chose à dire', () => {
    // Une région live créée en même temps que son texte n'est pas annoncée :
    // les deux doivent être montées dès le départ, vides.
    render(EtatEcran, {});

    expect(screen.getByRole('status').textContent?.trim()).toBe('');
    expect(screen.getByRole('alert').textContent?.trim()).toBe('');
  });

  it('prend le focus sur la région qui parle, quand on le lui demande', async () => {
    render(EtatEcran, {
      erreur: { genre: 'Serveur', message: 'boum' },
      onreessayer: () => {},
      prendLeFocus: true
    });

    expect(document.activeElement).toBe(await screen.findByRole('alert'));
  });

  it('laisse le focus où il est par défaut', () => {
    render(EtatEcran, { erreur: { genre: 'Serveur', message: 'boum' }, onreessayer: () => {} });

    expect(document.activeElement).toBe(document.body);
  });

  it('attend le délai annoncé avant d’offrir le réessai', async () => {
    vi.useFakeTimers();
    const reessayer = vi.fn();
    render(EtatEcran, {
      erreur: { genre: 'TropDeDemandes', message: 'trop vite', reessayerDansS: 2 },
      onreessayer: reessayer
    });

    expect(screen.getByText('Réessayer dans 2 s')).toBeDefined();
    expect(screen.queryByRole('button', { name: 'Réessayer' })).toBeNull();

    await vi.advanceTimersByTimeAsync(1000);
    expect(screen.getByText('Réessayer dans 1 s')).toBeDefined();
    expect(screen.queryByRole('button', { name: 'Réessayer' })).toBeNull();

    await vi.advanceTimersByTimeAsync(1000);
    const bouton = screen.getByRole('button', { name: 'Réessayer' });
    bouton.click();
    expect(reessayer).toHaveBeenCalledOnce();
  });

  it('offre le réessai tout de suite quand aucun délai n’est annoncé', () => {
    render(EtatEcran, { erreur: { genre: 'Serveur', message: 'boum' }, onreessayer: () => {} });

    expect(screen.getByRole('button', { name: 'Réessayer' })).toBeDefined();
  });
});
