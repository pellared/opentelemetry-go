// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package benchmark

import (
	"context"
	"io"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/noop"
)

var (
	ctx           = context.Background()
	testTimestamp = time.Date(1988, time.November, 17, 0, 0, 0, 0, time.UTC)
	testBody      = "log message"
	testSeverity  = log.SeverityInfo
	testFloat     = 1.2345
	testString    = "7e3b3b2aaeff56a7108fe11e154200dd/7819479873059528190"
	testInt       = 32768
	testBool      = true
)

// These benchmarks are based on slog/internal/benchmarks.
//
// They test a complete log record, from the user's call to its return.
//
// WriterLogger is an optimistic version of a real logger, doing real-world
// tasks as fast as possible . This gives us an upper bound
// on handler performance, so we can evaluate the (logger-independent) core
// activity of the package in an end-to-end context without concern that a
// slow logger implementation is skewing the results. The writerLogger
// allocates memory only when using strconv.
func BenchmarkEmit(b *testing.B) {
	for _, tc := range []struct {
		name   string
		logger log.Logger
	}{
		{"noop", noop.Logger{}},
		{"writer", &writerLogger{w: io.Discard}},
	} {
		b.Run(tc.name, func(b *testing.B) {
			for _, call := range []struct {
				name string
				f    func()
			}{
				{
					"no attrs",
					func() {
						r := log.Record{Timestamp: testTimestamp, Severity: testSeverity, Body: testBody}
						tc.logger.Emit(ctx, r)
					},
				},
				{
					"3 attrs",
					func() {
						r := log.Record{Timestamp: testTimestamp, Severity: testSeverity, Body: testBody}
						r.AddAttributes(
							attribute.String("string", testString),
							attribute.Float64("float", testFloat),
							attribute.Int("int", testInt),
						)
						tc.logger.Emit(ctx, r)
					},
				},
				{
					// The number should match nAttrsInline in record.go.
					// This should exercise the code path where no allocations
					// happen in Record or Attr. If there are allocations, they
					// should only be from strconv used in writerLogger.
					"5 attrs",
					func() {
						r := log.Record{Timestamp: testTimestamp, Severity: testSeverity, Body: testBody}
						r.AddAttributes(
							attribute.String("string", testString),
							attribute.Float64("float", testFloat),
							attribute.Int("int", testInt),
							attribute.Bool("bool", testBool),
							attribute.String("string", testString),
						)
						tc.logger.Emit(ctx, r)
					},
				},
				{
					"10 attrs",
					func() {
						r := log.Record{Timestamp: testTimestamp, Severity: testSeverity, Body: testBody}
						r.AddAttributes(
							attribute.String("string", testString),
							attribute.Float64("float", testFloat),
							attribute.Int("int", testInt),
							attribute.Bool("bool", testBool),
							attribute.String("string", testString),
							attribute.String("string", testString),
							attribute.Float64("float", testFloat),
							attribute.Int("int", testInt),
							attribute.Bool("bool", testBool),
							attribute.String("string", testString),
						)
						tc.logger.Emit(ctx, r)
					},
				},
				{
					"40 attrs",
					func() {
						r := log.Record{Timestamp: testTimestamp, Severity: testSeverity, Body: testBody}
						r.AddAttributes(
							attribute.String("string", testString),
							attribute.Float64("float", testFloat),
							attribute.Int("int", testInt),
							attribute.Bool("bool", testBool),
							attribute.String("string", testString),
							attribute.String("string", testString),
							attribute.Float64("float", testFloat),
							attribute.Int("int", testInt),
							attribute.Bool("bool", testBool),
							attribute.String("string", testString),
							attribute.String("string", testString),
							attribute.Float64("float", testFloat),
							attribute.Int("int", testInt),
							attribute.Bool("bool", testBool),
							attribute.String("string", testString),
							attribute.String("string", testString),
							attribute.Float64("float", testFloat),
							attribute.Int("int", testInt),
							attribute.Bool("bool", testBool),
							attribute.String("string", testString),
							attribute.String("string", testString),
							attribute.Float64("float", testFloat),
							attribute.Int("int", testInt),
							attribute.Bool("bool", testBool),
							attribute.String("string", testString),
							attribute.String("string", testString),
							attribute.Float64("float", testFloat),
							attribute.Int("int", testInt),
							attribute.Bool("bool", testBool),
							attribute.String("string", testString),
							attribute.String("string", testString),
							attribute.Float64("float", testFloat),
							attribute.Int("int", testInt),
							attribute.Bool("bool", testBool),
							attribute.String("string", testString),
							attribute.String("string", testString),
							attribute.Float64("float", testFloat),
							attribute.Int("int", testInt),
							attribute.Bool("bool", testBool),
							attribute.String("string", testString),
						)
						tc.logger.Emit(ctx, r)
					},
				},
			} {
				b.Run(call.name, func(b *testing.B) {
					b.ReportAllocs()
					for i := 0; i < b.N; i++ {
						call.f()
					}
				})
			}
		})
	}
}