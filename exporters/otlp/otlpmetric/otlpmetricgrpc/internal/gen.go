// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package internal provides internal functionally for the otlpmetricgrpc package.
package internal

//go:generate gotmpl --body=../../../../../internal/shared/otlp/partialsuccess.go.tmpl "--data={}" --out=partialsuccess.go
//go:generate gotmpl --body=../../../../../internal/shared/otlp/partialsuccess_test.go.tmpl "--data={}" --out=partialsuccess_test.go

//go:generate gotmpl --body=../../../../../internal/shared/otlp/observ/target.go.tmpl "--data={ \"pkg\": \"observ\" }" --out=observ/target.go
//go:generate gotmpl --body=../../../../../internal/shared/otlp/observ/target_test.go.tmpl "--data={ \"pkg\": \"observ\" }" --out=observ/target_test.go

//go:generate gotmpl --body=../../../../../internal/shared/otlp/retry/retry.go.tmpl "--data={}" --out=retry/retry.go
//go:generate gotmpl --body=../../../../../internal/shared/otlp/retry/retry_test.go.tmpl "--data={}" --out=retry/retry_test.go

//go:generate gotmpl --body=../../../../../internal/shared/otlp/envconfig/envconfig.go.tmpl "--data={}" --out=envconfig/envconfig.go
//go:generate gotmpl --body=../../../../../internal/shared/otlp/envconfig/envconfig_test.go.tmpl "--data={}" --out=envconfig/envconfig_test.go

//go:generate gotmpl --body=../../../../../internal/shared/otlp/otlpmetric/oconf/envconfig.go.tmpl "--data={\"envconfigImportPath\": \"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal/envconfig\", \"grpc\": true}" --out=oconf/envconfig.go
//go:generate gotmpl --body=../../../../../internal/shared/otlp/otlpmetric/oconf/envconfig_test.go.tmpl "--data={\"grpc\": true}" --out=oconf/envconfig_test.go
//go:generate gotmpl --body=../../../../../internal/shared/otlp/otlpmetric/oconf/options.go.tmpl "--data={\"retryImportPath\": \"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal/retry\", \"grpc\": true}" --out=oconf/options.go
//go:generate gotmpl --body=../../../../../internal/shared/otlp/otlpmetric/oconf/options_test.go.tmpl "--data={\"envconfigImportPath\": \"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal/envconfig\", \"grpc\": true}" --out=oconf/options_test.go
//go:generate gotmpl --body=../../../../../internal/shared/otlp/otlpmetric/oconf/optiontypes.go.tmpl "--data={\"grpc\": true}" --out=oconf/optiontypes.go
//go:generate gotmpl --body=../../../../../internal/shared/otlp/otlpmetric/oconf/tls.go.tmpl "--data={}" --out=oconf/tls.go

//go:generate gotmpl --body=../../../../../internal/shared/otlp/otlpmetric/otest/client.go.tmpl "--data={\"internalImportPath\": \"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal\", \"protoImportPath\": \"go.opentelemetry.io/proto/otlp\"}" --out=otest/client.go
//go:generate gotmpl --body=../../../../../internal/shared/otlp/otlpmetric/otest/client_test.go.tmpl "--data={\"internalImportPath\": \"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal\", \"protoImportPath\": \"go.opentelemetry.io/proto/otlp\"}" --out=otest/client_test.go
//go:generate gotmpl --body=../../../../../internal/shared/otlp/otlpmetric/otest/collector.go.tmpl "--data={\"oconfImportPath\": \"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc/internal/oconf\", \"protoImportPath\": \"go.opentelemetry.io/proto/otlp\", \"grpc\": true}" --out=otest/collector.go

//go:generate gotmpl --body=../../../../../internal/shared/otlp/otlpmetric/transform/attribute.go.tmpl "--data={\"protoImportPath\": \"go.opentelemetry.io/proto/otlp\"}" --out=transform/attribute.go
//go:generate gotmpl --body=../../../../../internal/shared/otlp/otlpmetric/transform/attribute_test.go.tmpl "--data={\"protoImportPath\": \"go.opentelemetry.io/proto/otlp\"}" --out=transform/attribute_test.go
//go:generate gotmpl --body=../../../../../internal/shared/otlp/otlpmetric/transform/error.go.tmpl "--data={\"protoImportPath\": \"go.opentelemetry.io/proto/otlp\"}" --out=transform/error.go
//go:generate gotmpl --body=../../../../../internal/shared/otlp/otlpmetric/transform/error_test.go.tmpl "--data={}" --out=transform/error_test.go
//go:generate gotmpl --body=../../../../../internal/shared/otlp/otlpmetric/transform/metricdata.go.tmpl "--data={\"protoImportPath\": \"go.opentelemetry.io/proto/otlp\"}" --out=transform/metricdata.go
//go:generate gotmpl --body=../../../../../internal/shared/otlp/otlpmetric/transform/metricdata_test.go.tmpl "--data={\"protoImportPath\": \"go.opentelemetry.io/proto/otlp\"}" --out=transform/metricdata_test.go

//go:generate gotmpl --body=../../../../../internal/shared/counter/counter.go.tmpl "--data={}" --out=counter/counter.go
//go:generate gotmpl --body=../../../../../internal/shared/counter/counter_test.go.tmpl "--data={}" --out=counter/counter_test.go

//go:generate gotmpl --body=../../../../../internal/shared/x/x.go.tmpl "--data={ \"pkg\": \"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc\" }" --out=x/x.go
//go:generate gotmpl --body=../../../../../internal/shared/x/x_test.go.tmpl "--data={}" --out=x/x_test.go

// Keep protocol-conditional template output canonical.
//go:generate gofmt -w oconf otest
