# Durcir l'architecture du back

Refactoring sans changement observable. L'étape 1 a livré une architecture en
couches correcte et gardée par un test ; quatre failles subsistent, dont trois
que le gardien actuel ne peut pas voir. Le front va typer ses DTO sur ce que
l'API renvoie — autant que ce contrat soit explicite et tenu par un adaptateur
avant, plutôt que par le domaine.

## Le contrat : rien ne change de visible

**Critère de succès :** les réponses HTTP sont identiques, octet pour octet.
Mêmes noms de champs, même ordre, mêmes valeurs. Aucune version d'API, aucune
migration, aucun client à prévenir.

Ce qui changera, et qu'il ne faut pas se cacher : les tests internes qui nomment
`generateloop.Request` ou `csr.Provenance` suivront le déplacement de ces types.
C'est mécanique, mais ce n'est pas « aucun test modifié ».

## L'audit

### 1. Le domaine sait qu'il est sérialisé en JSON

```go
// internal/domain/score.go
type Score struct {
    DistanceM     float64 `json:"distance_m"`
    PartNonBitume float64 `json:"part_non_bitume"`
    PartTrafic    float64 `json:"part_trafic"`
    PartRetracee  float64 `json:"part_retracee"`
    EcartCible    float64 `json:"ecart_cible"`
}
```

L'en-tête du paquet `httpapi` promet pourtant l'inverse, noir sur blanc : « Les
DTO définis ici sont volontairement distincts des types du domaine : le contrat
public doit pouvoir rester stable pendant que le modèle interne évolue. » Et le
commentaire de `sourceDTO` explique précisément pourquoi la provenance est
découplée — puis `loopDTO` expose `domain.Score` brut. La règle est écrite, elle
n'est simplement pas tenue partout.

`Score` décrit un fait métier. Que ce fait s'appelle `part_non_bitume` dans une
réponse HTTP est une décision d'adaptateur, prise ailleurs et pour d'autres
raisons. Conséquence concrète : renommer un champ du domaine casse l'API
publique sans qu'aucun test ne prévienne — donc on cesse de renommer, et le
vocabulaire métier se fige sur des choix d'exposition.

La fuite est contagieuse : `part_retracee` a été ajouté directement dans le
domaine, sans que la question se pose.

### 2. Il n'y a pas de port entrant

```go
func New(gen *generateloop.Generator, prov csr.Provenance, bbox domain.BBox,
         trustedProxies map[string]struct{}) http.Handler
```

`RouteNetwork` ferme le côté sortant de l'hexagone — le graphe est derrière un
port, et `testsupport.Grille` le prouve. Rien ne ferme le côté entrant :
l'adaptateur HTTP dépend du type concret `*generateloop.Generator`.

Ce qui l'empêche aujourd'hui n'est pas un oubli mais un cycle : une interface
qui décrirait la génération devrait référencer `generateloop.Request`, donc
`internal/app/port` importerait `internal/app/generateloop`.

### 3. Un adaptateur en importe un autre

`csr.Provenance` figure dans la signature de `httpapi.New` : l'adaptateur
entrant connaît l'adaptateur sortant. Le type décrit la provenance des données —
date de construction, sources, empreintes, `config_hash` — c'est-à-dire le socle
de l'engagement ODbL de reconstructibilité (§9), pas un détail du format CSR.

Les règles de couches actuelles raisonnent par couche entière et autorisent donc
`adapter → adapter`. C'est le trou par lequel ce couplage est passé.

### 4. Le gardien lui-même compare approximativement

```go
// internal/architecture_test.go
if strings.HasPrefix(pkg, modulePath+"/"+b) {
```

La comparaison ne vérifie pas la frontière de segment : une future couche
`internal/app2` serait détectée comme important `internal/app`. Constat déjà
consigné dans l'état des lieux. Le test qui garantit l'architecture est le
dernier endroit où tolérer une comparaison de chaînes approximative.

## La cible

### Le filet, écrit en premier

Avant toute modification, un test qui fige la réponse JSON courante et la
compare à une référence versionnée : `POST /v1/loops` et `GET /v1/regions` sur
la grille synthétique, sérialisées et confrontées à un fichier témoin.

Ce test doit être **écrit et vert avant le premier déplacement de type**. C'est
lui qui distingue un refactoring réussi d'un refactoring qui a dérapé, et il ne
prouve rien s'il est écrit après coup sur le résultat obtenu.

Précaution : la génération étant déterministe à requête égale, la référence est
stable. Si elle ne l'était pas, le témoin serait inutilisable et il faudrait le
découvrir maintenant, pas au milieu du chantier.

### 1. La sérialisation descend dans l'adaptateur

`domain.Score` perd ses tags. `httpapi` gagne un `scoreDTO` qui les porte, et
une conversion explicite depuis `domain.Score`. `loopDTO.Score` change de type
en conséquence.

Les noms JSON publics sont recopiés à l'identique. Le témoin le vérifie.

### 2. `Request` descend dans le domaine, le port entrant devient possible

`generateloop.Request` devient `domain.LoopRequest`. Son contenu est déjà
intégralement métier — un départ, une distance, une tolérance, des préférences,
un nombre de résultats, une variante — et rien n'y relève de la stratégie de
génération.

Le cycle levé, le port se définit :

```go
// internal/app/port
type LoopGenerator interface {
    Generate(ctx context.Context, req domain.LoopRequest) ([]domain.Loop, error)
    Stats() (exploredNodes, droppedCandidates int64)
}
```

Port grossier, comme `RouteNetwork` : deux méthodes qui décrivent une seule
chose — un moteur de boucles observable. `httpapi.New` le prend en paramètre.

