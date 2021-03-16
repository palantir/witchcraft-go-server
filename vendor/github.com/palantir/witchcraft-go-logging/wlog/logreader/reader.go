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

package logreader

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
)

// Entry is a single JSON entry in the log.
type Entry map[string]interface{}

// EntriesFromFile returns a slice of all of the log entries in the given file. Assumes that each line in the file
// is a JSON object that represents a log entry.
func EntriesFromFile(file string) ([]Entry, error) {
	logFile, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer func() {
		// file is opened for reads only, so nothing to be done if there is an error closing it
		_ = logFile.Close()
	}()
	return entriesFromReader(logFile)
}

func EntriesFromContent(content []byte) ([]Entry, error) {
	return entriesFromReader(bytes.NewReader(content))
}

func entriesFromReader(r io.Reader) ([]Entry, error) {
	var entries []Entry
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		var currEntry Entry
		dec := json.NewDecoder(bytes.NewReader(scanner.Bytes()))
		dec.UseNumber()

		if err := dec.Decode(&currEntry); err != nil {
			return nil, err
		}
		entries = append(entries, currEntry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}
