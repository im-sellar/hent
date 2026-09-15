import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [sveltekit()],
  server: {
    // En développement, Vite sert le front et relaie l'API : même origine,
    // donc aucune configuration de CORS, comme en production derrière Caddy.
    proxy: { '/v1': 'http://localhost:8080' }
  },
  test: { environment: 'node', include: ['src/**/*.test.ts'] }
});
