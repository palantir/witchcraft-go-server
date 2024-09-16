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
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nmiyake/pkg/dirs"
	"github.com/palantir/pkg/refreshable/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testStr1              = "renderConf1"
	testStr2              = "renderConf2"
	refreshableSyncPeriod = time.Millisecond * 100
	sleepPeriod           = time.Millisecond * 175
)

// Verifies that a file that is written to and then update returns the updated value when Refreshable.Current() is called
// We ensure we wait long enough (80ms) such that the refreshable duration (50ms) will read the change
func TestRefreshableFileChanges(t *testing.T) {
	tempDir, cleanup, err := dirs.TempDir("", "")
	require.NoError(t, err)
	defer cleanup()
	fileToWrite := filepath.Join(tempDir, "file")
	writeFileHelper(t, fileToWrite, testStr1)
	r, err := NewFileRefreshableWithDuration(context.Background(), fileToWrite, refreshableSyncPeriod)
	assert.NoError(t, err)
	str := getStringFromRefreshable(t, r)
	assert.Equal(t, str, "renderConf1")
	writeFileHelper(t, fileToWrite, testStr2)
	time.Sleep(sleepPeriod)
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
	r, err := NewFileRefreshableWithDuration(context.Background(), fileToWrite, refreshableSyncPeriod)
	assert.NoError(t, err)
	str := getStringFromRefreshable(t, r)
	assert.Equal(t, str, "renderConf1")
	var count int32
	mappingFunc := func(mapped []byte) {
		atomic.AddInt32(&count, 1)
		str := getStringFromInterface(t, mapped)
		assert.Equal(t, str, "renderConf2")
	}
	r.Subscribe(mappingFunc)
	writeFileHelper(t, fileToWrite, testStr2)
	time.Sleep(sleepPeriod)
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
	r, err := NewFileRefreshableWithDuration(context.Background(), fileToWritePointingAtActual, refreshableSyncPeriod)
	assert.NoError(t, err)
	str := getStringFromRefreshable(t, r)
	assert.Equal(t, str, "renderConf1")
	// Update the actual file
	writeFileHelper(t, fileToWriteActual, testStr2)
	time.Sleep(sleepPeriod)
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
	r, err := NewFileRefreshableWithDuration(context.Background(), fileToWritePointingAtSymlink, refreshableSyncPeriod)
	assert.NoError(t, err)
	str := getStringFromRefreshable(t, r)
	assert.Equal(t, str, "renderConf1")
	// Update the symlink file
	writeFileHelper(t, fileToWriteActual, testStr2)
	time.Sleep(sleepPeriod)
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
	r, err := NewFileRefreshableWithDuration(context.Background(), fileToWritePointingAtSymlink, refreshableSyncPeriod)
	assert.NoError(t, err)
	str := getStringFromRefreshable(t, r)
	assert.Equal(t, str, "renderConf1")
	// Change where the symlink points
	err = os.Remove(fileToWritePointingAtActual)
	assert.NoError(t, err)
	err = os.Symlink(fileToWriteActualUpdated, fileToWritePointingAtActual)
	assert.NoError(t, err)

	// Update the symlink file
	time.Sleep(sleepPeriod)
	str = getStringFromRefreshable(t, r)
	assert.Equal(t, str, "renderConf2")
}

func writeFileHelper(t *testing.T, path, value string) {
	err := ioutil.WriteFile(path, []byte(value), 0644)
	assert.NoError(t, err)
}

func getStringFromRefreshable(t *testing.T, r refreshable.Refreshable[[]byte]) string {
	return getStringFromInterface(t, r.Current())
}

func getStringFromInterface(t *testing.T, current []byte) string {
	b := make([]byte, len(current))
	for i, v := range current {
		b[i] = v
	}
	return string(b)
}
