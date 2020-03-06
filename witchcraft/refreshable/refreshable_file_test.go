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
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	testStr1 = "renderConf1"
	testStr2 = "renderConf2"
)

func TestRefreshableChanges(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	assert.NoError(t, err)
	fileToWrite := path.Join(tempDir, "file")
	writeFileHelper(t, fileToWrite, testStr1)
	r, err := NewFileRefreshable(context.Background(), fileToWrite, time.Millisecond * 50)
	assert.NoError(t, err)
	str := getStringFromRefreshable(t, r)
	assert.Equal(t, str, "renderConf1")
	writeFileHelper(t, fileToWrite, testStr2)
	time.Sleep(time.Millisecond * 80)
	str = getStringFromRefreshable(t, r)
	assert.Equal(t, str, "renderConf2")
	err = os.RemoveAll(tempDir)
	assert.NoError(t, err)
}

func TestRefreshableCanFollowSymLink(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	// We will have fileToWriteNew point at fileToWriteOld
	assert.NoError(t, err)
	fileToWriteOld := path.Join(tempDir, "fileOld")
	fileToWriteNew := path.Join(tempDir, "fileNew")
	// Write the old file
	writeFileHelper(t, fileToWriteOld, testStr1)
	// Symlink the old file to point at the new file
	err = os.Symlink(fileToWriteOld, fileToWriteNew)
	assert.NoError(t, err)
	// Point the refreshable towards the new file
	r, err := NewFileRefreshable(context.Background(), fileToWriteNew, time.Millisecond * 50)
	assert.NoError(t, err)
	str := getStringFromRefreshable(t, r)
	assert.Equal(t, str, "renderConf1")
	// Update the symlink file
	writeFileHelper(t, fileToWriteOld, testStr2)
	time.Sleep(time.Millisecond * 80)
	str = getStringFromRefreshable(t, r)
	assert.Equal(t, str, "renderConf2")
	err = os.RemoveAll(tempDir)
	assert.NoError(t, err)
}

func TestComplexSymLink(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	fileToWriteOld := path.Join(tempDir, "fileOld")
	fileToWriteAgain := path.Join(tempDir, "fileOldAgain")
	fileToWriteNew := path.Join(tempDir, "fileNew")
	assert.NoError(t, err)
	v1 := []byte("renderConf1")
	v2 := []byte("renderConf2")
	err = ioutil.WriteFile(fileToWriteOld, v1, 0777)
	assert.NoError(t, err)
	err = ioutil.WriteFile(fileToWriteAgain, v1, 0777)
	assert.NoError(t, err)
	err = os.Symlink(fileToWriteOld, fileToWriteNew)
	assert.NoError(t, err)
	r, err := NewFileRefreshable(context.Background(), fileToWriteNew, time.Millisecond * 50)
	assert.NoError(t, err)
	str := getStringFromRefreshable(t, r)
	assert.Equal(t, str, "renderConf1")
	err = os.Remove(fileToWriteNew)
	assert.NoError(t, err)
	err = os.Symlink(fileToWriteAgain, fileToWriteNew)
	assert.NoError(t, err)
	str = getStringFromRefreshable(t, r)
	assert.Equal(t, str, "renderConf1")
	err = ioutil.WriteFile(fileToWriteAgain, v2, 0777)
	assert.NoError(t, err)
	time.Sleep(time.Second)
	str = getStringFromRefreshable(t, r)
	assert.Equal(t, str, "renderConf2")
	err = os.RemoveAll(tempDir)
	assert.NoError(t, err)
}

func writeFileHelper(t *testing.T, path, value string) {
	err := ioutil.WriteFile(path, []byte(value), 0777)
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
