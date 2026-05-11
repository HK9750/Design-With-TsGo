# Geospatial Index

> **A spatial data structure for answering "what's near me?" — stores locations with latitude/longitude coordinates and supports bounding-box range queries and nearest-neighbor search via Haversine distance.**

## The Problem It Solves

You're building a food delivery app. A hungry user opens the app and expects to see restaurants within 3 km, sorted by distance. Your database has 500,000 restaurant locations, and this query runs 10,000 times per minute at peak dinner hours. The naive approach — compute the distance from the user to every restaurant, sort, and return the top 50 — is O(n) distance calculations per query. Each Haversine formula requires ~10 trigonometric operations. At 500,000 × 10,000 queries/minute, you'd need ~8 trillion trig operations per minute. Even with vectorized math, that's not feasible.

The fundamental challenge is that latitude and longitude form a 2D continuous space, while traditional database indexes (B-Tree, hash) are designed for 1D ordered keys. You can't just index on latitude — two points with identical latitude could be on opposite sides of the globe. You can't just box a B-Tree with `WHERE lat BETWEEN x AND y AND lon BETWEEN a AND b` efficiently either, because a 2D range requires a 2D index.

A geospatial index solves this by organizing points spatially. The simplest approach — and the one our implementation uses — is a *flat index with bounding-box filtering*: for range queries, we filter points by lat/lon bounds in a single pass. For nearest-neighbor, we compute Haversine distances exhaustively. This is O(n) but works well up to ~100,000 points. Production systems use more sophisticated structures: **R-trees** (PostGIS) that partition space into nested bounding rectangles, **Geohashes** (Redis GEO) that encode 2D coordinates into 1D strings suitable for B-Tree range scans, and **S2 cells** (Google Maps) that project the sphere onto a cube and use Hilbert curves for spatial locality.

## Architecture & Internals

```
                   GEOGRAPHIC COORDINATES
    ┌─────────────────────────────────────────────────────┐
    │                                                     │
    │  Latitude:  -90° (South Pole) to +90° (North Pole)  │
    │  Longitude: -180° (West) to +180° (East)            │
    │                                                     │
    │  London:      (51.5074, -0.1278)                   │
    │  Paris:       (48.8566,  2.3522)                   │
    │  New York:    (40.7128, -74.0060)                  │
    │  East London: (51.5200, -0.1000)                   │
    │                                                     │
    └─────────────────────────────────────────────────────┘

                  BOUNDING BOX QUERY
    Query: Range(minLat=51.45, minLon=-0.20,
                 maxLat=51.55, maxLon=-0.05)
    ┌─────────────────────────────────────────────────────┐
    │                                                     │
    │  maxLat (51.55) ┌────────────────────┐              │
    │                 │   driver-002 ★      │              │
    │                 │  (51.5150,-0.1150)  │              │
    │                 │                     │              │
    │                 │   driver-001 ★      │ driver-005 ★ │
    │                 │  (51.5074,-0.1278)  │(51.520,-0.10)│
    │                 │                     │              │
    │                 │   driver-006 ★      │              │
    │  minLat (51.45) │  (51.4900,-0.1300)  │              │
    │                 └────────────────────┘              │
    │                minLon (-0.20)    maxLon (-0.05)     │
    │                                                     │
    │  Excluded: driver-003 (Paris), driver-004 (NY)     │
    │                                                     │
    └─────────────────────────────────────────────────────┘

                HAVERSINE DISTANCE
    d = 2 × R × arcsin(√(hav(Δlat) + cos(lat1)×cos(lat2)×hav(Δlon)))

    where:
      hav(θ) = sin²(θ/2)
      R = 6371 km (Earth's mean radius)

    Example: London (51.5074, -0.1278) to Paris (48.8566, 2.3522)
      → ~343 km

    ┌─────────────────────────────────────────────────────┐
    │  Why Haversine and not Euclidean?                   │
    │                                                     │
    │  Euclidean (√(Δx² + Δy²)) treats lat/lon as a       │
    │  flat plane. At the equator, 1° lon ≈ 111 km.       │
    │  At London (51°N), 1° lon ≈ 69 km. The error        │
    │  grows with latitude — useless for global services. │
    │                                                     │
    │  Haversine accounts for Earth's curvature —         │
    │  accurate to within 0.5% for any distance.          │
    │                                                     │
    └─────────────────────────────────────────────────────┘
```

**Key types:**

- `GeoPoint` (`main.go:15`): A simple struct with an `ID` string for identification and `Lat`/`Lon` float64 coordinates. In production, you'd also store a payload (restaurant name, driver status, vehicle type) alongside the location.
- `GeoIndex` (`main.go:25`): A flat slice of `GeoPoint` values — the simplest possible spatial index. This is a *linear scan* index: every query scans all points. This works for up to ~100,000 points (roughly 0.8 MB of data) before the O(n) cost becomes noticeable.

## Production Use Cases

