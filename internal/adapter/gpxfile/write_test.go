package gpxfile_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/im-sellar/hent/internal/adapter/gpxfile"
	"github.com/im-sellar/hent/internal/domain"
)

func TestWriteGPX(t *testing.T) {
	l := domain.Loop{
		Coords: []domain.Coord{
			{Lat: 48.1173, Lon: -1.6778},
			{Lat: 48.1200, Lon: -1.6700},
			{Lat: 48.1173, Lon: -1.6778},
		},
		LengthM: 1500,
	}

	var buf bytes.Buffer
	if err := gpxfile.Write(&buf, l, "Boucle test"); err != nil {
		t.Fatalf("Write : %v", err)
	}
	out := buf.String()

	for _, attendu := range []string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<gpx`, `creator="hent"`, `<trkseg>`,
		`lat="48.1173"`, `lon="-1.6778"`,
		`<name>Boucle test</name>`,
		`OpenStreetMap`, // l'attribution voyage avec le fichier
	} {
		if !strings.Contains(out, attendu) {
			t.Errorf("le GPX ne contient pas %q", attendu)
		}
	}

	if n := strings.Count(out, "<trkpt "); n != 3 {
		t.Errorf("%d points de trace, attendu 3", n)
	}
}

func TestWriteGPXBoucleVide(t *testing.T) {
	var buf bytes.Buffer
	if err := gpxfile.Write(&buf, domain.Loop{}, "vide"); err == nil {
		t.Fatal("exporter une boucle sans point doit échouer")
	}
}
