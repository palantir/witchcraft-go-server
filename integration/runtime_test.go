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
	"github.com/palantir/witchcraft-go-server/config"
	"github.com/palantir/witchcraft-go-server/status"
	"github.com/palantir/witchcraft-go-server/witchcraft"
	"github.com/palantir/witchcraft-go-server/witchcraft/refreshable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		WithSelfSignedCertificate().
		WithInitFunc(func(ctx context.Context, router witchcraft.ConfigurableRouter, installConfig interface{}, runtimeConfig refreshable.Refreshable) (deferFn func(), rErr error) {
			setCfg := func(cfgI interface{}) {
				cfg, ok := cfgI.(testRuntimeConfig)
				if !ok {
					panic(fmt.Errorf("unable to cast runtime config of type %T to testRuntimeConfig", cfgI))
				}
				currCfg = cfg
			}
			setCfg(runtimeConfig.Current())
			runtimeConfig.Subscribe(setCfg)
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
