<script lang="ts">
  import { goto } from '$app/navigation';
  import { appEtat } from '$lib/assemblage.svelte';
  import { DISTANCE_MAX_M, DISTANCE_MIN_M } from '$lib/domaine/reglages';
  import { formatDistance } from '$lib/domaine/format';
  import Bouton from '$lib/ui/Bouton.svelte';
  import Curseur from '$lib/ui/Curseur.svelte';
  import EtatEcran from '$lib/ui/EtatEcran.svelte';
  import SelecteurTheme from '$lib/ui/SelecteurTheme.svelte';

  // Le bouton qui vient d'être activé est remplacé par l'état de recherche :
  // sans reprise, le focus retombe sur le document.
  let focusApresAction = $state(false);

  const etat = $derived(appEtat.resultats.etat());

  const motsBitume = ['jamais', 'un peu', 'moyennement', 'beaucoup', 'autant que possible'];
  const motBitume = $derived(
    motsBitume[Math.min(motsBitume.length - 1, Math.floor(appEtat.reglages.eviterBitume * motsBitume.length))]!
  );

  // Régler sans départ n'a pas de sens : la personne est renvoyée là où il se pose.
  $effect(() => {
    if (!appEtat.depart) void goto('/depart');
  });

  async function tracer() {
    const depart = appEtat.depart;
    if (!depart) return;
    focusApresAction = true;
    await appEtat.resultats.lancer({
      depart: depart.coord,
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

  <section class="depart" aria-labelledby="depart-titre">
    <h2 id="depart-titre">Départ</h2>
    <p class="libelle">{appEtat.depart?.libelle ?? ''}</p>
    <a class="changer" href="/depart">Changer</a>
  </section>

  <Curseur
    id="distance"
    etiquette="Distance"
    valeur={appEtat.reglages.distanceM}
    min={DISTANCE_MIN_M}
    max={DISTANCE_MAX_M}
    pas={500}
    texteValeur={formatDistance(appEtat.reglages.distanceM)}
    onchange={(v) => appEtat.regler({ ...appEtat.reglages, distanceM: v })}
  />

  <Curseur
    id="eviter-bitume"
    etiquette="Éviter le bitume"
    valeur={appEtat.reglages.eviterBitume}
    min={0}
    max={1}
    pas={0.05}
    texteValeur={motBitume}
    onchange={(v) => appEtat.regler({ ...appEtat.reglages, eviterBitume: v })}
  />

  <EtatEcran
    enAttente={etat.statut === 'calcul'}
    erreur={etat.statut === 'erreur' ? etat.erreur : undefined}
    onannuler={() => appEtat.resultats.annuler()}
    onreessayer={tracer}
    onassouplir={assouplir}
    hrefAutreDepart="/depart"
    prendLeFocus={focusApresAction}
  />
  {#if etat.statut !== 'calcul' && etat.statut !== 'erreur'}
    <Bouton onclick={tracer}>Tracer ma boucle</Bouton>
  {/if}

  <SelecteurTheme valeur={appEtat.theme} onchange={(t) => appEtat.changerTheme(t)} />
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
    min-height: 100dvh;
  }
  h1 {
    font-family: Spectral, Georgia, serif;
    font-size: 1.5625rem;
    font-weight: 600;
    margin: 0;
  }
  .depart {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: 2px 12px;
    padding: 13px;
    border: 1px solid var(--trait);
    border-radius: 4px;
  }
  .depart h2 {
    grid-column: 1 / -1;
    margin: 0;
    font-size: 0.875rem;
    font-weight: 400;
    color: var(--texte-gris);
  }
  .libelle {
    margin: 0;
    font-family: Spectral, Georgia, serif;
    font-size: 1.1875rem;
  }
  .changer {
    display: inline-flex;
    align-items: center;
    min-height: 44px;
    color: var(--accent);
    font-size: 0.875rem;
    text-underline-offset: 3px;
  }
  .changer:focus-visible {
    outline: 2px solid var(--accent-vif);
    outline-offset: 2px;
  }
</style>