`withDefaults` reste dans `generateloop`, mais cesse d'être une méthode — elle
devient `func withDefaults(domain.LoopRequest) domain.LoopRequest`. Appliquer des
valeurs par défaut est une décision de stratégie, pas une règle du domaine, et le
domaine n'a pas à porter le comportement d'une couche au-dessus de lui.

Point de vigilance assumé : `MaxResults` et `Variant` sont plus proches de la
requête utilisateur que du métier pur. Ils restent dans `LoopRequest` parce que
les en extraire imposerait deux types de requête et une conversion de plus, pour
un gain théorique. Si un second appelant apparaît un jour avec d'autres besoins,
c'est là qu'il faudra trancher à nouveau.

### 3. `Provenance` rejoint le domaine — sans toucher au format de l'artefact

**Contrainte découverte à la rédaction du plan :** les tags de `csr.Provenance`
et `csr.Source` ne sont pas décoratifs. L'en-tête de `graph.bin` est cet objet
sérialisé en JSON (`json.Marshal` dans `Write`, `json.Unmarshal` dans
`ReadGraph`). Les supprimer rendrait illisible tout artefact existant — 294 Mo à
reconstruire, et l'engagement ODbL de reconstructibilité avec.

La cible tient donc en deux types, pas un :

- `domain.Provenance` et `domain.Source`, **sans tags** : le concept métier —
  d'où viennent les données, avec quelles empreintes.
- Dans `csr`, un `provenanceHeader` **privé**, portant les tags actuels **à
  l'identique**, et deux conversions vers et depuis `domain.Provenance`. Le
  format binaire ne bouge pas d'un octet.

C'est le même motif que `scoreDTO` au point 1, appliqué à une sortie binaire au
lieu d'une sortie HTTP : le type qui décrit une sérialisation appartient à
l'adaptateur qui la produit.

**Vérification obligatoire :** un test qui relit un artefact écrit *avant* le
refactoring et retrouve la même provenance. Sans lui, la régression est
silencieuse jusqu'au prochain déploiement.

Après ce déplacement, `httpapi` n'importe plus `csr`. Les DTO HTTP de la
provenance existent déjà côté adaptateur (`provenanceDTOOf`, `sourceDTO`) : il
n'y a rien à inventer de ce côté.

### 4. Le gardien se durcit sur trois points

- **Frontière de segment :** `pkg == p || strings.HasPrefix(pkg, p+"/")`.
- **Règle nouvelle :** un paquet sous `internal/adapter/X` ne doit pas importer
  `internal/adapter/Y`. C'est la règle qui manquait pour attraper le point 3 ;
  sans elle, le couplage reviendra.
- **Gardien des tags :** échec si `json:`, `xml:`, `db:` ou `yaml:` apparaît
  dans une déclaration de structure de `internal/domain`. C'est le seul moyen
  d'empêcher la récidive au prochain champ ajouté — le point 1 s'est produit
  exactement comme ça.

Les trois se vérifient par lecture de l'AST, comme le test actuel.

## Ordre d'exécution

L'ordre n'est pas indifférent : chaque étape doit laisser la suite verte.

1. Le témoin JSON. Vert avant de continuer. `httpapi` se teste déjà sur la grille
   synthétique (`testnetwork_test.go`) : le témoin s'y greffe sans monter de
   nouvelle infrastructure.
2. Le gardien durci — frontière de segment et règle `adapter → adapter`. **Il
   échouera** sur `httpapi → csr` : c'est attendu, et c'est ce qui rend l'étape 4
   nécessaire plutôt qu'optionnelle. La règle est introduite d'abord, la
   violation réparée ensuite.
3. Le gardien des tags. Il échouera sur `domain.Score` — même logique.
4. `Provenance` et `Source` vers `domain`. Le gardien de l'étape 2 repasse.
5. Les tags hors du domaine, `scoreDTO` côté `httpapi`. Le gardien de l'étape 3
   repasse, le témoin reste vert.
6. `Request` vers `domain.LoopRequest`, puis `port.LoopGenerator`, puis
   `httpapi.New` qui prend le port.

Les étapes 2 et 3 introduisent délibérément des tests rouges. C'est ce qui
distingue une règle appliquée d'une règle affichée : si le gardien est écrit
après la correction, rien ne prouve qu'il attrape quoi que ce soit.

## Hors périmètre

- Aucune couche `platform/`. Les règles la mentionnent, le test gère son
  absence, rien ne la réclame.
- Aucun découpage de `RouteNetwork`. La granularité des ports n'est pas un
  critère de clean architecture ; ce port est cohérent et a deux
  implémentations utiles.
- Aucun changement de l'API publique. La route `GET /v1/loops/{id}` existe
  déjà et sert la boucle en GPX ; une variante rendant du JSON, dont le front
  a besoin, appartient au chantier suivant.

## Risques

**Le témoin JSON pourrait ne pas être reproductible.** La génération est
déterministe à requête égale — c'est la propriété qui permet déjà de régénérer
un GPX depuis son identifiant — mais `built_at` dans `/v1/regions` porte une
date. Le témoin doit donc neutraliser ce champ, sans neutraliser le reste.

**Le format de `graph.bin` est le point le plus dangereux du chantier.** Un
artefact illisible ne se voit pas en test unitaire si le test écrit et relit
avec le même code. Le témoin doit être un fichier produit *avant* le
refactoring, versionné dans `testdata/`.

**Le déplacement de `Request` touche beaucoup de fichiers.** C'est le seul point
du chantier où le diff sera large. Il est placé en dernier pour que tout le
reste soit déjà vert quand il arrive.

**Le gardien des tags pourrait être trop zélé.** Il ne doit inspecter que les
tags de structure de `internal/domain`, pas les chaînes littérales ni les
commentaires. Lecture de l'AST, pas d'expression régulière sur le texte.
