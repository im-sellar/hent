package domain

// Provenance dit d'où viennent les données d'un graphe et avec quelle
// configuration il a été construit. C'est ce qui rend le build reproductible,
// donc ce qui honore l'obligation de partage à l'identique de l'ODbL : sans
// elle, personne ne peut refaire l'artefact à partir des mêmes sources.
//
// Le type ne porte aucun tag de sérialisation. L'en-tête de graph.bin et la
// réponse de /v1/regions ont chacun leur propre représentation, définie par
// l'adaptateur qui la produit.
type Provenance struct {
	BuiltAt    string
	Sources    []Source
	ConfigHash string
}

// Source identifie un fichier d'entrée par son empreinte, pour qu'un tiers
// puisse vérifier qu'il refait le graphe à partir des mêmes octets.
type Source struct {
	Name      string
	File      string
	SHA256    string
	SizeBytes int64
}
