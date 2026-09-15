<script lang="ts">
  import { appEtat } from '$lib/assemblage.svelte';
  import { dureeMinutes } from '$lib/domaine/boucle';
  import { formatDistance, formatDuree, formatPourcent } from '$lib/domaine/format';

  const etat = $derived(appEtat.resultats.etat());
</script>

<svelte:head><title>Les boucles — hent</title></svelte:head>

<main>
  <a class="retour" href="/reglage">← Changer les réglages</a>

  {#if etat.statut === 'ok'}
    <h1>{etat.boucles.length === 1 ? 'Une boucle' : `${etat.boucles.length} boucles`}</h1>
    <p class="tri">la plus verte d’abord</p>
    <ul>
      {#each etat.boucles as boucle (boucle.id)}
        <li>
          <a href="/b/{encodeURIComponent(boucle.id)}">
            <span class="distance">{formatDistance(boucle.score.distanceM)}</span>
            <span class="mesures">
              {formatPourcent(boucle.score.partNonBitume)} hors bitume ·
              {formatDuree(dureeMinutes(boucle.score.distanceM, 8))}
            </span>
          </a>
        </li>
      {/each}
    </ul>
  {:else}
    <h1>Les boucles</h1>
    <p class="vide">Aucune recherche en cours. <a href="/reglage">Régler une boucle</a></p>
  {/if}
</main>

<style>
  main {
    max-width: 390px;
    margin: 0 auto;
    padding: 24px 20px 40px;
    background: var(--fond);
    color: var(--texte);
    min-height: 100vh;
  }
  h1 {
    font-family: Spectral, Georgia, serif;
    font-size: 1.5625rem;
    font-weight: 600;
    margin: 22px 0 0;
  }
  .tri,
  .vide {
    color: var(--texte-gris);
    font-size: 0.8125rem;
  }
  ul {
    list-style: none;
    padding: 0;
    margin: 22px 0 0;
  }
  li a {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-height: 52px;
    padding: 11px 13px;
    border-bottom: 1px solid var(--trait);
    color: inherit;
    text-decoration: none;
  }
  li a:focus-visible {
    outline: 2px solid var(--accent-vif);
    outline-offset: 2px;
  }
  .distance {
    font-family: Spectral, Georgia, serif;
    font-size: 1.5625rem;
  }
  .mesures {
    font-size: 0.875rem;
    color: var(--texte-gris);
  }
  .retour {
    color: var(--accent);
    font-size: 0.875rem;
    text-underline-offset: 3px;
  }
</style>
