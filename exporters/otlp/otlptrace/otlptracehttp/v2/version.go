// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptracehttp

import "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp/v2/internal"

// Version returns the current exporter version.
func Version() string {
	return internal.Version
}
