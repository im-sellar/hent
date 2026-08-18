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
}
