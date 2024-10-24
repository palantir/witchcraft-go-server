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

package integration

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"
	"time"

	"github.com/nmiyake/pkg/dirs"
	"github.com/palantir/conjure-go-runtime/v2/conjure-go-client/httpclient"
	"github.com/palantir/pkg/httpserver"
	"github.com/palantir/pkg/metrics"
	"github.com/palantir/pkg/refreshable"
	"github.com/palantir/witchcraft-go-logging/wlog"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/palantir/witchcraft-go-server/v2/config"
	"github.com/palantir/witchcraft-go-server/v2/status"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft"
	refreshablefile "github.com/palantir/witchcraft-go-server/v2/witchcraft/refreshable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

type testRuntimeConfig struct {
	config.Runtime `yaml:",inline"`
	SecretGreeting string `yaml:"secret-greeting"`
	Exclamations   int    `yaml:"exclamations"`
}

// TestRuntimeReloadWithEncryptedConfig verifies behavior of refreshable configuration.
// 1. assert that the configuration printed at startup is that passed to the server's InitFunc
// 2. assert that removing the configuration does not send an update with empty values.
// 3. assert that writing a changed configuration does trigger subscribers to the refreshable.
func TestRuntimeReloadWithEncryptedConfig(t *testing.T) {
	testDir, cleanup, err := dirs.TempDir("", "")
	require.NoError(t, err)
	defer cleanup()

	wd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(wd)
		require.NoError(t, err)
	}()

	err = os.Chdir(testDir)
	require.NoError(t, err)

	port, err := httpserver.AvailablePort()
	require.NoError(t, err)

	err = os.MkdirAll("var/conf", 0755)
	require.NoError(t, err)

	installCfgYml := fmt.Sprintf(`product-name: %s
use-console-log: true
server:
 address: localhost
 port: %d
 context-path: %s`,
		productName, port, basePath)
	err = ioutil.WriteFile(installYML, []byte(installCfgYml), 0644)
	require.NoError(t, err)

	const ecvKey = `AES:Nu2OInDbOHhXCNqqt1yyDuPwZwaJrSjV+IAypbZhw6Y=`
	err = ioutil.WriteFile("var/conf/encrypted-config-value.key", []byte(ecvKey), 0644)
	require.NoError(t, err)

	cfg1 := testRuntimeConfig{SecretGreeting: "hello, world!", Exclamations: 3}
	const cfg1YML = `
secret-greeting: ${enc:/pSQ0v8R3QR8WOLnxoAWTsnI6kkjGgQMbqFcU9UC+LxStdGbfg1i3R9mlVZjEuXuecVG5AK1Sq109YxUcg==}
exclamations: 3
`
	cfg2 := testRuntimeConfig{SecretGreeting: "hello, world!", Exclamations: 4}
	const cfg2YML = `
secret-greeting: ${enc:/pSQ0v8R3QR8WOLnxoAWTsnI6kkjGgQMbqFcU9UC+LxStdGbfg1i3R9mlVZjEuXuecVG5AK1Sq109YxUcg==}
exclamations: 4
`
	err = ioutil.WriteFile(runtimeYML, []byte(cfg1YML), 0644)
	require.NoError(t, err)

	var currCfg testRuntimeConfig
	server := witchcraft.NewServer().
		WithRuntimeConfigType(testRuntimeConfig{}).
		WithDisableGoRuntimeMetrics().
		WithRuntimeConfigProvider(getConfiguredFileRefreshable(t)).
		WithSelfSignedCertificate().
		WithInitFunc(func(ctx context.Context, info witchcraft.InitInfo) (cleanupFn func(), rErr error) {
			setCfg := func(cfgI interface{}) {
				cfg, ok := cfgI.(testRuntimeConfig)
				if !ok {
					panic(fmt.Errorf("unable to cast runtime config of type %T to testRuntimeConfig", cfgI))
				}
				currCfg = cfg
			}
			setCfg(info.RuntimeConfig.Current())
			info.RuntimeConfig.Subscribe(setCfg)
			return nil, nil
		})

	serverChan := make(chan error)
	go func() {
		serverChan <- server.Start()
	}()

	select {
	case err := <-serverChan:
		require.NoError(t, err)
	default:
	}

	ready := <-waitForTestServerReady(port, path.Join(basePath, status.LivenessEndpoint), 5*time.Second)
	if !ready {
		errMsg := "timed out waiting for server to start"
		select {
		case err := <-serverChan:
			errMsg = fmt.Sprintf("%s: %+v", errMsg, err)
		}
		require.Fail(t, errMsg)
	}

	defer func() {
		require.NoError(t, server.Close())
	}()

	// Assert our configuration was set to the initial values
	assert.Equal(t, cfg1, currCfg)

	// Remove file and assert that we do not change the stored config
	err = os.Remove(runtimeYML)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, cfg1, currCfg)

	// Update config and assert that our subscription overwrites the value
	err = ioutil.WriteFile(runtimeYML, []byte(cfg2YML), 0644)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, cfg2, currCfg)
}

