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
	"testing"
	"time"

	"github.com/palantir/pkg/metrics"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryGatherer_Gather(t *testing.T) {
	tag := metrics.MustNewTag("key", "value")
	reg := metrics.NewRootMetricsRegistry()
	reg.Counter("counter.test", tag).Inc(100)
	reg.Counter("0123.counter").Inc(100)
	reg.Gauge("gauge.test", tag).Update(101)
	reg.Histogram("histogram.test", tag).Update(15)
	reg.Histogram("histogram.test", tag).Update(25)
	reg.Histogram("histogram.test", tag).Update(35)
	reg.Meter("meter.test", tag).Mark(70)
	reg.Meter("meter.test", tag).Mark(60)
	reg.Meter("meter.test", tag).Mark(50)
	reg.Timer("timer.test", tag).Update(time.Second * 30)
	reg.Timer("timer.test", tag).Update(time.Second * 10)

	gatherer := NewRegistryGatherer(reg)
	expectedNow := time.Now()
	metricFamilies, err := gatherer.Gather()
	require.NoError(t, err, "Gather failed to collect metric families")

	mfs := map[string]*dto.MetricFamily{}
	for _, mf := range metricFamilies {
		mfs[*mf.Name] = mf
	}

	t.Run("Tags to Labels", func(t *testing.T) {
		mf, ok := mfs[convertName("counter.test")]
		require.True(t, ok, "Expected to find metric family for counter.test")
		require.Len(t, mf.Metric, 1, "expected 1 metric for the counter metric family")
		require.NotNil(t, mf.Metric[0].Counter, "Counter metric should not be nil")
		require.Len(t, mf.Metric[0].Label, 1, "Expect 1 label for the 1 tag")
		assert.Equal(t, tag.Key(), *mf.Metric[0].Label[0].Name)
		assert.Equal(t, tag.Value(), *mf.Metric[0].Label[0].Value)

	})
	t.Run("Valid counter", func(t *testing.T) {
		mf, ok := mfs[convertName("counter.test")]
		require.True(t, ok, "Expected to find metric family for counter.test")
		assert.Equal(t, *mf.Name, "counter_test")
		require.Len(t, mf.Metric, 1, "expected 1 metric for the counter metric family")
		require.NotNil(t, mf.Metric[0].Counter, "Counter metric should not be nil")
		assert.Equal(t, float64(100), *mf.Metric[0].Counter.Value)
		assert.True(t, *mf.Metric[0].TimestampMs-expectedNow.Unix() < 10, "Timestamp should be less than 10ms after expected TS")
	})
	t.Run("invalid counter", func(t *testing.T) {
		_, ok := mfs[convertName("0123.counter")]
		assert.False(t, ok, "counter with invalid name should be missing from metric families")
	})
	t.Run("valid gauge", func(t *testing.T) {
		mf, ok := mfs[convertName("gauge.test")]
		require.True(t, ok, "Expected to find metric family for gauge.test")
		assert.Equal(t, *mf.Name, "gauge_test")
		require.Len(t, mf.Metric, 1, "expected 1 metric for the gauge metric family")
		require.NotNil(t, mf.Metric[0].Gauge, "Gauge metric should not be nil")
		assert.Equal(t, float64(101), *mf.Metric[0].Gauge.Value)
		assert.True(t, *mf.Metric[0].TimestampMs-expectedNow.Unix() < 10, "Timestamp should be less than 10ms after expected TS")
	})
	t.Run("valid histogram", func(t *testing.T) {
		mf, ok := mfs[convertName("histogram.test")]
		require.True(t, ok, "Expected to find metric family for histogram.test")
		assert.Equal(t, *mf.Name, "histogram_test")
		require.Len(t, mf.Metric, 1, "expected 1 metric for the histogram metric family")
		require.NotNil(t, mf.Metric[0].Summary, "summary metric for histogram should not be nil")

		histo := reg.Histogram("histogram.test", tag)
		for _, quantile := range mf.Metric[0].Summary.Quantile {
			require.NotNil(t, quantile.Quantile, "quantile value should not be nil")
			assert.Equal(t, histo.Percentile(*quantile.Quantile), *quantile.Value)
		}
		assert.True(t, *mf.Metric[0].TimestampMs-expectedNow.Unix() < 10, "Timestamp should be less than 10ms after expected TS")
		expected := map[string]float64{
			"stddev": histo.StdDev(),
			"min":    float64(histo.Min()),
			"max":    float64(histo.Max()),
			"mean":   float64(histo.Mean()),
		}
		for suffix, expectedMF := range expected {
			expectedName := convertName("histogram.test_" + suffix)
			mf, ok := mfs[expectedName]
			require.True(t, ok, "Expected to find metric family for %s", expectedName)
			assert.Equal(t, expectedName, *mf.Name)
			require.Len(t, mf.Metric, 1, "expected 1 metric for the histogram metric family")
			require.NotNil(t, mf.Metric[0].Gauge, "gauge metric for histogram should not be nil")
			assert.Equal(t, expectedMF, *mf.Metric[0].Gauge.Value, "Expected value for %s", expectedName)
			assert.True(t, *mf.Metric[0].TimestampMs-expectedNow.Unix() < 10, "Timestamp should be less than 10ms after expected TS")
		}
	})
	t.Run("valid meter", func(t *testing.T) {
		meter := reg.Meter("meter.test", tag)

		expected := map[string]float64{
			"1m":   meter.Rate1(),
			"5m":   meter.Rate5(),
			"15m":  meter.Rate15(),
			"mean": meter.RateMean(),
		}
		for suffix, expectedMF := range expected {
			expectedName := convertName("meter.test_" + suffix)
			mf, ok := mfs[expectedName]
			require.True(t, ok, "Expected to find metric family for %s", expectedName)
			assert.Equal(t, expectedName, *mf.Name)
			require.Len(t, mf.Metric, 1, "expected 1 metric for the meter metric family")
			require.NotNil(t, mf.Metric[0].Gauge, "gauge metric for meter should not be nil")
			assert.Equal(t, expectedMF, *mf.Metric[0].Gauge.Value, "Expected value for %s", expectedName)
			assert.True(t, *mf.Metric[0].TimestampMs-expectedNow.Unix() < 10, "Timestamp should be less than 10ms after expected TS")
		}
	})
	t.Run("valid timer", func(t *testing.T) {
		mf, ok := mfs[convertName("timer.test")]
		require.True(t, ok, "Expected to find metric family for timer.test")
		assert.Equal(t, *mf.Name, "timer_test")
		require.Len(t, mf.Metric, 1, "expected 1 metric for the timer metric family")
		require.NotNil(t, mf.Metric[0].Summary, "summary metric for timer should not be nil")

		timer := reg.Timer("timer.test", tag)
		for _, quantile := range mf.Metric[0].Summary.Quantile {
			require.NotNil(t, quantile.Quantile, "quantile value should not be nil")
			assert.Equal(t, timer.Percentile(*quantile.Quantile), *quantile.Value)
		}
		assert.True(t, *mf.Metric[0].TimestampMs-expectedNow.Unix() < 10, "Timestamp should be less than 10ms after expected TS")

		expected := map[string]float64{
			"1m":       timer.Rate1(),
			"5m":       timer.Rate5(),
			"15m":      timer.Rate15(),
			"meanrate": timer.RateMean(),
			"stddev":   timer.StdDev(),
			"min":      float64(timer.Min()),
			"max":      float64(timer.Max()),
			"mean":     float64(timer.Mean()),
		}
		for suffix, expectedMF := range expected {
			expectedName := convertName("timer.test_" + suffix)
			mf, ok := mfs[expectedName]
			require.True(t, ok, "Expected to find metric family for %s", expectedName)
			assert.Equal(t, expectedName, *mf.Name)
			require.Len(t, mf.Metric, 1, "expected 1 metric for the timer metric family")
			require.NotNil(t, mf.Metric[0].Gauge, "gauge metric for timer should not be nil")
			assert.Equal(t, expectedMF, *mf.Metric[0].Gauge.Value, "Expected value for %s", expectedName)
			assert.True(t, *mf.Metric[0].TimestampMs-expectedNow.Unix() < 10, "Timestamp should be less than 10ms after expected TS")
		}
	})
}
