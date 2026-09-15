<script lang="ts">
  import type { ErreurMoteur } from '$lib/app/ports';
  import Bouton from './Bouton.svelte';

  type Props = {
    erreur?: ErreurMoteur;
    enAttente?: boolean;
    onannuler?: () => void;
    onreessayer?: () => void;
    onassouplir?: () => void;
    /**
     * Où envoyer quelqu'un dont le départ sort de la zone couverte. Laissé vide
     * par l'écran de départ lui-même : un composant d'état ne sait pas quelle
     * page le rend, et proposer un lien vers celle où l'on se trouve déjà est
     * un cul-de-sac.
     */
    hrefAutreDepart?: string;
    /**
     * Déplace le focus sur la région qui vient de parler. À poser quand cet
     * état remplace le bouton qu'on vient d'activer — le focus disparaîtrait
     * avec lui —, jamais à l'ouverture d'une page.
     */
    prendLeFocus?: boolean;
  };
  let {
    erreur,
    enAttente = false,
    onannuler,
    onreessayer,
    onassouplir,
    hrefAutreDepart,
    prendLeFocus = false
  }: Props = $props();

  const titres: Record<ErreurMoteur['genre'], string> = {
    HorsZone: 'Ce point est en dehors de la Bretagne.',
    AucuneBoucle: 'Pas de boucle à cette distance par ici.',
    TropDeDemandes: 'Une minute, le temps de souffler.',
    DelaiDepasse: 'La recherche a pris trop de temps.',
    Serveur: 'Le service a rencontré un problème.',
    Reseau: 'Impossible de joindre le service.'
  };

  const details: Record<ErreurMoteur['genre'], string> = {
    HorsZone: "C'est la seule région cartographiée pour l'instant. Le reste viendra.",
    AucuneBoucle: 'Les chemins autour de ce point ne se referment pas sur cette distance.',
    TropDeDemandes: 'Trop de demandes d’un coup. Le service est petit et gratuit ; il tient debout comme ça.',
    DelaiDepasse: 'Elle aboutit parfois au second essai.',
    Serveur: 'Ce n’est pas toi : quelque chose a échoué de notre côté. Réessaie dans un instant.',
    Reseau: 'Vérifie ta connexion, puis réessaie.'
  };

  let blocAttente: HTMLDivElement | null = $state(null);
  let blocErreur: HTMLDivElement | null = $state(null);
  let restantS = $state(0);

  // Le serveur annonce un délai ; tant qu'il court, aucun bouton n'est offert.
  // Un bouton « Réessayer dans 3 s » cliquable tout de suite ne fait que
  // reprendre un refus, et promet une attente qu'il n'observe pas.
  $effect(() => {
    const delai = erreur?.reessayerDansS ?? 0;
    restantS = delai;
    if (delai <= 0) return;

    const tic = setInterval(() => {
      restantS -= 1;
      if (restantS <= 0) clearInterval(tic);
    }, 1000);
    return () => clearInterval(tic);
  });

  $effect(() => {
    if (!prendLeFocus) return;
    (erreur ? blocErreur : enAttente ? blocAttente : null)?.focus();
  });
</script>

<div class="bloc" class:vide={!enAttente} role="status" tabindex="-1" bind:this={blocAttente}>
  {#if enAttente}
    <h2>Je parcours les chemins.</h2>
    <p>Une poignée de secondes, le temps de comparer les boucles.</p>
    {#if onannuler}
      <div class="actions" aria-live="off">
        <Bouton variante="secondaire" onclick={onannuler}>Annuler</Bouton>
      </div>
    {/if}
  {/if}
</div>

<div class="bloc" class:vide={!erreur} role="alert" tabindex="-1" bind:this={blocErreur}>
  {#if erreur}
    <h2>{titres[erreur.genre]}</h2>
    <p>{details[erreur.genre]}</p>
    <!-- Hors de la région d'alerte : un décompte qui change chaque seconde y
         serait relu en entier à chaque seconde. -->
    <div class="actions" aria-live="off">
      {#if erreur.genre === 'AucuneBoucle' && onassouplir}
        <Bouton onclick={onassouplir}>Accepter plus de bitume</Bouton>
      {:else if erreur.genre === 'HorsZone' && hrefAutreDepart}
        <Bouton href={hrefAutreDepart}>Choisir un autre départ</Bouton>
      {:else if onreessayer}
        {#if restantS > 0}
          <p class="decompte">Réessayer dans {restantS} s</p>
        {:else}
          <Bouton variante="secondaire" onclick={onreessayer}>Réessayer</Bouton>
        {/if}
      {/if}
    </div>
  {/if}
</div>

<style>
  .bloc {
    display: flex;
    flex-direction: column;
    gap: 18px;
    padding: 22px 20px;
  }
  /* Hors du flux plutôt que démontée : la région doit rester dans l'arbre
     d'accessibilité pour que sa prochaine mutation soit annoncée. */
  .vide {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
  }
  .bloc:focus-visible {
    outline: 2px solid var(--accent-vif);
    outline-offset: 2px;
  }
  .actions {
    display: flex;
    flex-direction: column;
    gap: 18px;
  }
  h2 {
    font-family: Spectral, Georgia, serif;
    font-size: 1.1875rem;
    font-weight: 400;
    margin: 0;
  }
  p {
    margin: 0;
    color: var(--texte-doux);
    font-size: 0.9375rem;
    line-height: 1.55;
  }
  .decompte {
    color: var(--texte-gris);
  }
</style>
