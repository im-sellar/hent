import { contratAttendu } from '$lib/app/version';

export function baseAPI(): string {
  return `/${contratAttendu()}`;
}