- **PostGIS**: The gold standard for geospatial queries. A PostgreSQL extension that adds geometry types, R-tree spatial indexes (GiST), and hundreds of functions (ST_Distance, ST_Contains, ST_Intersects). Used by OpenStreetMap, Foursquare, and every government GIS system.
- **Redis GEO**: Redis's geospatial commands (GEOADD, GEORADIUS, GEODIST) backed by a **Geohash** implementation. Geohash encodes lat/lon into a 1D string where nearby locations share string prefixes, enabling B-Tree range scans for proximity. Lightweight and ideal for caching nearby drivers or restaurants.
- **MongoDB Geospatial Queries**: Supports 2dsphere indexes (spherical geometry) and 2d indexes (flat geometry). Used by companies like Foursquare and The Weather Channel for location-based services. Queries like `$near`, `$geoWithin`, and `$geoIntersects` are backed by S2-style cell indexes.
- **Uber H3**: A hexagonal hierarchical geospatial indexing system. Divides the Earth into hexagons at 16 resolutions (from ~4 million km² to ~0.5 m²). Used by Uber for surge pricing (aggregate supply/demand per hex), trip visualization, and ETA prediction.
- **Google Maps / S2 Geometry**: Google's S2 library projects the sphere onto a cube, then maps 2D cells to 1D Hilbert curve indices. Every Google Maps feature (points of interest, roads, buildings) is indexed in S2 cells for efficient proximity and containment queries.

## When to Use It

| Use Geospatial Index when... | Don't use when... |
|---|---|
| Queries filter by geographic bounds ("restaurants near me") | All locations share the same point (co-located data center) |
| Distance between points matters for ranking | Latitude/longitude are just metadata you never query by |
| You need nearest-neighbor or radius queries | You only need to display lat/lon on a map (no spatial queries) |
| Points move and need frequent reindexing (drivers, assets) | Points are static and you can precompute all distances |
| You have 1,000–100,000 points (our linear scan is fine) | You have millions of points (use R-tree, Geohash, or S2) |

**Flat index vs. spatial tree**: Our linear-scan index is O(n) per query. For 10,000 drivers, nearest-neighbor takes ~0.1ms — perfectly fine. For 10 million points of interest, it takes ~100ms — unacceptable. The threshold for switching to an R-tree or geohash index is roughly 50,000–100,000 points. Below that, the simplicity of a linear scan beats the overhead of tree maintenance.

**Haversine is the right formula for most apps**: There are more accurate formulas (Vincenty's, which accounts for Earth's ellipsoid shape) and simpler ones (Equirectangular approximation, for very short distances). Haversine hits the sweet spot: accurate to ~0.5% globally, numerically stable for both small and large distances (no catastrophic cancellation near antipodes), and computationally reasonable.

## Complexity Analysis

| Operation | Time | Space |
|---|---|---|
| Insert | O(1) amortized | O(1) per point |
| Range (bbox) | O(n) | O(matching points) |
| Nearest (Haversine) | O(n) | O(1) |
| Haversine formula | O(1) | O(1) |
| Space (total) | O(n) | ~48 bytes per point |

## Implementation Deep Dive

### 1. Haversine Distance Formula (`main.go:83-88`)

The `haversine` function computes great-circle distance in kilometers. It converts degrees to radians, computes the differences, and applies the formula: `a = sin²(Δlat/2) + cos(lat1) × cos(lat2) × sin²(Δlon/2)`, then `d = 2 × R × atan2(√a, √(1-a))`. The `atan2` form is numerically stable for all distances — unlike `2 × R × asin(√a)` which can have precision issues for very small distances. We use Earth's mean radius of 6371 km. For applications requiring higher precision, you'd use the WGS84 ellipsoid parameters (equatorial radius 6378.137 km, flattening 1/298.257223563) with Vincenty's formula.

### 2. Bounding-Box Range Query (`main.go:38-52`)

`Range()` performs a single-pass linear scan, checking four conditions per point: `lat >= minLat && lat <= maxLat && lon >= minLon && lon <= maxLon`. This is a simple rectangular filter. It doesn't account for the fact that longitude lines converge near the poles (a 1° × 1° box near the North Pole covers far less area than near the equator). Production systems use spherical geometry for containment tests (ST_Contains with a polygon), but for most mid-latitude applications, the rectangular approximation works well.

### 3. Brute-Force Nearest Neighbor (`main.go:58-77`)

`Nearest()` initializes the best candidate as the first point, then iterates over all remaining points, computing Haversine distance for each one. When a closer point is found, it replaces the best candidate. This is O(n) distance calculations — each requiring ~10 trigonometric operations. For our 6-driver demo, this is trivial. For a production ride-hailing service with 100,000 drivers, this takes about 10ms per query — borderline but usable. Beyond 100,000, you need a spatial tree (R-tree) that prunes regions to reduce the search to O(log n) distance calculations.

## Running the Demo

```bash
go run ./29-Geospatial-Index/
```

The demo simulates a ride-hailing platform with 6 drivers in London, Paris, and New York. It demonstrates nearest-driver matching for a rider in central London, bounding-box queries to find all drivers in the Greater London area, and bounding-box queries for the New York area.

## Further Reading

- **Google S2 Geometry Library (s2geometry.io)** — The most sophisticated open-source geospatial library. Covers S2 cell hierarchy, Hilbert curve space-filling curves, and how S2 is used internally at Google for Maps, Street View, and location services.
- **Uber H3 Documentation (h3geo.org)** — Explains the hexagonal grid system: why hexagons (uniform neighbors, better for binning than squares), resolution levels (0-15), and how Uber uses H3 for market analysis, dispatch optimization, and data visualization at global scale.

---

*Part of the Design-With-TsGo system design curriculum*
