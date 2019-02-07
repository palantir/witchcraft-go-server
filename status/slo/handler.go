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

package slo

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-server/status"
)

const (
	CheckType      = "SERVICE_LEVEL_OBJECTIVES"
	P95Violations  = "p95Violations"
	p99Violations  = "p99Violations"
	p999Violations = "p999Violations"
)

// ServiceLevelObjectives is an interface that implements the `status.HealthCheckSource`.
//
// For every instrumented endpoint, ServiceLevelObjectives computes performance relative to the objectives by
// maintaining four buckets: the total number of requests and the number of requests that violate the target for each
// of p95, p99 and p999. When summarizing the results over an hour, the implementation checks if no more than 5% of the
// requests violate the p95 target, no more than 1% of the requests violate the p99 target, and no more than 0.1% of
// the requests violate the p999 target.
//
// When a violation of hourly targets has been detected, the provided health check declares a WARNING state and encodes
// which methods have violated the target objective.
type ServiceLevelObjectives interface {
	status.HealthCheckSource

	Register(name string, handlerFn http.HandlerFunc, p95, p99, p999 time.Duration) http.HandlerFunc
}

type serviceLevelObjectives struct {
	endpoints map[string]*endpoint
	rwMutext  sync.RWMutex

	// healthStatus represents the HealthStatus for the SLOs defined.
	healthStatus health.HealthStatus
}

// NewSLOs returns a ServiceLevelObjectives interface that can be used to register endpoints and their corresponding SLOs
func NewSLOs() ServiceLevelObjectives {
	return &serviceLevelObjectives{
		endpoints: map[string]*endpoint{},
		rwMutext:  sync.RWMutex{},
		healthStatus: health.HealthStatus{
			Checks: map[health.CheckType]health.HealthCheckResult{
				CheckType: {
					Type:    CheckType,
					Message: nil,
					Params:  map[string]interface{}{},
					State:   health.HealthStateHealthy,
				},
			},
		},
	}
}

// Register takes an endpoint name, http.HandlerFunc and expected p95, p99 and p999 durations and returns an
// instrumented http.HandlerFunc
func (s *serviceLevelObjectives) Register(endpointName string, handlerFn http.HandlerFunc, p95, p99, p999 time.Duration) http.HandlerFunc {
	e := &endpoint{
		handlerFn:    handlerFn,
		p95:          p95,
		p99:          p99,
		p999:         p999,
		totalCounter: NewHourCounter(),
		p95Counter:   NewHourCounter(),
		p99Counter:   NewHourCounter(),
		p999Counter:  NewHourCounter(),
	}
	s.rwMutext.Lock()
	defer s.rwMutext.Unlock()
	s.endpoints[endpointName] = e

	return e.Handle
}

// HealthStatus returns a HealthStatus result which reflects whether the defined SLOs have been met for a given
// endpoint in the last hour.
//
// When summarizing the results over an hour, the implementation checks if no more than 5% of the requests violate the
// p95 target, no more than 1% of the requests violate the p99 target, and no more than 0.1% of the requests violate
// the p999 target.
//
// When a violation of hourly targets has been detected, the provided health check declares a WARNING state and encodes
// which methods have violated the target objective.
func (s *serviceLevelObjectives) HealthStatus(ctx context.Context) health.HealthStatus {
	s.rwMutext.RLock()
	defer s.rwMutext.RUnlock()

	check := s.healthStatus.Checks[CheckType]

	for name, endpoint := range s.endpoints {
		violations := endpoint.sloViolations()
		check.Params[name] = violations
	}

	s.healthStatus.Checks[CheckType] = check
	return s.healthStatus
}

type endpoint struct {
	handlerFn http.HandlerFunc

	// p95 is a duration signifying the expected p95 of an endpoint
	p95 time.Duration

	// p99 is a duration signifying the expected p99 of an endpoint
	p99 time.Duration

	// p999 is a duration signifying the expected p999 of an endpoint
	p999 time.Duration

	// totalCounter tracks the total number of requests per hour
	totalCounter *HourCounter

	// p95Counter tracks the number of requests per hour whose duration exceeded the p95 duration
	p95Counter *HourCounter

	// p99Counter tracks the number of requests per hour whose duration exceeded the p99 duration
	p99Counter *HourCounter

	// p999Counter tracks the number of requests per hour whose duration exceeded the p999 duration
	p999Counter *HourCounter
}

func (s *endpoint) Handle(rw http.ResponseWriter, req *http.Request) {
	start := time.Now()
	s.handlerFn(rw, req)
	dur := time.Since(start)
	if dur > s.p95 {
		s.p95Counter.Mark(start)
	}
	if dur > s.p99 {
		s.p99Counter.Mark(start)
	}
	if dur > s.p999 {
		s.p999Counter.Mark(start)
	}
	s.totalCounter.Mark(start)
}

func (s *endpoint) sloViolations() []string {
	if float64(s.p999Counter.HourCount())/float64(s.totalCounter.HourCount()) <= 0.001 {
		return []string{}
	}
	violations := make([]string, 1, 3)
	violations[0] = p999Violations

	if float64(s.p99Counter.HourCount())/float64(s.totalCounter.HourCount()) > 0.01 {
		violations = append(violations, p99Violations)
	}
	if float64(s.p95Counter.HourCount())/float64(s.totalCounter.HourCount()) > 0.05 {
		violations = append(violations, P95Violations)
	}
	return violations
}
