package main

import (
	"os"
	"sort"
	"strings"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// TimePoint represents a single data point in a time series. It includes a
// Unix timestamp, a metric name, optional key-value tags for dimensionality,
// and a float64 value.
type TimePoint struct {
	Timestamp int64
	Metric    string
	Tags      map[string]string
	Value     float64
}

// TimeSeriesStore is an in-memory time series database that organizes data
// points by metric+tag combination. It supports ingestion with automatic
// timestamp ordering and range/aggregation queries.
// Time complexity: O(n log n) per ingest (due to sort), O(k) per query
// where k is the number of matching points.
type TimeSeriesStore struct{ series map[string][]TimePoint }

// NewTimeSeriesStore creates an empty time series store.
// Time complexity: O(1).
func NewTimeSeriesStore() *TimeSeriesStore {
	log.Debug("Created TimeSeriesStore")
	return &TimeSeriesStore{series: make(map[string][]TimePoint)}
}

// Ingest adds a data point to the store and re-sorts the series by timestamp.
// Each unique combination of metric+tags forms its own series.
// Time complexity: O(n log n) due to sort after append.
func (s *TimeSeriesStore) Ingest(point TimePoint) {
	defer log.Operation("TimeSeriesStore.Ingest", "metric=%s ts=%d value=%.2f", point.Metric, point.Timestamp, point.Value)()
	key := seriesKey(point.Metric, point.Tags)
	s.series[key] = append(s.series[key], point)
	sort.Slice(s.series[key], func(i, j int) bool { return s.series[key][i].Timestamp < s.series[key][j].Timestamp })
	log.Debug("Series %q now has %d points", key, len(s.series[key]))
}

// Query retrieves all data points for a given metric+tag combination within
// the inclusive time range [start, end].
// Time complexity: O(k) where k is the number of points in the series.
func (s *TimeSeriesStore) Query(metric string, tags map[string]string, start, end int64) []TimePoint {
	defer log.Operation("TimeSeriesStore.Query", "metric=%s range=[%d,%d]", metric, start, end)()
	var result []TimePoint
	for _, point := range s.series[seriesKey(metric, tags)] {
		if point.Timestamp >= start && point.Timestamp <= end {
			result = append(result, point)
		}
	}
	log.Info("Returned %d points", len(result))
	return result
}

// Average computes the arithmetic mean of all data points for a given
// metric+tag combination within the time range [start, end]. Returns 0 and
// false if no points match.
// Time complexity: O(k) where k is the number of matching points.
func (s *TimeSeriesStore) Average(metric string, tags map[string]string, start, end int64) (float64, bool) {
	defer log.Operation("TimeSeriesStore.Average", "metric=%s range=[%d,%d]", metric, start, end)()
	points := s.Query(metric, tags, start, end)
	if len(points) == 0 {
		log.Warn("No data points for %s in range [%d, %d]", metric, start, end)
		return 0, false
	}
	sum := 0.0
	for _, point := range points {
		sum += point.Value
	}
	avg := sum / float64(len(points))
	log.Info("Average: %.4f over %d points", avg, len(points))
	return avg, true
}

// seriesKey generates a deterministic string key from a metric name and tag
// map. Tags are sorted alphabetically to ensure consistent key generation.
// Time complexity: O(t log t) where t is the number of tags.
func seriesKey(metric string, tags map[string]string) string {
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, key := range keys {
		parts[i] = key + "=" + tags[key]
	}
	return metric + "|" + strings.Join(parts, ",")
}

func main() {
	defer log.Operation("main", "Running Time Series Store demo")()
	defer log.Info("Time Series Store demo completed")

	logger.Section("Time Series Store — Infrastructure Monitoring")
	log.Info("Simulating a monitoring system ingesting CPU, memory, and disk metrics")
	log.Info("Each metric is tagged by host and region for multi-dimensional queries")

	store := NewTimeSeriesStore()

	logger.Section("Data Ingestion — Simulating 1-Minute Collector Intervals")
	points := []TimePoint{
		{Timestamp: 1, Metric: "cpu", Tags: map[string]string{"host": "web-01", "region": "us-east"}, Value: 0.45},
		{Timestamp: 2, Metric: "cpu", Tags: map[string]string{"host": "web-01", "region": "us-east"}, Value: 0.72},
		{Timestamp: 3, Metric: "cpu", Tags: map[string]string{"host": "web-01", "region": "us-east"}, Value: 0.38},
		{Timestamp: 1, Metric: "cpu", Tags: map[string]string{"host": "web-02", "region": "us-west"}, Value: 0.61},
		{Timestamp: 2, Metric: "cpu", Tags: map[string]string{"host": "web-02", "region": "us-west"}, Value: 0.55},
		{Timestamp: 1, Metric: "mem", Tags: map[string]string{"host": "web-01", "region": "us-east"}, Value: 0.82},
		{Timestamp: 2, Metric: "mem", Tags: map[string]string{"host": "web-01", "region": "us-east"}, Value: 0.79},
		{Timestamp: 3, Metric: "mem", Tags: map[string]string{"host": "web-01", "region": "us-east"}, Value: 0.86},
		{Timestamp: 1, Metric: "disk", Tags: map[string]string{"host": "web-01", "region": "us-east"}, Value: 0.33},
	}
	start := time.Now()
	for _, point := range points {
		store.Ingest(point)
	}
	log.Info("Ingested %d data points in %v", len(points), time.Since(start))

	logger.Section("Query — Time-Range Aggregation for Alerting")
	avgCPU, ok := store.Average("cpu", map[string]string{"host": "web-01", "region": "us-east"}, 1, 3)
	if ok {
		log.Info("CPU avg for web-01 (us-east): %.4f", avgCPU)
		if avgCPU > 0.5 {
			log.Warn("CPU threshold exceeded! Avg=%.4f > 0.5", avgCPU)
		}
	}

	avgMem, ok := store.Average("mem", map[string]string{"host": "web-01", "region": "us-east"}, 1, 3)
	if ok {
		log.Info("Memory avg for web-01 (us-east): %.4f", avgMem)
		if avgMem > 0.8 {
			log.Warn("Memory pressure detected! Avg=%.4f > 0.8", avgMem)
		}
	}

	logger.Section("Query — Point-in-Time Retrieval (Debugging)")
	cpuPoints := store.Query("cpu", map[string]string{"host": "web-02", "region": "us-west"}, 1, 2)
	for _, p := range cpuPoints {
		log.Info("cpu@host=web-02 ts=%d value=%.2f", p.Timestamp, p.Value)
	}

	logger.Section("Edge Case — Query on Non-Existent Series")
	_, ok = store.Average("network", map[string]string{"host": "web-01", "region": "us-east"}, 1, 3)
	if !ok {
		log.Warn("network metric not found — alerting pipeline skips (normal behavior)")
	}

	logger.Section("Stats Summary")
	logger.KeyValue("total_points_ingested", len(points))
	logger.KeyValue("unique_series", len(store.series))
	logger.KeyValue("cpu_avg_web01", avgCPU)
	logger.KeyValue("mem_avg_web01", avgMem)
}
