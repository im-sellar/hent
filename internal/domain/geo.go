// Package domain contient les types métier de hent.
// Il n'importe rien d'autre que la bibliothèque standard : c'est la couche
// dont tout le reste dépend, et qui ne dépend de rien.
package domain

import "math"

// Rayon moyen de la Terre (IUGG), en mètres.
const earthRadiusM = 6371008.8

type Coord struct {
	Lat, Lon float64
}

type BBox struct {
	Min, Max Coord
}

func (b BBox) Contains(c Coord) bool {
	return c.Lat >= b.Min.Lat && c.Lat <= b.Max.Lat &&
		c.Lon >= b.Min.Lon && c.Lon <= b.Max.Lon
}

// HaversineM retourne la distance orthodromique en mètres entre a et b.
func HaversineM(a, b Coord) float64 {
	lat1, lat2 := radians(a.Lat), radians(b.Lat)
	dLat := radians(b.Lat - a.Lat)
	dLon := radians(b.Lon - a.Lon)

	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)

	return 2 * earthRadiusM * math.Asin(math.Sqrt(h))
}

func radians(deg float64) float64 { return deg * math.Pi / 180 }

// mPerDegLat : longueur d'un degré de latitude, en mètres. Constante à la
// précision qui nous intéresse (le placement de waypoints tolère largement
// l'aplatissement terrestre).
const mPerDegLat = 111320.0

// Offset retourne le point situé à distM mètres de c dans la direction
// bearingRad, comptée en radians depuis le nord et dans le sens horaire.
func Offset(c Coord, distM, bearingRad float64) Coord {
	dLat := distM * math.Cos(bearingRad) / mPerDegLat
	dLon := distM * math.Sin(bearingRad) / (mPerDegLat * math.Cos(radians(c.Lat)))
	return Coord{Lat: c.Lat + dLat, Lon: c.Lon + dLon}
}
