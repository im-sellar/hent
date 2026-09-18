<script lang="ts">
  import { onMount, tick, untrack } from 'svelte';
  import { goto } from '$app/navigation';
  import { appEtat } from '$lib/assemblage.svelte';
  import { nommerPoint } from '$lib/app/depart/poser';
  import { libelleParDefaut, memePoint, type Coord, type Lieu } from '$lib/domaine/depart';
  import { CENTRE_BRETAGNE, contient, type Zone } from '$lib/domaine/zone';
  import { formatDistance } from '$lib/domaine/format';
  import Bouton from '$lib/ui/Bouton.svelte';
  import Feuille from '$lib/ui/Feuille.svelte';

  const recherche = appEtat.recherche;
  const lieux = $derived(recherche.etat());

  let texte = $state('');
  let centre = $state<Coord>(appEtat.depart?.coord ?? CENTRE_BRETAGNE);
  let libelle = $state(appEtat.depart?.libelle ?? '');
  let zone = $state<Zone | null>(null);
  let position = $state<'refusee' | 'indisponible' | null>(null);

  let champ: HTMLInputElement | null = $state(null);
  let panneauPosition: HTMLDivElement | null = $state(null);
  let avertissement: HTMLParagraphElement | null = $state(null);

  let commande: Coord | null = null;
  let nommageEnCours: AbortController | null = null;

  const horsZone = $derived(zone !== null && !contient(zone, centre));

  const titresPosition = {
    refusee: 'Je n’ai pas accès à ta position.',
    indisponible: 'Ta position n’est pas disponible pour l’instant.'
  };

  /**
   * Nomme un point par géocodage inverse. Seul le dernier nommage compte : un
   * déplacement qui en suit un autre abandonne le premier, sans quoi une
   * réponse lente donnerait au réticule le nom d'un endroit qu'il a quitté.
   */
  async function nommer(point: Coord) {
    nommageEnCours?.abort();
    const controleur = new AbortController();
    nommageEnCours = controleur;
    try {
      const depart = await nommerPoint(appEtat.geocodeur, point, controleur.signal);
      if (!controleur.signal.aborted) libelle = depart.libelle;
    } catch {
      // Seule une annulation arrive ici : nommerPoint se replie sur les coordonnées pour tout le reste.
    } finally {
      if (nommageEnCours === controleur) nommageEnCours = null;
    }
  }

  /** Déplace la carte et retient l'ordre, pour ne pas renommer un endroit qu'on vient de choisir. */
  function aller(point: Coord, zoom: number) {
    commande = point;
    centre = point;
    appEtat.carte?.centrer(point, zoom);
  }

  onMount(() => {
    appEtat.zone().then((z) => (zone = z)).catch(() => undefined);
    if (!libelle) libelle = libelleParDefaut(centre);
    return () => nommageEnCours?.abort();
  });

  // Le layout crée la carte dans son propre onMount, qui s'exécute après celui
  // de cette page : le câblage doit donc attendre qu'elle existe, et se refaire
  // si elle apparaît plus tard.
  $effect(() => {
    const carte = appEtat.carte;
    if (!carte) return;
    carte.afficherBoucles([], null);
    carte.marquerDepart(null);
    untrack(() => aller(centre, appEtat.depart ? 14 : 7));
    const arreter = carte.surDeplacement((c) => {
      const commandee = commande !== null && memePoint(c, commande);
      commande = null;
      if (commandee) return;
      centre = c;
      void nommer(c);
    });
    return () => {
      arreter();
      carte.montrerZone(null);
    };
  });

  $effect(() => {
    appEtat.carte?.montrerZone(horsZone ? zone : null);
  });

  function saisir(e: Event) {
    texte = (e.currentTarget as HTMLInputElement).value;
    recherche.saisir(texte, centre);
  }

  function fermerListe() {
    texte = '';
    recherche.effacer();
  }

  function choisir(lieu: Lieu) {
    fermerListe();
    libelle = lieu.libelle;
    aller(lieu.coord, 14);
  }

  async function autourDeMoi() {
    position = null;
    const resultat = await appEtat.position.obtenir();
    if (resultat.statut === 'ok') {
      aller(resultat.coord, 14);
      void nommer(resultat.coord);
      return;
    }
    position = resultat.statut;
    await tick();
    panneauPosition?.focus();
  }

  /**
   * Pose le départ et passe au réglage. Hors zone, le bouton reste actif et
   * refuse en le disant : un bouton désactivé n'est pas focusable et n'annonce
   * pas pourquoi.
   */
  function partir() {
    if (horsZone) {
      avertissement?.focus();
      return;
    }
    appEtat.poserDepart({ coord: centre, libelle: libelle || libelleParDefaut(centre) });
    appEtat.resultats.reinitialiser();
    void goto('/reglage');
  }
</script>

<svelte:head><title>Partir d’où ? — hent</title></svelte:head>

