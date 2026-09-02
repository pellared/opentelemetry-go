// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptracehttp

import (
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coltracepb "go.opentelemetry.io/proto/slim/otlp/collector/trace/v1"
	tracepb "go.opentelemetry.io/proto/slim/otlp/trace/v1"
	"google.golang.org/protobuf/proto"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp/v2/internal"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp/v2/internal/otlpjson"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestExporterPayloadEncoding(t *testing.T) {
	tests := []struct {
		name        string
		encoding    Encoding
		compression Compression
		contentType string
	}{
		{
			name:        "protobuf",
			encoding:    EncodingProtobuf,
			compression: NoCompression,
			contentType: contentTypeProto,
		},
		{
			name:        "JSON with gzip",
			encoding:    EncodingJSON,
			compression: GzipCompression,
			contentType: contentTypeJSON,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := &coltracepb.ExportTraceServiceResponse{}
			var responseData []byte
			var err error
			if test.encoding == EncodingJSON {
				responseData, err = otlpjson.MarshalExportTraceServiceResponse(response)
			} else {
				responseData, err = proto.Marshal(response)
			}
			require.NoError(t, err)

			type result struct {
				request         *coltracepb.ExportTraceServiceRequest
				path            string
				contentType     string
				contentEncoding string
				header          string
				err             error
			}
			requests := make(chan result, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				request, err := decodeRequest(r)
				requests <- result{
					request:         request,
					path:            r.URL.Path,
					contentType:     r.Header.Get("Content-Type"),
					contentEncoding: r.Header.Get("Content-Encoding"),
					header:          r.Header.Get("X-Test-Header"),
					err:             err,
				}
				w.Header().Set("Content-Type", test.contentType)
				_, _ = w.Write(responseData)
			}))
			t.Cleanup(server.Close)

			exporter, err := New(t.Context(),
				WithEndpointURL(server.URL+"/custom/traces"),
				WithEncoding(test.encoding),
				WithCompression(test.compression),
				WithHeaders(map[string]string{"X-Test-Header": "present"}),
				WithRetry(RetryConfig{Enabled: false}),
			)
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, exporter.Shutdown(context.WithoutCancel(t.Context())))
			})

			traceID := trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
			spanID := trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8}
			start := time.Unix(100, 200)
			span := tracetest.SpanStub{
				Name: "test-span",
				SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
					TraceID:    traceID,
					SpanID:     spanID,
					TraceFlags: trace.FlagsSampled,
				}),
				SpanKind:  trace.SpanKindServer,
				StartTime: start,
				EndTime:   start.Add(time.Second),
				Attributes: []attribute.KeyValue{
					attribute.String("http.request.method", "GET"),
				},
				Events: []sdktrace.Event{
					{
						Name: "exception",
						Time: start.Add(time.Millisecond),
						Attributes: []attribute.KeyValue{
							attribute.String("exception.message", "boom"),
						},
					},
				},
				Status: sdktrace.Status{Code: codes.Error, Description: "failed"},
				Resource: resource.NewSchemaless(
					attribute.String("service.name", "trace-v2-test"),
				),
				InstrumentationScope: instrumentation.Scope{Name: "test.scope", Version: "1.2.3"},
			}
			err = exporter.ExportSpans(t.Context(), []sdktrace.ReadOnlySpan{span.Snapshot()})
			require.NoError(t, err)

			got := <-requests
			require.NoError(t, got.err)
			assert.Equal(t, "/custom/traces", got.path)
			assert.Equal(t, test.contentType, got.contentType)
			assert.Equal(t, "present", got.header)
			if test.compression == GzipCompression {
				assert.Equal(t, "gzip", got.contentEncoding)
			} else {
				assert.Empty(t, got.contentEncoding)
			}

			require.NotNil(t, got.request)
			require.Len(t, got.request.ResourceSpans, 1)
			resourceSpans := got.request.ResourceSpans[0]
			require.Len(t, resourceSpans.Resource.Attributes, 1)
			assert.Equal(t, "service.name", resourceSpans.Resource.Attributes[0].Key)
			assert.Equal(t, "trace-v2-test", resourceSpans.Resource.Attributes[0].Value.GetStringValue())
			require.Len(t, resourceSpans.ScopeSpans, 1)
			scopeSpans := resourceSpans.ScopeSpans[0]
			assert.Equal(t, "test.scope", scopeSpans.Scope.Name)
			assert.Equal(t, "1.2.3", scopeSpans.Scope.Version)
			require.Len(t, scopeSpans.Spans, 1)
			gotSpan := scopeSpans.Spans[0]
			assert.Equal(t, "test-span", gotSpan.Name)
			assert.Equal(t, traceID[:], gotSpan.TraceId)
			assert.Equal(t, spanID[:], gotSpan.SpanId)
			assert.Equal(t, tracepb.Span_SPAN_KIND_SERVER, gotSpan.Kind)
			require.Len(t, gotSpan.Attributes, 1)
			assert.Equal(t, "GET", gotSpan.Attributes[0].Value.GetStringValue())
			require.Len(t, gotSpan.Events, 1)
			assert.Equal(t, "exception", gotSpan.Events[0].Name)
			assert.Equal(t, tracepb.Status_STATUS_CODE_ERROR, gotSpan.Status.Code)
			assert.Equal(t, "failed", gotSpan.Status.Message)
		})
	}
}

