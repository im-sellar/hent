<script lang="ts">
  /**
   * La feuille basse : le `<main>` des écrans qui partagent l'écran avec la
   * carte. Elle prend la place que la carte lui laisse, défile seule, et garde
   * sa lèvre arrondie des maquettes.
   */
  type Props = { children: import('svelte').Snippet };
  let { children }: Props = $props();
</script>

<main class="feuille">
  {@render children()}
</main>

<style>
  .feuille {
    position: relative;
    z-index: 1;
    max-height: 62dvh;
    overflow-y: auto;
    margin-top: -16px;
    padding: 20px 20px 32px;
    display: flex;
    flex-direction: column;
    gap: 16px;
    background: var(--feuille);
    color: var(--texte);
    border-top: 1px solid var(--voie);
    border-radius: 16px 16px 0 0;
    box-shadow: var(--ombre-feuille);
  }
  @media (min-width: 720px) {
    .feuille {
      max-width: 420px;
      margin: -16px auto 0;
      border-radius: 16px 16px 0 0;
    }
  }
</style>
