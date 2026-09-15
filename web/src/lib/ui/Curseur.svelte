<script lang="ts">
  type Props = {
    etiquette: string;
    valeur: number;
    min: number;
    max: number;
    pas?: number;
    /** Valeur lue à voix haute, quand un nombre nu ne veut rien dire. */
    texteValeur: string;
    onchange: (v: number) => void;
  };
  let { etiquette, valeur, min, max, pas = 1, texteValeur, onchange }: Props = $props();
</script>

<div class="reglage">
  <div class="ligne">
    <label for={etiquette}>{etiquette}</label>
    <span class="valeur">{texteValeur}</span>
  </div>
  <input
    id={etiquette}
    type="range"
    {min}
    {max}
    step={pas}
    value={valeur}
    aria-valuetext={texteValeur}
    oninput={(e) => onchange(Number(e.currentTarget.value))}
  />
</div>

<style>
  .ligne {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    margin-bottom: 11px;
  }
  label {
    font-size: 0.875rem;
    color: var(--texte-gris);
    letter-spacing: 0.04em;
  }
  .valeur {
    font-family: Spectral, Georgia, serif;
    font-size: 1.5625rem;
  }
  input[type='range'] {
    width: 100%;
    accent-color: var(--accent-vif);
  }
  input:focus-visible {
    outline: 2px solid var(--accent-vif);
    outline-offset: 2px;
  }
</style>