// TestRuntimeReloadWithNilLoggerConfig verifies that reloading runtime configuration with nil logger config works.
func TestRuntimeReloadWithNilLoggerConfig(t *testing.T) {
	testDir, cleanup, err := dirs.TempDir("", "")
	require.NoError(t, err)
	defer cleanup()

	wd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(wd)
		require.NoError(t, err)
	}()

	err = os.Chdir(testDir)
	require.NoError(t, err)

	port, err := httpserver.AvailablePort()
	require.NoError(t, err)

	err = os.MkdirAll("var/conf", 0755)
	require.NoError(t, err)

	runtimeConfigWithLoggingYML, err := yaml.Marshal(config.Runtime{
		LoggerConfig: &config.LoggerConfig{
			Level: wlog.DebugLevel,
		},
	})
	require.NoError(t, err)

	err = ioutil.WriteFile(runtimeYML, runtimeConfigWithLoggingYML, 0644)
	require.NoError(t, err)

	runtimeConfigUpdatedChan := make(chan struct{})

	server := witchcraft.NewServer().
		WithInstallConfig(config.Install{
			ProductName:   productName,
			UseConsoleLog: true,
			Server: config.Server{
				Address:     "localhost",
				Port:        port,
				ContextPath: basePath,
			},
		}).
		WithDisableGoRuntimeMetrics().
		WithSelfSignedCertificate().
		WithInitFunc(func(ctx context.Context, info witchcraft.InitInfo) (cleanupFn func(), rErr error) {
			info.RuntimeConfig.Subscribe(func(cfgI interface{}) {
				runtimeConfigUpdatedChan <- struct{}{}
			})
			return nil, nil
		})

	serverChan := make(chan error)
	go func() {
		serverChan <- server.Start()
	}()

	select {
	case err := <-serverChan:
		require.NoError(t, err)
	default:
	}

	ready := <-waitForTestServerReady(port, path.Join(basePath, status.LivenessEndpoint), 5*time.Second)
	if !ready {
		errMsg := "timed out waiting for server to start"
		select {
		case err := <-serverChan:
			errMsg = fmt.Sprintf("%s: %+v", errMsg, err)
		}
		require.Fail(t, errMsg)
	}

	defer func() {
		require.NoError(t, server.Close())
	}()

	err = ioutil.WriteFile(runtimeYML, []byte(""), 0644)
	require.NoError(t, err)

	select {
	case <-runtimeConfigUpdatedChan:
		break
	case <-time.After(5 * time.Second):
		assert.Fail(t, "timed out waiting for runtime configuration to be updated")
	}
}

