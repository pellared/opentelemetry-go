// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package logtest // import "go.opentelemetry.io/otel/sdk/log/logtest"

import (
	"slices"
	"time"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
)

// attributesInlineCount is the number of attributes that are efficiently
// stored in an array within a record. This value is borrowed from slog which
// performed a quantitative survey of log library use and found this value to
// cover 95% of all use-cases (https://go.dev/blog/slog#performance).
const attributesInlineCount = 5

// record is a log record emitted by the Logger.
type record struct {
	// Do not embed the log.Record. Attributes need to be overwrite-able and
	// deep-copying needs to be possible.

	timestamp         time.Time
	observedTimestamp time.Time
	severity          log.Severity
	severityText      string
	body              log.Value

	// The fields below are for optimizing the implementation of Attributes and
	// AddAttributes. This design is borrowed from the slog Record type:
	// https://cs.opensource.google/go/go/+/refs/tags/go1.22.0:src/log/slog/record.go;l=20

	// Allocation optimization: an inline array sized to hold
	// the majority of log calls (based on examination of open-source
	// code). It holds the start of the list of attributes.
	front [attributesInlineCount]log.KeyValue

	// The number of attributes in front.
	nFront int

	// The list of attributes except for those in front.
	// Invariants:
	//   - len(back) > 0 if nFront == len(front)
	//   - Unused array elements are zero-ed. Used to detect mistakes.
	back []log.KeyValue

	traceID    trace.TraceID
	spanID     trace.SpanID
	traceFlags trace.TraceFlags

	// resource represents the entity that collected the log.
	resource *resource.Resource

	// scope is the Scope that the Logger was created with.
	scope *instrumentation.Scope

	attributeValueLengthLimit int
	attributeCountLimit       int
}

// Timestamp returns the time when the log record occurred.
func (r *record) Timestamp() time.Time {
	return r.timestamp
}

// SetTimestamp sets the time when the log record occurred.
func (r *record) SetTimestamp(t time.Time) {
	r.timestamp = t
}

// ObservedTimestamp returns the time when the log record was observed.
func (r *record) ObservedTimestamp() time.Time {
	return r.observedTimestamp
}

// SetObservedTimestamp sets the time when the log record was observed.
func (r *record) SetObservedTimestamp(t time.Time) {
	r.observedTimestamp = t
}

// Severity returns the severity of the log record.
func (r *record) Severity() log.Severity {
	return r.severity
}

// SetSeverity sets the severity level of the log record.
func (r *record) SetSeverity(level log.Severity) {
	r.severity = level
}

// SeverityText returns severity (also known as log level) text. This is the
// original string representation of the severity as it is known at the source.
func (r *record) SeverityText() string {
	return r.severityText
}

// SetSeverityText sets severity (also known as log level) text. This is the
// original string representation of the severity as it is known at the source.
func (r *record) SetSeverityText(text string) {
	r.severityText = text
}

// Body returns the body of the log record.
func (r *record) Body() log.Value {
	return r.body
}

// SetBody sets the body of the log record.
func (r *record) SetBody(v log.Value) {
	r.body = v
}

// WalkAttributes walks all attributes the log record holds by calling f for
// each on each [log.KeyValue] in the [record]. Iteration stops if f returns false.
func (r *record) WalkAttributes(f func(log.KeyValue) bool) {
	for i := 0; i < r.nFront; i++ {
		if !f(r.front[i]) {
			return
		}
	}
	for _, a := range r.back {
		if !f(a) {
			return
		}
	}
}

// AddAttributes adds attributes to the log record.
func (r *record) AddAttributes(attrs ...log.KeyValue) {
	var i int
	for i = 0; i < len(attrs) && r.nFront < len(r.front); i++ {
		a := attrs[i]
		r.front[r.nFront] = a
		r.nFront++
	}

	r.back = slices.Grow(r.back, len(attrs[i:]))
	r.back = append(r.back, attrs[i:]...)
}

// SetAttributes sets (and overrides) attributes to the log record.
func (r *record) SetAttributes(attrs ...log.KeyValue) {
	r.nFront = 0
	var i int
	for i = 0; i < len(attrs) && r.nFront < len(r.front); i++ {
		a := attrs[i]
		r.front[r.nFront] = a
		r.nFront++
	}

	r.back = slices.Clone(attrs[i:])
}

// AttributesLen returns the number of attributes in the log record.
func (r *record) AttributesLen() int {
	return r.nFront + len(r.back)
}

// TraceID returns the trace ID or empty array.
func (r *record) TraceID() trace.TraceID {
	return r.traceID
}

// SetTraceID sets the trace ID.
func (r *record) SetTraceID(id trace.TraceID) {
	r.traceID = id
}

// SpanID returns the span ID or empty array.
func (r *record) SpanID() trace.SpanID {
	return r.spanID
}

// SetSpanID sets the span ID.
func (r *record) SetSpanID(id trace.SpanID) {
	r.spanID = id
}

// TraceFlags returns the trace flags.
func (r *record) TraceFlags() trace.TraceFlags {
	return r.traceFlags
}

// SetTraceFlags sets the trace flags.
func (r *record) SetTraceFlags(flags trace.TraceFlags) {
	r.traceFlags = flags
}

// Resource returns the entity that collected the log.
func (r *record) Resource() resource.Resource {
	if r.resource == nil {
		return *resource.Empty()
	}
	return *r.resource
}

// InstrumentationScope returns the scope that the Logger was created with.
func (r *record) InstrumentationScope() instrumentation.Scope {
	if r.scope == nil {
		return instrumentation.Scope{}
	}
	return *r.scope
}

// AttributeValueLengthLimit is the maximum allowed attribute value length.
//
// This limit only applies to string and string slice attribute values.
// Any string longer than this value should be truncated to this length.
//
// Negative value means no limit should be applied.
func (r *record) AttributeValueLengthLimit() int {
	return r.attributeValueLengthLimit
}

// AttributeCountLimit is the maximum allowed log record attribute count. Any
// attribute added to a log record once this limit is reached should be dropped.
//
// Zero means no attributes should be recorded.
//
// Negative value means no limit should be applied.
func (r *record) AttributeCountLimit() int {
	return r.attributeCountLimit
}

// Clone returns a copy of the record with no shared state. The original record
// and the clone can both be modified without interfering with each other.
func (r *record) Clone() record {
	res := *r
	res.back = slices.Clone(r.back)
	return res
}
