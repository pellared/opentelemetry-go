// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package benchmark

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
)

type spyLogger struct {
	embedded.Logger
	Record log.Record
	Attrs  []attribute.KeyValue
}

func (l *spyLogger) Emit(_ context.Context, r log.Record) {
	l.Record = r
}

func (l *spyLogger) WithAttributes(attrs ...attribute.KeyValue) log.Logger {
	l.Attrs = attrs
	return l
}