// TestRuntimeReloadWithConfigWithExtraKeyDefaultUnmarshal verifies that reloading runtime configuration with an unknown
// key succeeds when server is in its default mode.
func TestRuntimeReloadWithConfigWithUnknownKeyDefaultUnmarshal(t *testing.T) {
	testDir, cleanup, err := dirs.TempDir("", "")
	require.NoError(t, err)
	defer cleanup()

	wd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(wd)
		require.NoError(t, err)
	}()

	err = os.Chdir(testDir)
	require.NoError(t, err)

	port, err := httpserver.AvailablePort()
	require.NoError(t, err)

	err = os.MkdirAll("var/conf", 0755)
	require.NoError(t, err)

	const validCfg1YML = `
logging:
  level: debug
exclamations: 3
`
	validCfg1 := testRuntimeConfig{
		Runtime: config.Runtime{
			LoggerConfig: &config.LoggerConfig{
				Level: wlog.DebugLevel,
			},
		},
		Exclamations: 3,
	}
	const invalidYML = `
invalid-key: invalid-value
`
	const validCfg2YML = `
logging:
  level: info
exclamations: 4
`
	validCfg2 := testRuntimeConfig{
		Runtime: config.Runtime{
			LoggerConfig: &config.LoggerConfig{
				Level: wlog.InfoLevel,
			},
		},
		Exclamations: 4,
	}

	err = ioutil.WriteFile(runtimeYML, []byte(validCfg1YML), 0644)
	require.NoError(t, err)

	var currCfg testRuntimeConfig

	server := witchcraft.NewServer().
		WithRuntimeConfigType(testRuntimeConfig{}).
		WithInstallConfig(config.Install{
			ProductName:   productName,
			UseConsoleLog: true,
			Server: config.Server{
				Address:     "localhost",
				Port:        port,
				ContextPath: basePath,
			},
		}).
		WithDisableGoRuntimeMetrics().
		WithRuntimeConfigProvider(getConfiguredFileRefreshable(t)).
		WithSelfSignedCertificate().
		WithInitFunc(func(ctx context.Context, info witchcraft.InitInfo) (cleanupFn func(), rErr error) {
			setCfg := func(cfgI interface{}) {
				cfg, ok := cfgI.(testRuntimeConfig)
				if !ok {
					panic(fmt.Errorf("unable to cast runtime config of type %T to testRuntimeConfig", cfgI))
				}
				currCfg = cfg
			}
			setCfg(info.RuntimeConfig.Current())
			info.RuntimeConfig.Subscribe(setCfg)
			return nil, nil
		})

	serverChan := make(chan error)
	go func() {
		serverChan <- server.Start()
	}()

	select {
	case err := <-serverChan:
		require.NoError(t, err)
	default:
	}

	ready := <-waitForTestServerReady(port, path.Join(basePath, status.LivenessEndpoint), 5*time.Second)
	if !ready {
		errMsg := "timed out waiting for server to start"
		select {
		case err := <-serverChan:
			errMsg = fmt.Sprintf("%s: %+v", errMsg, err)
		}
		require.Fail(t, errMsg)
	}

	defer func() {
		require.NoError(t, server.Close())
	}()

	// Assert our configuration was set to the initial values
	assert.Equal(t, validCfg1, currCfg)

	// Assert that, in default mode, configuration with extra key is considered valid: extra value should be ignored,
	// and "missing" values should be default values
	invalidRuntimeConfig := testRuntimeConfig{
		Runtime:        config.Runtime{},
		SecretGreeting: "",
		Exclamations:   0,
	}

	err = ioutil.WriteFile(runtimeYML, []byte("invalid-key: \"invalid-value\""), 0644)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, invalidRuntimeConfig, currCfg)

	// Update config to different config and assert that our subscription overwrites the value
	err = ioutil.WriteFile(runtimeYML, []byte(validCfg2YML), 0644)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, validCfg2, currCfg)
}

