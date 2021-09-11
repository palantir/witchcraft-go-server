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

package zeroimpl

import (
	"reflect"

	"github.com/palantir/witchcraft-go-logging/wlog"
	"github.com/rs/zerolog"
)

type zeroLogEntry struct {
	evt             *zerolog.Event
	keys            map[string]struct{}
	stringMapValues map[string]map[string]string
	anyMapValues    map[string]map[string]interface{}
}

func (e *zeroLogEntry) keyExists(key string) bool {
	if _, exists := e.keys[key]; exists {
		return true
	}
	e.keys[key] = struct{}{}
	return false
}

func (e *zeroLogEntry) StringValue(key, value string) {
	if e.keyExists(key) {
		return
	}
	e.evt = e.evt.Str(key, value)
}

func (e *zeroLogEntry) OptionalStringValue(key, value string) {
	if value != "" {
		e.StringValue(key, value)
	}
}

func (e *zeroLogEntry) StringListValue(k string, v []string) {
	if len(v) > 0 {
		if e.keyExists(k) {
			return
		}
		e.evt.Strs(k, v)
	}
}

func (e *zeroLogEntry) SafeLongValue(key string, value int64) {
	if e.keyExists(key) {
		return
	}
	e.evt = e.evt.Int64(key, value)
}

func (e *zeroLogEntry) IntValue(key string, value int32) {
	if e.keyExists(key) {
		return
	}
	e.evt = e.evt.Int32(key, value)
}

func (e *zeroLogEntry) ObjectValue(k string, v interface{}, marshalerType reflect.Type) {
	if e.keyExists(k) {
		return
	}
	e.evt.Interface(k, v)
}

//StringMapValue adds or merges the strings in values
//Since wlog overrides duplicates with a preference for the last parameter
//The parameters should not replace an existing key because parameters are passed to zerolog in reverse
//This differs from the default wlog StringMapValue since parameters are not reversed
func (e *zeroLogEntry) StringMapValue(key string, values map[string]string) {
	if len(values) == 0 {
		return
	}
	if e.stringMapValues == nil {
		e.stringMapValues = make(map[string]map[string]string)
	}
	entryMapVals, ok := e.stringMapValues[key]
	if !ok {
		entryMapVals = make(map[string]string)
		e.stringMapValues[key] = entryMapVals
	}
	for k, v := range values {
		if _, exists := entryMapVals[k]; !exists {
			entryMapVals[k] = v
		}
	}
}

//AnyMapValue adds or merges the values in values
//Since wlog overrides duplicates with a preference for the last parameter
//The parameters should not replace an existing key because parameters are passed to zerolog in reverse
//This differs from the default wlog AnyMapValue since parameters are not reversed
func (e *zeroLogEntry) AnyMapValue(key string, values map[string]interface{}) {
	if len(values) == 0 {
		return
	}
	if e.anyMapValues == nil {
		e.anyMapValues = make(map[string]map[string]interface{})
	}
	entryMapVals, ok := e.anyMapValues[key]
	if !ok {
		entryMapVals = make(map[string]interface{})
		e.anyMapValues[key] = entryMapVals
	}
	for k, v := range values {
		if _, exists := entryMapVals[k]; !exists {
			entryMapVals[k] = v
		}
	}
}

func (e *zeroLogEntry) StringMapValues() map[string]map[string]string {
	return e.stringMapValues
}

func (e *zeroLogEntry) AnyMapValues() map[string]map[string]interface{} {
	return e.anyMapValues
}

func (e *zeroLogEntry) Evt() *zerolog.Event {
	evt := e.evt
	for key, values := range e.StringMapValues() {
		key := key
		values := values
		dictEvt := zerolog.Dict()
		for k, v := range values {
			dictEvt = dictEvt.Str(k, v)
		}
		evt = evt.Dict(key, dictEvt)
	}
	for key, values := range e.AnyMapValues() {
		key := key
		values := values
		dictEvt := zerolog.Dict()
		for k, v := range values {
			dictEvt = dictEvt.Interface(k, v)
		}
		evt = evt.Dict(key, dictEvt)
	}
	return evt
}

type zeroLogger struct {
	logger zerolog.Logger
	level  zerolog.Level
}

func (l *zeroLogger) should(level zerolog.Level) bool {
	if level < l.level {
		return false
	}
	return true
}

func (l *zeroLogger) Log(params ...wlog.Param) {
	if !l.should(zerolog.NoLevel) {
		return
	}
	logOutput(l.logger.Log, "", params)
}

func (l *zeroLogger) Debug(msg string, params ...wlog.Param) {
	if !l.should(zerolog.DebugLevel) {
		return
	}
	logOutput(l.logger.Log, msg, params)
}

func (l *zeroLogger) Info(msg string, params ...wlog.Param) {
	if !l.should(zerolog.InfoLevel) {
		return
	}
	logOutput(l.logger.Log, msg, params)
}

func (l *zeroLogger) Warn(msg string, params ...wlog.Param) {
	if !l.should(zerolog.WarnLevel) {
		return
	}
	logOutput(l.logger.Log, msg, params)
}

func (l *zeroLogger) Error(msg string, params ...wlog.Param) {
	if !l.should(zerolog.ErrorLevel) {
		return
	}
	logOutput(l.logger.Log, msg, params)
}

func (l *zeroLogger) SetLevel(level wlog.LogLevel) {
	l.level = toZeroLevel(level)
	l.logger = l.logger.Level(toZeroLevel(level))
}

func reverseParams(params []wlog.Param) {
	for i, j := 0, len(params)-1; i < j; i, j = i+1, j-1 {
		params[i], params[j] = params[j], params[i]
	}
}

func logOutput(newEvt func() *zerolog.Event, msg string, params []wlog.Param) {
	entry := &zeroLogEntry{
		evt:  newEvt(),
		keys: make(map[string]struct{}),
	}
	if !entry.evt.Enabled() {
		return
	}
	reverseParams(params)
	wlog.ApplyParams(entry, params)
	entry.Evt().Msg(msg)
}
