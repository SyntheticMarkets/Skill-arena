package observability

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var durationBuckets = [...]float64{
	0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
}

type durationHistogram struct {
	count   uint64
	sum     float64
	buckets [len(durationBuckets)]uint64
}

// PrometheusRecorder records bounded-cardinality component durations.
type PrometheusRecorder struct {
	mu         sync.RWMutex
	durations  map[string]*durationHistogram
	httpCounts map[string]uint64
}

func NewPrometheusRecorder() *PrometheusRecorder {
	return &PrometheusRecorder{
		durations:  make(map[string]*durationHistogram),
		httpCounts: make(map[string]uint64),
	}
}

func (r *PrometheusRecorder) ObserveTiming(name string, duration time.Duration) {
	if r == nil || strings.TrimSpace(name) == "" {
		return
	}
	seconds := duration.Seconds()
	r.mu.Lock()
	histogram := r.durations[name]
	if histogram == nil {
		histogram = &durationHistogram{}
		r.durations[name] = histogram
	}
	histogram.count++
	histogram.sum += seconds
	for index, boundary := range durationBuckets {
		if seconds <= boundary {
			histogram.buckets[index]++
		}
	}
	r.mu.Unlock()
}

func (r *PrometheusRecorder) ObserveHTTPRequest(method string, status int, duration time.Duration) {
	if r == nil {
		return
	}
	statusClass := strconv.Itoa(status/100) + "xx"
	key := strings.ToUpper(method) + "\x00" + statusClass
	r.mu.Lock()
	r.httpCounts[key]++
	r.mu.Unlock()
	r.ObserveTiming("http.request."+strings.ToLower(method)+"."+statusClass, duration)
}

func (r *PrometheusRecorder) WritePrometheus(w io.Writer) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, err := fmt.Fprintln(w, "# HELP skill_arena_component_duration_seconds Internal component duration."); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# TYPE skill_arena_component_duration_seconds histogram"); err != nil {
		return err
	}
	names := make([]string, 0, len(r.durations))
	for name := range r.durations {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		histogram := r.durations[name]
		label := prometheusLabel(name)
		for index, boundary := range durationBuckets {
			if _, err := fmt.Fprintf(
				w,
				"skill_arena_component_duration_seconds_bucket{component=\"%s\",le=\"%g\"} %d\n",
				label, boundary, histogram.buckets[index],
			); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(
			w,
			"skill_arena_component_duration_seconds_bucket{component=\"%s\",le=\"+Inf\"} %d\n",
			label, histogram.count,
		); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(
			w, "skill_arena_component_duration_seconds_sum{component=\"%s\"} %g\n",
			label, histogram.sum,
		); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(
			w, "skill_arena_component_duration_seconds_count{component=\"%s\"} %d\n",
			label, histogram.count,
		); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(w, "# HELP skill_arena_http_requests_total HTTP requests by method and status class."); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# TYPE skill_arena_http_requests_total counter"); err != nil {
		return err
	}
	keys := make([]string, 0, len(r.httpCounts))
	for key := range r.httpCounts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		parts := strings.SplitN(key, "\x00", 2)
		if _, err := fmt.Fprintf(
			w,
			"skill_arena_http_requests_total{method=\"%s\",status_class=\"%s\"} %d\n",
			prometheusLabel(parts[0]), prometheusLabel(parts[1]), r.httpCounts[key],
		); err != nil {
			return err
		}
	}
	return nil
}

func prometheusLabel(value string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "\n", "\\n", "\"", "\\\"")
	return replacer.Replace(value)
}
