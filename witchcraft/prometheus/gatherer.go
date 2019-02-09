// Copyright (c) 2019 Palantir Technologies. All rights reserved.
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

package prometheus

import (
	"fmt"
	"regexp"
	"time"

	"github.com/palantir/pkg/metrics"
	"github.com/palantir/witchcraft-go-error"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"go.uber.org/multierr"
)

const (
	metricNameReplacePattern = `[^a-zA-Z0-9_:]`
	metricNameStartPattern   = `[a-zA-Z_:]`
)

var (
	metricNameReplaceRegex = regexp.MustCompile(metricNameReplacePattern)
	metricNameStartRegex   = regexp.MustCompile(metricNameStartPattern)
)

// registryGatherer is a `prometheus.Gatherer` for converting metrics in a `metrics.Registry` to prometheus metrics
type registryGatherer struct {
	registry metrics.Registry
}

// NewRegistryGatherer returns a `prometheus.Gatherer` for the provided `metrics.Registry`
func NewRegistryGatherer(registry metrics.Registry) prometheus.Gatherer {
	return &registryGatherer{
		registry: registry,
	}
}

func (r *registryGatherer) Gather() ([]*dto.MetricFamily, error) {
	now := time.Now().Unix()
	var registryErr []error
	var metricFamilies []*dto.MetricFamily
	r.registry.Each(func(name string, tags metrics.Tags, value metrics.MetricVal) {
		// If the name does not start with a valid character, skip it
		if !metricNameStartRegex.MatchString(name[:1]) {
			return
		}
		name = convertName(name)
		mf := dto.MetricFamily{
			Name: stringp(name),
		}
		switch value.Type() {
		case "counter":
			m, err := toCounter(value, tags, now)
			if err != nil {
				registryErr = append(registryErr, err)
				return
			}
			mf.Metric = append(mf.Metric, m)
			mf.Type = dto.MetricType_COUNTER.Enum()
			metricFamilies = append(metricFamilies, &mf)
		case "gauge":
			m, err := toGauge(value, tags, now)
			if err != nil {
				registryErr = append(registryErr, err)
				return
			}
			mf.Metric = append(mf.Metric, m)
			mf.Type = dto.MetricType_GAUGE.Enum()
			metricFamilies = append(metricFamilies, &mf)
		case "histogram":
			mfs, err := toHistogram(name, value, tags, now)
			if err != nil {
				registryErr = append(registryErr, err)
				return
			}
			metricFamilies = append(metricFamilies, mfs...)
		case "meter":
			mfs, err := toMeter(name, value, tags, now)
			if err != nil {
				registryErr = append(registryErr, err)
				return
			}
			metricFamilies = append(metricFamilies, mfs...)
		case "timer":
			mfs, err := toTimer(name, value, tags, now)
			if err != nil {
				registryErr = append(registryErr, err)
				return
			}
			metricFamilies = append(metricFamilies, mfs...)
		default:
			panic("Unknown type: " + value.Type())
		}
	})
	if len(registryErr) != 0 {
		return nil, multierr.Combine(registryErr...)
	}
	return metricFamilies, nil
}

func stringp(s string) *string {
	return &s
}

func toCounter(m metrics.MetricVal, tags metrics.Tags, ts int64) (*dto.Metric, error) {
	f, err := getFloat64p(m.Values(), "count")
	if err != nil {
		return nil, err
	}
	return &dto.Metric{
		Counter: &dto.Counter{
			Value: f,
		},
		TimestampMs: &ts,
		Label:       labelPairs(tags),
	}, nil
}

func toGauge(m metrics.MetricVal, tags metrics.Tags, ts int64) (*dto.Metric, error) {
	f, err := getFloat64p(m.Values(), "value")
	if err != nil {
		return nil, err
	}
	return &dto.Metric{
		Gauge: &dto.Gauge{
			Value: f,
		},
		TimestampMs: &ts,
		Label:       labelPairs(tags),
	}, nil
}

func toHistogram(name string, m metrics.MetricVal, tags metrics.Tags, ts int64) ([]*dto.MetricFamily, error) {
	vals := m.Values()
	labels := labelPairs(tags)
	c, err := getInt64p(vals, "count")
	if err != nil {
		return nil, err
	}
	count := uint64(*c)
	p95, err := getFloat64p(vals, "p95")
	if err != nil {
		return nil, err
	}
	p99, err := getFloat64p(vals, "p99")
	if err != nil {
		return nil, err
	}
	median, err := getFloat64p(vals, "p50")
	if err != nil {
		return nil, err
	}
	min, err := getFloat64p(vals, "min")
	if err != nil {
		return nil, err
	}
	max, err := getFloat64p(vals, "max")
	if err != nil {
		return nil, err
	}
	mean, err := getFloat64p(vals, "mean")
	if err != nil {
		return nil, err
	}
	stddev, err := getFloat64p(vals, "stddev")
	if err != nil {
		return nil, err
	}
	return []*dto.MetricFamily{
		{
			Name: stringp(name),
			Metric: []*dto.Metric{
				{
					Summary: &dto.Summary{
						SampleCount: &count,
						Quantile: []*dto.Quantile{
							{
								Quantile: float64p(0.5),
								Value:    median,
							},
							{
								Quantile: float64p(0.95),
								Value:    p95,
							},
							{
								Quantile: float64p(0.99),
								Value:    p99,
							},
						},
					},
					Label:       labels,
					TimestampMs: &ts,
				},
			},
			Type: dto.MetricType_SUMMARY.Enum(),
		},
		{
			Name: stringp(name + "_mean"),
			Metric: []*dto.Metric{
				{
					Gauge: &dto.Gauge{
						Value: mean,
					},
					Label:       labels,
					TimestampMs: &ts,
				},
			},
			Type: dto.MetricType_GAUGE.Enum(),
		},
		{
			Name: stringp(name + "_min"),
			Metric: []*dto.Metric{
				{
					Gauge: &dto.Gauge{
						Value: min,
					},
					Label:       labels,
					TimestampMs: &ts,
				},
			},
			Type: dto.MetricType_GAUGE.Enum(),
		},
		{
			Name: stringp(name + "_max"),
			Metric: []*dto.Metric{
				{
					Gauge: &dto.Gauge{
						Value: max,
					},
					Label:       labels,
					TimestampMs: &ts,
				},
			},
			Type: dto.MetricType_GAUGE.Enum(),
		},
		{
			Name: stringp(name + "_stddev"),
			Metric: []*dto.Metric{
				{
					Gauge: &dto.Gauge{
						Value: stddev,
					},
					Label:       labels,
					TimestampMs: &ts,
				},
			},
			Type: dto.MetricType_GAUGE.Enum(),
		},
	}, nil
}

