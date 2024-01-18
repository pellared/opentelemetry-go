module go.opentelemetry.io/otel/log/internal

go 1.20

require (
	github.com/go-logr/logr v1.4.1
	github.com/stretchr/testify v1.8.4
	go.opentelemetry.io/otel v1.23.0-rc.1
	go.opentelemetry.io/otel/log v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel/trace v1.23.0-rc.1
	golang.org/x/exp v0.0.0-20231127185646-65229373498e
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/otel/metric v1.23.0-rc.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/otel/log => ../

replace go.opentelemetry.io/otel => ../..

replace go.opentelemetry.io/otel/trace => ../../trace

replace go.opentelemetry.io/otel/metric => ../../metric
