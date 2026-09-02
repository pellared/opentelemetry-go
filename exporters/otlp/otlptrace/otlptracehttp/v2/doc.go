// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otlptracehttp provides a self-contained OTLP trace exporter using
// HTTP with protobuf or JSON payloads.
//
// Unlike the v1 exporter, this module does not expose a transport Client API.
// Keeping OTLP protobuf types internal allows the module to avoid any
// dependency on gRPC.
package otlptracehttp
