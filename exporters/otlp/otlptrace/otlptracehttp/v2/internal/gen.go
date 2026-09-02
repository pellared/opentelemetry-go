// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package internal provides internal implementation details for the exporter.
package internal

//go:generate gotmpl --body=../../../../../../internal/shared/otlp/partialsuccess.go.tmpl "--data={}" --out=partialsuccess.go
//go:generate gotmpl --body=../../../../../../internal/shared/otlp/partialsuccess_test.go.tmpl "--data={}" --out=partialsuccess_test.go

//go:generate gotmpl --body=../../../../../../internal/shared/otlp/retry/retry.go.tmpl "--data={}" --out=retry/retry.go
//go:generate gotmpl --body=../../../../../../internal/shared/otlp/retry/retry_test.go.tmpl "--data={}" --out=retry/retry_test.go

//go:generate gotmpl --body=../../../../../../internal/shared/otlp/envconfig/envconfig.go.tmpl "--data={}" --out=envconfig/envconfig.go
//go:generate gotmpl --body=../../../../../../internal/shared/otlp/envconfig/envconfig_test.go.tmpl "--data={}" --out=envconfig/envconfig_test.go
