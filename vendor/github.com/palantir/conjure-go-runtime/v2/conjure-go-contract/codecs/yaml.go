// Copyright (c) 2025 Palantir Technologies. All rights reserved.
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

package codecs

import (
	"fmt"
	"io"

	"gopkg.in/yaml.v2"
)

const (
	contentTypeYAML = "application/yaml"
)

var _ Codec = codecYAML{}

// YAML codec encodes and decodes YAML using gopkg.in/yaml.v2.
func YAML() Codec {
	return &codecYAML{}
}

type codecYAML struct{}

func (codecYAML) Accept() string {
	return contentTypeYAML
}

func (c codecYAML) Decode(r io.Reader, v interface{}) error {
	return yaml.NewDecoder(r).Decode(v)
}

func (c codecYAML) Unmarshal(data []byte, v interface{}) error {
	return yaml.Unmarshal(data, *&v) // work around outparamcheck
}

func (codecYAML) ContentType() string {
	return contentTypeYAML
}

func (c codecYAML) Encode(w io.Writer, v interface{}) error {
	encoder := yaml.NewEncoder(w)
	err := encoder.Encode(v)
	if err != nil {
		return err
	}
	err = encoder.Close()
	if err != nil {
		return fmt.Errorf("failed to close yaml.v2 encoder: %s", err.Error())
	}
	return nil
}

func (c codecYAML) Marshal(v interface{}) ([]byte, error) {
	return yaml.Marshal(v)
}
