// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package log // import "go.opentelemetry.io/otel/log"

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"unsafe"
)

// A Value can represent a structured value.
// The zero Value corresponds to nil.
type Value struct {
	_ [0]func() // disallow ==
	// num holds the value for Kinds: Int64, Float64, and Bool,
	// the length for String, Bytes, List, Group.
	num uint64
	// If any is of type Kind, then the value is in num as described above.
	// Otherwise (if is of type stringptr, listptr, sliceptr or groupptr) it contains the value.
	any any
}

type (
	stringptr *byte     // used in Value.any when the Value is a string
	bytesptr  *byte     // used in Value.any when the Value is a []byte
	listptr   *Value    // used in Value.any when the Value is a []Value
	groupptr  *KeyValue // used in Value.any when the Value is a []KeyValue
)

// Kind is the kind of a [Value].
type Kind int

// Kind values.
const (
	KindEmpty Kind = iota
	KindBool
	KindFloat64
	KindInt64
	KindString
	KindBytes
	KindList
	KindGroup
)

var kindStrings = []string{
	"KindEmpty",
	"Bool",
	"Float64",
	"Int64",
	"String",
	"Bytes",
	"List",
	"Group",
}

var emptyString = []byte("<nil>")

func (k Kind) String() string {
	if k >= 0 && int(k) < len(kindStrings) {
		return kindStrings[k]
	}
	return "<unknown log.Kind>"
}

// Kind returns v's Kind.
func (v Value) Kind() Kind {
	switch x := v.any.(type) {
	case Kind:
		return x
	case stringptr:
		return KindString
	case bytesptr:
		return KindBytes
	case listptr:
		return KindList
	case groupptr:
		return KindGroup
	default:
		return KindEmpty
	}
}

// StringValue returns a new [Value] for a string.
func StringValue(value string) Value {
	return Value{num: uint64(len(value)), any: stringptr(unsafe.StringData(value))}
}

// IntValue returns a [Value] for an int.
func IntValue(v int) Value {
	return Int64Value(int64(v))
}

// Int64Value returns a [Value] for an int64.
func Int64Value(v int64) Value {
	return Value{num: uint64(v), any: KindInt64}
}

// Float64Value returns a [Value] for a floating-point number.
func Float64Value(v float64) Value {
	return Value{num: math.Float64bits(v), any: KindFloat64}
}

// BoolValue returns a [Value] for a bool.
func BoolValue(v bool) Value { //nolint:revive // We are passing bool as this is a constructor for bool.
	u := uint64(0)
	if v {
		u = 1
	}
	return Value{num: u, any: KindBool}
}

// BytesValue returns a [Value] for bytes.
// The caller must not subsequently mutate the argument slice.
func BytesValue(v []byte) Value {
	return Value{num: uint64(len(v)), any: bytesptr(unsafe.SliceData(v))}
}

// ListValue returns a [Value] for a list of [Value].
// The caller must not subsequently mutate the argument slice.
func ListValue(vs ...Value) Value {
	return Value{num: uint64(len(vs)), any: listptr(unsafe.SliceData(vs))}
}

// GroupValue returns a new [Value] for a list of key-value pairs.
// The caller must not subsequently mutate the argument slice.
func GroupValue(kvs ...KeyValue) Value {
	// Remove empty groups.
	// It is simpler overall to do this at construction than
	// to check each Group recursively for emptiness.
	if n := countEmptyGroups(kvs); n > 0 {
		as2 := make([]KeyValue, 0, len(kvs)-n)
		for _, a := range kvs {
			if !a.Value.isEmptyGroup() {
				as2 = append(as2, a)
			}
		}
		kvs = as2
	}
	return Value{num: uint64(len(kvs)), any: groupptr(unsafe.SliceData(kvs))}
}

// countEmptyGroups returns the number of empty group values in its argument.
func countEmptyGroups(as []KeyValue) int {
	n := 0
	for _, a := range as {
		if a.Value.isEmptyGroup() {
			n++
		}
	}
	return n
}

// Any returns v's value as an any.
func (v Value) Any() any {
	switch v.Kind() {
	case KindGroup:
		return v.group()
	case KindList:
		return v.list()
	case KindInt64:
		return int64(v.num)
	case KindFloat64:
		return v.float()
	case KindString:
		return v.str()
	case KindBool:
		return v.bool()
	case KindBytes:
		return v.bytes()
	case KindEmpty:
		return nil
	default:
		panic(fmt.Sprintf("bad kind: %s", v.Kind()))
	}
}

// String returns Value's value as a string, formatted like [fmt.Sprint]. Unlike
// the methods Int64, Float64, and so on, which panic if v is of the
// wrong kind, String never panics.
func (v Value) String() string {
	if sp, ok := v.any.(stringptr); ok {
		return unsafe.String(sp, v.num)
	}
	var buf []byte
	return string(v.append(buf))
}

func (v Value) str() string {
	return unsafe.String(v.any.(stringptr), v.num)
}

