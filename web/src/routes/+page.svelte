<script lang="ts">
  import Bouton from '$lib/ui/Bouton.svelte';
</script>

<svelte:head>
  <title>hent — une boucle, jamais un aller-retour</title>
  <meta
    name="description"
    content="Dis où tu pars et combien de kilomètres. hent te rend une boucle qui fuit le bitume et revient à ton point de départ. Bretagne, sans compte."
  />
</svelte:head>

<main>
  <h1>Une boucle.<br />Jamais un aller‑retour.</h1>
  <p>
    Dis où tu pars et combien de kilomètres. <em>hent</em> te rend une boucle qui fuit le bitume et
    revient à ton point de départ.
  </p>
  <Bouton href="/reglage">Tracer ma boucle</Bouton>

  <section>
    <h2>Ce que fait hent</h2>
    <dl>
      <dt>Des vraies boucles</dt>
      <dd>Le tracé ne repasse pas sur ses pas. C’est plus dur à calculer qu’un simple trajet, et c’est tout l’intérêt.</dd>
      <dt>Le bitume en dernier recours</dt>
      <dd>Sentiers, chemins creux, pistes forestières. Tu règles le curseur, le tracé s’écarte des routes.</dd>
      <dt>Un GPX, et tu pars</dt>
      <dd>Le fichier se charge dans ta montre ou ton téléphone. Rien à installer, pas de compte.</dd>
    </dl>
  </section>

  <footer>
    Données <a href="https://www.openstreetmap.org/copyright">© les contributeurs OpenStreetMap</a>,
    sous licence ODbL.<br />
    <em>hent</em> — « chemin », en breton.
  </footer>
</main>

<style>
  main {
    max-width: 390px;
    margin: 0 auto;
    padding: 56px 24px 44px;
    display: flex;
    flex-direction: column;
    gap: 22px;
    background: var(--fond);
    color: var(--texte);
    min-height: 100vh;
  }
  h1 {
    font-family: Spectral, Georgia, serif;
    font-size: 2.5rem;
    line-height: 1.1;
    font-weight: 600;
    margin: 0;
  }
  h2 {
    font-family: Spectral, Georgia, serif;
    font-size: 0.8125rem;
    font-weight: 400;
    letter-spacing: 0.16em;
    text-transform: uppercase;
    color: var(--texte-gris);
  }
  p {
    margin: 0;
    color: var(--texte-doux);
    font-size: 1.0625rem;
  }
  dt {
    font-family: Spectral, Georgia, serif;
    font-size: 1.1875rem;
    margin-top: 22px;
  }
  dd {
    margin: 3px 0 0;
    color: var(--texte-doux);
    font-size: 0.9375rem;
  }
  footer {
    font-size: 0.75rem;
    color: var(--texte-gris);
    line-height: 1.6;
  }
  a {
    color: var(--accent);
    text-underline-offset: 3px;
  }
</style>
