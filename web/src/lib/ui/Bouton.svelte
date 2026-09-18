<script lang="ts">
  type Props = {
    variante?: 'primaire' | 'secondaire';
    href?: string;
    onclick?: () => void;
    children: import('svelte').Snippet;
  };
  let { variante = 'primaire', href, onclick, children }: Props = $props();
</script>

{#if href}
  <a class="bouton {variante}" {href}>{@render children()}</a>
{:else}
  <button class="bouton {variante}" type="button" {onclick}>{@render children()}</button>
{/if}

<style>
  .bouton {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 52px;
    padding: 0 22px;
    border-radius: 4px;
    border: none;
    font: inherit;
    font-weight: 700;
    font-size: 1.0625rem;
    text-decoration: none;
    cursor: pointer;
  }
  .primaire {
    background: var(--accent);
    color: var(--sur-accent);
  }
  .secondaire {
    background: transparent;
    color: var(--accent);
    border: 1px solid var(--trait-vif);
  }
  .bouton:focus-visible {
    outline: 2px solid var(--accent-vif);
    outline-offset: 2px;
  }
</style>
