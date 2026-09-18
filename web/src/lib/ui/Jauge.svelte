<script lang="ts">
  type Props = { etiquette: string; part: number; texte: string; alerte?: boolean };
  let { etiquette, part, texte, alerte = false }: Props = $props();
</script>

<div>
  <div class="ligne">
    <span class="etiquette">{etiquette}</span>
    <span class="valeur">{texte}</span>
  </div>
  <div class="piste">
    <div class="remplissage" class:alerte style="width: {Math.min(100, Math.max(0, part * 100))}%"></div>
  </div>
</div>

<style>
  .ligne {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    margin-bottom: 6px;
  }
  .etiquette {
    font-size: 0.875rem;
    color: var(--texte-doux);
  }
  .valeur {
    font-family: Spectral, Georgia, serif;
    font-size: 1.0625rem;
  }
  .piste {
    height: 6px;
    background: var(--trait);
    border-radius: 3px;
    overflow: hidden;
  }
  .remplissage {
    height: 100%;
    min-width: 4px;
    background: var(--accent);
  }
  .alerte {
    background: var(--alerte);
  }
</style>
