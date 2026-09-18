import { describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/svelte';
import SelecteurTheme from './SelecteurTheme.svelte';

describe('SelecteurTheme', () => {
  it('offre trois boutons radio nommés, celui du thème courant coché', () => {
    render(SelecteurTheme, { valeur: 'clair', onchange: () => {} });

    const radios = screen.getAllByRole('radio');
    expect(radios.map((r) => (r as HTMLInputElement).value)).toEqual(['auto', 'sombre', 'clair']);
    expect(screen.getByRole('radio', { name: 'Clair' })).toHaveProperty('checked', true);
    expect(screen.getByRole('radio', { name: 'Automatique' })).toHaveProperty('checked', false);
    expect(screen.getByRole('group', { name: 'Thème' })).toBeDefined();
  });

  it('annonce le choix', async () => {
    const onchange = vi.fn();
    render(SelecteurTheme, { valeur: 'auto', onchange });

    await fireEvent.click(screen.getByRole('radio', { name: 'Sombre' }));

    expect(onchange).toHaveBeenCalledWith('sombre');
    expect(onchange).toHaveBeenCalledOnce();
  });
});