// Int64 returns v's value as an int64. It panics
// if v is not a signed integer.
func (v Value) Int64() int64 {
	if g, w := v.Kind(), KindInt64; g != w {
		panic(fmt.Sprintf("Value kind is %s, not %s", g, w))
	}
	return int64(v.num)
}

// Bool returns v's value as a bool. It panics
// if v is not a bool.
func (v Value) Bool() bool {
	if g, w := v.Kind(), KindBool; g != w {
		panic(fmt.Sprintf("Value kind is %s, not %s", g, w))
	}
	return v.bool()
}

func (v Value) bool() bool {
	return v.num == 1
}

// Float64 returns v's value as a float64. It panics
// if v is not a float64.
func (v Value) Float64() float64 {
	if g, w := v.Kind(), KindFloat64; g != w {
		panic(fmt.Sprintf("Value kind is %s, not %s", g, w))
	}

	return v.float()
}

func (v Value) float() float64 {
	return math.Float64frombits(v.num)
}

// Group returns v's value as a []byte.
// It panics if v's [Kind] is not [KindBytes].
func (v Value) Bytes() []byte {
	if sp, ok := v.any.(bytesptr); ok {
		return unsafe.Slice((*byte)(sp), v.num)
	}
	panic("Bytes: bad kind")
}

func (v Value) bytes() []byte {
	return unsafe.Slice((*byte)(v.any.(bytesptr)), v.num)
}

// List returns v's value as a []Value.
// It panics if v's [Kind] is not [KindList].
func (v Value) List() []Value {
	if sp, ok := v.any.(listptr); ok {
		return unsafe.Slice((*Value)(sp), v.num)
	}
	panic("List: bad kind")
}

func (v Value) list() []Value {
	return unsafe.Slice((*Value)(v.any.(listptr)), v.num)
}

// Group returns v's value as a []KeyValue.
// It panics if v's [Kind] is not [KindGroup].
func (v Value) Group() []KeyValue {
	if sp, ok := v.any.(groupptr); ok {
		return unsafe.Slice((*KeyValue)(sp), v.num)
	}
	panic("Group: bad kind")
}

func (v Value) group() []KeyValue {
	return unsafe.Slice((*KeyValue)(v.any.(groupptr)), v.num)
}

// Empty reports whether the value is empty (coresponds to nil).
func (v Value) Empty() bool {
	return v.Kind() == KindEmpty
}

// Equal reports whether v and w represent the same Go value.
func (v Value) Equal(w Value) bool {
	k1 := v.Kind()
	k2 := w.Kind()
	if k1 != k2 {
		return false
	}
	switch k1 {
	case KindInt64, KindBool:
		return v.num == w.num
	case KindString:
		return v.str() == w.str()
	case KindFloat64:
		return v.float() == w.float()
	case KindList:
		return sliceEqualFunc(v.list(), w.list(), Value.Equal)
	case KindGroup:
		return sliceEqualFunc(v.group(), w.group(), KeyValue.Equal)
	case KindBytes:
		return bytes.Equal(v.bytes(), w.bytes())
	case KindEmpty:
		return true
	default:
		panic(fmt.Sprintf("bad kind: %s", k1))
	}
}

// isEmptyGroup reports whether v is a group that has no attributes.
func (v Value) isEmptyGroup() bool {
	if v.Kind() != KindGroup {
		return false
	}
	// We do not need to recursively examine the group's key-value pairs for emptiness,
	// because GroupValue removed them when the group was constructed, and
	// groups are immutable.
	return len(v.group()) == 0
}

// append appends a text representation of v to dst.
// v is formatted as with fmt.Sprint.
func (v Value) append(dst []byte) []byte {
	switch v.Kind() {
	case KindString:
		return append(dst, v.str()...)
	case KindInt64:
		return strconv.AppendInt(dst, int64(v.num), 10)
	case KindFloat64:
		return strconv.AppendFloat(dst, v.float(), 'g', -1, 64)
	case KindBool:
		return strconv.AppendBool(dst, v.bool())
	case KindBytes:
		return fmt.Append(dst, v.bytes())
	case KindGroup:
		return fmt.Append(dst, v.group())
	case KindList:
		return fmt.Append(dst, v.list())
	case KindEmpty:
		return append(dst, emptyString...)
	default:
		panic(fmt.Sprintf("bad kind: %s", v.Kind()))
	}
}

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

// Float64 returns an KeyValue for a floating-point number.
func Float64(key string, v float64) KeyValue {
	return KeyValue{key, Float64Value(v)}
}

// Bool returns an KeyValue for a bool.
func Bool(key string, v bool) KeyValue {
	return KeyValue{key, BoolValue(v)}
}

// Bytes returns an KeyValue for a bytes.
func Bytes(key string, v []byte) KeyValue {
	return KeyValue{key, BytesValue(v)}
}

// Bytes returns an KeyValue for a list of [Value].
func List(key string, args ...Value) KeyValue {
	return KeyValue{key, ListValue(args...)}
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
