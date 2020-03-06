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
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"
)

func TestSimple(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	assert.NoError(t, err)
	v1 := []byte("renderConf1")
	v2 := []byte("renderConf2")
	fileToWrite := path.Join(tempDir, "renderConf")
	err = ioutil.WriteFile(fileToWrite, v1, 0777)
	assert.NoError(t, err)
	r, err := NewFileRefreshable(context.Background(), fileToWrite)
	assert.NoError(t, err)
	str := getStringFromRefreshable(t, r)
	fmt.Println(str)
	err = ioutil.WriteFile(fileToWrite, v2, 0777)
	assert.NoError(t, err)
	time.Sleep(time.Second)
	str = getStringFromRefreshable(t, r)
	fmt.Println(str)
	err = os.RemoveAll(tempDir)
	assert.NoError(t, err)
}

func getStringFromRefreshable(t *testing.T, r Refreshable) string {
	current := r.Current()
	currentCasted, ok := current.([]uint8)
	assert.True(t, ok)
	b := make([]byte, len(currentCasted))
	for i, v := range currentCasted {
		b[i] = v
	}
	return string(b)
}
