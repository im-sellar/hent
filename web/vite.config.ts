import { sveltekit } from '@sveltejs/kit/vite';
// vitest/config réexporte le defineConfig de Vite en élargissant son type à
// la clé `test` : importer depuis 'vite' fait échouer la vérification des
// types sur cette config, sans toucher au comportement à l'exécution.
import { defineConfig } from 'vitest/config';

export default defineConfig({
  plugins: [sveltekit()],
  server: {
    // En développement, Vite sert le front et relaie l'API : même origine,
    // donc aucune configuration de CORS, comme en production derrière Caddy.
    proxy: { '/v1': 'http://localhost:8080' }
  },
  test: { environment: 'node', include: ['src/**/*.test.ts'] }
});
