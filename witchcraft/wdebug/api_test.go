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

package wdebug

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiagnosticType_Validate(t *testing.T) {
	for _, test := range []struct {
		name      string
		in        string
		expectErr bool
	}{
		{
			name:      "valid",
			in:        "a0.b1.c2.v1",
			expectErr: false,
		},
		{
			name:      "invalid: no version suffix",
			in:        "a.b.c",
			expectErr: true,
		},
		{
			name:      "invalid: segments preceding version suffix include non-alphanumeric characters",
			in:        "segment-with-a-bang!.v1",
			expectErr: true,
		},
		{
			name:      "invalid: empty segment preceding version suffix",
			in:        "segment..v1",
			expectErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := DiagnosticType(test.in).Validate()
			if test.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
