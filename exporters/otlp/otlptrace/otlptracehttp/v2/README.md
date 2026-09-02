# OTLP Trace HTTP Exporter v2

[![PkgGoDev](https://pkg.go.dev/badge/go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp/v2)](https://pkg.go.dev/go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp/v2)

This prototype is a self-contained OTLP/HTTP trace exporter. It supports
protobuf and JSON payloads without importing the gRPC runtime.

The v2 API intentionally omits the v1 `Client`, `NewClient`, and
`NewUnstarted` APIs. The exporter owns transformation and transport so its
protobuf implementation remains private.

Experimental exporter self-observability metrics are not included in this
prototype. Export behavior, environment configuration, retry handling,
partial-success responses, and lifecycle semantics are retained.

## Prototype dependency

The descriptor-distinct protobuf module is currently available only from the
companion fork prototype. Because Go does not inherit `replace` directives
from dependency modules, consumers testing this branch need this directive in
their main module:

```go
replace go.opentelemetry.io/proto/otlphttp => github.com/pellared/opentelemetry-proto-go/otlphttp v0.0.0-20260902183618-9b820c222ae8
```
