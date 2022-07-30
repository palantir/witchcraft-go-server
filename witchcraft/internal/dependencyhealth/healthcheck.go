// Copyright (c) 2022 Palantir Technologies. All rights reserved.
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

package dependencyhealth

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sort"
	"time"

	"github.com/palantir/conjure-go-runtime/v2/conjure-go-client/httpclient"
	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-health/conjure/witchcraft/api/health"
)

const (
	serviceDependencyCheckType health.CheckType = "SERVICE_DEPENDENCY"
)

// Messages are vars so their address can be used for *string fields.
var (
	serviceDependencyMsgHealthy         = "All remote services are healthy"
	serviceDependencyMsgServiceFailed   = "All nodes of a remote service have a high failure rate"
	serviceDependencyMsgServiceImpaired = "Some nodes of a remote service have a high failure rate"
)

type ServiceDependencyHealthCheck struct {
	hosts HostMetricsRegistry
}

// NewServiceDependencyHealthCheck returns the SERVICE_DEPENDENCY health check.
// For each service, it measures the ratio of submitted results representing errors from each host.
// When a host returns fewer 2xx responses than 5xx and network errors, it is unhealthy.
//
// This should be a singleton within an application.
// No (known) support exists for merging multiple service dependency checks.
func NewServiceDependencyHealthCheck() *ServiceDependencyHealthCheck {
	return &ServiceDependencyHealthCheck{hosts: NewHostRegistry(5 * time.Minute)}
}

func (h *ServiceDependencyHealthCheck) Middleware(serviceName string) httpclient.Middleware {
	return ServiceDependencyMiddleware{serviceName: serviceName, hosts: h.hosts}
}

type ServiceDependencyMiddleware struct {
	serviceName string
	hosts       HostMetricsRegistry
}

func (m ServiceDependencyMiddleware) RoundTrip(req *http.Request, next http.RoundTripper) (*http.Response, error) {
	resp, respErr := next.RoundTrip(req)
	if m.serviceName != "" && req != nil && req.URL != nil {
		metric := m.hosts.HostMetrics(m.serviceName, req.URL.Hostname(), req.URL.Port())
		switch {
		case respErr != nil && errors.As(werror.RootCause(respErr), new(net.Error)):
			metric.MarkError()
		case resp == nil, resp.StatusCode < 200, resp.StatusCode > 599:
			metric.MarkOther()
		case resp.StatusCode < 300:
			metric.Mark2xx()
		case resp.StatusCode < 400:
			metric.Mark3xx()
		case resp.StatusCode < 500:
			metric.Mark4xx()
		case resp.StatusCode < 600:
			metric.Mark5xx()
		}
	}
	return resp, respErr
}

// HealthStatus iterates through all tracked hosts for each service.
// If there are both healthy and unhealthy hosts, the check is HEALTHY
// but details are included in the message and params.
// If there are only unhealthy hosts for at least one service, the check is WARNING.
func (h *ServiceDependencyHealthCheck) HealthStatus(context.Context) health.HealthStatus {
	allServices := h.hosts.AllServices()
	if len(allServices) == 0 {
		return health.HealthStatus{}
	}
	// Start with a healthy result
	result := health.HealthCheckResult{
		Type:    serviceDependencyCheckType,
		State:   health.New_HealthState(health.HealthState_HEALTHY),
		Message: &serviceDependencyMsgHealthy,
		Params:  map[string]interface{}{},
	}
	var containsUnhealthyService bool
	for serviceName, serviceHosts := range allServices {
		healthy, unhealthy := collectServiceStatus(serviceHosts)
		if len(unhealthy) == 0 {
			continue
		}
		result.Message = &serviceDependencyMsgServiceImpaired
		result.Params[string(serviceName)] = unhealthy
		if len(healthy) == 0 {
			containsUnhealthyService = true
		}
	}
	if containsUnhealthyService {
		result.Message = &serviceDependencyMsgServiceFailed
		result.State = health.New_HealthState(health.HealthState_WARNING)
	}

	return health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			serviceDependencyCheckType: result,
		},
	}
}

func collectServiceStatus(hosts map[HostPort]HostStatus) (healthy, unhealthy []string) {
	for hostPort, hostStatus := range hosts {
		// If a host was previously failing and we've since failed over and stopped using it,
		// there's no reason to continue reporting the issue as it has been handled.
		if !hostStatus.IsActive() {
			continue
		}
		if hostStatus.IsHealthy() {
			healthy = append(healthy, string(hostPort))
		} else {
			unhealthy = append(unhealthy, string(hostPort))
		}
	}
	sort.Strings(healthy)
	sort.Strings(unhealthy)
	return healthy, unhealthy
}
