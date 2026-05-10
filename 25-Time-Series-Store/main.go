package main

import (
	"fmt"
	"sort"
	"strings"
)

type TimePoint struct {
	Timestamp int64
	Metric    string
	Tags      map[string]string
	Value     float64
}

type TimeSeriesStore struct{ series map[string][]TimePoint }

func NewTimeSeriesStore() *TimeSeriesStore {
	return &TimeSeriesStore{series: make(map[string][]TimePoint)}
}

func (s *TimeSeriesStore) Ingest(point TimePoint) {
	key := seriesKey(point.Metric, point.Tags)
	s.series[key] = append(s.series[key], point)
	sort.Slice(s.series[key], func(i, j int) bool { return s.series[key][i].Timestamp < s.series[key][j].Timestamp })
}

func (s *TimeSeriesStore) Query(metric string, tags map[string]string, start, end int64) []TimePoint {
	var result []TimePoint
	for _, point := range s.series[seriesKey(metric, tags)] {
		if point.Timestamp >= start && point.Timestamp <= end {
			result = append(result, point)
		}
	}
	return result
}

func (s *TimeSeriesStore) Average(metric string, tags map[string]string, start, end int64) (float64, bool) {
	points := s.Query(metric, tags, start, end)
	if len(points) == 0 {
		return 0, false
	}
	sum := 0.0
	for _, point := range points {
		sum += point.Value
	}
	return sum / float64(len(points)), true
}

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
	store := NewTimeSeriesStore()
	store.Ingest(TimePoint{Timestamp: 1, Metric: "cpu", Tags: map[string]string{"host": "a"}, Value: 0.7})
	store.Ingest(TimePoint{Timestamp: 2, Metric: "cpu", Tags: map[string]string{"host": "a"}, Value: 0.9})
	avg, _ := store.Average("cpu", map[string]string{"host": "a"}, 1, 2)
	fmt.Println(avg)
}
