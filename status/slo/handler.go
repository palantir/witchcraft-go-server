package slo

import (
	"context"
	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-server/status"
	"net/http"
	"sync"
	"time"
)

const (
	CheckType = "SERVICE_LEVEL_OBJECTIVES"
	P95Violations = "p95Violations"
	p99Violations = "p99Violations"
	p999Violations = "p999Violations"

)

type Endpoint interface {
	Handle(rw http.ResponseWriter, req *http.Request)
	SLOViolations() []string
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

func (s *endpoint) SLOViolations() []string {
	if float64(s.p999Counter.HourCount()) / float64(s.totalCounter.HourCount()) <= 0.001 {
		return []string{}
	}
	violations := make([]string, 1, 3)
	violations[0] = p999Violations


	if float64(s.p99Counter.HourCount()) / float64(s.totalCounter.HourCount()) > 0.01 {
		violations = append(violations, p99Violations)
	}
	if float64(s.p95Counter.HourCount()) / float64(s.totalCounter.HourCount()) > 0.05 {
		violations = append(violations, P95Violations)
	}
	return violations
}

type ServiceLevelObjective interface {
	status.HealthCheckSource

	AddEndpoint(name string, handlerFn http.HandlerFunc, p95, p99, p999 time.Duration) Endpoint
}

type serviceLevelObjective struct {
	endpoints map[string]Endpoint
	rwMutext sync.RWMutex

	// healthStatus represents the HealthStatus for the SLOs defined.
	healthStatus health.HealthStatus
}

func (s *serviceLevelObjective) AddEndpoint(endpointName string, handlerFn http.HandlerFunc, p95, p99, p999 time.Duration) Endpoint {
	e := &endpoint{
		endpointName: endpointName,
		handlerFn: handlerFn,
		p95: p95,
		p99: p99,
		p999: p999,
		totalCounter: NewHourCounter(),
		p95Counter: NewHourCounter(),
		p99Counter: NewHourCounter(),
		p999Counter: NewHourCounter(),
		healthStatus: health.HealthStatus{
			Checks: map[health.CheckType]health.HealthCheckResult{
				CheckType: {
					Type: CheckType,
					Message: nil,
					Params: map[string]interface{}{
						endpointName: []string{},
					},
					State: health.HealthStateHealthy,
				},
			},
		},
	}
	s.rwMutext.Lock()
	defer s.rwMutext.Unlock()
	s.endpoints[endpointName] = e

	return e
}


// HealthStatus returns a HealthStatus result which reflects whether the defined SLOs have been met for a given
// endpoint in the last hour.
//
//
func (s *serviceLevelObjective) HealthStatus(ctx context.Context) health.HealthStatus {
	s.rwMutext.RLock()
	defer s.rwMutext.RUnlock()

	check := s.healthStatus.Checks[CheckType]

	for name, endpoint := range s.endpoints {
		violations := endpoint.SLOViolations()
		check.Params[name] = violations
	}

	s.healthStatus.Checks[CheckType] = check
	return s.healthStatus
}

