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

package wrouter_test

import (
	"testing"

	"github.com/palantir/witchcraft-go-server/v2/wrouter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidPathSegments(t *testing.T) {
	for i, currCase := range []struct {
		name string
		path string
		want []wrouter.PathSegment
	}{
		{
			"alphanumeric strings are valid path param names",
			"/a/{b1}",
			[]wrouter.PathSegment{
				{Type: wrouter.LiteralSegment, Value: "a"},
				{Type: wrouter.PathParamSegment, Value: "b1"},
			},
		},
		{
			"alphanumeric strings are valid trailing path param names",
			"/a/{b1*}",
			[]wrouter.PathSegment{
				{Type: wrouter.LiteralSegment, Value: "a"},
				{Type: wrouter.TrailingPathParamSegment, Value: "b1"},
			},
		},
		{
			"empty path is valid",
			"/",
			[]wrouter.PathSegment{
				{Type: wrouter.LiteralSegment, Value: ""},
			},
		},
		{
			"trailing param can be empty",
			"/a/b/",
			[]wrouter.PathSegment{
				{Type: wrouter.LiteralSegment, Value: "a"},
				{Type: wrouter.LiteralSegment, Value: "b"},
				{Type: wrouter.LiteralSegment, Value: ""},
			},
		},
	} {
		got, err := wrouter.NewPathTemplate(currCase.path)
		require.NoError(t, err, "Case %d: %s", i, currCase.name)
		assert.Equal(t, currCase.want, got.Segments(), "Case %d: %s", i, currCase.name)
	}
}

func TestInvalidPathSegments(t *testing.T) {
	for i, currCase := range []struct {
		Name      string
		Path      string
		WantError string
	}{
		{
			"path must start with a '/'",
			"a",
			"path \"a\" must match regexp ^/[a-zA-Z0-9_{}/.\\-*]*$",
		},
		{
			"regular path segment may not contain special characters besides '-', '_'",
			"/a/:b",
			"path \"/a/:b\" must match regexp ^/[a-zA-Z0-9_{}/.\\-*]*$",
		},
		{
			"path segment starting with an opening curly brace must have a matching close brace",
			"/a/{b/",
			"invalid path param {b in path /a/{b/",
		},
		{
			"path segment may not have unmatched close curly brace",
			"/a/b}/",
			"segment b} of path /a/b}/ includes special characters",
		},
		{
			"slashes may not appear between open and close braces",
			"/a{/b}/",
			"segment a{ of path /a{/b}/ includes special characters",
		},
		{
			"path param names must be alphanumeric: cannot contain '.'",
			"/a/{b.}/",
			"invalid path param {b.} in path /a/{b.}/",
		},
		{
			"path param names must be alphanumeric: cannot contain '-'",
			"/a/{b-}/",
			"invalid path param {b-} in path /a/{b-}/",
		},
		{
			"trailing match path params may only appear at the end of a path",
			"/a/{b*}/c",
			"trailing match path param {b*} does not appear at end of path /a/{b*}/c",
		},
		{
			"If matching curly braces appear in a path segment, they must enclose the entire segment",
			"/a/b{c}",
			"segment b{c} of path /a/b{c} includes special characters",
		},
		{
			"duplicate nested braces may not appear in path segments",
			"/{{a}}",
			"invalid path param {{a}} in path /{{a}}",
		},
		{
			"path param names must be nonempty strings",
			"/{}",
			"invalid path param {} in path /{}",
		},
		{
			"trailing path param names must be nonempty strings",
			"/{*}",
			"invalid path param {*} in path /{*}",
		},
		{
			"only one '*' may occur in a trailing match path param",
			"/{a**}",
			"invalid path param {a**} in path /{a**}",
		},
		{
			"path param names must be unique within a path",
			"/{a}/{a}",
			"path param \"a\" appears more than once in path /{a}/{a}",
		},
		{
			"path param names, trailing match or not, must be unique within a path",
			"/{a}/{a*}",
			"path param \"a\" appears more than once in path /{a}/{a*}",
		},
		{
			"empty segments can only occur at the end",
			"//",
			"segment at index 0 of path // with segments [ ] was empty",
		},
	} {
		_, err := wrouter.NewPathTemplate(currCase.Path)
		assert.EqualError(t, err, currCase.WantError, "Case %d: %s\n%v", i, currCase.Name, err)
	}
}
