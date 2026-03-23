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
	"testing"
	"time"

	"github.com/palantir/conjure-go-runtime/v3/conjure-go-client/httpclient"
	"github.com/palantir/pkg/refreshable/v2"
	"github.com/palantir/witchcraft-go-server/v3/config"
	"github.com/stretchr/testify/require"
)

func TestServiceDiscovery_RefreshableClientConfig(t *testing.T) {
	const serviceName = "blank"
	startingConfig := httpclient.ServicesConfig{}
	defaultRefreshable := refreshable.New(startingConfig)
	discovery := NewServiceDiscovery(config.Install{}, defaultRefreshable).(*serviceDiscovery)
	blankConfig := discovery.serviceConfig(serviceName)
	require.Equal(t, httpclient.ClientConfig{ServiceName: serviceName}, blankConfig.Current())
	t.Run("update default config", func(t *testing.T) {
		defaultRefreshable.Update(httpclient.ServicesConfig{
			Default: httpclient.ClientConfig{
				APIToken: new("secret"),
			},
		})
		require.Equal(t, httpclient.ClientConfig{ServiceName: serviceName, APIToken: new("secret")}, blankConfig.Current())
	})
	t.Run("update services config", func(t *testing.T) {
		defaultRefreshable.Update(httpclient.ServicesConfig{
			Services: map[string]httpclient.ClientConfig{serviceName: {
				APIToken: new("different secret"),
			}},
		})
		require.Equal(t, httpclient.ClientConfig{ServiceName: serviceName, APIToken: new("different secret")}, blankConfig.Current())
	})
	t.Run("add extra configs", func(t *testing.T) {
		discovery.WithDefaultConfig(httpclient.ClientConfig{
			ReadTimeout: new(time.Second),
		})
		discovery.WithServiceConfig(serviceName, httpclient.ClientConfig{
			WriteTimeout: new(time.Second),
		})
		defaultRefreshable.Update(httpclient.ServicesConfig{
			Services: map[string]httpclient.ClientConfig{serviceName: {
				APIToken: new("new secret"),
			}},
		})
		require.Equal(t, httpclient.ClientConfig{
			ServiceName:  serviceName,
			APIToken:     new("new secret"),
			ReadTimeout:  new(time.Second),
			WriteTimeout: new(time.Second),
		}, blankConfig.Current())
	})
	t.Run("revert to empty config", func(t *testing.T) {
		defaultRefreshable.Update(httpclient.ServicesConfig{
			Services: map[string]httpclient.ClientConfig{},
		})
		require.Equal(t, httpclient.ClientConfig{
			ServiceName:  serviceName,
			ReadTimeout:  new(time.Second),
			WriteTimeout: new(time.Second),
		}, blankConfig.Current())
	})
}

func TestServiceDiscovery_ClientOverrides(t *testing.T) {
	const serviceName = "blank"
	ctx := context.Background()
	startingConfig := httpclient.ServicesConfig{}
	defaultRefreshable := refreshable.New(startingConfig)
	discovery := NewServiceDiscovery(config.Install{}, defaultRefreshable).(*serviceDiscovery)
	t.Run("update default param", func(t *testing.T) {
		discovery.WithDefaultParams(func(serviceName string) ([]httpclient.ClientParam, error) {
			return []httpclient.ClientParam{httpclient.WithHTTPTimeout(time.Second)}, nil
		})
		client, err := discovery.NewHTTPClient(ctx, serviceName)
		require.NoError(t, err)
		require.Equal(t, time.Second, client.Current().Timeout)
	})
	t.Run("update service param", func(t *testing.T) {
		discovery.WithServiceParams(serviceName, httpclient.WithHTTPTimeout(2*time.Second))
		client, err := discovery.NewHTTPClient(ctx, serviceName)
		require.NoError(t, err)
		require.Equal(t, 2*time.Second, client.Current().Timeout)
	})
}
