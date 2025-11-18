// Copyright (c) 2018 Palantir Technologies. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package witchcraft

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"sort"
	"sync"
	"time"

	gometrics "github.com/palantir/go-metrics"
	"github.com/palantir/pkg/metrics"
	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-logging/wlog/metriclog/metric1log"
	"github.com/palantir/witchcraft-go-logging/wlog/wapp"
	"github.com/palantir/witchcraft-go-server/v3/config"
)

func defaultMetricTypeValuesBlocklist() map[string]map[string]struct{} {
	return map[string]map[string]struct{}{
		"histogram": {
			"min":    {},
			"mean":   {},
			"stddev": {},
			"p50":    {},
		},
		"meter": {
			"1m":   {},
			"5m":   {},
			"15m":  {},
			"mean": {},
		},
		"timer": {
			"1m":       {},
			"5m":       {},
			"15m":      {},
			"meanRate": {},
			"min":      {},
			"mean":     {},
			"stddev":   {},
			"p50":      {},
		},
	}
}

var (
	initTime = time.Now()
)

// seenMetricsSet pairs a set with a lock that can be used to control access to the set.
type seenMetricsSet struct {
	// protects access to seenSet
	lock sync.Mutex
	// records metrics that have been logged
	seenSet map[string]struct{}
}

func (s *Server[I, R]) initMetrics(ctx context.Context, installCfg config.Install) (rRegistry metrics.RootRegistry, rDeferFn func(), rErr error) {
	metricsRegistry := metrics.DefaultMetricsRegistry
	metricsEmitFreq := defaultMetricEmitFrequency
	if freq := installCfg.MetricsEmitFrequency; freq > 0 {
		metricsEmitFreq = freq
	}

	var collectGoRuntimeMetrics func()
	if !s.disableGoRuntimeMetrics {
		var ok bool
		collectGoRuntimeMetrics, ok = metrics.CaptureRuntimeMemStatsFunc(metricsRegistry)
		if !ok {
			return nil, nil, werror.Error("metricsRegistry does not support capturing runtime memory statistics")
		}
	}

	if s.metricTypeValuesBlocklist == nil {
		s.metricTypeValuesBlocklist = defaultMetricTypeValuesBlocklist()
	}

	// seenMetrics tracks the metrics that have been seen so far. Uses a lock to protect access because emitFn that
	// reads and writes this map can be called in the "emit" goroutine and in the goroutine that runs the rDeferFn
	// returned by this function.
	seenMetrics := &seenMetricsSet{
		// keys should be created by tagMapKey function
		seenSet: make(map[string]struct{}),
	}

	// start goroutine that logs metrics at the given frequency
	go wapp.RunWithRecoveryLogging(ctx, func(ctx context.Context) {
		tick := time.Tick(metricsEmitFreq)
		for {
			select {
			case <-tick:
				s.emitMetricsOnce(metricsRegistry, collectGoRuntimeMetrics, seenMetrics)
			case <-ctx.Done():
				return
			}
		}
	})

	return metricsRegistry, func() {
		// emit all metrics a final time on termination
		s.emitMetricsOnce(metricsRegistry, collectGoRuntimeMetrics, seenMetrics)
	}, nil
}

// emitMetricsOnce records updated values for runtime, uptime, and cardinality metrics,
// then logs all metrics stored in the registry using the Server's metricLogger.
func (s *Server[I, R]) emitMetricsOnce(metricsRegistry metrics.Registry, collectGoRuntimeMetrics func(), seenMetrics *seenMetricsSet) {
	if collectGoRuntimeMetrics != nil {
		collectGoRuntimeMetrics()
	}
	markServerUptimeMetric(metricsRegistry)
	s.markCardinalityMetric(metricsRegistry)

	metricsRegistry.Each(func(metricID string, tags metrics.Tags, metricVal metrics.MetricVal) {
		metricType := metricVal.Type()
		valuesToUse := s.collectMetricValues(metricID, metricVal)
		if len(valuesToUse) == 0 {
			// do not record metric if it does not have any values
			return
		}

		// note that s.metricLogger is used rather than extracting metric logger from the context to ensure that
		// most up-to-date metric logger is used (s.metricLogger may be updated during initialization).
		// s.metricLogger is not guaranteed to be non-nil at this point.
		localMetricLogger := s.metricLogger
		if localMetricLogger == nil {
			return
		}

		tagsMap := tags.ToMap()
		metricTagsParam := metric1log.Tags(tagsMap)

		// if metric is one for which a zero value should be logged on first observation, log a zero value if necessary
		if isZeroValueMetric(metricType) {
			zeroValuesLogged := s.logZeroValueMetric(localMetricLogger, metricID, metricType, tagsMap, seenMetrics, metricTagsParam)

			// if the zeroValueLogged is equivalent to valuesToUse, then there's no need to log the metric again
			if zeroValuesLogged != nil && reflect.DeepEqual(zeroValuesLogged, valuesToUse) {
				return
			}
		}
		localMetricLogger.Metric(metricID, metricType, metric1log.Values(valuesToUse), metricTagsParam)
	})
}

func (s *Server[I, R]) collectMetricValues(metricName string, metricVal metrics.MetricVal) map[string]any {
	valuesToUse := make(map[string]any)
	s.visitMetricValues(metricName, metricVal, func(key string) {
		valuesToUse[key] = metricVal.Value(key)
	})
	return valuesToUse
}

