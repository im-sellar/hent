// Package gpxfile sérialise une boucle au format GPX 1.1, lisible par les
// montres et applications de trace.
package gpxfile

import (
	"errors"
	"fmt"
	"io"

	"github.com/im-sellar/hent/internal/domain"
)

// Attribution : l'ODbL impose de créditer la source, et le fichier circule
// indépendamment de l'API — l'attribution doit donc voyager avec lui.
const attribution = "Données © les contributeurs OpenStreetMap, sous licence ODbL"

var ErrEmptyLoop = errors.New("boucle sans point, rien à exporter")

func Write(w io.Writer, l domain.Loop, name string) error {
	if len(l.Coords) == 0 {
		return ErrEmptyLoop
	}

	if _, err := fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="hent" xmlns="http://www.topografix.com/GPX/1/1">
  <metadata>
    <name>%s</name>
    <desc>%s — %.0f m</desc>
    <copyright author="OpenStreetMap contributors"><license>https://opendatacommons.org/licenses/odbl/</license></copyright>
  </metadata>
  <trk>
    <name>%s</name>
    <trkseg>
`, escape(name), attribution, l.LengthM, escape(name)); err != nil {
		return err
	}

	for _, c := range l.Coords {
		if _, err := fmt.Fprintf(w, "      <trkpt lat=\"%.7g\" lon=\"%.7g\"></trkpt>\n", c.Lat, c.Lon); err != nil {
			return err
		}
	}

	_, err := io.WriteString(w, "    </trkseg>\n  </trk>\n</gpx>\n")
	return err
}

func escape(s string) string {
	var out []rune
	for _, r := range s {
		switch r {
		case '&':
			out = append(out, []rune("&amp;")...)
		case '<':
			out = append(out, []rune("&lt;")...)
		case '>':
			out = append(out, []rune("&gt;")...)
		default:
			out = append(out, r)
		}
	}
	return string(out)
}
