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

package diag1log

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"

	"github.com/palantir/witchcraft-go-logging/conjure/sls/spec/logging"
	"github.com/palantir/witchcraft-go-logging/internal/conjuretype"
)

// ThreadDumpV1FromGoroutines unmarshals a "goroutine dump" (as formatted by panic or the runtime package)
// and returns a conjured logging.ThreadDumpV1 object.
func ThreadDumpV1FromGoroutines(goroutinesContent []byte) logging.ThreadDumpV1 {
	// Goroutines are separated by an empty line
	goroutines := bytes.Split(goroutinesContent, []byte("\n\n"))

	threads := logging.ThreadDumpV1{Threads: make([]logging.ThreadInfoV1, len(goroutines))}
	for i, goroutine := range goroutines {
		threads.Threads[i] = unmarshalThreadDump(goroutine)
	}
	return threads
}

var titleLinePattern = regexp.MustCompile(`^(goroutine (\d+)) \[([^]]+)]:$`)

func unmarshalThreadDump(goroutine []byte) logging.ThreadInfoV1 {
	lines := bytes.Split(bytes.TrimSpace(goroutine), []byte("\n"))
	if len(lines) == 0 {
		return logging.ThreadInfoV1{}
	}

	info := logging.ThreadInfoV1{Params: map[string]interface{}{}}

	// Unmarshal title (first) line
	matches := titleLinePattern.FindSubmatch(lines[0])
	if len(matches) >= 4 {
		name := string(matches[1])
		info.Name = &name

		idInt, err := strconv.ParseInt(string(matches[2]), 0, 64)
		if err == nil {
			id, err := conjuretype.NewSafeLong(idInt)
			if err == nil {
				info.Id = &id
			}
		}

		info.Params["status"] = string(matches[3])
	}

	// Go through stack frames two lines at a time
	stackLines := lines[1:]
	for i := 0; i < len(stackLines); i += 2 {
		funcLine := stackLines[i]
		fileLine := stackLines[i+1]

		frame := logging.StackFrameV1{Params: map[string]interface{}{}}

		unmarshalFuncLine(funcLine, &frame)
		unmarshalFileLine(fileLine, &frame)

		info.StackTrace = append(info.StackTrace, frame)
	}

	return info
}

func unmarshalFuncLine(funcLine []byte, frame *logging.StackFrameV1) {
	if bytes.HasPrefix(funcLine, []byte("created by ")) {
		// creators do not include arguments
		procedure := strings.TrimPrefix(string(funcLine), "created by ")
		frame.Procedure = &procedure
		frame.Params["goroutineCreator"] = true
		return
	}

	argIndex := bytes.LastIndex(funcLine, []byte("("))
	if argIndex != -1 {
		procedure := string(funcLine[:argIndex])
		frame.Procedure = &procedure

		// TODO(bmoylan): Would including function arguments be safe?
		//args := bytes.Split(funcLine[argIndex+1:len(funcLine)-1], []byte(", "))
		//for i, arg := range args {
		//	frame.Params[fmt.Sprintf("arg%d", i)] = string(arg)
		//}
	}
}

func unmarshalFileLine(fileLine []byte, frame *logging.StackFrameV1) {
	segments := strings.Split(string(bytes.TrimSpace(fileLine)), " +")
	frame.Address = &segments[1]

	sepIdx := strings.LastIndex(segments[0], ":")
	absPath := segments[0][:sepIdx]

	// Trim everything up to the first /src/ as a hueristic for limiting the file
	// to the go import path. This will break if someone's GOPATH exists in a directory
	// with /src/ in the path (e.g. ~/src/go/src/github...), but we should not be
	// deploying binaries compiled in non-standard build environments.
	const srcDir = `/src/`
	idx := strings.Index(absPath, srcDir)
	if idx >= 0 {
		file := absPath[idx+len(srcDir):]
		frame.File = &file
	}

	lineNumStr := segments[0][sepIdx+1:]
	lineNum, err := strconv.Atoi(lineNumStr)
	if err == nil {
		frame.Line = &lineNum
	}
}
