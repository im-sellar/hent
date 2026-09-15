import adapter from '@sveltejs/adapter-static';

export default {
  kit: {
    // Sortie 100 % statique : Caddy la sert telle quelle, aucun Node en
    // production. `fallback` renvoie les routes non prérendues vers une page
    // unique, ce qui permet à /b/<id> de fonctionner sans serveur applicatif.
    // Nommée `200.html`, pas `index.html` : l'accueil est prérendu et produit
    // son propre `index.html` — les deux fichiers sous le même nom
    // s'écraseraient, et le fallback gagnerait, rendant l'accueil indexable
    // illisible pour un robot qui n'exécute pas le JS.
    adapter: adapter({ fallback: '200.html', strict: false })
  }
};
