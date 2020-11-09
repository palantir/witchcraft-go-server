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
	"os"
	"path"
	"testing"
	"time"

	"github.com/nmiyake/pkg/dirs"
	"github.com/palantir/pkg/httpserver"
	"github.com/palantir/witchcraft-go-logging/wlog"
	"github.com/palantir/witchcraft-go-server/v2/config"
	"github.com/palantir/witchcraft-go-server/v2/status"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft/refreshable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
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

func getConfiguredFileRefreshable(t *testing.T) refreshable.Refreshable {
	r, err := refreshable.NewFileRefreshableWithDuration(context.Background(), "var/conf/runtime.yml", time.Millisecond*30)
	assert.NoError(t, err)
	return r
}
