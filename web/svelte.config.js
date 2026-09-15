import adapter from '@sveltejs/adapter-static';

export default {
  kit: {
    // Sortie 100 % statique : Caddy la sert telle quelle, aucun Node en
    // production. `fallback` renvoie les routes non prérendues vers une page
    // unique, ce qui permet à /b/<id> de fonctionner sans serveur applicatif.
    adapter: adapter({ fallback: 'index.html', strict: false })
  }
};
