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

package integration

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/palantir/witchcraft-go-server/v2/config"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptedConfig(t *testing.T) {
	const (
		decryptedValue          = "hello world"
		encryptionKey           = "AES:T6H7a4WvQS9ITcNIihyUIj30K4SIrD6dB39ENJQ7oAo="
		encryptedValue          = "${enc:eyJ0eXBlIjoiQUVTIiwibW9kZSI6IkdDTSIsImNpcGhlcnRleHQiOiJqcGl0bThQRStRekd2YlE9IiwiaXYiOiJrTHlBOEZBNzFnTDVpdkswIiwidGFnIjoicmxpcXY3amYwbWVnaGU1N0pyQ3ZzZz09In0=}"
		decryptedMultilineValue = `-----BEGIN CERTIFICATE-----
cert
-----END CERTIFICATE-----
`
		encryptedMultilineValue = "${enc:eyJ0eXBlIjoiQUVTIiwibW9kZSI6IkdDTSIsImNpcGhlcnRleHQiOiJuTXNWOEV1L1JpWGN1TGFDbVNSRVVQUG1NbFpab0paMEN6QlNuL3BWUWwzWUhkWXlVSEZYaGY5c3N5NlFTbzVXYnp5bTlaTCtmQytxQTZNPSIsIml2IjoiRzNtSHp1SmRVU0NncXRLcyIsInRhZyI6IlI2UWZPb2UvS1hiSFZiTmZDaWNkVlE9PSJ9}"
	)

	type message struct {
		config.Runtime `yaml:",inline"`
		Message        string `yaml:"message"`
	}

	type messageInstall struct {
		config.Install `yaml:",inline"`
		Message        string `yaml:"message"`
	}

	for _, test := range []struct {
		Name          string
		ECVKeyContent string
		InstallConfig string
		RuntimeConfig string
		// Use InitFn to verify config
		Verify    func(info witchcraft.InitInfo) error
		VerifyLog func(t *testing.T, logOutput []byte)
	}{
		{
			Name:          "init function sees decrypted config",
			ECVKeyContent: encryptionKey,
			InstallConfig: fmt.Sprintf("message: %s\nuse-console-log: true\n", encryptedValue),
			RuntimeConfig: fmt.Sprintf("message: %s\nlogging:\n  level: warn\n", encryptedValue),
			Verify: func(info witchcraft.InitInfo) error {
				if msg := info.InstallConfig.(*messageInstall).Message; msg != decryptedValue {
					return fmt.Errorf("expected %q got %q", decryptedValue, msg)
				}
				if msg := info.RuntimeConfig.Current().(*message).Message; msg != decryptedValue {
					return fmt.Errorf("expected %q got %q", decryptedValue, msg)
				}
				return nil
			},
			VerifyLog: func(t *testing.T, logOutput []byte) {
				assert.Equal(t, []string{"abort startup"}, getLogFileMessages(t, logOutput))
			},
		},
		{
			Name:          "encrypted config can be concatenated with other strings",
			ECVKeyContent: encryptionKey,
			InstallConfig: fmt.Sprintf("message: prefix%ssuffix\nuse-console-log: true\n", encryptedValue),
			RuntimeConfig: fmt.Sprintf("message: prefix%ssuffix%s\nlogging:\n  level: warn\n", encryptedValue, encryptedMultilineValue),
			Verify: func(info witchcraft.InitInfo) error {
				expectedInstallValue := fmt.Sprintf("prefix%ssuffix", decryptedValue)
				if msg := info.InstallConfig.(*messageInstall).Message; msg != expectedInstallValue {
					return fmt.Errorf("expected %q got %q", expectedInstallValue, msg)
				}
				expectedRuntimeValue := fmt.Sprintf("prefix%ssuffix%s", decryptedValue, decryptedMultilineValue)
				if msg := info.RuntimeConfig.Current().(*message).Message; msg != expectedRuntimeValue {
					return fmt.Errorf("expected %q got %q", expectedRuntimeValue, msg)
				}
				return nil
			},
			VerifyLog: func(t *testing.T, logOutput []byte) {
				assert.Equal(t, []string{"abort startup"}, getLogFileMessages(t, logOutput))
			},
		},
		{
			Name:          "encrypted multi-line value is decrypted properly",
			ECVKeyContent: encryptionKey,
			InstallConfig: fmt.Sprintf("message: %s\nuse-console-log: true\n", encryptedMultilineValue),
			RuntimeConfig: fmt.Sprintf("message: %s\nlogging:\n  level: warn\n", encryptedMultilineValue),
			Verify: func(info witchcraft.InitInfo) error {
				if msg := info.InstallConfig.(*messageInstall).Message; msg != decryptedMultilineValue {
					return fmt.Errorf("expected %q got %q", decryptedValue, msg)
				}
				if msg := info.RuntimeConfig.Current().(*message).Message; msg != decryptedMultilineValue {
					return fmt.Errorf("expected %q got %q", decryptedValue, msg)
				}
				return nil
			},
			VerifyLog: func(t *testing.T, logOutput []byte) {
				assert.Equal(t, []string{"abort startup"}, getLogFileMessages(t, logOutput))
			},
		},
		{
			Name:          "no ecv required if no encrypted config",
			ECVKeyContent: "test_invalid_key_material",
			InstallConfig: "message: hello install\nuse-console-log: true\n",
			RuntimeConfig: "message: hello runtime\nlogging:\n  level: warn\n",
			Verify: func(info witchcraft.InitInfo) error {
				if msg := info.InstallConfig.(*messageInstall).Message; msg != "hello install" {
					return fmt.Errorf("expected %q got %q", "hello install", msg)
				}
				if msg := info.RuntimeConfig.Current().(*message).Message; msg != "hello runtime" {
					return fmt.Errorf("expected %q got %q", "hello runtime", msg)
				}
				return nil
			},
			VerifyLog: func(t *testing.T, logOutput []byte) {
				assert.Equal(t, []string{"abort startup"}, getLogFileMessages(t, logOutput))
			},
		},
		{
			Name:          "warning printed if ecv key fails in runtime",
			ECVKeyContent: "test_invalid_key_material",
			InstallConfig: "message: hello install\nuse-console-log: true\n",
			RuntimeConfig: fmt.Sprintf("message: %s\nlogging:\n  level: warn\n", encryptedValue),
			Verify: func(info witchcraft.InitInfo) error {
				if msg := info.InstallConfig.(*messageInstall).Message; msg != "hello install" {
					return fmt.Errorf("expected %q got %q", "hello install", msg)
				}
				if msg := info.RuntimeConfig.Current().(*message).Message; msg != encryptedValue {
					return fmt.Errorf("expected %q got %q", encryptedValue, msg)
				}
				return nil
			},
			VerifyLog: func(t *testing.T, logOutput []byte) {
				assert.Equal(t, []string{"Failed to decrypt encrypted runtime configuration", "abort startup"}, getLogFileMessages(t, logOutput))
			},
		},
		{
			Name:          "warning printed if encrypted config cannot be decrypted in runtime",
			ECVKeyContent: encryptionKey,
			InstallConfig: "message: hello install\nuse-console-log: true\n",
			RuntimeConfig: fmt.Sprintf("message: %s\nlogging:\n  level: warn\n", "${enc:invalid}"),
			Verify: func(info witchcraft.InitInfo) error {
				if msg := info.InstallConfig.(*messageInstall).Message; msg != "hello install" {
					return fmt.Errorf("expected %q got %q", "hello install", msg)
				}
				if msg := info.RuntimeConfig.Current().(*message).Message; msg != "${enc:invalid}" {
					return fmt.Errorf("expected %q got %q", "${enc:invalid}", msg)
				}
				return nil
			},
			VerifyLog: func(t *testing.T, logOutput []byte) {
				assert.Equal(t, []string{"Failed to decrypt encrypted runtime configuration", "abort startup"}, getLogFileMessages(t, logOutput))
			},
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			tmpDir, err := ioutil.TempDir("", "TestEncryptedConfig_")
			require.NoError(t, err)
			defer func() {
				_ = os.RemoveAll(tmpDir)
			}()
			ecvKeyFile := filepath.Join(tmpDir, "ecv.key")
			err = ioutil.WriteFile(ecvKeyFile, []byte(test.ECVKeyContent), 0600)
			require.NoError(t, err)
			installFile := filepath.Join(tmpDir, "install.yml")
			err = ioutil.WriteFile(installFile, []byte(test.InstallConfig), 0644)
			require.NoError(t, err)
			runtimeFile := filepath.Join(tmpDir, "runtime.yml")
			err = ioutil.WriteFile(runtimeFile, []byte(test.RuntimeConfig), 0644)
			require.NoError(t, err)

			logOutputBuffer := &bytes.Buffer{}
			server := witchcraft.NewServer().
				WithECVKeyFromFile(ecvKeyFile).
				WithInstallConfigFromFile(installFile).
				WithInstallConfigType(&messageInstall{}).
				WithRuntimeConfigFromFile(runtimeFile).
				WithRuntimeConfigType(&message{}).
				WithLoggerStdoutWriter(logOutputBuffer).
				WithDisableGoRuntimeMetrics().
				WithSelfSignedCertificate().
				WithInitFunc(func(ctx context.Context, info witchcraft.InitInfo) (cleanup func(), rErr error) {
					if err := test.Verify(info); err != nil {
						return nil, err
					}
					return nil, fmt.Errorf("abort startup")
				})
			// Execute server, expecting error
			err = server.Start()
			require.EqualError(t, err, "abort startup")

			if test.VerifyLog != nil {
				test.VerifyLog(t, logOutputBuffer.Bytes())
			}
		})
	}
}
