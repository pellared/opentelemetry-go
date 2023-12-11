// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package benchmark

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
)

func TestWriterLogger(t *testing.T) {
	sb := &strings.Builder{}
	var l log.Logger = &writerLogger{w: sb}

	r := log.Record{
		Timestamp: testTimestamp,
		Severity:  testSeverity,
		Body:      testBody,
	}
	l = l.WithAttributes(
		attribute.String("string", testString),
		attribute.Float64("float", testFloat),
		attribute.Int("int", testInt),
		attribute.Bool("bool", testBool),
	)
	l.Emit(ctx, r)

	want := "timestamp=595728000 severity=9 body=log message string=7e3b3b2aaeff56a7108fe11e154200dd/7819479873059528190 float=1.2345 int=32768 bool=true\n"
	assert.Equal(t, want, sb.String())
}

// writerLogger is a logger that writes to a provided io.Writer without any locking.
// It is intended to represent a high-performance logger that synchronously
// writes text.
type writerLogger struct {
	embedded.Logger
	w io.Writer

	// The fields below are for optimizing the implementation of
	// Attributes and AddAttributes.

	// Allocation optimization: an inline array sized to hold
	// the majority of log calls (based on examination of open-source
	// code). It holds the start of the list of attributes.
	front [attributesInlineCount]attribute.KeyValue

	// The number of attributes in front.
	nFront int

	// The list of attributes except for those in front.
	// Invariants:
	//   - len(back) > 0 if nFront == len(front)
	//   - Unused array elements are zero. Used to detect mistakes.
	back []attribute.KeyValue
}

const attributesInlineCount = 5

// WithAttributes appends attributes that would be emitted by the logger.
func (l *writerLogger) WithAttributes(attrs ...attribute.KeyValue) log.Logger {
	cl := *l // shallow copy of the logger

	var i int
	for i = 0; i < len(attrs) && cl.nFront < len(cl.front); i++ {
		a := attrs[i]
		if !a.Valid() {
			continue
		}
		cl.front[cl.nFront] = a
		cl.nFront++
	}

	var attrsToSlice int
	for _, a := range attrs[i:] {
		if a.Valid() {
			attrsToSlice++
		}
	}

	if attrsToSlice == 0 {
		return &cl
	}

	cl.back = sliceGrow(cl.back, attrsToSlice)
	for _, a := range attrs[i:] {
		if a.Valid() {
			cl.back = append(cl.back, a)
		}
	}
	cl.back = sliceClip(cl.back) // prevent append from mutating shared array

	return &cl
}

func (l *writerLogger) Emit(_ context.Context, r log.Record) {
	if !r.Timestamp.IsZero() {
		l.write("timestamp=")
		l.write(strconv.FormatInt(r.Timestamp.Unix(), 10))
		l.write(" ")
	}
	l.write("severity=")
	l.write(strconv.FormatInt(int64(r.Severity), 10))
	l.write(" ")
	l.write("body=")
	l.write(r.Body)
	l.walkAttributes(func(kv attribute.KeyValue) bool {
		l.write(" ")
		l.write(string(kv.Key))
		l.write("=")
		l.appendValue(kv.Value)
		return true
	})
	l.write("\n")
}

// walkAttributes calls f on each [attribute.KeyValue].
// Iteration stops if f returns false.
func (l *writerLogger) walkAttributes(f func(attribute.KeyValue) bool) {
	for i := 0; i < l.nFront; i++ {
		if !f(l.front[i]) {
			return
		}
	}
	for _, a := range l.back {
		if !f(a) {
			return
		}
	}
}

func (l *writerLogger) appendValue(v attribute.Value) {
	switch v.Type() {
	case attribute.STRING:
		l.write(v.AsString())
	case attribute.INT64:
		l.write(strconv.FormatInt(v.AsInt64(), 10)) // strconv.FormatInt allocates memory.
	case attribute.FLOAT64:
		l.write(strconv.FormatFloat(v.AsFloat64(), 'g', -1, 64)) // strconv.FormatFloat allocates memory.
	case attribute.BOOL:
		l.write(strconv.FormatBool(v.AsBool()))
	default:
		panic(fmt.Sprintf("unhandled attribute type: %s", v.Type()))
	}
}

func (l *writerLogger) write(s string) {
	_, _ = io.WriteString(l.w, s)
}

// sliceGrow increases the slice's capacity, if necessary, to guarantee space for
// another n elements. After Grow(n), at least n elements can be appended
// to the slice without another allocation. If n is negative or too large to
// allocate the memory, Grow panics.
//
// This is a copy from https://pkg.go.dev/slices as it is not available in Go 1.20.
func sliceGrow[S ~[]E, E any](s S, n int) S {
	if n < 0 {
		panic("cannot be negative")
	}
	if n -= cap(s) - len(s); n > 0 {
		s = append(s[:cap(s)], make([]E, n)...)[:len(s)]
	}
	return s
}

// sliceClip removes unused capacity from the slice, returning s[:len(s):len(s)].
//
// This is a copy from https://pkg.go.dev/slices as it is not available in Go 1.20.
func sliceClip[S ~[]E, E any](s S) S {
	return s[:len(s):len(s)]
}
