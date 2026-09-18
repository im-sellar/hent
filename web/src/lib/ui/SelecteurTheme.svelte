<script lang="ts">
  import { THEMES, type Theme } from '$lib/domaine/theme';

  /**
   * Trois boutons radio dans un groupe nommé. Des radios plutôt qu'un
   * interrupteur : trois états ne se basculent pas, ils se choisissent.
   */
  type Props = { valeur: Theme; onchange: (t: Theme) => void };
  let { valeur, onchange }: Props = $props();

  const noms: Record<Theme, string> = { auto: 'Automatique', sombre: 'Sombre', clair: 'Clair' };
</script>

<fieldset>
  <legend>Thème</legend>
  <div class="choix">
    {#each THEMES as theme (theme)}
      <label>
        <input type="radio" name="theme" value={theme} checked={valeur === theme} onchange={() => onchange(theme)} />
        <span>{noms[theme]}</span>
      </label>
    {/each}
  </div>
</fieldset>

<style>
  fieldset {
    border: 1px solid var(--trait);
    border-radius: 4px;
    padding: 13px;
  }
  legend {
    font-size: 0.875rem;
    color: var(--texte-gris);
  }
  .choix {
    display: flex;
    gap: 6px;
  }
  label {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 44px;
    border: 1px solid var(--trait-vif);
    border-radius: 4px;
    font-size: 0.9375rem;
    cursor: pointer;
  }
  label:has(input:checked) {
    background: var(--accent);
    color: var(--sur-accent);
    border-color: var(--accent);
  }
  input {
    position: absolute;
    width: 1px;
    height: 1px;
    margin: -1px;
    opacity: 0;
  }
  label:has(input:focus-visible) {
    outline: 2px solid var(--accent-vif);
    outline-offset: 2px;
  }
</style>