// visitMetricValues calls the provided visit function for each key in the provided metric value.
// If the metric with the provided metricID is in the metrics blocklist, the visit function is not called for any keys.
// The visit function is not called for keys that are in the metric type value blocklist for the type.
func (s *Server[I, R]) visitMetricValues(metricID string, metricVal metrics.MetricVal, visit func(key string)) {
	if _, blocklisted := s.metricsBlocklist[metricID]; blocklisted {
		// skip emitting metric if it is blocklisted
		return
	}
	for key := range metricVal.Keys() {
		if s.metricTypeValuesBlocklist != nil {
			if disallowedKeysForType, ok := s.metricTypeValuesBlocklist[metricVal.Type()]; ok {
				if _, disallowed := disallowedKeysForType[key]; disallowed {
					continue
				}
			}
		}
		visit(key)
	}
}

// isZeroValueMetric returns true if the provided metric type is a type of metric for which a zero value should be
// emitted before a regular value is emitted. This is true for metrics for which analysis often requires determining
// when the value has changed: for such metrics, if a value for a given metric, type, and set of tags has not been
// emitted before, a zero-value version of the metric is emitted before the regular metric is emitted to ensure that a
// value change can be observed. Currently, this is done for "meter", "timer" and "histogram" types.
func isZeroValueMetric(metricType string) bool {
	return metricType == "meter" || metricType == "timer" || metricType == "histogram"
}

// zeroValuesForMetricType returns the metric values for the "zero" value of the metric of the given type. The provided
// metric type must be a valid type that returns true when provided to the "isZeroValueMetric" function: panics
// otherwise.
func (s *Server[I, R]) zeroValuesForMetricType(metricName string, metricType string) map[string]any {
	var zeroMetric any
	switch metricType {
	case "meter":
		zeroMetric = gometrics.NewMeter()
	case "timer":
		zeroMetric = gometrics.NewTimer()
	case "histogram":
		zeroMetric = gometrics.NewHistogram(metrics.DefaultSample())
	default:
		panic(fmt.Sprintf("should never happen: unsupported metricType %s", metricType))
	}
	return s.collectMetricValues(metricName, metrics.ToMetricVal(zeroMetric))
}

// logZeroValueMetric logs a "zero" value for metric with the provided metricID, metricType, and tags to the provided
// metricLogger if an entry for the metric has not yet been logged (which is determined based on whether or not an entry
// for the metric exists in the provided seenMetrics set). Returns the zero values of the logged metric. If a zero value
// is logged, an entry for the metric is added to seenMetrics.
func (s *Server[I, R]) logZeroValueMetric(
	metricLogger metric1log.Logger,
	metricID string,
	metricType string,
	tags map[string]string,
	seenMetrics *seenMetricsSet,
	metricTagsParam metric1log.Param,
) (zeroValuesLogged map[string]any) {

	// Acquire lock to ensure that map access is safe. Safe to lock for the entirety of the function rather than
	// selectively locking just the read and write of the map because this function will be called in a single goroutine
	// in all instances except when the server shuts down.
	seenMetrics.lock.Lock()
	defer seenMetrics.lock.Unlock()

	// if metric has been seen before, no need to log zero value
	mapKey := tagMapKey(metricID, metricType, tags)

	_, metricSeen := seenMetrics.seenSet[mapKey]
	if metricSeen {
		return nil
	}

	zeroValuesToUse := s.zeroValuesForMetricType(metricID, metricType)

	// if removing the disallowed keys causes the values to be empty, no need to log zero value
	if len(zeroValuesToUse) == 0 {
		return nil
	}

	// metric not seen before: emit zero-value and record
	metricLogger.Metric(metricID, metricType, metric1log.Values(zeroValuesToUse), metricTagsParam)
	seenMetrics.seenSet[mapKey] = struct{}{}
	return zeroValuesToUse
}

// tagMapKey returns a string that provides a unique key for the provided metricID, metricType and tagsMap. The content
// of the map is put into a slice where each element is a string of the form "{key}:{value}" and the slice is then
// sorted. The tags map (rather than slice) is used to construct the key because the map is what the metric logger
// ultimately emits.
func tagMapKey(metricID, metricType string, tagsMap map[string]string) string {
	tagsSlice := make([]string, len(tagsMap))
	idx := 0
	for k, v := range tagsMap {
		tagsSlice[idx] = fmt.Sprintf("%s:%s", k, v)
		idx++
	}
	sort.Strings(tagsSlice)
	return fmt.Sprintf("%s:%s:%v", metricID, metricType, tagsSlice)
}

func markServerUptimeMetric(metricsRegistry metrics.Registry) {
	metricsRegistry.Gauge(
		"server.uptime",
		metrics.MustNewTag("go_version", runtime.Version()),
		metrics.MustNewTag("go_os", runtime.GOOS),
		metrics.MustNewTag("go_arch", runtime.GOARCH),
	).Update(time.Since(initTime).Microseconds())
}

func (s *Server[I, R]) markCardinalityMetric(metricsRegistry metrics.Registry) {
	var count int64
	metricsRegistry.Each(func(metricName string, tags metrics.Tags, val metrics.MetricVal) {
		s.visitMetricValues(metricName, val, func(key string) {
			count++
		})
	})
	metricsRegistry.Gauge("server.metric_cardinality").Update(count)
}
