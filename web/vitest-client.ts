import { afterEach } from 'vitest';
import { cleanup } from '@testing-library/svelte';

// jsdom garde un seul document pour tout un fichier de test : sans démontage,
// le composant du test précédent reste dans le DOM et les requêtes par rôle ou
// par texte en trouvent deux.
afterEach(cleanup);
