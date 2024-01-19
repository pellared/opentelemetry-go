// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package log // import "go.opentelemetry.io/otel/log"

import (
	"fmt"
)

// An KeyValue is a key-value pair.
type KeyValue struct {
	Key   string
	Value Value
}

// String returns an KeyValue for a string value.
func String(key, value string) KeyValue {
	return KeyValue{key, StringValue(value)}
}

// Int64 returns an KeyValue for an int64.
func Int64(key string, value int64) KeyValue {
	return KeyValue{key, Int64Value(value)}
}

// Int converts an int to an int64 and returns
// an KeyValue with that value.
func Int(key string, value int) KeyValue {
	return Int64(key, int64(value))
}

// Uint64 returns an KeyValue for a uint64.
func Uint64(key string, v uint64) KeyValue {
	return KeyValue{key, Uint64Value(v)}
}

// Float64 returns an KeyValue for a floating-point number.
func Float64(key string, v float64) KeyValue {
	return KeyValue{key, Float64Value(v)}
}

// Bool returns an KeyValue for a bool.
func Bool(key string, v bool) KeyValue {
	return KeyValue{key, BoolValue(v)}
}

// Group returns an KeyValue for a Group [Value].
//
// Use Group to collect several key-value pairs under a single
// key.
func Group(key string, args ...KeyValue) KeyValue {
	return KeyValue{key, GroupValue(args...)}
}

// Invalid reports whether the key-value has empty key or value.
func (a KeyValue) Invalid() bool {
	return a.Key == "" || a.Value.Empty()
}

// Equal reports whether a and b have equal keys and values.
func (a KeyValue) Equal(b KeyValue) bool {
	return a.Key == b.Key && a.Value.Equal(b.Value)
}

func (a KeyValue) String() string {
	return fmt.Sprintf("%s=%s", a.Key, a.Value)
}
