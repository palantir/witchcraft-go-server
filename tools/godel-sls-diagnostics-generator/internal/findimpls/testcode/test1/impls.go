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

package test1

const MyInterfaceImpl1IntConst = 42

// MyInterfaceImpl1 implements MyInterface
type MyInterfaceImpl1 struct{}

func (m MyInterfaceImpl1) String() string {
	return "my string"
}

func (m MyInterfaceImpl1) Int() int {
	return MyInterfaceImpl1IntConst
}

// MyInterfaceImpl2 implements MyInterface and myPrivateInterface
type MyInterfaceImpl2 struct{}

func (m MyInterfaceImpl2) String() string {
	return "my string"
}

func (m MyInterfaceImpl2) Int() int {
	return MyInterfaceImpl1IntConst
}

func (m MyInterfaceImpl2) PublicString() string {
	return "public string"
}

func (m MyInterfaceImpl2) privateString() string {
	return "private string"
}

var _ = myInterfaceImpl3{}

// myInterfaceImpl3 is a private type that implements MyInterface and myPrivateInterface
type myInterfaceImpl3 struct{}

func (m myInterfaceImpl3) String() string {
	return "my string"
}

func (m myInterfaceImpl3) Int() int {
	return MyInterfaceImpl1IntConst
}

func (m myInterfaceImpl3) PublicString() string {
	return "public string"
}

func (m myInterfaceImpl3) privateString() string {
	return "private string"
}

type MyInterfaceImpl4 int

func (m MyInterfaceImpl4) String() string {
	return "my string"
}

func (m MyInterfaceImpl4) Int() int {
	return MyInterfaceImpl1IntConst
}

type MyInterfaceImpl5 func(string)

func (m MyInterfaceImpl5) String() string {
	return "my string"
}

func (m MyInterfaceImpl5) Int() int {
	return MyInterfaceImpl1IntConst
}
