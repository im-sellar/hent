<script lang="ts">
  import { page } from '$app/state';
  import { appEtat, formatEcartCible } from '$lib/app/etat.svelte';
  import { dureeMinutes, type Boucle, type Demande } from '$lib/domaine/boucle';
  import { formatDistance, formatDuree, formatPourcent } from '$lib/domaine/format';
  import type { ErreurMoteur } from '$lib/app/ports';
  import Bouton from '$lib/ui/Bouton.svelte';
  import EtatEcran from '$lib/ui/EtatEcran.svelte';
  import Jauge from '$lib/ui/Jauge.svelte';

  const id = $derived(page.params.id ?? '');

  let boucle = $state<Boucle | null>(null);
  let demande = $state<Demande | null>(null);
  let erreur = $state<ErreurMoteur | null>(null);
  let chargement = $state(false);

  // Deux chemins d'entrée : on arrive de la liste, ou par un lien partagé. Le
  // second impose un appel, puisque rien n'est en mémoire.
  $effect(() => {
    const etat = appEtat.resultats.etat();
    if (etat.statut === 'ok') {
      const connue = etat.boucles.find((b) => b.id === id);
      if (connue) {
        boucle = connue;
        demande = etat.demande;
        erreur = null;
        return;
      }
    }
    if (!id || boucle?.id === id) return;

    chargement = true;
    erreur = null;
    appEtat.moteur
      .ouvrir(id)
      .then((r) => {
        boucle = r.boucle;
        demande = r.demande;
      })
      .catch((e: unknown) => {
        const g = e && typeof e === 'object' && 'genre' in e ? e : null;
        erreur = g
          ? { genre: (g as ErreurMoteur).genre, message: (g as ErreurMoteur).message }
          : { genre: 'Reseau', message: String(e) };
      })
      .finally(() => {
        chargement = false;
      });
  });

  const ecart = $derived(boucle && demande ? formatEcartCible(demande.distanceM, boucle.score.distanceM) : '');
</script>

<svelte:head><title>Une boucle — hent</title></svelte:head>

<main>
  <a class="retour" href="/boucles">← Les boucles</a>

  {#if chargement}
    <EtatEcran enAttente />
  {:else if erreur}
    <EtatEcran erreur={erreur} />
  {:else if boucle}
    <div class="titre">
      <span class="distance">{formatDistance(boucle.score.distanceM)}</span>
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
  {/if}
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
    font-family: Spectral, Georgia, serif;
    font-size: 2.25rem;
    line-height: 1;
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
    color: var(--accent);
    font-size: 0.875rem;
    text-underline-offset: 3px;
  }
</style>
