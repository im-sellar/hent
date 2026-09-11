// Command graphbuild construit l'artefact graph.bin à partir d'un extrait
// OpenStreetMap. Il tourne hors ligne, sur le poste de développement : toutes
// les opérations coûteuses se paient ici, une fois, et jamais en production.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/im-sellar/hent/internal/adapter/network/csr"
	"github.com/im-sellar/hent/internal/adapter/osmsource"
)

func main() {
	in := flag.String("in", "", "extrait OpenStreetMap au format .osm.pbf")
	out := flag.String("out", "graph.bin", "artefact à produire")
	name := flag.String("source-name", "geofabrik/bretagne", "nom de la source, inscrit dans la provenance")
	flag.Parse()

	if *in == "" {
		log.Fatal("le drapeau -in est obligatoire")
	}
	if err := run(*in, *out, *name); err != nil {
		log.Fatal(err)
	}
}

func run(in, out, sourceName string) error {
	start := time.Now()

	log.Printf("lecture de %s", in)
	g, stats, err := osmsource.Read(context.Background(), in)
	if err != nil {
		return fmt.Errorf("lecture de l'extrait : %w", err)
	}
	log.Printf("%d tronçons retenus, %d nœuds, %d arêtes dirigées",
		stats.Ways, g.NumNodes(), g.NumEdges())

	digest, size, err := hashFile(in)
	if err != nil {
		return err
	}

	prov := csr.Provenance{
		BuiltAt: time.Now().UTC().Format(time.RFC3339),
		Sources: []csr.Source{{
			Name: sourceName, File: in, SHA256: digest, SizeBytes: size,
		}},
		ConfigHash: osmsource.ConfigHash(),
	}

	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := csr.Write(f, g, prov); err != nil {
		return fmt.Errorf("écriture de l'artefact : %w", err)
	}

	info, err := f.Stat()
	if err != nil {
		return err
	}
	log.Printf("%s écrit (%.1f Mo) en %s", out, float64(info.Size())/(1<<20), time.Since(start).Round(time.Second))
	return nil
}

func hashFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	h := sha256.New()
	size, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), size, nil
}

// Le hash de configuration est calculé par osmsource à partir du contenu réel
// de ses tables de classification : voir osmsource.ConfigHash.
