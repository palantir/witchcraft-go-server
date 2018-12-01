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

package wtracing

import (
	"net/http"
	"strings"

	"github.com/palantir/witchcraft-go-error"
)

const (
	b3TraceID      = "X-B3-TraceId"
	b3SpanID       = "X-B3-SpanId"
	b3ParentSpanID = "X-B3-ParentSpanId"
	b3Sampled      = "X-B3-Sampled"
	b3Flags        = "X-B3-Flags"
)

// InjectB3HeaderVals takes the provided SpanContext and sets the appropriate B3 values in the provided header.
func InjectB3HeaderVals(req *http.Request, sc SpanContext) {
	if len(sc.TraceID) > 0 && len(sc.ID) > 0 {
		req.Header.Set(b3TraceID, string(sc.TraceID))
		req.Header.Set(b3SpanID, string(sc.ID))
		if parentID := sc.ParentID; parentID != nil {
			req.Header.Set(b3ParentSpanID, string(*sc.ParentID))
		}
	}

	if sc.Debug {
		req.Header.Set(b3Flags, "1")
	} else if sampled := sc.Sampled; sampled != nil {
		sampledVal := "0"
		if *sampled {
			sampledVal = "1"
		}
		req.Header.Set(b3Sampled, sampledVal)
	}
}

// ExtractB3HeaderVals returns a SpanContext created by extracting the B3 values from the provided header. If the values
// in the provided header do not constitute a valid SpanContext (for example, if it is missing a TraceID or SpanID,
// has an unsupported "Sampled" value, etc.), the "Err" field of the returned SpanContext will be non-nil and will
// contain the error. However, even if the "Err" field is set, all of the values that could be extracted from the header
// and set on the returned context.
func ExtractB3HeaderVals(req *http.Request) SpanContext {
	sc := SpanContext{}

	traceID := strings.ToLower(req.Header.Get(b3TraceID))
	if traceID == "" {
		sc.Err = werror.Error("TraceID must be present",
			werror.SafeParam("traceId", traceID),
		)
	}
	sc.TraceID = TraceID(traceID)

	spanID := strings.ToLower(req.Header.Get(b3SpanID))
	if spanID == "" {
		sc.Err = werror.Error("SpanID must be present",
			werror.SafeParam("spanId", spanID),
		)
	}
	sc.ID = SpanID(spanID)

	var parentIDVal *SpanID
	if parentID := strings.ToLower(req.Header.Get(b3ParentSpanID)); parentID != "" {
		if traceID == "" || spanID == "" {
			sc.Err = werror.Error("ParentID was present but TraceID or SpanID was not",
				werror.SafeParam("parentId", parentID),
				werror.SafeParam("traceId", traceID),
				werror.SafeParam("spanId", spanID),
			)
		}
		parentIDVal = (*SpanID)(&parentID)
	}
	sc.ParentID = parentIDVal

	var sampledVal *bool
	switch sampledHeader := strings.ToLower(req.Header.Get(b3Sampled)); sampledHeader {
	case "0", "false":
		boolVal := false
		sampledVal = &boolVal
	case "1", "true":
		boolVal := true
		sampledVal = &boolVal
	case "":
		// keep nil
	default:
		sc.Err = werror.Error("Sampled value was invalid", werror.SafeParam("sampledHeaderVal", sampledHeader))
	}
	debug := req.Header.Get(b3Flags) == "1"
	if debug {
		sampledVal = nil
	}
	sc.Sampled = sampledVal
	sc.Debug = debug

	return sc
}
