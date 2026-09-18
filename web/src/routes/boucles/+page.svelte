<script lang="ts">
  import { appEtat } from '$lib/assemblage.svelte';
  import { dureeMinutes } from '$lib/domaine/boucle';
  import { formatDistance, formatDuree, formatKilometres, formatPourcent } from '$lib/domaine/format';
  import Feuille from '$lib/ui/Feuille.svelte';

  const etat = $derived(appEtat.resultats.etat());
  const boucles = $derived(etat.statut === 'ok' ? etat.boucles : []);

  let survolee = $state<string | null>(null);
  const selectionnee = $derived(
    survolee && boucles.some((b) => b.id === survolee) ? survolee : (boucles[0]?.id ?? null)
  );

  const titre = $derived.by(() => {
    if (etat.statut !== 'ok') return 'Les boucles';
    const distance = formatKilometres(etat.demande.distanceM);
    return appEtat.depart ? `${distance} au départ de ${appEtat.depart.libelle}` : distance;
  });

  $effect(() => {
    const carte = appEtat.carte;
    if (!carte) return;
    carte.montrerZone(null);
    carte.afficherBoucles(boucles, selectionnee);
    carte.marquerDepart(etat.statut === 'ok' ? etat.demande.depart : null);
  });
</script>

<svelte:head><title>Les boucles — hent</title></svelte:head>

<Feuille>
  <a class="retour" href="/reglage">← Changer les réglages</a>

  <h1>{titre}</h1>
  {#if etat.statut === 'ok' && boucles.length > 0}
    <p class="tri">{boucles.length === 1 ? 'Une boucle' : `${boucles.length} boucles`}, la plus verte d’abord</p>
    <ul>
      {#each boucles as boucle (boucle.id)}
        <li>
          <a
            href="/b/{encodeURIComponent(boucle.id)}"
            class:selectionnee={boucle.id === selectionnee}
            onmouseenter={() => (survolee = boucle.id)}
            onfocus={() => (survolee = boucle.id)}
          >
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
    <p class="vide">Aucune recherche en cours. <a href="/reglage">Régler une boucle</a></p>
  {/if}
</Feuille>

<style>
  h1 {
    font-family: Spectral, Georgia, serif;
    font-size: 1.5625rem;
    font-weight: 600;
    margin: 0;
  }
  .tri,
  .vide {
    margin: 0;
    color: var(--texte-gris);
    font-size: 0.8125rem;
  }
  .vide a {
    color: var(--accent);
    text-underline-offset: 3px;
  }
  ul {
    list-style: none;
    padding: 0;
    margin: 0;
  }
  li a {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-height: 52px;
    padding: 11px 13px;
    border-bottom: 1px solid var(--trait);
    border-radius: 4px;
    color: inherit;
    text-decoration: none;
  }
  li a.selectionnee {
    background: var(--tuile);
    box-shadow: inset 0 0 0 1px var(--trait-vif);
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
    display: inline-flex;
    align-items: center;
    min-height: 44px;
    width: fit-content;
    color: var(--accent);
    font-size: 0.875rem;
    text-underline-offset: 3px;
  }
</style>
