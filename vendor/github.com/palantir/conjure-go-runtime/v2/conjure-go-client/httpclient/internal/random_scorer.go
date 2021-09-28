// Copyright (c) 2021 Palantir Technologies. All rights reserved.
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

package internal

import (
	"math/rand"
	"net/http"
)

type randomScorer struct {
	uris      []string
	nanoClock func() int64
}

func (n *randomScorer) GetURIsInOrderOfIncreasingScore() []string {
	uris := make([]string, len(n.uris))
	copy(uris, n.uris)
	rand.New(rand.NewSource(n.nanoClock())).Shuffle(len(uris), func(i, j int) {
		uris[i], uris[j] = uris[j], uris[i]
	})
	return uris
}

func (n *randomScorer) RoundTrip(req *http.Request, next http.RoundTripper) (*http.Response, error) {
	return next.RoundTrip(req)
}

// NewRandomURIScoringMiddleware returns a URI scorer that randomizes the order of URIs when scoring using a rand.Rand
// seeded by the nanoClock function. The middleware no-ops on each request.
func NewRandomURIScoringMiddleware(uris []string, nanoClock func() int64) URIScoringMiddleware {
	return &randomScorer{
		uris:      uris,
		nanoClock: nanoClock,
	}
}
