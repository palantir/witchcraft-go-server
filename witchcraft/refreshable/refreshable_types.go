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

package refreshable

import (
	"time"
)

type String interface {
	Refreshable
	CurrentString() string
}

type StringPtr interface {
	Refreshable
	CurrentStringPtr() *string
}

type StringSlice interface {
	Refreshable
	CurrentStringSlice() []string
}

type Int interface {
	Refreshable
	CurrentInt() int
}

type IntPtr interface {
	Refreshable
	CurrentIntPtr() *int
}

type Bool interface {
	Refreshable
	CurrentBool() bool
}

type BoolPtr interface {
	Refreshable
	CurrentBoolPtr() *bool
}

// Duration is a Refreshable that can return the current time.Duration.
type Duration interface {
	Refreshable
	CurrentDuration() time.Duration
}

type DurationPtr interface {
	Refreshable
	CurrentDurationPtr() *time.Duration
}

func NewBool(in Refreshable) Bool {
	return refreshableTyped{
		Refreshable: in,
	}
}

func NewBoolPtr(in Refreshable) BoolPtr {
	return refreshableTyped{
		Refreshable: in,
	}
}

func NewDuration(in Refreshable) Duration {
	return refreshableTyped{
		Refreshable: in,
	}
}

func NewDurationPtr(in Refreshable) Duration {
	return refreshableTyped{
		Refreshable: in,
	}
}

func NewString(in Refreshable) String {
	return refreshableTyped{
		Refreshable: in,
	}
}

func NewStringPtr(in Refreshable) StringPtr {
	return refreshableTyped{
		Refreshable: in,
	}
}

func NewStringSlice(in Refreshable) StringSlice {
	return refreshableTyped{
		Refreshable: in,
	}
}

func NewInt(in Refreshable) Int {
	return refreshableTyped{
		Refreshable: in,
	}
}

func NewIntPtr(in Refreshable) IntPtr {
	return refreshableTyped{
		Refreshable: in,
	}
}

var _ Bool = (*refreshableTyped)(nil)
var _ BoolPtr = (*refreshableTyped)(nil)
var _ Duration = (*refreshableTyped)(nil)
var _ Int = (*refreshableTyped)(nil)
var _ IntPtr = (*refreshableTyped)(nil)
var _ String = (*refreshableTyped)(nil)
var _ StringPtr = (*refreshableTyped)(nil)
var _ StringSlice = (*refreshableTyped)(nil)

type refreshableTyped struct {
	Refreshable
}

func (rt refreshableTyped) CurrentString() string {
	return rt.Current().(string)
}

func (rt refreshableTyped) CurrentStringPtr() *string {
	return rt.Current().(*string)
}

func (rt refreshableTyped) CurrentStringSlice() []string {
	return rt.Current().([]string)
}

func (rt refreshableTyped) CurrentInt() int {
	return rt.Current().(int)
}

func (rt refreshableTyped) CurrentIntPtr() *int {
	return rt.Current().(*int)
}

func (rt refreshableTyped) CurrentBool() bool {
	return rt.Current().(bool)
}

func (rt refreshableTyped) CurrentBoolPtr() *bool {
	return rt.Current().(*bool)
}

func (rt refreshableTyped) CurrentDuration() time.Duration {
	return rt.Current().(time.Duration)
}
