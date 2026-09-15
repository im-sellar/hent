<script lang="ts">
  import { goto } from '$app/navigation';
  import { appEtat } from '$lib/assemblage.svelte';
  import { DISTANCE_MAX_M, DISTANCE_MIN_M } from '$lib/domaine/reglages';
  import { estCoordValide } from '$lib/domaine/depart';
  import { formatDistance } from '$lib/domaine/format';
  import Bouton from '$lib/ui/Bouton.svelte';
  import Curseur from '$lib/ui/Curseur.svelte';
  import EtatEcran from '$lib/ui/EtatEcran.svelte';

  // Saisie provisoire du départ : le plan 2 la remplace par la carte et la
  // recherche d'adresse. Elle existe pour que la chaîne soit utilisable de bout
  // en bout dès maintenant.
  let lat = $state(48.117);
  let lon = $state(-1.677);

  const etat = $derived(appEtat.resultats.etat());
  const coordValide = $derived(estCoordValide({ lat, lon }));

  const motsBitume = ['jamais', 'un peu', 'moyennement', 'beaucoup', 'autant que possible'];
  const motBitume = $derived(
    motsBitume[Math.min(motsBitume.length - 1, Math.floor(appEtat.reglages.eviterBitume * motsBitume.length))]!
  );

  async function tracer() {
    appEtat.poserDepart({ coord: { lat, lon }, libelle: `${lat.toFixed(4)}, ${lon.toFixed(4)}` });
    await appEtat.resultats.lancer({
      depart: { lat, lon },
      distanceM: appEtat.reglages.distanceM,
      eviterBitume: appEtat.reglages.eviterBitume,
      maxResultats: 5,
      variante: 0
    });
    if (appEtat.resultats.etat().statut === 'ok') await goto('/boucles');
  }

  function assouplir() {
    appEtat.regler({ ...appEtat.reglages, eviterBitume: Math.max(0, appEtat.reglages.eviterBitume - 0.3) });
    void tracer();
  }
</script>

<svelte:head><title>Régler — hent</title></svelte:head>

<main>
  <h1>Ta boucle</h1>

  <Curseur
    etiquette="Distance"
    valeur={appEtat.reglages.distanceM}
    min={DISTANCE_MIN_M}
    max={DISTANCE_MAX_M}
    pas={500}
    texteValeur={formatDistance(appEtat.reglages.distanceM)}
    onchange={(v) => appEtat.regler({ ...appEtat.reglages, distanceM: v })}
  />

  <Curseur
    etiquette="Éviter le bitume"
    valeur={appEtat.reglages.eviterBitume}
    min={0}
    max={1}
    pas={0.05}
    texteValeur={motBitume}
    onchange={(v) => appEtat.regler({ ...appEtat.reglages, eviterBitume: v })}
  />

  <fieldset>
    <legend>Départ</legend>
    <p class="provisoire">Saisie temporaire : la carte et la recherche d’adresse arrivent ensuite.</p>
    <label>Latitude <input type="number" step="0.0001" bind:value={lat} /></label>
    <label>Longitude <input type="number" step="0.0001" bind:value={lon} /></label>
    {#if !coordValide}
      <p class="invalide">Ces coordonnées ne sont pas valides.</p>
    {/if}
  </fieldset>

  {#if etat.statut === 'calcul'}
    <EtatEcran enAttente onannuler={() => appEtat.resultats.annuler()} />
  {:else if etat.statut === 'erreur'}
    <EtatEcran erreur={etat.erreur} onreessayer={tracer} onassouplir={assouplir} />
  {:else}
    <Bouton onclick={tracer}>Tracer ma boucle</Bouton>
  {/if}
</main>

<style>
  main {
    max-width: 390px;
    margin: 0 auto;
    padding: 24px 20px 40px;
    display: flex;
    flex-direction: column;
    gap: 22px;
    background: var(--fond);
    color: var(--texte);
    min-height: 100vh;
  }
  h1 {
    font-family: Spectral, Georgia, serif;
    font-size: 1.5625rem;
    font-weight: 600;
    margin: 0;
  }
  fieldset {
    border: 1px solid var(--trait);
    border-radius: 4px;
    padding: 13px;
  }
  legend {
    font-size: 0.875rem;
    color: var(--texte-gris);
  }
  label {
    display: block;
    margin-top: 9px;
    font-size: 0.875rem;
    color: var(--texte-doux);
  }
  input[type='number'] {
    width: 100%;
    min-height: 44px;
    padding: 0 11px;
    background: var(--tuile);
    color: var(--texte);
    border: 1px solid var(--trait-vif);
    border-radius: 4px;
    font: inherit;
  }
  input:focus-visible {
    outline: 2px solid var(--accent-vif);
    outline-offset: 2px;
  }
  .provisoire,
  .invalide {
    margin: 0;
    font-size: 0.8125rem;
    color: var(--texte-gris);
  }
  .invalide {
    color: var(--alerte);
  }
</style>