func toMeter(name string, value metrics.MetricVal, tags metrics.Tags, now int64) ([]*dto.MetricFamily, error) {
	vals := value.Values()
	labels := labelPairs(tags)
	count, err := getFloat64p(vals, "count")
	if err != nil {
		return nil, err
	}
	rate1m, err := getFloat64p(vals, "1m")
	if err != nil {
		return nil, err
	}
	rate5m, err := getFloat64p(vals, "5m")
	if err != nil {
		return nil, err
	}
	rate15m, err := getFloat64p(vals, "15m")
	if err != nil {
		return nil, err
	}
	mean, err := getFloat64p(vals, "mean")
	if err != nil {
		return nil, err
	}
	return []*dto.MetricFamily{
		{
			Name: stringp(name + "_count"),
			Metric: []*dto.Metric{
				{
					Counter: &dto.Counter{
						Value: count,
					},
					TimestampMs: &now,
					Label:       labels,
				},
			},
			Type: dto.MetricType_COUNTER.Enum(),
		}, {
			Name: stringp(name + "_1m"),
			Metric: []*dto.Metric{
				{
					Gauge: &dto.Gauge{
						Value: rate1m,
					},
					Label:       labels,
					TimestampMs: &now,
				},
			},
			Type: dto.MetricType_GAUGE.Enum(),
		}, {
			Name: stringp(name + "_5m"),
			Metric: []*dto.Metric{
				{
					Gauge: &dto.Gauge{
						Value: rate5m,
					},
					Label:       labels,
					TimestampMs: &now,
				},
			},
			Type: dto.MetricType_GAUGE.Enum(),
		},
		{
			Name: stringp(name + "_15m"),
			Metric: []*dto.Metric{
				{
					Gauge: &dto.Gauge{
						Value: rate15m,
					},
					Label:       labels,
					TimestampMs: &now,
				},
			},
			Type: dto.MetricType_GAUGE.Enum(),
		},
		{
			Name: stringp(name + "_mean"),
			Metric: []*dto.Metric{
				{
					Gauge: &dto.Gauge{
						Value: mean,
					},
					Label:       labels,
					TimestampMs: &now,
				},
			},
			Type: dto.MetricType_GAUGE.Enum(),
		},
	}, nil
}

func toTimer(name string, value metrics.MetricVal, tags metrics.Tags, ts int64) ([]*dto.MetricFamily, error) {
	histo, err := toHistogram(name, value, tags, ts)
	if err != nil {
		return nil, err
	}
	meter, err := toMeter(name, value, tags, ts)
	if err != nil {
		return nil, err
	}
	meanRate, err := getFloat64p(value.Values(), "meanRate")
	if err != nil {
		return nil, err
	}
	meanRateMF := &dto.MetricFamily{
		Name: stringp(name + "_meanrate"),
		Metric: []*dto.Metric{
			{
				Gauge: &dto.Gauge{
					Value: meanRate,
				},
				Label:       labelPairs(tags),
				TimestampMs: &ts,
			},
		},
		Type: dto.MetricType_GAUGE.Enum(),
	}
	return append(histo, append(meter, meanRateMF)...), nil
}

func getFloat64p(m map[string]interface{}, key string) (*float64, error) {
	v, ok := m[key]
	if !ok {
		return nil, werror.Error("no value in map for key", werror.SafeParam("expectedKey", key),
			werror.SafeParam("values", m))
	}
	var f64 float64
	switch value := v.(type) {
	case int64:
		f64 = float64(value)
	case float64:
		f64 = value
	default:
		return nil, werror.Error("Cannot convert value to float64", werror.SafeParam("type", fmt.Sprintf("%T", v)))
	}
	return &f64, nil
}

func getInt64p(m map[string]interface{}, key string) (*int64, error) {
	v, ok := m[key]
	if !ok {
		return nil, werror.Error("Counter had no value", werror.SafeParam("expectedKey", key),
			werror.SafeParam("values", m))
	}
	var i64 int64
	switch value := v.(type) {
	case int64:
		i64 = value
	case float64:
		i64 = int64(value)
	default:
		return nil, werror.Error("Cannot convert value to int64", werror.SafeParam("type", fmt.Sprintf("%T", v)))
	}
	return &i64, nil
}

func float64p(f float64) *float64 {
	return &f
}

func labelPairs(tags metrics.Tags) []*dto.LabelPair {
	labels := make([]*dto.LabelPair, len(tags))
	for i, tag := range tags {
		labels[i] = &dto.LabelPair{
			Name:  stringp(tag.Key()),
			Value: stringp(tag.Value()),
		}
	}
	return labels
}

func convertName(name string) string {
	return metricNameReplaceRegex.ReplaceAllString(name, "_")
}
