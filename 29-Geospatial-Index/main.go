package main

import (
	"fmt"
	"math"
)

type GeoPoint struct {
	ID  string
	Lat float64
	Lon float64
}

type GeoIndex struct{ points []GeoPoint }

func (g *GeoIndex) Insert(point GeoPoint) { g.points = append(g.points, point) }

func (g *GeoIndex) Range(minLat, minLon, maxLat, maxLon float64) []GeoPoint {
	var result []GeoPoint
	for _, point := range g.points {
		if point.Lat >= minLat && point.Lat <= maxLat && point.Lon >= minLon && point.Lon <= maxLon {
			result = append(result, point)
		}
	}
	return result
}

func (g *GeoIndex) Nearest(lat, lon float64) (GeoPoint, bool) {
	if len(g.points) == 0 {
		return GeoPoint{}, false
	}
	best := g.points[0]
	bestDistance := haversine(lat, lon, best.Lat, best.Lon)
	for _, point := range g.points[1:] {
		distance := haversine(lat, lon, point.Lat, point.Lon)
		if distance < bestDistance {
			best = point
			bestDistance = distance
		}
	}
	return best, true
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	toRad := func(v float64) float64 { return v * math.Pi / 180 }
	dLat, dLon := toRad(lat2-lat1), toRad(lon2-lon1)
	a := math.Pow(math.Sin(dLat/2), 2) + math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Pow(math.Sin(dLon/2), 2)
	return 6371 * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func main() {
	index := &GeoIndex{}
	index.Insert(GeoPoint{ID: "london", Lat: 51.5, Lon: -0.1})
	point, _ := index.Nearest(51.49, -0.11)
	fmt.Println(point.ID)
}
