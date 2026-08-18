# État des lieux — 18 août 2026

Où en est `hent`, et comment reprendre.

## En un coup d'œil

**2 tâches terminées sur 9** pour l'étape 1. Le graphe se construit et se parcourt ; il ne sait pas encore lire OpenStreetMap ni calculer d'itinéraire.

| # | Tâche | Code | Revue |
|---|---|---|---|
| 1 | Squelette, géométrie, garde-fou d'architecture | ✅ `4621bc4` | ✅ approuvée |
| 2 | Graphe CSR et son constructeur | ✅ `0126ae2` | ⚠️ **non faite** |
| 3 | Profil, pondérations, fonction de coût | — | — |
| 4 | A*, index spatial, benchmark, port réseau | — | — |
| 5 | Lecture OSM et construction du graphe | — | — |
| 6 | Sérialisation `graph.bin` et binaire `graphbuild` | — | — |
| 7 | Génération de boucles | — | — |
| 8 | Score lisible | — | — |
| 9 | API HTTP, export GPX, binaire `routed` | — | — |

`go vet ./... && go test ./...` passe sur les trois paquets existants.

**La Task 2 n'a pas été relue.** Son code est commité et ses tests passent, mais aucun relecteur ne l'a examinée — la session s'est arrêtée entre l'implémentation et la revue. À reprendre en premier.

## Ce qui existe

```
internal/domain/geo.go            Coord, BBox, HaversineM
internal/domain/track.go          NodeRef, EdgeRef, Path
internal/adapter/network/csr/     Graph (CSR), Builder
internal/architecture_test.go     garde-fou de la règle de dépendance
```

Le garde-fou d'architecture a été vérifié comme mordant réellement : import interdit ajouté dans `domain`, échec constaté, import retiré.

## Les documents

- **`docs/design.md`** — la spec. Autorité sur toutes les décisions techniques : modèle de coût, invariant des pénalités, contrainte ODbL, feuille de route en 5 étapes.
- **Le plan d'implémentation** vit hors du dépôt, dans le vault Obsidian :
  `~/Library/CloudStorage/OneDrive-Hellowork/obsidian-vault/claude/hent/2026-08-18-hent-plan-etape1.md`
  9 tâches, tout le code Go à écrire, en TDD. C'est lui qu'on déroule.
- **Le journal d'exécution** est dans `.superpowers/sdd/2026-08-18-hent-plan-etape1/progress.md` (ignoré par git, local à la machine) : briefs, rapports, revues, et les décisions prises en cours de route.

## Décisions prises pendant la mise en route

Cinq points que le plan laissait ouverts ont été tranchés pour ne pas bloquer. Les deux premiers ont depuis été résolus pour de bon ; les trois autres restent des choix réversibles.

1. ~~**Identité git**~~ — **résolu.** Les commits ont été réécrits sous `aurelien.morice@ik.me` avant toute publication, et le dépôt porte cette identité en local.
2. **Chemin du module** — `github.com/im-sellar/hent`, supposé. Il est codé en dur dans `internal/architecture_test.go` : à changer là aussi si le compte GitHub diffère.
3. **Tests HTTP de la Task 9** — le plan faisait partir les boucles de test d'un point situé à 160 m du bord de la grille synthétique. Les waypoints d'une boucle de 4 km en seraient sortis et le test aurait échoué par intermittence. Le point de départ passe à `48.135 / -1.628`, au centre de la grille.
4. **`TestGenerateVariantDonneAutreChose` (Task 7)** — le plan comparait deux boucles par leur longueur et leur nombre de nœuds. Sur une grille régulière, deux boucles distinctes ont très souvent ces deux valeurs identiques. Le test comparera les ensembles d'arêtes.
5. **Sérialisation (Task 6)** — l'écriture élément par élément via `binary.Write` est conservée telle que le plan la spécifie. `graphbuild` tourne hors ligne, sa lenteur ne touche jamais le service. Si le build de la Bretagne dépasse deux minutes, passer à un `bufio.Writer` avec encodage en tampon.

Les points 3 et 4 sont de vrais défauts du plan, trouvés avant exécution. Le plan dans le vault n'a **pas** été modifié : au moment d'attaquer les Tasks 7 et 9, appliquer ces deux corrections.

## Constats mineurs laissés de côté

- `internal/architecture_test.go:38` — la comparaison de préfixe d'import ne vérifie pas la frontière de segment. Une future couche `internal/app2` serait faussement détectée comme important `internal/app`. Aucun impact sur les quatre couches actuelles.
- `internal/domain/geo_test.go` — `TestBBoxContains` ne couvre pas les points situés exactement sur `Min` ou `Max`, alors que `Contains` utilise des comparaisons inclusives.

## Reprendre

1. Faire relire la Task 2 (diff `4621bc4..0126ae2`), ou l'accepter en connaissance de cause.
2. Enchaîner sur la Task 3 du plan : profil, pondérations et fonction de coût.

L'invariant à ne jamais perdre de vue, quel que soit l'ordre choisi ensuite : **tout critère se formule comme une pénalité positive, jamais comme une récompense**. Un coût négatif rend l'A* faux silencieusement, sans erreur, avec des itinéraires absurdes. C'est expliqué au §6 de `docs/design.md`.
