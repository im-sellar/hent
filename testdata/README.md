# Données de test

Les fichiers `.osm.pbf` de ce répertoire sont des extraits d'OpenStreetMap,
obtenus via [Geofabrik](https://download.geofabrik.de/) puis découpés avec
`osmium extract`. Ils sont versionnés pour que la suite de tests s'exécute
sans accès réseau.

**Données © les contributeurs OpenStreetMap**, sous licence
[ODbL](https://opendatacommons.org/licenses/odbl/).

La licence MIT du code de ce dépôt **ne s'applique pas** à ces fichiers.

## Reconstruire un extrait

```sh
# -L : Geofabrik redirige « latest » vers le fichier daté du jour.
curl -L -o data/bretagne-latest.osm.pbf \
  https://download.geofabrik.de/europe/france/bretagne-latest.osm.pbf

osmium extract --bbox -1.70,48.10,-1.655,48.13 \
  data/bretagne-latest.osm.pbf -o testdata/rennes-centre.osm.pbf
```
