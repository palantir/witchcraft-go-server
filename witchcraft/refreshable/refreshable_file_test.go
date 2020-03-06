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
	r, err := NewFileRefreshableWithDuration(context.Background(), fileToWrite, time.Millisecond*50)
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

func TestRefreshableSubscribes(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	assert.NoError(t, err)
	fileToWrite := path.Join(tempDir, "file")
	writeFileHelper(t, fileToWrite, testStr1)
	r, err := NewFileRefreshableWithDuration(context.Background(), fileToWrite, time.Millisecond*50)
	assert.NoError(t, err)
	str := getStringFromRefreshable(t, r)
	assert.Equal(t, str, "renderConf1")
	count := 0
	mappingFunc := func(mapped interface{}) {
		count = count + 1
		str := getStringFromInterface(t, mapped)
		assert.Equal(t, str, "renderConf2")
	}
	r.Subscribe(mappingFunc)
	writeFileHelper(t, fileToWrite, testStr2)
	time.Sleep(time.Millisecond * 80)
	assert.Equal(t, 1, count)

	err = os.RemoveAll(tempDir)
	assert.NoError(t, err)
}


func TestRefreshableCanFollowSymLink(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	// We will have fileToWritePointingAtActual point at fileToWriteActual
	assert.NoError(t, err)
	fileToWriteActual := path.Join(tempDir, "fileToWriteActual")
	fileToWritePointingAtActual := path.Join(tempDir, "fileToWritePointingAtActual")
	// Write the old file
	writeFileHelper(t, fileToWriteActual, testStr1)
	// Symlink the old file to point at the new file
	err = os.Symlink(fileToWriteActual, fileToWritePointingAtActual)
	assert.NoError(t, err)
	// Point the refreshable towards the new file
	r, err := NewFileRefreshableWithDuration(context.Background(), fileToWritePointingAtActual, time.Millisecond*50)
	assert.NoError(t, err)
	str := getStringFromRefreshable(t, r)
	assert.Equal(t, str, "renderConf1")
	// Update the actual file
	writeFileHelper(t, fileToWriteActual, testStr2)
	time.Sleep(time.Millisecond * 80)
	str = getStringFromRefreshable(t, r)
	assert.Equal(t, str, "renderConf2")
	err = os.RemoveAll(tempDir)
	assert.NoError(t, err)
}

func TestRefreshableCanFollowMultipleSymLinks(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	// We will have fileToWritePointingAtActual point at fileToWriteActual
	assert.NoError(t, err)
	fileToWriteActual := path.Join(tempDir, "fileToWriteActual")
	fileToWritePointingAtActual := path.Join(tempDir, "fileToWritePointingAtActual")
	fileToWritePointingAtSymlink := path.Join(tempDir, "fileToWritePointingAtSymlink")
	// Write the old file
	writeFileHelper(t, fileToWriteActual, testStr1)
	// Symlink the old file to point at the new file
	err = os.Symlink(fileToWriteActual, fileToWritePointingAtActual)
	assert.NoError(t, err)
	// Symlink a to a symlink
	err = os.Symlink(fileToWritePointingAtActual, fileToWritePointingAtSymlink)
	assert.NoError(t, err)
	// Point the refreshable towards the new file
	r, err := NewFileRefreshableWithDuration(context.Background(), fileToWritePointingAtSymlink, time.Millisecond*50)
	assert.NoError(t, err)
	str := getStringFromRefreshable(t, r)
	assert.Equal(t, str, "renderConf1")
	// Update the symlink file
	writeFileHelper(t, fileToWriteActual, testStr2)
	time.Sleep(time.Millisecond * 80)
	str = getStringFromRefreshable(t, r)
	assert.Equal(t, str, "renderConf2")
	err = os.RemoveAll(tempDir)
	assert.NoError(t, err)
}

func TestRefreshableCanFollowMovingSymLink(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	// We will have fileToWritePointingAtActual point at fileToWriteActual
	assert.NoError(t, err)
	fileToWriteActualOriginal := path.Join(tempDir, "fileToWriteActualOriginal")
	fileToWriteActualUpdated := path.Join(tempDir, "fileToWriteActualUpdated")
	fileToWritePointingAtActual := path.Join(tempDir, "fileToWritePointingAtActual")
	fileToWritePointingAtSymlink := path.Join(tempDir, "fileToWritePointingAtSymlink")
	// Write the old file
	writeFileHelper(t, fileToWriteActualOriginal, testStr1)
	// Write the old file
	writeFileHelper(t, fileToWriteActualUpdated, testStr2)
	// Symlink the old file to point at the new file
	err = os.Symlink(fileToWriteActualOriginal, fileToWritePointingAtActual)
	assert.NoError(t, err)
	// Symlink a to a symlink
	err = os.Symlink(fileToWritePointingAtActual, fileToWritePointingAtSymlink)
	assert.NoError(t, err)
	// Point the refreshable towards the new file
	r, err := NewFileRefreshableWithDuration(context.Background(), fileToWritePointingAtSymlink, time.Millisecond*50)
	assert.NoError(t, err)
	str := getStringFromRefreshable(t, r)
	assert.Equal(t, str, "renderConf1")
	// Change where the symlink points
	err = os.Remove(fileToWritePointingAtActual)
	assert.NoError(t, err)
	err = os.Symlink(fileToWriteActualUpdated, fileToWritePointingAtActual)
	assert.NoError(t, err)

	// Update the symlink file
	time.Sleep(time.Millisecond * 80)
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
	return getStringFromInterface(t, r.Current())
}

func getStringFromInterface(t *testing.T, current interface{}) string {
	currentCasted, ok := current.([]uint8)
	assert.True(t, ok)
	b := make([]byte, len(currentCasted))
	for i, v := range currentCasted {
		b[i] = v
	}
	return string(b)
}
