#!/usr/bin/env python3
"""Écrit la planche Thèmes à partir de `_themes.json`, pour qu'elle ne puisse pas mentir.

Les valeurs affichées sont lues dans le même fichier que celui qui alimente les
maquettes : une couleur changée là-bas apparaît ici sans retouche. Les ratios sont
calculés, pas recopiés.
"""
import json, pathlib

ROLES = [
    ("--fond",        "Fond général de l'écran"),
    ("--feuille",     "Panneaux, feuille basse, blocs"),
    ("--tuile",       "Champs et tuiles de chiffres"),
    ("--carte",       "Fond cartographique"),
    ("--carte-vegetation", "Bois et prairies sur la carte"),
    ("--carte-eau",   "Cours d'eau et plans d'eau"),
    ("--voile",       "Pastilles posées sur la carte"),
    ("--trait",       "Séparateurs, contours de tuiles — décoratif"),
    ("--trait-vif",   "Contour d'un composant — tient les 3:1"),
    ("--rail",        "Piste de curseur, non parcourue"),
    ("--voie",        "Chemins sur la carte, lèvre de la feuille"),
    ("--grille",      "Maillage de la carte, arbres de fond"),
    ("--texte",       "Texte principal, poignée de curseur"),
    ("--texte-doux",  "Explications, texte secondaire"),
    ("--texte-gris",  "Étiquettes, unités, mentions"),
    ("--accent",      "Action, tracé, symbole, liens"),
    ("--sur-accent",  "Texte posé sur un aplat d'accent"),
    ("--accent-vif",  "Survol, focus, portion parcourue"),
    ("--alerte",      "Ce qu'on subit : trafic, bitume, erreur"),
    ("--trace-fond",  "Remplissage de la boucle sur la carte"),
]


def lum(couleur: str) -> float:
    h = couleur.lstrip("#")
    canaux = [int(h[i:i + 2], 16) / 255 for i in (0, 2, 4)]
    lineaire = [c / 12.92 if c <= 0.04045 else ((c + 0.055) / 1.055) ** 2.4 for c in canaux]
    return 0.2126 * lineaire[0] + 0.7152 * lineaire[1] + 0.0722 * lineaire[2]


def ratio(a: str, b: str) -> float:
    x, y = sorted((lum(a), lum(b)), reverse=True)
    return (x + 0.05) / (y + 0.05)


def pastille(valeur: str, sur: str) -> str:
    return (f'<span style="display:inline-block;width:26px;height:26px;border-radius:4px;'
            f'background:{sur};box-shadow:inset 0 0 0 26px {valeur}, 0 0 0 1px #2f3d24;'
            f'vertical-align:middle;flex-shrink:0;"></span>')


def lignes(themes: dict) -> str:
    out = []
    for jeton, role in ROLES:
        s, c = themes["sombre"][jeton], themes["clair"][jeton]
        out.append(f'''
      <div style="display: contents;">
        <div style="padding: 9px 0; border-bottom: 1px solid #2f3d24; font-family: 'IBM Plex Mono', monospace; font-size: 12px;">{jeton}</div>
        <div style="padding: 9px 0; border-bottom: 1px solid #2f3d24; display: flex; align-items: center; gap: 10px;">
          {pastille(s, "#191f13")}<span style="font-family: 'IBM Plex Mono', monospace; font-size: 12px; color: #a9b096;">{s}</span>
        </div>
        <div style="padding: 9px 0; border-bottom: 1px solid #2f3d24; display: flex; align-items: center; gap: 10px;">
          {pastille(c, "#fbfaf6")}<span style="font-family: 'IBM Plex Mono', monospace; font-size: 12px; color: #a9b096;">{c}</span>
        </div>
        <div style="padding: 9px 0; border-bottom: 1px solid #2f3d24; font-size: 14px; color: #8d9678;">{role}</div>
      </div>''')
    return "".join(out)


CONTROLES = [
    ("Texte principal",          "--texte",      ["--feuille", "--fond", "--tuile", "--carte"], 4.5),
    ("Texte secondaire",         "--texte-doux", ["--feuille", "--fond", "--tuile", "--carte"], 4.5),
    ("Étiquettes",               "--texte-gris", ["--feuille", "--fond", "--tuile", "--carte"], 4.5),
    ("Lien et icône d'accent",   "--accent",     ["--feuille", "--fond", "--tuile", "--carte"], 4.5),
    ("Texte sur aplat d'accent", "--sur-accent", ["--accent"],                                  4.5),
    ("Contour de composant",     "--trait-vif",  ["--feuille", "--fond", "--tuile", "--carte"], 3),
    ("Aplat d'accent / page",    "--accent",     ["--feuille", "--fond"],                       3),
    ("Piste de curseur",         "--rail",       ["--feuille"],                                 3),
    ("Portion parcourue",        "--accent-vif", ["--rail"],                                    3),
    ("Anneau de focus",          "--accent-vif", ["--feuille", "--fond", "--tuile", "--carte"], 3),
    ("Tracé sur la carte",       "--accent",     ["--carte", "--voie"],                         3),
    ("Chiffre en alerte",        "--alerte",     ["--feuille", "--tuile"],                      4.5),
]


