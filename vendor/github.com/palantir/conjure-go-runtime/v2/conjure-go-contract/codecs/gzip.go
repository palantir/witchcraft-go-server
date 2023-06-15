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

package codecs

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

var _ Codec = codecGZIP{}

// GZIP wraps an existing Codec and uses gzip for compression and decompression.
func GZIP(codec Codec) Codec {
	return &codecGZIP{contentCodec: codec}
}

type codecGZIP struct {
	contentCodec Codec
}

func (c codecGZIP) Accept() string {
	return c.contentCodec.Accept()
}

func (c codecGZIP) Decode(r io.Reader, v interface{}) error {
	gzipReader, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %s", err.Error())
	}
	return c.contentCodec.Decode(gzipReader, v)
}

func (c codecGZIP) Unmarshal(data []byte, v interface{}) error {
	return c.Decode(bytes.NewBuffer(data), v)
}

func (c codecGZIP) ContentType() string {
	return c.contentCodec.ContentType()
}

func (c codecGZIP) Encode(w io.Writer, v interface{}) (err error) {
	gzipWriter := gzip.NewWriter(w)
	defer func() {
		if closeErr := gzipWriter.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()
	return c.contentCodec.Encode(gzipWriter, v)
}

func (c codecGZIP) Marshal(v interface{}) ([]byte, error) {
	var buffer bytes.Buffer
	err := c.Encode(&buffer, v)
	if err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buffer.Bytes(), []byte{'\n'}), nil
}
