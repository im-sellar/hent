package domain

// NodeRef et EdgeRef sont des identifiants opaques manipulés par le domaine
// mais interprétés uniquement par l'adaptateur réseau. Ils permettent de
// désigner des arêtes (pour interdire leur réutilisation dans une boucle)
// sans que le domaine connaisse la structure du graphe.
type NodeRef uint32

type EdgeRef uint32

// Path est un chemin résolu entre deux nœuds.
type Path struct {
	Nodes   []NodeRef
	Edges   []EdgeRef
	Coords  []Coord
	LengthM float64
	Cost    float64

	// ExploredNodes : nœuds dépilés par la recherche. Donnée d'observabilité,
	// et non résultat métier — elle rend visible l'arbitrage du modèle de
	// coût : plus les pondérations s'écartent, moins l'heuristique informe, et
	// plus A* dérive vers Dijkstra.
	ExploredNodes int
}

// PathOptions porte les contraintes de recherche d'un chemin.
type PathOptions struct {
	// UsedEdges liste les arêtes déjà consommées par les segments précédents
	// d'une boucle. Leur coût est multiplié par ReuseFactor : c'est ce qui
	// évite qu'une boucle revienne sur ses pas, tout en laissant l'A* les
	// réemprunter quand il n'existe réellement pas d'alternative — un pont,
	// un col.
	UsedEdges   map[EdgeRef]struct{}
	ReuseFactor float64

	// MaxNodes plafonne l'exploration. Au-delà, la recherche est abandonnée
	// plutôt que de monopoliser le serveur. Zéro = valeur par défaut.
	MaxNodes int
}
