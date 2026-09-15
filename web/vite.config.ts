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
  test: {
    // Deux projets, parce que Svelte compile différemment selon la cible. En
    // environnement `node`, tout `.svelte.ts` est compilé pour le serveur, où
    // `$effect` ne s'exécute jamais : un composant y est inobservable. Le
    // projet `client` monte donc les tests d'interface sous jsdom, avec la
    // condition de résolution `browser` qui donne le runtime client de Svelte.
    projects: [
      {
        extends: './vite.config.ts',
        resolve: { conditions: ['browser'] },
        test: {
          name: 'client',
          environment: 'jsdom',
          include: ['src/**/*.svelte.test.ts'],
          setupFiles: ['./vitest-client.ts']
        }
      },
      {
        extends: './vite.config.ts',
        test: {
          name: 'serveur',
          environment: 'node',
          include: ['src/**/*.test.ts'],
          exclude: ['src/**/*.svelte.test.ts']
        }
      }
    ]
  }
});
