// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package log // import "go.opentelemetry.io/otel/log"

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"unsafe"
)

// A Value can represent a structured value.
// The zero Value corresponds to nil.
type Value struct {
	_ [0]func() // disallow ==
	// num holds the value for Kinds Int64, Uint64, Float64, and Bool,
	// the string length for KindString.
	num uint64
	// If any is of type Kind, then the value is in num as described above.
	// If any is of type stringptr, then the Kind is String and the string value
	// consists of the length in num and the pointer in any.
	// (This implies that key-value pairs cannot store values of type stringptr.)
	any any
}

type (
	stringptr *byte     // used in Value.any when the Value is a string
	groupptr  *KeyValue // used in Value.any when the Value is a []Attr
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
	KindUint64
	KindGroup
)

var kindStrings = []string{
	"KindEmpty",
	"Bool",
	"Float64",
	"Int64",
	"String",
	"Uint64",
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
	case groupptr:
		return KindGroup
	default:
		return KindEmpty
	}
}

//////////////// Constructors

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

// Uint64Value returns a [Value] for a uint64.
func Uint64Value(v uint64) Value {
	return Value{num: v, any: KindUint64}
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

// GroupValue returns a new [Value] for a list of key-value pairs.
// The caller must not subsequently mutate the argument slice.
func GroupValue(as ...KeyValue) Value {
	// Remove empty groups.
	// It is simpler overall to do this at construction than
	// to check each Group recursively for emptiness.
	if n := countEmptyGroups(as); n > 0 {
		as2 := make([]KeyValue, 0, len(as)-n)
		for _, a := range as {
			if !a.Value.isEmptyGroup() {
				as2 = append(as2, a)
			}
		}
		as = as2
	}
	return Value{num: uint64(len(as)), any: groupptr(unsafe.SliceData(as))}
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

//////////////// Accessors

// Any returns v's value as an any.
func (v Value) Any() any {
	switch v.Kind() {
	case KindGroup:
		return v.group()
	case KindInt64:
		return int64(v.num)
	case KindUint64:
		return v.num
	case KindFloat64:
		return v.float()
	case KindString:
		return v.str()
	case KindBool:
		return v.bool()
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

// Uint64 returns v's value as a uint64. It panics
// if v is not an unsigned integer.
func (v Value) Uint64() uint64 {
	if g, w := v.Kind(), KindUint64; g != w {
		panic(fmt.Sprintf("Value kind is %s, not %s", g, w))
	}
	return v.num
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

// Group returns v's value as a []Attr.
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

//////////////// Other

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
	case KindInt64, KindUint64, KindBool:
		return v.num == w.num
	case KindString:
		return v.str() == w.str()
	case KindFloat64:
		return v.float() == w.float()
	case KindGroup:
		return slices.EqualFunc(v.group(), w.group(), KeyValue.Equal)
	case KindEmpty:
		return k1 == k2
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
	case KindUint64:
		return strconv.AppendUint(dst, v.num, 10)
	case KindFloat64:
		return strconv.AppendFloat(dst, v.float(), 'g', -1, 64)
	case KindBool:
		return strconv.AppendBool(dst, v.bool())
	case KindGroup:
		return fmt.Append(dst, v.group())
	case KindEmpty:
		return append(dst, emptyString...)
	default:
		panic(fmt.Sprintf("bad kind: %s", v.Kind()))
	}
}
