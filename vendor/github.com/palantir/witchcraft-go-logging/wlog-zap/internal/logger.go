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

package zapimpl

import (
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/palantir/witchcraft-go-logging/wlog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var _ wlog.LogEntry = (*zapLogEntry)(nil)

type zapLogEntry struct {
	fields              map[string]*zapcore.Field
	mutableValueEntries wlog.MutableValueEntries
}

func newZapLogEntry() *zapLogEntry {
	return &zapLogEntry{
		fields: make(map[string]*zapcore.Field),
	}
}

func zapLogEntrySetValue[ValT any](entry *zapLogEntry, fn func(k string, v ValT) zap.Field, k string, v ValT) {
	entry.delete(k)
	field := fn(k, v)
	entry.fields[k] = &field
}

func (e *zapLogEntry) delete(k string) {
	delete(e.fields, k)
	e.mutableValueEntries.Delete(k)
}

func (e *zapLogEntry) StringValue(k, v string) {
	zapLogEntrySetValue(e, zap.String, k, v)
}

func (e *zapLogEntry) OptionalStringValue(k, v string) {
	if v == "" {
		e.delete(k)
	} else {
		e.StringValue(k, v)
	}
}

func (e *zapLogEntry) StringListValue(k string, v []string) {
	e.delete(k)
	e.mutableValueEntries.StringListValue(k, v)
}

func (e *zapLogEntry) StringListAppendValue(k string, v []string) {
	e.StringListValue(k, append(e.mutableValueEntries.StringListValues()[k], v...))
}

func (e *zapLogEntry) SafeLongValue(k string, v int64) {
	zapLogEntrySetValue(e, zap.Int64, k, v)
}

func (e *zapLogEntry) IntValue(k string, v int32) {
	zapLogEntrySetValue(e, zap.Int32, k, v)
}

func (e *zapLogEntry) StringMapValue(k string, v map[string]string) {
	delete(e.fields, k)
	e.mutableValueEntries.StringMapValue(k, v)
}

func (e *zapLogEntry) AnyMapValue(k string, v map[string]any) {
	delete(e.fields, k)
	e.mutableValueEntries.AnyMapValue(k, v)
}

func (e *zapLogEntry) ObjectValue(k string, v interface{}, marshalerType reflect.Type) {
	zapLogEntrySetValue(e, zap.Reflect, k, v)
}

func (e *zapLogEntry) ObjectListValue(k string, v []any) {
	e.delete(k)
	e.mutableValueEntries.ObjectListValue(k, v)
}

func (e *zapLogEntry) ObjectListAppendValue(k string, v []any) {
	e.ObjectListValue(k, append(e.mutableValueEntries.ObjectListValues()[k], v...))
}

func (e *zapLogEntry) Fields() []zapcore.Field {
	stringListValues := e.mutableValueEntries.StringListValues()
	stringMapValues := e.mutableValueEntries.StringMapValues()
	anyMapValues := e.mutableValueEntries.AnyMapValues()
	objectListValues := e.mutableValueEntries.ObjectListValues()
	fields := make([]zapcore.Field, 0, len(e.fields)+len(stringMapValues)+len(anyMapValues)+len(stringListValues)+len(objectListValues))
	for _, field := range e.fields {
		fields = append(fields, *field)
	}
	for k, v := range stringListValues {
		fields = append(fields, zap.Strings(k, v))
	}
	for k, v := range objectListValues {
		fields = append(fields, zap.Any(k, v))
	}
	fields = append(fields, zapLogEntryMapValuesToFields(stringMapValues, func(k string, v string, enc zapcore.ObjectEncoder) error {
		enc.AddString(k, v)
		return nil
	})...)
	fields = append(fields, zapLogEntryMapValuesToFields(anyMapValues, func(k string, v any, enc zapcore.ObjectEncoder) error {
		if err := encodeField(k, v, enc); err != nil {
			return fmt.Errorf("failed to encode field %s: %v", k, err)
		}
		return nil
	})...)
	return fields
}

func zapLogEntryMapValuesToFields[ValType any](inputMap map[string]map[string]ValType, valFn func(k string, v ValType, enc zapcore.ObjectEncoder) error) []zapcore.Field {
	var fields []zapcore.Field
	for key, values := range inputMap {
		if len(values) == 0 {
			// this logic makes it such that "values" is encoded in a manner that matches the actual value, whether that
			// be nil or empty.
			fields = append(fields, zap.Any(key, values))
		} else {
			fields = append(fields, zap.Object(key, zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
				keys := make([]string, 0, len(values))
				for k := range values {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					if err := valFn(k, values[k], enc); err != nil {
						return err
					}
				}
				return nil
			})))
		}
	}
	return fields
}

type zapLogger struct {
	logger *zap.Logger
	*wlog.AtomicLogLevel
}

func (l *zapLogger) Log(params ...wlog.Param) {
	logOutput(l.logger.Info, "", params)
}

func (l *zapLogger) Debug(msg string, params ...wlog.Param) {
	if l.Enabled(wlog.DebugLevel) {
		logOutput(l.logger.Debug, msg, params)
	}
}

func (l *zapLogger) Info(msg string, params ...wlog.Param) {
	if l.Enabled(wlog.InfoLevel) {
		logOutput(l.logger.Info, msg, params)
	}
}

func (l *zapLogger) Warn(msg string, params ...wlog.Param) {
	if l.Enabled(wlog.WarnLevel) {
		logOutput(l.logger.Warn, msg, params)
	}
}

func (l *zapLogger) Error(msg string, params ...wlog.Param) {
	if l.Enabled(wlog.ErrorLevel) {
		logOutput(l.logger.Error, msg, params)
	}
}

func logOutput(logFn func(string, ...zap.Field), msg string, params []wlog.Param) {
	entry := newZapLogEntry()
	wlog.ApplyParams(entry, wlog.ParamsWithMessage(msg, params))
	// Empty string is used for the "message" because the message is added to params above if present
	logFn("", entry.Fields()...)
}

func encodeField(key string, value interface{}, enc zapcore.ObjectEncoder) error {
	switch v := value.(type) {
	case string:
		enc.AddString(key, v)
	case int:
		enc.AddInt(key, v)
	case int8:
		enc.AddInt8(key, v)
	case int16:
		enc.AddInt16(key, v)
	case int32:
		enc.AddInt32(key, v)
	case int64:
		enc.AddInt64(key, v)
	case uint:
		enc.AddUint(key, v)
	case uint8:
		enc.AddUint8(key, v)
	case uint16:
		enc.AddUint16(key, v)
	case uint32:
		enc.AddUint32(key, v)
	case uint64:
		enc.AddUint64(key, v)
	case bool:
		enc.AddBool(key, v)
	case float32:
		enc.AddFloat32(key, v)
	case float64:
		enc.AddFloat64(key, v)
	case []byte:
		enc.AddBinary(key, v)
	case time.Duration:
		enc.AddDuration(key, v)
	case time.Time:
		enc.AddTime(key, v)
		// support string and int slices explicitly because they are common slice types
	case []string:
		return enc.AddArray(key, zapcore.ArrayMarshalerFunc(func(enc zapcore.ArrayEncoder) error {
			for _, k := range v {
				enc.AppendString(k)
			}
			return nil
		}))
	case []int:
		return enc.AddArray(key, zapcore.ArrayMarshalerFunc(func(enc zapcore.ArrayEncoder) error {
			for _, k := range v {
				enc.AppendInt(k)
			}
			return nil
		}))
	default:
		// add non-primitive types using reflection
		return enc.AddReflected(key, v)
	}
	return nil
}
