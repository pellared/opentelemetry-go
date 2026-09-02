// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptracehttp

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigSignalEnvironmentTakesPrecedence(t *testing.T) {
	clearConfigEnvironment(t)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://generic.example/base")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "https://trace.example/custom/traces")
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_INSECURE", "false")
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "source=generic")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_HEADERS", "source=trace,escaped=value%20with%20spaces")
	t.Setenv("OTEL_EXPORTER_OTLP_COMPRESSION", "none")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_COMPRESSION", "gzip")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL", "http/json")
	t.Setenv("OTEL_EXPORTER_OTLP_TIMEOUT", "1000")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_TIMEOUT", "2500")

	cfg := newConfig()
	assert.Equal(t, "trace.example", cfg.endpoint)
	assert.Equal(t, "/custom/traces", cfg.urlPath)
	assert.False(t, cfg.insecure)
	assert.Equal(t, map[string]string{
		"source":  "trace",
		"escaped": "value with spaces",
	}, cfg.headers)
	assert.Equal(t, GzipCompression, cfg.compression)
	assert.Equal(t, EncodingJSON, cfg.encoding)
	assert.Equal(t, 2500*time.Millisecond, cfg.timeout)
}

func TestOptionsOverrideEnvironment(t *testing.T) {
	clearConfigEnvironment(t)
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "https://environment.example/environment")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL", "http/json")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_TIMEOUT", "5000")

	cfg := newConfig(
		WithEndpointURL("http://option.example/option"),
		WithEncoding(EncodingProtobuf),
		WithTimeout(time.Second),
		WithURLPath("relative/traces"),
		WithHeaders(map[string]string{"source": "option"}),
	)
	assert.Equal(t, "option.example", cfg.endpoint)
	assert.Equal(t, "/relative/traces", cfg.urlPath)
	assert.True(t, cfg.insecure)
	assert.Equal(t, EncodingProtobuf, cfg.encoding)
	assert.Equal(t, time.Second, cfg.timeout)
	assert.Equal(t, map[string]string{"source": "option"}, cfg.headers)
}

func TestGenericEndpointAppendsDefaultTracePath(t *testing.T) {
	clearConfigEnvironment(t)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://collector.example/base/")

	cfg := newConfig()
	assert.Equal(t, "collector.example", cfg.endpoint)
	assert.Equal(t, "/base/v1/traces", cfg.urlPath)
	assert.True(t, cfg.insecure)
}

func TestHTTPClientTakesPrecedence(t *testing.T) {
	clearConfigEnvironment(t)
	custom := &http.Client{Timeout: 37 * time.Second}

	client, err := newClient(t.Context(), newConfig(
		WithHTTPClient(custom),
		WithTimeout(time.Millisecond),
		WithProxy(func(*http.Request) (*url.URL, error) { return nil, nil }),
	))
	assert.NoError(t, err)
	assert.Same(t, custom, client.httpClient)
	assert.Equal(t, 37*time.Second, custom.Timeout)
}

func clearConfigEnvironment(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"ENDPOINT",
		"TRACES_ENDPOINT",
		"INSECURE",
		"TRACES_INSECURE",
		"CERTIFICATE",
		"TRACES_CERTIFICATE",
		"CLIENT_CERTIFICATE",
		"CLIENT_KEY",
		"TRACES_CLIENT_CERTIFICATE",
		"TRACES_CLIENT_KEY",
		"HEADERS",
		"TRACES_HEADERS",
		"COMPRESSION",
		"TRACES_COMPRESSION",
		"PROTOCOL",
		"TRACES_PROTOCOL",
		"TIMEOUT",
		"TRACES_TIMEOUT",
	} {
		t.Setenv("OTEL_EXPORTER_OTLP_"+name, "")
	}
}
