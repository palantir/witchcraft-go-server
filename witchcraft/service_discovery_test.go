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
	"testing"
	"time"

	"github.com/palantir/conjure-go-runtime/v2/conjure-go-client/httpclient"
	"github.com/palantir/pkg/refreshable"
	"github.com/palantir/witchcraft-go-server/v2/config"
	"github.com/stretchr/testify/require"
)

func TestServiceDiscovery_RefreshableClientConfig(t *testing.T) {
	const serviceName = "blank"
	startingConfig := httpclient.ServicesConfig{}
	defaultRefreshable := refreshable.NewDefaultRefreshable(startingConfig)
	refreshingConfig := config.NewRefreshingServicesConfig(defaultRefreshable)
	discovery := NewServiceDiscovery(config.Install{}, refreshingConfig).(*serviceDiscovery)
	blankConfig := discovery.serviceConfig(serviceName)
	require.Equal(t, httpclient.ClientConfig{ServiceName: serviceName}, blankConfig.CurrentClientConfig())
	t.Run("update default config", func(t *testing.T) {
		require.NoError(t, defaultRefreshable.Update(httpclient.ServicesConfig{
			Default: httpclient.ClientConfig{
				APIToken: stringPtr("secret"),
			},
		}))
		require.Equal(t, httpclient.ClientConfig{ServiceName: serviceName, APIToken: stringPtr("secret")}, blankConfig.CurrentClientConfig())
	})
	t.Run("update services config", func(t *testing.T) {
		require.NoError(t, defaultRefreshable.Update(httpclient.ServicesConfig{
			Services: map[string]httpclient.ClientConfig{serviceName: {
				APIToken: stringPtr("different secret"),
			}},
		}))
		require.Equal(t, httpclient.ClientConfig{ServiceName: serviceName, APIToken: stringPtr("different secret")}, blankConfig.CurrentClientConfig())
	})
	t.Run("add extra configs", func(t *testing.T) {
		discovery.WithDefaultConfig(httpclient.ClientConfig{
			ReadTimeout: durationPtr(time.Second),
		})
		discovery.WithServiceConfig(serviceName, httpclient.ClientConfig{
			WriteTimeout: durationPtr(time.Second),
		})
		require.NoError(t, defaultRefreshable.Update(httpclient.ServicesConfig{
			Services: map[string]httpclient.ClientConfig{serviceName: {
				APIToken: stringPtr("new secret"),
			}},
		}))
		require.Equal(t, httpclient.ClientConfig{
			ServiceName:  serviceName,
			APIToken:     stringPtr("new secret"),
			ReadTimeout:  durationPtr(time.Second),
			WriteTimeout: durationPtr(time.Second),
		}, blankConfig.CurrentClientConfig())
	})
	t.Run("revert to empty config", func(t *testing.T) {
		require.NoError(t, defaultRefreshable.Update(httpclient.ServicesConfig{
			Services: map[string]httpclient.ClientConfig{},
		}))
		require.Equal(t, httpclient.ClientConfig{
			ServiceName:  serviceName,
			ReadTimeout:  durationPtr(time.Second),
			WriteTimeout: durationPtr(time.Second),
		}, blankConfig.CurrentClientConfig())
	})
}

func durationPtr(d time.Duration) *time.Duration { return &d }
func stringPtr(s string) *string                 { return &s }
