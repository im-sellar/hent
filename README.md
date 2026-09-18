# hent

Générateur de boucles trail « nature-aware ». Donne-lui un point de départ et une
distance, il te rend une boucle qui privilégie les chemins et évite le bitume.

`hent` signifie « chemin » en breton.

**En cours de construction** — voir [docs/etat-des-lieux.md](docs/etat-des-lieux.md)
pour l'avancement, et [docs/design.md](docs/design.md) pour la conception.

## Développer

```sh
make serve   # l'API, sur le graphe local (une quinzaine de secondes au démarrage)
make dev     # le front, qui relaie /v1 vers l'API — dans un second terminal
make test    # la suite Go, puis celle du front
```

La carte et la recherche d'adresse appellent `data.geopf.fr` et
`api-adresse.data.gouv.fr` : `make dev` a besoin du réseau, `make test` non.

## Licence des données

Données © les contributeurs OpenStreetMap, sous licence
[ODbL](https://opendatacommons.org/licenses/odbl/).

## Licence

Code sous licence [MIT](LICENSE).
