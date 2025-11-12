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

	werror "github.com/palantir/witchcraft-go-error"
	"google.golang.org/protobuf/proto"
)

const (
	contentTypeProtobuf = "Application/x-protobuf"
)

// Protobuf codec encodes and decodes protobuf requests and responses using
// google.golang.org/protobuf.
var Protobuf Codec = codecProtobuf{}

type codecProtobuf struct{}

func (codecProtobuf) Accept() string {
	return contentTypeProtobuf
}

func (codecProtobuf) Decode(r io.Reader, v interface{}) error {
	msg, ok := v.(proto.Message)
	if !ok {
		return werror.Error("failed to decode protobuf data from type which does not implement proto.Message")
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return proto.Unmarshal(data, msg)
}

func (c codecProtobuf) Unmarshal(data []byte, v interface{}) error {
	msg, ok := v.(proto.Message)
	if !ok {
		return werror.Error("failed to decode protobuf data from type which does not implement proto.Message")
	}
	return proto.Unmarshal(data, msg)
}

func (codecProtobuf) ContentType() string {
	return contentTypeProtobuf
}

func (c codecProtobuf) Encode(w io.Writer, v interface{}) error {
	buf, err := c.Marshal(v)
	if err != nil {
		return err
	}

	_, err = w.Write(buf)
	return err
}

func (c codecProtobuf) Marshal(v interface{}) ([]byte, error) {
	msg, ok := v.(proto.Message)
	if !ok {
		return nil, werror.Error("failed to encode protobuf data from type which does not implement proto.Message")
	}
	return proto.Marshal(msg)
}
