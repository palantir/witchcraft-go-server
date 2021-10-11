// Copyright (c) 2021 Palantir Technologies. All rights reserved.
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
	"sync"

	"github.com/palantir/conjure-go-runtime/v2/conjure-go-client/httpclient"
	"github.com/palantir/witchcraft-go-server/v2/config"
)

type ServiceDiscovery interface {
	// NewClient is an alias for httpclient.NewClientFromRefreshableConfig based on the 'service-discovery' block in runtime configuration.
	NewClient(ctx context.Context, serviceName string, additionalParams ...httpclient.ClientParam) (httpclient.Client, error)
	// NewHTTPClient is an alias for httpclient.NewHTTPClientFromRefreshableConfig based on the 'service-discovery' block in runtime configuration.
	NewHTTPClient(ctx context.Context, serviceName string, additionalParams ...httpclient.HTTPClientParam) (httpclient.RefreshableHTTPClient, error)
}

// ConfigurableServiceDiscovery is an extension to the ServiceDiscovery interface which allows for injecting external
// configuration to be used when constructing clients. New usages should prefer the runtime.service-discovery block
// but in some cases reading from deprecated config locations is necessary to avoid breaking changes when config files
// are loosely coupled to code versions.
type ConfigurableServiceDiscovery interface {
	ServiceDiscovery
	// WithDefaultConfig adds the provided ClientConfig to a list of extra defaults to be merged together for the final
	// client configurations returned by NewClient and NewHTTPClient.
	WithDefaultConfig(defaults httpclient.ClientConfig)
	// WithServiceConfig allows for additional configuration from outside the default runtime.service-discovery block.
	// Use this option to consume configuration overrides from a 'legacy' location in configuration when migrating
	// to live-reloadable clients. Users should strive to move overrides to the default location in witchcraft config
	// to benefit from live-reloadability. All config provided by WithServiceConfig will be merged into the default
	// refreshable ServicesConfig. In case of conflict, the config provided here overrides the defaults.
	WithServiceConfig(serviceName string, cfg httpclient.ClientConfig)
	// WithUserAgent overrides the default user-agent header constructed from the product name and version in install config.
	WithUserAgent(userAgent string)
}

type serviceDiscovery struct {
	sync.RWMutex
	Services  config.RefreshableServicesConfig
	Extra     *httpclient.ServicesConfig
	UserAgent string
}

func NewServiceDiscovery(install config.Install, services config.RefreshableServicesConfig) ConfigurableServiceDiscovery {
	return &serviceDiscovery{Services: services, UserAgent: userAgent(install)}
}

func (s *serviceDiscovery) NewClient(ctx context.Context, serviceName string, additionalParams ...httpclient.ClientParam) (httpclient.Client, error) {
	s.RLock()
	defer s.RUnlock()
	params := append([]httpclient.ClientParam{httpclient.WithUserAgent(s.UserAgent)}, additionalParams...)
	return httpclient.NewClientFromRefreshableConfig(ctx, s.serviceConfig(serviceName), params...)
}

func (s *serviceDiscovery) NewHTTPClient(ctx context.Context, serviceName string, additionalParams ...httpclient.HTTPClientParam) (httpclient.RefreshableHTTPClient, error) {
	s.RLock()
	defer s.RUnlock()
	params := append([]httpclient.HTTPClientParam{httpclient.WithUserAgent(s.UserAgent)}, additionalParams...)
	return httpclient.NewHTTPClientFromRefreshableConfig(ctx, s.serviceConfig(serviceName), params...)
}

func (s *serviceDiscovery) WithDefaultConfig(defaults httpclient.ClientConfig) {
	s.Lock()
	defer s.Unlock()
	if s.Extra == nil {
		s.Extra = &httpclient.ServicesConfig{Default: defaults}
	} else {
		s.Extra.Default = httpclient.MergeClientConfig(defaults, s.Extra.Default)
	}
}

func (s *serviceDiscovery) WithServiceConfig(serviceName string, cfg httpclient.ClientConfig) {
	s.Lock()
	defer s.Unlock()
	if s.Extra == nil {
		s.Extra = &httpclient.ServicesConfig{Services: map[string]httpclient.ClientConfig{serviceName: cfg}}
	} else if s.Extra.Services == nil {
		s.Extra.Services = map[string]httpclient.ClientConfig{serviceName: cfg}
	} else if existing, exists := s.Extra.Services[serviceName]; exists {
		s.Extra.Services[serviceName] = httpclient.MergeClientConfig(cfg, existing)
	} else {
		s.Extra.Services[serviceName] = cfg
	}
}

func (s *serviceDiscovery) WithUserAgent(userAgent string) {
	s.Lock()
	defer s.Unlock()
	s.UserAgent = userAgent
}

// serviceConfig returns a RefreshableClientConfig which merges the configured RefreshableServicesConfig with any
// additional configuration in s.Extra. Precedence order is:
//      s.Extra.Services -> s.Services.Services -> s.Extra.Default -> s.Services.Default
func (s *serviceDiscovery) serviceConfig(serviceName string) httpclient.RefreshableClientConfig {
	return httpclient.NewRefreshingClientConfig(s.Services.MapServicesConfig(func(servicesConfig httpclient.ServicesConfig) interface{} {
		if s.Extra == nil {
			return servicesConfig.ClientConfig(serviceName)
		}
		serviceCfg, serviceCfgOk := servicesConfig.Services[serviceName]
		serviceCfg.ServiceName = serviceName
		extraCfg, extraCfgOk := s.Extra.Services[serviceName]
		extraCfg.ServiceName = serviceName
		var cfg httpclient.ClientConfig
		switch {
		case serviceCfgOk && extraCfgOk:
			cfg = httpclient.MergeClientConfig(extraCfg, serviceCfg)
		case serviceCfgOk:
			cfg = serviceCfg
		case extraCfgOk:
			cfg = extraCfg
		}
		defaults := httpclient.MergeClientConfig(s.Extra.Default, servicesConfig.Default)

		return httpclient.MergeClientConfig(cfg, defaults)
	}))
}

func userAgent(install config.Install) string {
	if install.ProductName != "" {
		agent := install.ProductName
		if install.ProductVersion != "" {
			agent += "/" + install.ProductVersion
		}
		return agent
	}
	return "witchcraft-go-server"
}
