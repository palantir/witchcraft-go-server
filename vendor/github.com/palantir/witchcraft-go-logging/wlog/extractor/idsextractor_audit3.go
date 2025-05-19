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

package extractor

import (
	"net/http"
	"strings"
)

const (
	UserAgentKey = "userAgent"
	SourceIPKey  = "sourceIp"
	ForwardIPs   = "forwardedForIps"
)

// newAudit3IDsFromHeaderExtractor returns an extractor that sets the keys for values used by audit.3 logs.
func newAudit3IDsFromHeaderExtractor() IDsFromRequest {
	return &audit3IDsExtractor{}
}

type audit3IDsExtractor struct{}

func (e *audit3IDsExtractor) ExtractIDs(req *http.Request) map[string]string {
	sourceIP := req.RemoteAddr
	if idx := strings.LastIndex(sourceIP, ":"); idx != -1 {
		sourceIP = sourceIP[:idx]
	}
	return map[string]string{
		UserAgentKey: req.Header.Get("User-Agent"),
		ForwardIPs:   strings.Join(req.Header.Values("X-Forwarded-For"), ","),
		SourceIPKey:  sourceIP,
	}
}