// TestRuntimeReloadWithConfigWithExtraKeyStrictUnmarshal verifies that reloading runtime configuration with an unknown
// key fails and uses last known valid config when server is in strict unmarshal mode.
func TestRuntimeReloadWithConfigWithExtraKeyStrictUnmarshal(t *testing.T) {
	testDir, cleanup, err := dirs.TempDir("", "")
	require.NoError(t, err)
	defer cleanup()

	wd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(wd)
		require.NoError(t, err)
	}()

	err = os.Chdir(testDir)
	require.NoError(t, err)

	port, err := httpserver.AvailablePort()
	require.NoError(t, err)

	err = os.MkdirAll("var/conf", 0755)
	require.NoError(t, err)

	const validCfg1YML = `
logging:
  level: debug
exclamations: 3
`
	validCfg1 := testRuntimeConfig{
		Runtime: config.Runtime{
			LoggerConfig: &config.LoggerConfig{
				Level: wlog.DebugLevel,
			},
		},
		Exclamations: 3,
	}
	const invalidYML = `
invalid-key: invalid-value
`
	const validCfg2YML = `
logging:
  level: info
exclamations: 4
`
	validCfg2 := testRuntimeConfig{
		Runtime: config.Runtime{
			LoggerConfig: &config.LoggerConfig{
				Level: wlog.InfoLevel,
			},
		},
		Exclamations: 4,
	}

	err = ioutil.WriteFile(runtimeYML, []byte(validCfg1YML), 0644)
	require.NoError(t, err)

	var currCfg testRuntimeConfig

	server := witchcraft.NewServer().
		WithRuntimeConfigType(testRuntimeConfig{}).
		WithInstallConfig(config.Install{
			ProductName:   productName,
			UseConsoleLog: true,
			Server: config.Server{
				Address:     "localhost",
				Port:        port,
				ContextPath: basePath,
			},
		}).
		WithDisableGoRuntimeMetrics().
		WithSelfSignedCertificate().
		WithRuntimeConfigProvider(getConfiguredFileRefreshable(t)).
		WithInitFunc(func(ctx context.Context, info witchcraft.InitInfo) (cleanupFn func(), rErr error) {
			setCfg := func(cfgI interface{}) {
				cfg, ok := cfgI.(testRuntimeConfig)
				if !ok {
					panic(fmt.Errorf("unable to cast runtime config of type %T to testRuntimeConfig", cfgI))
				}
				currCfg = cfg
			}
			setCfg(info.RuntimeConfig.Current())
			info.RuntimeConfig.Subscribe(setCfg)
			return nil, nil
		}).
		WithStrictUnmarshalConfig()

	serverChan := make(chan error)
	go func() {
		serverChan <- server.Start()
	}()

	select {
	case err := <-serverChan:
		require.NoError(t, err)
	default:
	}

	ready := <-waitForTestServerReady(port, path.Join(basePath, status.LivenessEndpoint), 5*time.Second)
	if !ready {
		errMsg := "timed out waiting for server to start"
		select {
		case err := <-serverChan:
			errMsg = fmt.Sprintf("%s: %+v", errMsg, err)
		}
		require.Fail(t, errMsg)
	}

	defer func() {
		require.NoError(t, server.Close())
	}()

	// Assert our configuration was set to the initial values
	assert.Equal(t, validCfg1, currCfg)

	// Assert that introducing invalid config does not change the stored config
	err = ioutil.WriteFile(runtimeYML, []byte("invalid-key: \"invalid-value\""), 0644)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, validCfg1, currCfg)

	// Update config to a valid, but different config and assert that our subscription overwrites the value
	err = ioutil.WriteFile(runtimeYML, []byte(validCfg2YML), 0644)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, validCfg2, currCfg)
}

