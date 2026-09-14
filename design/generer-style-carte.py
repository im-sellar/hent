#!/usr/bin/env python3
"""Écrit les styles MapLibre des deux thèmes à partir de `_themes.json`.

Le fond vient des tuiles vectorielles de la Géoplateforme IGN (PLAN.IGN), qui
sont servies sans clé ni quota et portent `routier_chemin` comme couche
distincte de `routier_route` — la distinction même que fait hent. Le style ne
dessine donc pas une carte générique assombrie : il hiérarchise le chemin
au-dessus de la route, à l'inverse d'un fond routier ordinaire.

Sortie : carte/hent-<theme>.json. Le tracé de la boucle n'est pas ici — il est
ajouté par l'application comme source GeoJSON par-dessus ce fond.
"""
from __future__ import annotations

import json, pathlib, sys, urllib.request

BASE = "https://data.geopf.fr"
TUILES = f"{BASE}/tms/1.0.0/PLAN.IGN/{{z}}/{{x}}/{{y}}.pbf"
GLYPHES = f"{BASE}/annexes/ressources/vectorTiles/fonts/{{fontstack}}/{{range}}.pbf"
METADONNEES = f"{BASE}/tms/1.0.0/PLAN.IGN/metadata.json"
ATTRIBUTION = '<a href="https://geoservices.ign.fr/">© IGN</a>'
POLICE = ["Open Sans Regular"]


def palier(*couples):
    """Interpolation linéaire par zoom, au format des expressions MapLibre."""
    sortie = ["interpolate", ["linear"], ["zoom"]]
    for zoom, valeur in couples:
        sortie += [zoom, valeur]
    return sortie


def couches(t: dict) -> list[dict]:
    ligne = lambda **kw: {"type": "line", "source": "plan-ign",
                          "layout": {"line-cap": "round", "line-join": "round"}, **kw}
    return [
        {"id": "fond", "type": "background", "paint": {"background-color": t["--carte"]}},

        {"id": "vegetation", "type": "fill", "source": "plan-ign", "source-layer": "ocs_vegetation_surf",
         "paint": {"fill-color": t["--carte-vegetation"]}},
        {"id": "eau-surface", "type": "fill", "source": "plan-ign", "source-layer": "hydro_surf",
         "paint": {"fill-color": t["--carte-eau"]}},
        ligne(id="eau-lineaire", **{"source-layer": "hydro_reseau"},
              paint={"line-color": t["--carte-eau"], "line-width": palier((11, 0.7), (16, 2.4))}),

        {"id": "bati", "type": "fill", "source": "plan-ign", "source-layer": "bati_surf", "minzoom": 14,
         "paint": {"fill-color": t["--grille"], "fill-opacity": 0.55}},

        # La route reste un décor : c'est ce que hent cherche à éviter.
        ligne(id="routes", **{"source-layer": "routier_route"},
              paint={"line-color": t["--grille"], "line-width": palier((9, 0.5), (13, 1.4), (17, 3.4))}),
        ligne(id="routes-importantes", **{"source-layer": "routier_route_sup"},
              paint={"line-color": t["--grille"], "line-width": palier((9, 0.9), (13, 2.2), (17, 5))}),

        # Le chemin passe devant, plus épais que la route qui le croise.
        ligne(id="chemins", **{"source-layer": "routier_chemin"},
              paint={"line-color": t["--voie"], "line-width": palier((11, 0.9), (14, 2.2), (17, 4.2))}),
        ligne(id="chemins-importants", **{"source-layer": "routier_chemin_sup"},
              paint={"line-color": t["--voie"], "line-width": palier((11, 1.2), (14, 2.8), (17, 5))}),

        {"id": "toponymes-lieux", "type": "symbol", "source": "plan-ign",
         "source-layer": "toponyme_localite_ponc", "minzoom": 9,
         "layout": {"text-field": ["get", "graphie"], "text-font": POLICE,
                    "text-size": palier((9, 11), (14, 14)), "text-padding": 6},
         "paint": {"text-color": t["--texte-gris"], "text-halo-color": t["--carte"], "text-halo-width": 1.4}},
        {"id": "toponymes-rues", "type": "symbol", "source": "plan-ign",
         "source-layer": "toponyme_routier_odonyme_lin", "minzoom": 15,
         "layout": {"symbol-placement": "line", "text-field": ["get", "graphie"],
                    "text-font": POLICE, "text-size": 11},
         "paint": {"text-color": t["--texte-gris"], "text-halo-color": t["--carte"], "text-halo-width": 1.2}},
    ]


def style(nom: str, t: dict) -> dict:
    return {
        "version": 8,
        "name": f"hent — {nom}",
        "glyphs": GLYPHES,
        "sources": {"plan-ign": {"type": "vector", "tiles": [TUILES],
                                 "minzoom": 0, "maxzoom": 18, "attribution": ATTRIBUTION}},
        "layers": couches(t),
    }


def couches_servies() -> set[str] | None:
    """Les source-layer réellement présentes dans le jeu de tuiles, ou None hors ligne."""
    try:
        with urllib.request.urlopen(METADONNEES, timeout=20) as r:
            return {c["id"] for c in json.load(r)["vector_layers"]}
    except Exception as e:                                    # noqa: BLE001
        print(f"  (métadonnées IGN injoignables : {e}) — vérification des couches sautée", file=sys.stderr)
        return None


def main() -> int:
    themes = json.loads(pathlib.Path("_themes.json").read_text())
    servies = couches_servies()
    manquantes: set[str] = set()
    dossier = pathlib.Path("carte"); dossier.mkdir(exist_ok=True)

    for nom, t in themes.items():
        s = style(nom, t)
        if servies is not None:
            manquantes |= {c["source-layer"] for c in s["layers"] if "source-layer" in c} - servies
        chemin = dossier / f"hent-{nom}.json"
        chemin.write_text(json.dumps(s, ensure_ascii=False, indent=2) + "\n")
        print(f"  {chemin} — {len(s['layers'])} couches")

    if manquantes:
        print(f"ERREUR : couches absentes du jeu de tuiles : {sorted(manquantes)}", file=sys.stderr)
        return 1
    if servies is not None:
        print("  toutes les source-layer existent côté IGN")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
