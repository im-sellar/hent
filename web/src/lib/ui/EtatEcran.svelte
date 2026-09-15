<script lang="ts">
  import type { ErreurMoteur } from '$lib/app/ports';
  import Bouton from './Bouton.svelte';

  type Props = {
    erreur?: ErreurMoteur;
    enAttente?: boolean;
    onannuler?: () => void;
    onreessayer?: () => void;
    onassouplir?: () => void;
  };
  let { erreur, enAttente = false, onannuler, onreessayer, onassouplir }: Props = $props();

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
</script>

{#if enAttente}
  <div class="bloc" role="status">
    <h2>Je parcours les chemins.</h2>
    <p>Une poignée de secondes, le temps de comparer les boucles.</p>
    {#if onannuler}
      <Bouton variante="secondaire" onclick={onannuler}>Annuler</Bouton>
    {/if}
  </div>
{:else if erreur}
  <div class="bloc" role="alert">
    <h2>{titres[erreur.genre]}</h2>
    <p>{details[erreur.genre]}</p>
    {#if erreur.genre === 'AucuneBoucle' && onassouplir}
      <Bouton onclick={onassouplir}>Accepter plus de bitume</Bouton>
    {:else if erreur.genre === 'HorsZone'}
      <Bouton href="/reglage">Choisir un autre départ</Bouton>
    {:else if onreessayer}
      <Bouton variante="secondaire" onclick={onreessayer}>
        {erreur.reessayerDansS ? `Réessayer dans ${erreur.reessayerDansS} s` : 'Réessayer'}
      </Bouton>
    {/if}
  </div>
{/if}

<style>
  .bloc {
    display: flex;
    flex-direction: column;
    gap: 18px;
    padding: 22px 20px;
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
</style>
