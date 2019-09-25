// Copyright (c) 2019 Palantir Technologies. All rights reserved.
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

package window

import (
	"testing"
	"time"

	werror "github.com/palantir/witchcraft-go-error"
	"github.com/stretchr/testify/assert"
)

func TestSlidingWindowManager_NoErrors(t *testing.T) {
	manager := NewSlidingWindowManager(time.Millisecond)
	errors := manager.GetErrors()
	assert.Nil(t, errors)
}

func TestSlidingWindowManager_AllErrorsUpToDate(t *testing.T) {
	manager := NewSlidingWindowManager(time.Second)
	manager.SubmitError(werror.Error("error #1"))
	manager.SubmitError(werror.Error("error #2"))
	manager.SubmitError(werror.Error("error #3"))
	errors := manager.GetErrors()
	assert.EqualValues(t, len(errors), 3)
	assert.EqualValues(t, werror.Error("error #1").Error(), errors[0].Error())
	assert.EqualValues(t, werror.Error("error #2").Error(), errors[1].Error())
	assert.EqualValues(t, werror.Error("error #3").Error(), errors[2].Error())
}

func TestSlidingWindowManager_AllErrorsOutOfDate(t *testing.T) {
	manager := NewSlidingWindowManager(time.Millisecond)
	manager.SubmitError(werror.Error("error #1"))
	manager.SubmitError(werror.Error("error #2"))
	manager.SubmitError(werror.Error("error #3"))
	<-time.After(2 * time.Millisecond)
	errors := manager.GetErrors()
	assert.Nil(t, errors)
}

func TestSlidingWindowManager_SomeErrorsOutOfDate(t *testing.T) {
	manager := NewSlidingWindowManager(500 * time.Millisecond)
	manager.SubmitError(werror.Error("error #1"))
	manager.SubmitError(werror.Error("error #2"))
	<-time.After(time.Second)
	manager.SubmitError(werror.Error("error #3"))
	manager.SubmitError(werror.Error("error #4"))
	errors := manager.GetErrors()
	assert.EqualValues(t, len(errors), 2)
	assert.EqualValues(t, werror.Error("error #3").Error(), errors[0].Error())
	assert.EqualValues(t, werror.Error("error #4").Error(), errors[1].Error())
}
