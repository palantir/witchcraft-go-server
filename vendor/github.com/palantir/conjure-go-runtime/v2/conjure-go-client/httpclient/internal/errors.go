// Copyright (c) 2020 Palantir Technologies. All rights reserved.
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
	werror "github.com/palantir/witchcraft-go-error"
)

// StatusCodeFromError retrieves the 'statusCode' parameter from the provided werror.
// If the error is not a werror or does not have the statusCode param, ok is false.
//
// The default client error decoder sets the statusCode parameter on its returned errors. Note that, if a custom error
// decoder is used, this function will only return a status code for the error if the custom decoder sets a 'statusCode'
// parameter on the error.
func StatusCodeFromError(err error) (statusCode int, ok bool) {
	statusCodeI, ok := werror.ParamFromError(err, "statusCode")
	if statusCodeI == nil {
		return 0, false
	}
	statusCode, ok = statusCodeI.(int)
	return statusCode, ok
}

// LocationFromError retrieves the 'location' parameter from the provided werror.
// If the error is not a werror or does not have the location param, ok is false.
//
// The default client error decoder sets the location parameter on its returned errors
// if the status code is 3xx and a location is set in the response header
func LocationFromError(err error) (location string, ok bool) {
	locationI, _ := werror.ParamFromError(err, "location")
	if locationI == nil {
		return "", false
	}
	location, ok = locationI.(string)
	return location, ok
}
