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
	"runtime"
	"time"

	gometrics "github.com/palantir/go-metrics"
	"github.com/palantir/pkg/metrics"
	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-logging/wlog/metriclog/metric1log"
	"github.com/palantir/witchcraft-go-logging/wlog/wapp"
	"github.com/palantir/witchcraft-go-server/v2/config"
)

func defaultMetricTypeValuesBlacklist() map[string]map[string]struct{} {
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

func (s *Server) initMetrics(ctx context.Context, installCfg config.Install) (rRegistry metrics.RootRegistry, rDeferFn func(), rErr error) {
	metricsRegistry := metrics.DefaultMetricsRegistry
	metricsEmitFreq := defaultMetricEmitFrequency
	if freq := installCfg.MetricsEmitFrequency; freq > 0 {
		metricsEmitFreq = freq
	}

	initServerUptimeMetric(ctx, metricsRegistry)

	// start routine that capture Go runtime metrics
	if !s.disableGoRuntimeMetrics {
		if ok := metrics.CaptureRuntimeMemStatsWithContext(ctx, metricsRegistry, metricsEmitFreq); !ok {
			return nil, nil, werror.Error("metricsRegistry does not support capturing runtime memory statistics")
		}
	}

	metricTypeValuesBlacklist := s.metricTypeValuesBlacklist
	if metricTypeValuesBlacklist == nil {
		metricTypeValuesBlacklist = defaultMetricTypeValuesBlacklist()
	}

	// records metrics that have been logged. Key consists of "{metricID}-{metricType}-{tags}".
	seenMetrics := make(map[string]struct{})

	emitFn := func(metricID string, tags metrics.Tags, metricVal metrics.MetricVal) {
		if _, blackListed := s.metricsBlacklist[metricID]; blackListed {
			// skip emitting metric if it is blacklisted
			return
		}

		valuesToUse := metricVal.Values()
		metricType := metricVal.Type()
		removeDisallowedKeys(metricType, valuesToUse, metricTypeValuesBlacklist)
		if len(valuesToUse) == 0 {
			// do not record metric if it does not have any values
			return
		}

		// note that s.metricLogger is used rather than extracting metric logger from the context to ensure that
		// most up-to-date metric logger is used (s.metricLogger may be updated during initialization).
		// s.metricLogger is not guaranteed to be non-nil at this point.
		if s.metricLogger != nil {
			metricTagsParam := metric1log.Tags(tags.ToMap())

			if metricType == "meter" || metricType == "timer" || metricType == "histogram" {
				mapKey := fmt.Sprintf("%s:%s:%v", metricID, metricType, tags)
				if _, ok := seenMetrics[mapKey]; !ok {
					var zeroMetric interface{}
					switch metricType {
					case "meter":
						zeroMetric = gometrics.NewMeter()
					case "timer":
						zeroMetric = gometrics.NewTimer()
					case "histogram":
						zeroMetric = gometrics.NewHistogram(metrics.DefaultSample())
					}

					zeroValuesToUse := metrics.ToMetricVal(zeroMetric).Values()
					removeDisallowedKeys(metricType, zeroValuesToUse, metricTypeValuesBlacklist)
					if len(zeroValuesToUse) != 0 {
						fmt.Println("logging zero val for", mapKey)
						// metric not seen before: emit zero-value and record
						s.metricLogger.Metric(metricID, metricType, metric1log.Values(zeroValuesToUse), metricTagsParam)
						seenMetrics[mapKey] = struct{}{}
					}
				}
			}
			s.metricLogger.Metric(metricID, metricType, metric1log.Values(valuesToUse), metricTagsParam)
		}
	}

	// start goroutine that logs metrics at the given frequency
	go wapp.RunWithRecoveryLogging(ctx, func(ctx context.Context) {
		metrics.RunEmittingRegistry(ctx, metricsRegistry, metricsEmitFreq, emitFn)
	})

	return metricsRegistry, func() {
		// emit all metrics a final time on termination
		metricsRegistry.Each(emitFn)
	}, nil
}

func removeDisallowedKeys(metricType string, metricVals map[string]interface{}, disallowedKeys map[string]map[string]struct{}) {
	disallowedKeysForType, ok := disallowedKeys[metricType]
	if ! ok {
		return
	}
	// remove disallowed keys
	for blacklistedKey := range disallowedKeysForType {
		delete(metricVals, blacklistedKey)
	}
}

func initServerUptimeMetric(ctx context.Context, metricsRegistry metrics.Registry) {
	ctx = metrics.WithRegistry(ctx, metricsRegistry)
	ctx = metrics.AddTags(ctx, metrics.MustNewTag("go_version", runtime.Version()))
	ctx = metrics.AddTags(ctx, metrics.MustNewTag("go_os", runtime.GOOS))
	ctx = metrics.AddTags(ctx, metrics.MustNewTag("go_arch", runtime.GOARCH))

	metrics.FromContext(ctx).Gauge("server.uptime").Update(int64(time.Since(initTime) / time.Microsecond))

	// start goroutine that updates the uptime metric
	go wapp.RunWithRecoveryLogging(ctx, func(ctx context.Context) {
		t := time.NewTicker(5.0 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				metrics.FromContext(ctx).Gauge("server.uptime").Update(int64(time.Since(initTime) / time.Microsecond))
			}
		}
	})
}