def controles(themes: dict) -> str:
    out = []
    for label, jeton, fonds, seuil in CONTROLES:
        cases = []
        for nom in ("sombre", "clair"):
            t = themes[nom]
            v = min(ratio(t[jeton], t[f]) for f in fonds)
            couleur = "#a8c98a" if v >= seuil else "#c9a35e"
            cases.append(f'<div style="padding: 8px 0; border-bottom: 1px solid #2f3d24; font-family: \'IBM Plex Mono\', monospace; font-size: 13px; color: {couleur};">{v:.2f}</div>')
        out.append(f'''
      <div style="display: contents;">
        <div style="padding: 8px 0; border-bottom: 1px solid #2f3d24; font-size: 14px; color: #a9b096;">{label}</div>
        {cases[0]}{cases[1]}
        <div style="padding: 8px 0; border-bottom: 1px solid #2f3d24; font-family: 'IBM Plex Mono', monospace; font-size: 13px; color: #8d9678;">≥ {seuil}</div>
      </div>''')
    return "".join(out)


def main() -> int:
    themes = json.loads(pathlib.Path("_themes.json").read_text())
    entete = lambda t: f'<div style="padding: 0 0 9px; border-bottom: 1px solid #607a4a; font-size: 12px; color: #8d9678; letter-spacing: .16em; text-transform: uppercase;">{t}</div>'
    page = f'''<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <script src="./support.js"></script>
</head>
<body>
<x-dc>
<helmet>
  <link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Spectral:ital,wght@0,400;0,600&family=Karla:wght@400;500;700&family=IBM+Plex+Mono:wght@400&display=swap">
  <style>
    body {{ margin: 0; font-family: Karla, 'Helvetica Neue', sans-serif; }}
    .mono {{ font-family: 'IBM Plex Mono', ui-monospace, monospace; }}
  </style>
</helmet>

<div style="width: 1080px; background: #14190f; color: #e9ead9; padding: 44px 44px 52px; display: flex; flex-direction: column; gap: 36px;">

  <div>
    <h1 style="margin: 0 0 7px; font-family: Spectral, Georgia, serif; font-size: 25px; font-weight: 600;">Les deux thèmes</h1>
    <p style="margin: 0; color: #8d9678; font-size: 14px; max-width: 760px;">
      Dix-huit jetons, deux jeux de valeurs. Les maquettes ne connaissent que les noms — aucune couleur n'y est écrite en dur, ce qui rend le second thème gratuit et le troisième possible. Un thème clair n'hérite d'aucun contraste du sombre : les deux colonnes de droite sont recalculées, pas recopiées.
    </p>
  </div>

  <div style="display: grid; grid-template-columns: 190px 200px 200px 1fr; gap: 0 22px;">
    {entete("Jeton")}{entete("Sombre")}{entete("Clair")}{entete("Rôle")}
    {lignes(themes)}
  </div>

  <div style="display: flex; flex-direction: column; gap: 16px;">
    <div>
      <h2 style="margin: 0 0 5px; font-family: Spectral, Georgia, serif; font-size: 19px; font-weight: 600;">Contrôles RGAA</h2>
      <p style="margin: 0; color: #8d9678; font-size: 14px;">Calculés à chaque génération de cette planche. Le pire fond est retenu quand un rôle en côtoie plusieurs.</p>
    </div>
    <div style="display: grid; grid-template-columns: 1fr 110px 110px 90px; gap: 0 22px;">
      {entete("Couple")}{entete("Sombre")}{entete("Clair")}{entete("Seuil")}
      {controles(themes)}
    </div>
  </div>

  <div style="display: flex; gap: 22px;">
    <div style="flex: 1; padding: 20px 22px; background: #191f13; border: 1px solid #2f3d24; border-radius: 4px;">
      <div style="font-family: Spectral, Georgia, serif; font-size: 19px; margin-bottom: 10px;">Ce qui change entre les deux</div>
      <div style="font-size: 15px; color: #a9b096; line-height: 1.6;">
        L'accent bascule du vert pâle au vert profond : sur du papier, <span class="mono" style="font-size: 14px;">#a8c98a</span> tombe à 1,6:1 et ne peut être ni un lien, ni un tracé, ni un aplat. C'est le seul jeton dont le <em>caractère</em> change — les autres ne font que s'inverser.
      </div>
    </div>
    <div style="flex: 1; padding: 20px 22px; background: #191f13; border: 1px solid #2f3d24; border-radius: 4px;">
      <div style="font-family: Spectral, Georgia, serif; font-size: 19px; margin-bottom: 10px;">Comment on bascule</div>
      <div style="font-size: 15px; color: #a9b096; line-height: 1.6;">
        Le sombre est le défaut. <span class="mono" style="font-size: 14px;">prefers-color-scheme: light</span> pose le jeu clair, et un attribut <span class="mono" style="font-size: 14px;">data-theme</span> sur la racine tranche quand la personne a choisi elle-même. Aucun composant n'a à savoir lequel est actif.
      </div>
    </div>
  </div>

</div>
</x-dc>
</body>
</html>
'''
    pathlib.Path("Themes.dc.html").write_text(page)
    print("Themes.dc.html écrit —", len(ROLES), "jetons,", len(CONTROLES), "contrôles")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
