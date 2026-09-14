#!/usr/bin/env python3
"""Injecte un jeu de jetons de thème dans les maquettes et dérive les variantes claires.

Les maquettes sont écrites une seule fois, en `var(--jeton)`. Ce script est la seule
chose qui connaisse les valeurs : il pose le bloc `:root` du thème sombre dans le
fichier source et écrit la variante claire à côté. Relancer après toute modification
d'une maquette, sinon la variante claire diverge en silence.
"""
import json, pathlib, re, sys

DEBUT, FIN = "    /* theme:debut */", "    /* theme:fin */"
ECRANS = {"Main.dc.html": "MainClair.dc.html",
          "Depart.dc.html": "DepartClair.dc.html",
          "DepartRecherche.dc.html": "DepartRechercheClair.dc.html",
          "Generateur.dc.html": "GenerateurClair.dc.html",
          "Boucles.dc.html": "BouclesClair.dc.html",
          "Detail.dc.html": "DetailClair.dc.html"}

# Planches de documentation : elles consomment les mêmes jetons mais n'ont pas de
# variante claire — elles décrivent le système, elles ne sont pas le produit.
PLANCHES = ["Etats.dc.html"]


def bloc(jetons: dict[str, str]) -> str:
    lignes = "\n".join(f"      {k}: {v};" for k, v in jetons.items())
    return f"{DEBUT}\n    :root {{\n{lignes}\n    }}\n{FIN}"


def poser(source: str, jetons: dict[str, str]) -> str:
    """Remplace le bloc de jetons s'il existe, l'insère en tête du <style> sinon."""
    nouveau = bloc(jetons)
    if DEBUT in source:
        return re.sub(re.escape(DEBUT) + r".*?" + re.escape(FIN), lambda _: nouveau, source, flags=re.S)
    ancre = "  <style>\n"
    if ancre not in source:
        raise SystemExit(f"pas de <style> où insérer les jetons")
    return source.replace(ancre, ancre + nouveau + "\n", 1)


def main() -> int:
    themes = json.loads(pathlib.Path("_themes.json").read_text())
    for src, dst in ECRANS.items():
        p = pathlib.Path(src)
        s = poser(p.read_text(), themes["sombre"])
        p.write_text(s)
        pathlib.Path(dst).write_text(poser(s, themes["clair"]))
        print(f"  {src} -> {dst}")
    for nom in PLANCHES:
        q = pathlib.Path(nom)
        q.write_text(poser(q.read_text(), themes["sombre"]))
        print(f"  {nom} (sombre seul)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