<Feuille>
  <h1 class="cache-visuellement">Partir d’où ?</h1>
  <div class="barre">
    <a class="retour" href="/" aria-label="Retour à l’accueil">←</a>
    <label class="champ">
      <span class="cache-visuellement">Une adresse, une commune</span>
      <input
        type="search"
        placeholder="Une adresse, une commune…"
        autocomplete="off"
        value={texte}
        bind:this={champ}
        oninput={saisir}
        onkeydown={(e) => {
          if (e.key === 'Escape') fermerListe();
        }}
      />
    </label>
  </div>

  {#if lieux.statut === 'ok' && lieux.lieux.length > 0}
    <ul class="lieux">
      {#each lieux.lieux as lieu (`${lieu.coord.lat},${lieu.coord.lon}|${lieu.libelle}`)}
        <li>
          <button type="button" onclick={() => choisir(lieu)}>
            <span class="nom">{lieu.libelle}</span>
            <span class="complement">{lieu.complement}</span>
            {#if lieu.distanceM !== undefined}<span class="distance">{formatDistance(lieu.distanceM)}</span>{/if}
          </button>
        </li>
      {/each}
    </ul>
    <p class="source">
      Adresses : <a href="https://adresse.data.gouv.fr/">Base Adresse Nationale</a>, sous licence ouverte.
      Ce que tu tapes ici lui est envoyé.
    </p>
  {:else if lieux.statut === 'ok'}
    <p class="source">Aucun lieu trouvé pour « {texte} ».</p>
  {:else if lieux.statut === 'erreur'}
    <p class="source">La recherche d’adresse ne répond pas. La carte et ta position restent disponibles.</p>
  {/if}

  <button type="button" class="position" onclick={autourDeMoi}>Autour de moi</button>

  {#if position}
    <div class="panneau" role="alert" tabindex="-1" aria-labelledby="position-titre" bind:this={panneauPosition}>
      <h2 id="position-titre">{titresPosition[position]}</h2>
      <p>Ça n’empêche rien : cherche une adresse, ou déplace la carte.</p>
      <button type="button" class="secondaire" onclick={() => champ?.focus()}>Chercher une adresse</button>
    </div>
  {/if}

  <div class="depart">
    <p class="libelle" role="status">{libelle}</p>
    {#if appEtat.carte}
      <p class="aide">Fais glisser la carte pour ajuster : le point reste au centre.</p>
    {:else}
      <p class="aide">La carte n’est pas disponible sur cet appareil : cherche une adresse ou utilise ta position.</p>
    {/if}
    <!-- Montée en permanence : une région live créée avec son texte n'est pas annoncée. -->
    <p class="avertissement" role="alert" tabindex="-1" bind:this={avertissement}>
      {horsZone ? 'Ce point est en dehors de la Bretagne, la seule région couverte pour l’instant.' : ''}
    </p>
  </div>

  <Bouton onclick={partir}>Partir d’ici</Bouton>
</Feuille>

<style>
  .barre {
    display: flex;
    gap: 9px;
    align-items: center;
  }
  .retour {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 44px;
    min-height: 44px;
    border: 1px solid var(--trait-vif);
    border-radius: 4px;
    color: var(--texte);
    text-decoration: none;
  }
  .champ {
    flex: 1;
  }
  input[type='search'] {
    width: 100%;
    min-height: 44px;
    padding: 0 13px;
    background: var(--tuile);
    color: var(--texte);
    border: 1px solid var(--trait-vif);
    border-radius: 4px;
    font: inherit;
  }
  input:focus-visible,
  button:focus-visible,
  .retour:focus-visible,
  [tabindex='-1']:focus-visible {
    outline: 2px solid var(--accent-vif);
    outline-offset: 2px;
  }
  .lieux {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  .lieux button {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: 2px 12px;
    width: 100%;
    min-height: 52px;
    padding: 9px 13px;
    text-align: left;
    background: none;
    border: 0;
    border-bottom: 1px solid var(--trait);
    color: var(--texte);
    font: inherit;
    cursor: pointer;
  }
  .nom {
    font-family: Spectral, Georgia, serif;
    font-size: 1.0625rem;
  }
  .complement {
    grid-column: 1;
    font-size: 0.8125rem;
    color: var(--texte-gris);
  }
  .distance {
    grid-column: 2;
    grid-row: 1 / span 2;
    align-self: center;
    font-size: 0.8125rem;
    color: var(--texte-gris);
  }
  .source,
  .aide {
    margin: 0;
    font-size: 0.75rem;
    color: var(--texte-gris);
    line-height: 1.5;
  }
  .source a {
    color: var(--accent);
    text-underline-offset: 3px;
  }
  .position,
  .secondaire {
    align-self: flex-start;
    min-height: 48px;
    padding: 0 16px;
    background: var(--voile);
    color: var(--texte);
    border: 1px solid var(--trait-vif);
    border-radius: 4px;
    font: inherit;
    font-weight: 700;
    cursor: pointer;
  }
  .panneau {
    padding: 13px;
    background: var(--tuile);
    border: 1px solid var(--trait);
    border-radius: 4px;
  }
  .panneau h2 {
    margin: 0 0 4px;
    font-family: Spectral, Georgia, serif;
    font-size: 1.1875rem;
    font-weight: 600;
  }
  .panneau p {
    margin: 0 0 12px;
    color: var(--texte-doux);
    font-size: 0.9375rem;
  }
  .depart {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .libelle {
    margin: 0;
    font-family: Spectral, Georgia, serif;
    font-size: 1.1875rem;
  }
  .avertissement {
    margin: 0;
    font-size: 0.8125rem;
    color: var(--alerte);
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
</style>
