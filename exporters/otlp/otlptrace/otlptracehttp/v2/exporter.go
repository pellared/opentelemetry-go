// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptracehttp

import (
	"context"
	"fmt"
	"sync/atomic"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp/v2/internal/transform"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Exporter exports spans using OTLP over HTTP.
type Exporter struct {
	client  *client
	stopped atomic.Bool
}

var _ sdktrace.SpanExporter = (*Exporter)(nil)

// New creates an OTLP/HTTP trace Exporter.
func New(ctx context.Context, opts ...Option) (*Exporter, error) {
	client, err := newClient(ctx, newConfig(opts...))
	if err != nil {
		return nil, err
	}
	return &Exporter{client: client}, nil
}

// ExportSpans transforms and exports spans to an OTLP receiver.
func (e *Exporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	if e.stopped.Load() {
		return errShutdown
	}

	resourceSpans := transform.Spans(spans)
	if len(resourceSpans) == 0 {
		return nil
	}

	if e.stopped.Load() {
		return errShutdown
	}
	if err := e.client.uploadTraces(ctx, resourceSpans); err != nil {
		return fmt.Errorf("traces export: %w", err)
	}
	return nil
}

// Shutdown stops the Exporter and interrupts in-flight requests.
func (e *Exporter) Shutdown(ctx context.Context) error {
	if e.stopped.Swap(true) {
		return nil
	}
	return e.client.shutdown(ctx)
}

// MarshalLog returns logging data about the Exporter.
func (*Exporter) MarshalLog() any {
	return struct{ Type string }{Type: "OTLP/HTTP"}
}
