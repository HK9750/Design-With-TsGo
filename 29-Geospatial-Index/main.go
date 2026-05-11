package main

import (
	"math"
	"os"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// GeoPoint represents a geospatial location identified by a string ID with
// latitude and longitude coordinates in decimal degrees.
type GeoPoint struct {
	ID  string
	Lat float64
	Lon float64
}

// GeoIndex is a flat geospatial index supporting insertion, bounding-box range
// queries, and nearest-neighbor search using the Haversine formula for great-
// circle distance calculation.
// Time complexity: O(n) for Range and Nearest queries over all points.
type GeoIndex struct{ points []GeoPoint }

// Insert adds a geospatial point to the index.
// Time complexity: O(1) amortized.
func (g *GeoIndex) Insert(point GeoPoint) {
	defer log.Operation("GeoIndex.Insert", "id=%s lat=%.4f lon=%.4f", point.ID, point.Lat, point.Lon)()
	g.points = append(g.points, point)
	log.Debug("Index now contains %d points", len(g.points))
}

// Range returns all points within the bounding box defined by minimum and
// maximum latitude/longitude. Returns an empty slice if no points match.
// Time complexity: O(n) where n is the number of indexed points.
func (g *GeoIndex) Range(minLat, minLon, maxLat, maxLon float64) []GeoPoint {
	defer log.Operation("GeoIndex.Range", "bbox=[%.4f,%.4f]x[%.4f,%.4f]", minLat, minLon, maxLat, maxLon)()
	var result []GeoPoint
	for _, point := range g.points {
		if point.Lat >= minLat && point.Lat <= maxLat && point.Lon >= minLon && point.Lon <= maxLon {
			result = append(result, point)
		}
	}
	if len(result) == 0 {
		log.Warn("No points found in bounding box")
	} else {
		log.Info("Found %d points in bounding box", len(result))
	}
	return result
}

// Nearest finds the closest point to the given coordinates using the Haversine
// great-circle distance formula. Returns the point and true, or an empty
// GeoPoint and false if the index is empty.
// Time complexity: O(n) where n is the number of indexed points.
func (g *GeoIndex) Nearest(lat, lon float64) (GeoPoint, bool) {
	defer log.Operation("GeoIndex.Nearest", "target=[%.4f,%.4f]", lat, lon)()
	if len(g.points) == 0 {
		log.Warn("Index is empty, no nearest point available")
		return GeoPoint{}, false
	}
	best := g.points[0]
	bestDistance := haversine(lat, lon, best.Lat, best.Lon)
	log.Debug("Initial best: %s at %.2f km", best.ID, bestDistance)
	for _, point := range g.points[1:] {
		distance := haversine(lat, lon, point.Lat, point.Lon)
		if distance < bestDistance {
			best = point
			bestDistance = distance
			log.Debug("New best: %s at %.2f km", best.ID, bestDistance)
		}
	}
	log.Info("Nearest: %s at %.2f km", best.ID, bestDistance)
	return best, true
}

// haversine computes the great-circle distance in kilometers between two
// geographic coordinates using the Haversine formula. Assumes Earth radius
// of 6371 km.
// Time complexity: O(1).
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	toRad := func(v float64) float64 { return v * math.Pi / 180 }
	dLat, dLon := toRad(lat2-lat1), toRad(lon2-lon1)
	a := math.Pow(math.Sin(dLat/2), 2) + math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Pow(math.Sin(dLon/2), 2)
	return 6371 * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func main() {
	defer log.Operation("main", "Running Geospatial Index demo")()
	defer log.Info("Geospatial Index demo completed")

	logger.Section("Geospatial Index — Ride-Hailing Service Locator")
	log.Info("Simulating a ride-hailing platform matching riders to nearby drivers")
	log.Info("Each point represents a driver with their real-time GPS location")

	index := &GeoIndex{}

	logger.Section("Driver Fleet Registration")
	start := time.Now()
	index.Insert(GeoPoint{ID: "driver-001", Lat: 51.5074, Lon: -0.1278})  // London
	index.Insert(GeoPoint{ID: "driver-002", Lat: 51.5150, Lon: -0.1150})  // Near London
	index.Insert(GeoPoint{ID: "driver-003", Lat: 48.8566, Lon: 2.3522})   // Paris
	index.Insert(GeoPoint{ID: "driver-004", Lat: 40.7128, Lon: -74.0060}) // New York
	index.Insert(GeoPoint{ID: "driver-005", Lat: 51.5200, Lon: -0.1000})  // East London
	index.Insert(GeoPoint{ID: "driver-006", Lat: 51.4900, Lon: -0.1300})  // South London
	log.Info("Registered %d drivers in %v", 6, time.Since(start))

	logger.Section("Query 1 — Nearest Driver for a Rider in Central London")
	riderLat, riderLon := 51.5000, -0.1200
	log.Info("Rider requests pickup at [%.4f, %.4f]", riderLat, riderLon)
	nearest, ok := index.Nearest(riderLat, riderLon)
	if ok {
		distance := haversine(riderLat, riderLon, nearest.Lat, nearest.Lon)
		log.Info("Matched: %s is %.2f km away", nearest.ID, distance)
	}

	logger.Section("Query 2 — Bounding Box: All Drivers in Greater London")
	londonMinLat, londonMinLon := 51.45, -0.20
	londonMaxLat, londonMaxLon := 51.55, -0.05
	log.Info("Searching London area: [%.4f,%.4f] to [%.4f,%.4f]",
		londonMinLat, londonMinLon, londonMaxLat, londonMaxLon)
	londonDrivers := index.Range(londonMinLat, londonMinLon, londonMaxLat, londonMaxLon)
	for _, d := range londonDrivers {
		log.Info("  %s at [%.4f, %.4f]", d.ID, d.Lat, d.Lon)
	}

	logger.Section("Query 3 — Bounding Box: Drivers in New York Area")
	nyDrivers := index.Range(40.5, -74.5, 41.0, -73.5)
	for _, d := range nyDrivers {
		log.Info("  %s at [%.4f, %.4f]", d.ID, d.Lat, d.Lon)
	}

	logger.Section("Edge Case — Nearest Driver When Fleet is Nearby")
	driver, _ := index.Nearest(51.5100, -0.1150)
	log.Info("Rider near London Bridge: closest driver is %s", driver.ID)

	logger.Section("Stats Summary")
	logger.KeyValue("total_drivers_indexed", len(index.points))
	logger.KeyValue("london_drivers", len(londonDrivers))
	logger.KeyValue("ny_drivers", len(nyDrivers))
	logger.KeyValue("nearest_driver", nearest.ID)
}
