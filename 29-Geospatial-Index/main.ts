export type GeoPoint = { id: string; lat: number; lon: number };

export class GeoIndex {
  private readonly points: GeoPoint[] = [];

  insert(point: GeoPoint): void {
    this.points.push(point);
  }

  range(minLat: number, minLon: number, maxLat: number, maxLon: number): GeoPoint[] {
    return this.points.filter((p) => p.lat >= minLat && p.lat <= maxLat && p.lon >= minLon && p.lon <= maxLon);
  }

  nearest(lat: number, lon: number): GeoPoint | null {
    let best: GeoPoint | null = null;
    let bestDistance = Number.POSITIVE_INFINITY;
    for (const point of this.points) {
      const distance = haversine(lat, lon, point.lat, point.lon);
      if (distance < bestDistance) {
        best = point;
        bestDistance = distance;
      }
    }
    return best;
  }
}

function haversine(lat1: number, lon1: number, lat2: number, lon2: number): number {
  const toRad = (n: number) => n * Math.PI / 180;
  const dLat = toRad(lat2 - lat1);
  const dLon = toRad(lon2 - lon1);
  const a = Math.sin(dLat / 2) ** 2 + Math.cos(toRad(lat1)) * Math.cos(toRad(lat2)) * Math.sin(dLon / 2) ** 2;
  return 6371 * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
}

function main(): void {
  const index = new GeoIndex();
  index.insert({ id: "london", lat: 51.5, lon: -0.1 });
  console.log(index.nearest(51.49, -0.11));
}

if (require.main === module) main();
