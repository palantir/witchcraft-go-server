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
	"github.com/nmiyake/pkg/dirs"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	testStr1 = "renderConf1"
	testStr2 = "renderConf2"
)

// Verifies that a file that is written to and then update returns the updated value when Refreshable.Current() is called
// We ensure we wait long enough (80ms) such that the refreshable duration (50ms) will read the change
func TestRefreshableFileChanges(t *testing.T) {
	tempDir, cleanup, err := dirs.TempDir("", "")
	require.NoError(t, err)
	defer cleanup()
	fileToWrite := filepath.Join(tempDir, "file")
	writeFileHelper(t, fileToWrite, testStr1)
	r, err := NewFileRefreshableWithDuration(context.Background(), fileToWrite, time.Millisecond*50)
	assert.NoError(t, err)
	str := getStringFromRefreshable(t, r)
	assert.Equal(t, str, "renderConf1")
	writeFileHelper(t, fileToWrite, testStr2)
	time.Sleep(time.Millisecond * 80)
	str = getStringFromRefreshable(t, r)
	assert.Equal(t, str, "renderConf2")
}

// Verifies that a subscribing to a FileRefreshable will correctly call the callback provided
// We ensure we wait long enough (80ms) such that the refreshable duration (50ms) will read the change
func TestRefreshableFileSubscribes(t *testing.T) {
	tempDir, cleanup, err := dirs.TempDir("", "")
	require.NoError(t, err)
	defer cleanup()
	fileToWrite := filepath.Join(tempDir, "file")
	writeFileHelper(t, fileToWrite, testStr1)
	r, err := NewFileRefreshableWithDuration(context.Background(), fileToWrite, time.Millisecond*50)
	assert.NoError(t, err)
	str := getStringFromRefreshable(t, r)
	assert.Equal(t, str, "renderConf1")
	var count int32
	mappingFunc := func(mapped interface{}) {
		atomic.AddInt32(&count, 1)
		str := getStringFromInterface(t, mapped)
		assert.Equal(t, str, "renderConf2")
	}
	r.Subscribe(mappingFunc)
	writeFileHelper(t, fileToWrite, testStr2)
	time.Sleep(time.Millisecond * 80)
	assert.Equal(t, int32(1), count)
}

// Verifies that a RefreshableFile can follow a symlink when the original file updates
// We ensure we wait long enough (80ms) such that the refreshable duration (50ms) will read the change
func TestRefreshableFileCanFollowSymLink(t *testing.T) {
	tempDir, cleanup, err := dirs.TempDir("", "")
	require.NoError(t, err)
	defer cleanup()
	// We will have fileToWritePointingAtActual point at fileToWriteActual
	fileToWriteActual := filepath.Join(tempDir, "fileToWriteActual")
	fileToWritePointingAtActual := filepath.Join(tempDir, "fileToWritePointingAtActual")
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
}

// Verifies that a RefreshableFile can follow a symlink to a symlink when the original file updates
// We ensure we wait long enough (80ms) such that the refreshable duration (50ms) will read the change
func TestRefreshableFileCanFollowMultipleSymLinks(t *testing.T) {
	tempDir, cleanup, err := dirs.TempDir("", "")
	require.NoError(t, err)
	defer cleanup()
	// We will have fileToWritePointingAtActual point at fileToWriteActual
	fileToWriteActual := filepath.Join(tempDir, "fileToWriteActual")
	fileToWritePointingAtActual := filepath.Join(tempDir, "fileToWritePointingAtActual")
	fileToWritePointingAtSymlink := filepath.Join(tempDir, "fileToWritePointingAtSymlink")
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
}

// Verifies that a RefreshableFile can follow a symlink to a symlink when the symlink changes
// We ensure we wait long enough (80ms) such that the refreshable duration (50ms) will read the change
func TestRefreshableFileCanFollowMovingSymLink(t *testing.T) {
	tempDir, cleanup, err := dirs.TempDir("", "")
	require.NoError(t, err)
	defer cleanup()
	// We will have fileToWritePointingAtActual point at fileToWriteActual
	fileToWriteActualOriginal := filepath.Join(tempDir, "fileToWriteActualOriginal")
	fileToWriteActualUpdated := filepath.Join(tempDir, "fileToWriteActualUpdated")
	fileToWritePointingAtActual := filepath.Join(tempDir, "fileToWritePointingAtActual")
	fileToWritePointingAtSymlink := filepath.Join(tempDir, "fileToWritePointingAtSymlink")
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
}

func writeFileHelper(t *testing.T, path, value string) {
	err := ioutil.WriteFile(path, []byte(value), 0644)
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
