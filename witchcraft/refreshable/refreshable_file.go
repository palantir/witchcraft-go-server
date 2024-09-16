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
	"crypto/sha256"
	"os"
	"time"

	"github.com/palantir/pkg/refreshable/v2"
	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
)

const (
	defaultRefreshableSyncPeriod = time.Second
)

// NewFileRefreshable is identical to NewFileRefreshableWithDuration except it defaults to use defaultRefreshableSyncPeriod for how often the file is checked
func NewFileRefreshable(ctx context.Context, filePath string) (refreshable.Validated[[]byte], error) {
	return NewFileRefreshableWithDuration(ctx, filePath, defaultRefreshableSyncPeriod)
}

// NewFileRefreshableWithDuration returns a new Refreshable whose current value is the bytes of the file at the provided path.
// The file is checked every duration time.Duration as an argument.
// Calling this function also starts a goroutine which updates the value of the refreshable whenever the specified file
// is changed. The goroutine will terminate when the provided context is done or when the returned cancel function is
// called.
func NewFileRefreshableWithDuration(ctx context.Context, filePath string, duration time.Duration) (refreshable.Validated[[]byte], error) {
	// Create a refreshable containing the current time, to be updated with a ticker.
	// A mapped refreshable is used to ensure that the file is read once per tick.
	timeRefreshable := refreshable.New(time.Now())
	go func() {
		for t := range time.Tick(duration) {
			if ctx.Err() != nil {
				return
			}
			timeRefreshable.Update(t)
		}
	}()
	var fileChecksum [sha256.Size]byte
	fileRefreshable, _, err := refreshable.MapWithError[time.Time, []byte](timeRefreshable, func(time.Time) ([]byte, error) {
		return readFileForRefreshable(ctx, filePath, &fileChecksum)
	})
	if err != nil {
		return nil, err
	}
	return fileRefreshable, nil
}

func readFileForRefreshable(ctx context.Context, filePath string, fileChecksum *[sha256.Size]byte) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, werror.Convert(ctx.Err())
	default:
	}
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		err = werror.WrapWithContextParams(ctx, err, "failed to read refreshable file", werror.SafeParam("filePath", filePath))
		svc1log.FromContext(ctx).Error("Failed to read refreshable file.", svc1log.Stacktrace(err))
		return nil, err
	}
	loadedChecksum := sha256.Sum256(fileBytes)
	if *fileChecksum != loadedChecksum {
		*fileChecksum = loadedChecksum
		svc1log.FromContext(ctx).Info("Attempting to update file refreshable")
	}
	return fileBytes, nil
}
