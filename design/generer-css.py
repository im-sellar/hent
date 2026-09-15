#!/usr/bin/env python3
"""Écrit les variables CSS du front à partir de `_themes.json`.

Le front ne connaît que des noms de jetons ; leurs valeurs vivent ici, dans le
même fichier que celui qui alimente les maquettes et les styles de carte. Un
ajustement de palette se propage donc au code sans recopie.

Trois blocs sont produits : le thème sombre par défaut, le thème clair quand le
système le demande, et le thème clair forcé par un attribut sur la racine — la
bascule manuelle doit l'emporter sur la préférence système.
"""
from __future__ import annotations

import json
import pathlib

SORTIE = pathlib.Path(__file__).parent.parent / "web" / "src" / "styles" / "jetons.css"
SOURCE = pathlib.Path(__file__).parent / "_themes.json"


def bloc(jetons: dict[str, str], retrait: str = "  ") -> str:
    return "\n".join(f"{retrait}{nom}: {valeur};" for nom, valeur in jetons.items())


def main() -> int:
    themes = json.loads(SOURCE.read_text())
    sombre, clair = themes["sombre"], themes["clair"]

    css = f"""/* Généré par design/generer-css.py — ne pas modifier à la main.
   Les valeurs vivent dans design/_themes.json, avec les maquettes. */

:root {{
  color-scheme: dark light;
{bloc(sombre)}
}}

@media (prefers-color-scheme: light) {{
  :root:not([data-theme='sombre']) {{
{bloc(clair, "    ")}
  }}
}}

:root[data-theme='clair'] {{
{bloc(clair)}
}}
"""
    SORTIE.parent.mkdir(parents=True, exist_ok=True)
    SORTIE.write_text(css)
    print(f"  {SORTIE} — {len(sombre)} jetons par thème")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
