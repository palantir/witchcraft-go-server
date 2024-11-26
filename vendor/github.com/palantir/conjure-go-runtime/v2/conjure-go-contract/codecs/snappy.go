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

package codecs

import (
	"io"

	"github.com/golang/snappy"
)

var _ Codec = codecSNAPPY{}

// Snappy wraps an existing Codec and uses snappy with no-framing for
// compression and decompression using github.com/golang/snappy.
// Ref: https://github.com/google/snappy/blob/main/format_description.txt
func Snappy(codec Codec) Codec {
	return &codecSNAPPY{contentCodec: codec}
}

type codecSNAPPY struct {
	contentCodec Codec
}

func (c codecSNAPPY) Accept() string {
	return c.contentCodec.Accept()
}

func (c codecSNAPPY) Decode(r io.Reader, v interface{}) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	decoded, err := snappy.Decode(nil, data)
	if err != nil {
		return err
	}
	return c.contentCodec.Unmarshal(decoded, v)
}

func (c codecSNAPPY) Unmarshal(data []byte, v interface{}) error {
	decoded, err := snappy.Decode(nil, data)
	if err != nil {
		return err
	}
	return c.contentCodec.Unmarshal(decoded, v)
}

func (c codecSNAPPY) ContentType() string {
	return c.contentCodec.ContentType()
}

func (c codecSNAPPY) Encode(w io.Writer, v interface{}) error {
	data, err := c.contentCodec.Marshal(v)
	if err != nil {
		return err
	}
	encoded := snappy.Encode(nil, data)
	_, err = w.Write(encoded)
	return err
}

func (c codecSNAPPY) Marshal(v interface{}) ([]byte, error) {
	data, err := c.contentCodec.Marshal(v)
	if err != nil {
		return nil, err
	}
	encoded := snappy.Encode(nil, data)
	return encoded, nil
}
