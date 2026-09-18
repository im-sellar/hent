import { afterEach } from 'vitest';
import { cleanup } from '@testing-library/svelte';

// jsdom garde un seul document pour tout un fichier de test : sans démontage,
// le composant du test précédent reste dans le DOM et les requêtes par rôle ou
// par texte en trouvent deux.
afterEach(cleanup);

// jsdom n'implémente pas matchMedia ; le layout le consulte pour suivre le
// thème du système. Une doublure inerte suffit : les tests qui en dépendent la
// remplacent explicitement.
if (typeof window.matchMedia !== 'function') {
  window.matchMedia = (requete: string) =>
    ({
      matches: false,
      media: requete,
      onchange: null,
      addEventListener() {},
      removeEventListener() {},
      addListener() {},
      removeListener() {},
      dispatchEvent: () => false
    }) as MediaQueryList;
}
