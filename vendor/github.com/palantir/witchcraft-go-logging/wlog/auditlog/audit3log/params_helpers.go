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

package audit3log

import (
	"strings"
)

// OriginsFromXForwardedForHeaderValue returns the IP addresses from the provided value, which should be the value of
// the "X-Forwarded-For" header. Trims whitespace and omit any values that are blank or have the value "unknown".
func OriginsFromXForwardedForHeaderValue(forwardedFor string) []string {
	// https://tools.ietf.org/html/rfc7239#section-6.2
	const unknownIP = "unknown"
	var origins []string
	for _, forwardedForValue := range strings.Split(forwardedFor, ",") {
		for _, ip := range strings.Split(forwardedForValue, ",") {
			ip = strings.TrimSpace(ip)
			if ip == "" || strings.ToLower(ip) == unknownIP {
				continue
			}
			origins = append(origins, ip)
		}
	}
	return origins
}

// OriginFromForwardedOrSourceOrigin returns the value that should be used for the "Origin" parameter of an audit.3 log
// entry given the forwarded origins and sourceOrigin. If the forwarded origins are non-empty, the first one is used;
// otherwise, the sourceOrigin is used.
func OriginFromForwardedOrSourceOrigin(forwardedOrigins []string, sourceOrigin string) string {
	if len(forwardedOrigins) > 0 {
		return forwardedOrigins[0]
	}
	return sourceOrigin
}