func TestRuntimeReloadServiceDiscovery(t *testing.T) {
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, userAgent, req.Header.Get("User-Agent"), "expected user-agent based on config")
	}))

	testDir, cleanup, err := dirs.TempDir("", "")
	require.NoError(t, err)
	defer cleanup()

	wd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(wd)
		require.NoError(t, err)
	}()

	err = os.Chdir(testDir)
	require.NoError(t, err)

	port, err := httpserver.AvailablePort()
	require.NoError(t, err)

	err = os.MkdirAll("var/conf", 0755)
	require.NoError(t, err)

	type installConfig struct {
		config.Install `yaml:",inline"`
		Clients        httpclient.ServicesConfig `yaml:"clients"`
	}
	installCfgYml := fmt.Sprintf(`product-name: %s
product-version: %s
use-console-log: true
server:
  address: localhost
  port: %d
  context-path: %s
clients:
  metrics:
    enabled: true
    tags:
      deprecated: deprecated
`,
		productName, productVersion, port, basePath)
	err = ioutil.WriteFile(installYML, []byte(installCfgYml), 0644)
	require.NoError(t, err)

	runtimeCfgYml := fmt.Sprintf(`logging:
  level: info
service-discovery:
  metrics:
    enabled: true
    tags:
      default: default
  services:
    upstream-server:
      uris:
        - %s
      metrics:
        enabled: true
        tags:
          test: one
`,
		upstreamServer.URL)
	err = ioutil.WriteFile(runtimeYML, []byte(runtimeCfgYml), 0644)
	require.NoError(t, err)

	var ctx context.Context
	var client httpclient.Client
	server := witchcraft.NewServer().
		WithInstallConfigType(installConfig{}).
		WithDisableGoRuntimeMetrics().
		WithSelfSignedCertificate().
		WithInitFunc(func(initCtx context.Context, info witchcraft.InitInfo) (cleanupFn func(), rErr error) {
			ctx = initCtx
			info.Clients.WithDefaultConfig(info.InstallConfig.(installConfig).Clients.Default)
			var err error
			client, err = info.Clients.NewClient(initCtx, "upstream-server")
			return nil, err
		})

	serverChan := make(chan error)
	go func() {
		serverChan <- server.Start()
	}()

	select {
	case err := <-serverChan:
		require.NoError(t, err)
	default:
	}

	ready := <-waitForTestServerReady(port, path.Join(basePath, status.LivenessEndpoint), 5*time.Second)
	if !ready {
		errMsg := "timed out waiting for server to start"
		select {
		case err := <-serverChan:
			errMsg = fmt.Sprintf("%s: %+v", errMsg, err)
		}
		require.Fail(t, errMsg)
	}

	defer func() {
		require.NoError(t, server.Close())
	}()

	// Now that our server is ready and we know we are initialized, make a request to the upstream-server
	// and verify that our client metrics were updated according to the configured tags.

	_, err = client.Get(ctx)
	require.NoError(t, err)
	test1timer := metrics.FromContext(ctx).Timer("client.response", metrics.MustNewTags(map[string]string{
		"method-name":  "rpcmethodnamemissing",
		"method":       "get",
		"service-name": "upstream-server",
		"family":       "2xx",
		"deprecated":   "deprecated",
		"default":      "default",
		"test":         "one",
	})...)
	if !assert.Equal(t, int64(1), test1timer.Count()) {
		metrics.FromContext(ctx).Each(func(name string, tags metrics.Tags, value metrics.MetricVal) {
			t.Logf("%s %v", name, tags)
		})
	}

	// Update runtime config file (change tag value one->two)
	// Sleep 3 seconds for FileRefreshable to update
	// Execute request again, and check that new metric is updateds

	runtimeCfgYml = fmt.Sprintf(`logging:
  level: info
service-discovery:
  metrics:
    enabled: true
    tags:
      default: default
  services:
    upstream-server:
      uris:
        - %s
      metrics:
        enabled: true
        tags:
          test: two
`,
		upstreamServer.URL)
	err = ioutil.WriteFile(runtimeYML, []byte(runtimeCfgYml), 0644)
	require.NoError(t, err)
	time.Sleep(3 * time.Second)

	_, err = client.Get(ctx)
	require.NoError(t, err)
	test2timer := metrics.FromContext(ctx).Timer("client.response", metrics.MustNewTags(map[string]string{
		"method-name":  "rpcmethodnamemissing",
		"method":       "get",
		"service-name": "upstream-server",
		"family":       "2xx",
		"deprecated":   "deprecated",
		"default":      "default",
		"test":         "two",
	})...)
	if !assert.Equal(t, int64(1), test2timer.Count()) {
		metrics.FromContext(ctx).Each(func(name string, tags metrics.Tags, value metrics.MetricVal) {
			t.Logf("%s %v", name, tags)
		})
	}
}

func getConfiguredFileRefreshable(t *testing.T) refreshable.Refreshable {
	ctx := svc1log.WithLogger(context.Background(), svc1log.New(os.Stdout, wlog.DebugLevel))
	r, err := refreshablefile.NewFileRefreshableWithDuration(ctx, "var/conf/runtime.yml", time.Millisecond*30)
	assert.NoError(t, err)
	return r
}
