export type TimePoint = { timestamp: number; metric: string; tags: Record<string, string>; value: number };

export class TimeSeriesStore {
  private readonly series = new Map<string, TimePoint[]>();

  ingest(point: TimePoint): void {
    const key = seriesKey(point.metric, point.tags);
    const points = this.series.get(key) ?? [];
    points.push(point);
    points.sort((a, b) => a.timestamp - b.timestamp);
    this.series.set(key, points);
  }

  query(metric: string, tags: Record<string, string>, start: number, end: number): TimePoint[] {
    return (this.series.get(seriesKey(metric, tags)) ?? []).filter((p) => p.timestamp >= start && p.timestamp <= end);
  }

  average(metric: string, tags: Record<string, string>, start: number, end: number): number | null {
    const points = this.query(metric, tags, start, end);
    return points.length === 0 ? null : points.reduce((sum, p) => sum + p.value, 0) / points.length;
  }
}

function seriesKey(metric: string, tags: Record<string, string>): string {
  return `${metric}|${Object.keys(tags).sort().map((key) => `${key}=${tags[key]}`).join(",")}`;
}

function main(): void {
  const store = new TimeSeriesStore();
  store.ingest({ timestamp: 1, metric: "cpu", tags: { host: "a" }, value: 0.7 });
  store.ingest({ timestamp: 2, metric: "cpu", tags: { host: "a" }, value: 0.9 });
  console.log(store.average("cpu", { host: "a" }, 1, 2));
}

if (require.main === module) main();
