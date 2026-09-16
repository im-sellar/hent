<script lang="ts">
  import { untrack } from 'svelte';
  import { page } from '$app/state';
  import { appEtat } from '$lib/assemblage.svelte';
  import { versErreurMoteur } from '$lib/app/generation/resultats.svelte';
  import { dureeMinutes, type Boucle, type Demande } from '$lib/domaine/boucle';
  import { formatDistance, formatDuree, formatEcartCible, formatPourcent } from '$lib/domaine/format';
  import type { ErreurMoteur } from '$lib/app/ports';
  import Bouton from '$lib/ui/Bouton.svelte';
  import EtatEcran from '$lib/ui/EtatEcran.svelte';
  import Jauge from '$lib/ui/Jauge.svelte';

  const id = $derived(page.params.id ?? '');

  let boucle = $state<Boucle | null>(null);
  let demande = $state<Demande | null>(null);
  let erreur = $state<ErreurMoteur | null>(null);
  let chargement = $state(false);
  let tentative = $state(0);
  let focusApresAction = $state(false);

  // Deux chemins d'entrée : on arrive de la liste, ou par un lien partagé. Le
  // second impose un appel, puisque rien n'est en mémoire.
  //
  // L'effet ne dépend que de `id` et de `tentative`. L'état partagé des
  // résultats, qu'il alimente lui-même par `poser()`, est donc lu sous
  // `untrack` : un effet qui dépend de ce qu'il écrit se réordonnance pendant
  // son propre `.then`, son teardown pose `annulee` avant que le `.finally`
  // chaîné ne soit dépilé, et `chargement` ne redescend plus jamais. La branche
  // « connue » le redescend elle aussi explicitement, pour rester juste quelle
  // que soit la relance qui l'amène là.
  $effect(() => {
    const idVoulu = id;
    // Dépendance de lecture sans usage : c'est elle que reessayer() incrémente
    // pour relancer l'effet après une erreur, l'id restant inchangé.
    tentative;

    const connue = untrack(() => {
      const etat = appEtat.resultats.etat();
      if (etat.statut !== 'ok') return null;
      const trouvee = etat.boucles.find((b) => b.id === idVoulu);
      return trouvee ? { boucle: trouvee, demande: etat.demande } : null;
    });

    if (connue) {
      boucle = connue.boucle;
      demande = connue.demande;
      erreur = null;
      chargement = false;
      return;
    }

    if (!idVoulu) return;

    // Le contrôleur protège contre deux navigations rapides d'un détail à un
    // autre : `annulee` ignore toute réponse qui arriverait après que l'effet
    // a déjà repris, qu'elle soit tardive ou provoquée par l'abandon lui-même.
    let annulee = false;
    const controleur = new AbortController();

    chargement = true;
    erreur = null;
    appEtat.moteur
      .ouvrir(idVoulu, controleur.signal)
      .then((r) => {
        if (annulee) return;
        boucle = r.boucle;
        demande = r.demande;
        appEtat.resultats.poser([r.boucle], r.demande);
      })
      .catch((e: unknown) => {
        if (annulee) return;
        erreur = versErreurMoteur(e);
      })
      .finally(() => {
        if (annulee) return;
        chargement = false;
      });

    return () => {
      annulee = true;
      controleur.abort();
    };
  });

  function reessayer() {
    // Le bouton disparaît avec l'écran d'erreur : sans reprise, le focus
    // retombe sur le document.
    focusApresAction = true;
    erreur = null;
    tentative += 1;
  }

  const ecart = $derived(boucle && demande ? formatEcartCible(demande.distanceM, boucle.score.distanceM) : '');
</script>

<svelte:head><title>Une boucle — hent</title></svelte:head>

<main>
  <a class="retour" href="/boucles">← Les boucles</a>

  {#if !chargement && !erreur && boucle}
    <div class="titre">
      <h1 class="distance">{formatDistance(boucle.score.distanceM)}</h1>
      {#if ecart}<span class="ecart">{ecart}</span>{/if}
    </div>

    <Jauge
      etiquette="Hors bitume"
      part={boucle.score.partNonBitume}
      texte={formatPourcent(boucle.score.partNonBitume)}
    />
    <Jauge
      etiquette="Exposition au trafic"
      part={boucle.score.partTrafic}
      texte={formatPourcent(boucle.score.partTrafic)}
      alerte
    />

    <dl class="tuiles">
      <div><dt>{formatDuree(dureeMinutes(boucle.score.distanceM, 8))}</dt><dd>à 8 km/h</dd></div>
      <div><dt>{formatPourcent(boucle.score.partRetracee)}</dt><dd>de chemin refait</dd></div>
    </dl>

    <Bouton href={appEtat.moteur.urlGPX(boucle.id)}>Télécharger le GPX</Bouton>
    <p class="partage">
      Le lien de cette page contient ton point de départ : ne le partage qu’en connaissance de cause.
    </p>
  {:else}
    <h1 class="cache-visuellement">Une boucle</h1>
  {/if}

  <EtatEcran
    enAttente={chargement}
    erreur={erreur ?? undefined}
    onreessayer={reessayer}
    hrefAutreDepart="/reglage"
    prendLeFocus={focusApresAction}
  />
</main>

<style>
  main {
    max-width: 390px;
    margin: 0 auto;
    padding: 24px 20px 40px;
    display: flex;
    flex-direction: column;
    gap: 20px;
    background: var(--fond);
    color: var(--texte);
    min-height: 100vh;
  }
  .titre {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
  }
  .distance {
    margin: 0;
    font-family: Spectral, Georgia, serif;
    font-size: 2.25rem;
    font-weight: 600;
    line-height: 1;
  }
  .cache-visuellement {
    position: absolute;
    width: 1px;
    height: 1px;
    margin: -1px;
    padding: 0;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border: 0;
  }
  .ecart,
  .partage {
    font-size: 0.8125rem;
    color: var(--texte-gris);
  }
  .tuiles {
    display: flex;
    gap: 9px;
    margin: 0;
  }
  .tuiles > div {
    flex-grow: 1;
    padding: 11px 13px;
    background: var(--tuile);
    border: 1px solid var(--trait);
    border-radius: 4px;
  }
  dt {
    font-family: Spectral, Georgia, serif;
    font-size: 1.1875rem;
  }
  dd {
    margin: 2px 0 0;
    font-size: 0.75rem;
    color: var(--texte-gris);
    letter-spacing: 0.04em;
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