func TestExporterRetriesTransientResponse(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		if requests.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			http.Error(w, "try again", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", contentTypeProto)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	exporter, err := New(t.Context(),
		WithEndpointURL(server.URL),
		WithRetry(RetryConfig{
			Enabled:         true,
			InitialInterval: time.Nanosecond,
			MaxInterval:     time.Nanosecond,
			MaxElapsedTime:  time.Second,
		}),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, exporter.Shutdown(context.WithoutCancel(t.Context())))
	})

	err = exporter.ExportSpans(t.Context(), tracetest.SpanStubs{{Name: "retry"}}.Snapshots())
	require.NoError(t, err)
	assert.Equal(t, int32(2), requests.Load())
}

func TestExporterReportsJSONPartialSuccess(t *testing.T) {
	response := &coltracepb.ExportTraceServiceResponse{
		PartialSuccess: &coltracepb.ExportTracePartialSuccess{
			RejectedSpans: 2,
			ErrorMessage:  "collector dropped spans",
		},
	}
	responseData, err := otlpjson.MarshalExportTraceServiceResponse(response)
	require.NoError(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", contentTypeJSON+"; charset=utf-8")
		_, _ = w.Write(responseData)
	}))
	t.Cleanup(server.Close)

	exporter, err := New(t.Context(),
		WithEndpointURL(server.URL),
		WithRetry(RetryConfig{Enabled: false}),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, exporter.Shutdown(context.WithoutCancel(t.Context())))
	})

	err = exporter.ExportSpans(t.Context(), tracetest.SpanStubs{{Name: "partial"}}.Snapshots())
	require.Error(t, err)
	assert.ErrorIs(t, err, internal.PartialSuccess{})
	var partial internal.PartialSuccess
	require.ErrorAs(t, err, &partial)
	assert.Equal(t, int64(2), partial.RejectedItems)
	assert.Equal(t, "spans", partial.RejectedKind)
	assert.Equal(t, "collector dropped spans", partial.ErrorMessage)
}

func TestExporterShutdown(t *testing.T) {
	started := make(chan struct{})
	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			close(started)
			<-r.Context().Done()
			return nil, context.Cause(r.Context())
		}),
	}

	exporter, err := New(t.Context(),
		WithHTTPClient(httpClient),
		WithRetry(RetryConfig{Enabled: false}),
	)
	require.NoError(t, err)

	exportDone := make(chan error, 1)
	go func() {
		exportDone <- exporter.ExportSpans(t.Context(), tracetest.SpanStubs{{Name: "shutdown"}}.Snapshots())
	}()
	<-started
	require.NoError(t, exporter.Shutdown(t.Context()))

	require.ErrorIs(t, <-exportDone, context.Canceled)
	assert.ErrorIs(
		t,
		exporter.ExportSpans(t.Context(), tracetest.SpanStubs{{Name: "stopped"}}.Snapshots()),
		errShutdown,
	)
	require.NoError(t, exporter.Shutdown(t.Context()))
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestNewWithCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := New(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

func decodeRequest(r *http.Request) (*coltracepb.ExportTraceServiceRequest, error) {
	var body io.Reader = r.Body
	if r.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(r.Body)
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		body = reader
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	request := &coltracepb.ExportTraceServiceRequest{}
	switch mediaType(r.Header.Get("Content-Type")) {
	case contentTypeProto:
		err = proto.Unmarshal(data, request)
	case contentTypeJSON:
		err = otlpjson.UnmarshalExportTraceServiceRequest(data, request)
	default:
		err = errors.New("unexpected content type")
	}
	return request, err
}
