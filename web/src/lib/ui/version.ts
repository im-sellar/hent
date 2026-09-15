import { contratAttendu } from '$lib/app/version';

export function libelleContrat(): string {
  return `API ${contratAttendu()}`;
}
