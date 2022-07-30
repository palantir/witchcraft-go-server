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
	"net"
	"sync"
	"time"
)

type ServiceName string

// HostPort is a 'host:port' string.
type HostPort string

type HostMetricsRegistry interface {
	HostMetrics(serviceName string, hostname string, port string) HostStatus
	AllServices() map[ServiceName]map[HostPort]HostStatus
}

type HostStatus interface {
	IsActive() bool
	IsHealthy() bool

	Mark2xx()
	Mark3xx()
	Mark4xx()
	Mark5xx()
	MarkError()
	MarkOther()
}

// NewHostRegistry returns a new HostMetricsRegistry that is safe for concurrent use and should generally be used as a singleton per application.
// Metrics are used for internal tracking and not registered on the default registry emitted by witchcraft.
func NewHostRegistry(windowSize time.Duration) HostMetricsRegistry {
	return &hostMetricsRegistry{
		registry:   map[ServiceName]map[HostPort]*hostStatus{},
		windowSize: windowSize,
	}
}

type hostMetricsRegistry struct {
	mux        sync.Mutex
	registry   map[ServiceName]map[HostPort]*hostStatus
	windowSize time.Duration
}

func (h *hostMetricsRegistry) HostMetrics(serviceName string, hostname string, port string) HostStatus {
	h.mux.Lock()
	defer h.mux.Unlock()
	service := ServiceName(serviceName)
	hostport := HostPort(net.JoinHostPort(hostname, port))
	if _, ok := h.registry[service]; !ok {
		h.registry[service] = map[HostPort]*hostStatus{}
	}
	if _, ok := h.registry[service][hostport]; !ok {
		h.registry[service][hostport] = &hostStatus{
			meter2xx:   newTimeBucket(h.windowSize),
			meter3xx:   newTimeBucket(h.windowSize),
			meter4xx:   newTimeBucket(h.windowSize),
			meter5xx:   newTimeBucket(h.windowSize),
			meterErr:   newTimeBucket(h.windowSize),
			meterOther: newTimeBucket(h.windowSize),
		}
	}
	return h.registry[service][hostport]
}

func (h *hostMetricsRegistry) AllServices() map[ServiceName]map[HostPort]HostStatus {
	h.mux.Lock()
	defer h.mux.Unlock()
	services := map[ServiceName]map[HostPort]HostStatus{}
	for serviceName, serviceHosts := range h.registry {
		services[serviceName] = map[HostPort]HostStatus{}
		for hostPort, status := range serviceHosts {
			services[serviceName][hostPort] = status
		}
	}
	return services
}

type hostStatus struct {
	meter2xx   *timeBucket
	meter3xx   *timeBucket
	meter4xx   *timeBucket
	meter5xx   *timeBucket
	meterErr   *timeBucket
	meterOther *timeBucket
}

func (h *hostStatus) IsActive() bool {
	return h.meter2xx.Count() > 0 ||
		h.meter3xx.Count() > 0 ||
		h.meter4xx.Count() > 0 ||
		h.meter5xx.Count() > 0 ||
		h.meterErr.Count() > 0 ||
		h.meterOther.Count() > 0
}

func (h *hostStatus) IsHealthy() bool {
	return h.meter2xx.Count() >= (h.meter5xx.Count() + h.meterErr.Count())
}

func (h *hostStatus) Mark2xx() {
	h.meter2xx.Mark()
}

func (h *hostStatus) Mark3xx() {
	h.meter3xx.Mark()
}

func (h *hostStatus) Mark4xx() {
	h.meter4xx.Mark()
}

func (h *hostStatus) Mark5xx() {
	h.meter5xx.Mark()
}

func (h *hostStatus) MarkError() {
	h.meterErr.Mark()
}

func (h *hostStatus) MarkOther() {
	h.meterOther.Mark()
}
