<script lang="ts">
  import { onMount, tick } from 'svelte';
  import { page } from '$app/state';
  import { appEtat } from '$lib/assemblage.svelte';
  import '../styles/jetons.css';

  let { children } = $props();
  let conteneur = $state<HTMLDivElement | null>(null);

  const ROUTES_AVEC_CARTE = ['/depart', '/boucles', '/b/[id]'];
  const ROUTE_RETICULE: string = '/depart';
  const avecCarte = $derived(ROUTES_AVEC_CARTE.includes(page.route.id ?? ''));
  const avecReticule = $derived(page.route.id === ROUTE_RETICULE);

  onMount(() => {
    appEtat.appliquerTheme();
    if (conteneur) appEtat.monterCarte(conteneur);

    const systeme = matchMedia('(prefers-color-scheme: light)');
    const suivre = () => {
      if (appEtat.theme === 'auto') appEtat.appliquerTheme();
    };
    systeme.addEventListener('change', suivre);
    return () => {
      systeme.removeEventListener('change', suivre);
      appEtat.demonterCarte();
    };
  });

  // Un conteneur masqué a une taille nulle pour MapLibre : à chaque fois qu'un
  // écran remontre la carte, elle doit reprendre ses mesures.
  $effect(() => {
    if (!avecCarte) return;
    void tick().then(() => appEtat.carte?.redimensionner());
  });
</script>

<div class="ecran">
  <div class="carte" hidden={!avecCarte}>
    <div class="toile" bind:this={conteneur}></div>
    {#if avecReticule}
      <div class="reticule"></div>
    {/if}
  </div>
  {@render children?.()}
</div>

<style>
  :global(body) {
    margin: 0;
    background: var(--fond);
  }
  .ecran {
    display: flex;
    flex-direction: column;
    min-height: 100dvh;
  }
  .carte {
    position: relative;
    flex: 1 1 auto;
    min-height: 38dvh;
    background: var(--carte);
  }
  .carte[hidden] {
    display: none;
  }
  .toile {
    position: absolute;
    inset: 0;
  }
  .reticule {
    position: absolute;
    left: 50%;
    top: 50%;
    width: 22px;
    height: 22px;
    margin: -11px 0 0 -11px;
    border: 2px solid var(--accent-vif);
    border-radius: 50%;
    box-shadow: 0 0 0 2px var(--fond);
    pointer-events: none;
  }
  .reticule::after {
    content: '';
    position: absolute;
    left: 50%;
    top: 50%;
    width: 4px;
    height: 4px;
    margin: -2px 0 0 -2px;
    background: var(--accent-vif);
    border-radius: 50%;
  }
</style>
