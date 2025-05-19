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

package categories

type Classification string

const (
	Classification_RESOURCE     = Classification("RESOURCE")
	Classification_TOKEN        = Classification("TOKEN")
	Classification_UID          = Classification("UID")
	Classification_DATA         = Classification("DATA")
	Classification_METADATA     = Classification("METADATA")
	Classification_USER_INPUT   = Classification("USER_INPUT")
	Classification_CONSTANT     = Classification("CONSTANT")
	Classification_PASS_THROUGH = Classification("PASS_THROUGH")
)
