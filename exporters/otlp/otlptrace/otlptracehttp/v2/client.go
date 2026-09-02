// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptracehttp

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	coltracepb "go.opentelemetry.io/proto/slim/otlp/collector/trace/v1"
	tracepb "go.opentelemetry.io/proto/slim/otlp/trace/v1"
	"google.golang.org/protobuf/proto"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp/v2/internal"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp/v2/internal/otlpjson"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp/v2/internal/retry"
)

const (
	contentTypeProto = "application/x-protobuf"
	contentTypeJSON  = "application/json"
)

var (
	errInsecureEndpointWithTLS       = errors.New("insecure HTTP endpoint cannot use TLS client configuration")
	errShutdown                      = errors.New("HTTP exporter is shutdown")
	maxResponseBodySize        int64 = 4 * 1024 * 1024
)

var defaultTransport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: time.Second,
}

type client struct {
	cfg         config
	requestFunc retry.RequestFunc
	httpClient  *http.Client
	stopCh      chan struct{}
	stopOnce    sync.Once
}

func newClient(ctx context.Context, cfg config) (*client, error) {
	if cfg.insecure && cfg.tlsCfg != nil {
		return nil, errInsecureEndpointWithTLS
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	httpClient := cfg.httpClient
	if httpClient == nil {
		transport := defaultTransport
		if cfg.tlsCfg != nil || cfg.proxy != nil {
			transport = defaultTransport.Clone()
			transport.TLSClientConfig = cfg.tlsCfg
			transport.Proxy = cfg.proxy
		}
		httpClient = &http.Client{
			Transport: transport,
			Timeout:   cfg.timeout,
		}
	}

	return &client{
		cfg:         cfg,
		requestFunc: cfg.retryCfg.RequestFunc(evaluate),
		httpClient:  httpClient,
		stopCh:      make(chan struct{}),
	}, nil
}

func (c *client) uploadTraces(ctx context.Context, spans []*tracepb.ResourceSpans) (uploadErr error) {
	pbRequest := &coltracepb.ExportTraceServiceRequest{ResourceSpans: spans}
	request, err := c.newRequest(pbRequest)
	if err != nil {
		return err
	}

	ctx, cancel := c.contextWithStop(ctx)
	defer cancel()

	requestErr := c.requestFunc(ctx, func(requestCtx context.Context) error {
		if err := requestCtx.Err(); err != nil {
			return err
		}

		request.reset(requestCtx)
		resp, err := c.httpClient.Do(request.Request)
		var urlErr *url.Error
		if errors.As(err, &urlErr) && urlErr.Temporary() {
			return newResponseError(http.Header{}, err)
		}
		if err != nil {
			return err
		}
		if resp == nil {
			return errors.New("OTLP endpoint returned a nil response")
		}
		if resp.Body == nil {
			resp.Body = http.NoBody
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				uploadErr = errors.Join(uploadErr, err)
			}
		}()

		if resp.StatusCode >= http.StatusOK && resp.StatusCode <= 299 {
			data, err := readResponseBody(resp.Body)
			if err != nil {
				return err
			}
			if len(data) == 0 {
				return nil
			}

			var response coltracepb.ExportTraceServiceResponse
			switch mediaType(resp.Header.Get("Content-Type")) {
			case contentTypeProto:
				if err := proto.Unmarshal(data, &response); err != nil {
					return err
				}
			case contentTypeJSON:
				if err := otlpjson.UnmarshalExportTraceServiceResponse(data, &response); err != nil {
					return err
				}
			default:
				return nil
			}

			if partial := response.PartialSuccess; partial != nil {
				message := partial.GetErrorMessage()
				rejected := partial.GetRejectedSpans()
				if rejected != 0 || message != "" {
					uploadErr = errors.Join(uploadErr, internal.TracePartialSuccessError(rejected, message))
				}
			}
			return nil
		}

		data, err := readResponseBody(resp.Body)
		if err != nil {
			return err
		}
		responseBody := strings.TrimSpace(string(data))
		if responseBody == "" {
			responseBody = "(empty)"
		}
		bodyErr := fmt.Errorf("body: %s", responseBody)

		switch resp.StatusCode {
		case http.StatusTooManyRequests,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			return newResponseError(resp.Header, bodyErr)
		default:
			return fmt.Errorf("failed to send to %s: %s (%w)", request.URL, resp.Status, bodyErr)
		}
	})
	return errors.Join(uploadErr, requestErr)
}

func readResponseBody(body io.Reader) ([]byte, error) {
	var data bytes.Buffer
	if _, err := io.Copy(&data, http.MaxBytesReader(nil, io.NopCloser(body), maxResponseBodySize)); err != nil {
		if maxBytesErr, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return nil, fmt.Errorf("response body too large: exceeded %d bytes", maxBytesErr.Limit)
		}
		return nil, err
	}
	return data.Bytes(), nil
}

func mediaType(value string) string {
	value, _, _ = strings.Cut(value, ";")
	return strings.TrimSpace(value)
}

func (c *client) newRequest(message *coltracepb.ExportTraceServiceRequest) (request, error) {
	body, err := c.marshalRequest(message)
	if err != nil {
		return request{}, err
	}
	if c.cfg.maxRequestSize > 0 && len(body) > c.cfg.maxRequestSize {
		return request{}, fmt.Errorf("request body too large: exceeded %d bytes", c.cfg.maxRequestSize)
	}

	target := url.URL{
		Scheme: "https",
		Host:   c.cfg.endpoint,
		Path:   c.cfg.urlPath,
	}
	if c.cfg.insecure {
		target.Scheme = "http"
	}
	httpRequest, err := http.NewRequestWithContext(context.Background(), http.MethodPost, target.String(), http.NoBody)
	if err != nil {
		return request{}, err
	}
	httpRequest.Header.Set("User-Agent", "OTel Go OTLP over HTTP traces exporter/"+Version())
	for key, value := range c.cfg.headers {
		httpRequest.Header.Set(key, value)
	}

	req := request{Request: httpRequest}
	switch c.cfg.compression {
	case NoCompression:
		httpRequest.ContentLength = int64(len(body))
		req.bodyReader = bodyReader(body)
		httpRequest.GetBody = bodyReaderErr(body)
	case GzipCompression:
		httpRequest.ContentLength = -1
		httpRequest.Header.Set("Content-Encoding", "gzip")
		compressed, err := gzipBody(body)
		if err != nil {
			return request{}, err
		}
		req.bodyReader = bodyReader(compressed)
		httpRequest.GetBody = bodyReaderErr(compressed)
	default:
		return request{}, fmt.Errorf("unsupported compression: %d", c.cfg.compression)
	}

	if c.cfg.encoding == EncodingJSON {
		httpRequest.Header.Set("Content-Type", contentTypeJSON)
	} else {
		httpRequest.Header.Set("Content-Type", contentTypeProto)
	}
	return req, nil
}

func (c *client) marshalRequest(message *coltracepb.ExportTraceServiceRequest) ([]byte, error) {
	if c.cfg.encoding == EncodingJSON {
		body, err := otlpjson.MarshalExportTraceServiceRequest(message)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body in json: %w", err)
		}
		return body, nil
	}
	body, err := proto.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body in protobuf: %w", err)
	}
	return body, nil
}

var gzipWriterPool = sync.Pool{
	New: func() any {
		return gzip.NewWriter(io.Discard)
	},
}

func gzipBody(body []byte) ([]byte, error) {
	writer := gzipWriterPool.Get().(*gzip.Writer)
	defer func() {
		writer.Reset(io.Discard)
		gzipWriterPool.Put(writer)
	}()

	var compressed bytes.Buffer
	writer.Reset(&compressed)
	if _, err := writer.Write(body); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return compressed.Bytes(), nil
}

func (c *client) shutdown(ctx context.Context) error {
	c.stopOnce.Do(func() {
		close(c.stopCh)
	})
	return ctx.Err()
}

func (c *client) contextWithStop(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		select {
		case <-ctx.Done():
		case <-c.stopCh:
			cancel()
		}
	}()
	return ctx, cancel
}

func bodyReader(body []byte) func() io.ReadCloser {
	return func() io.ReadCloser {
		return io.NopCloser(bytes.NewReader(body))
	}
}

func bodyReaderErr(body []byte) func() (io.ReadCloser, error) {
	return func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
}

type request struct {
	*http.Request
	bodyReader func() io.ReadCloser
}

func (r *request) reset(ctx context.Context) {
	r.Body = r.bodyReader()
	r.Request = r.WithContext(ctx)
}

type retryableError struct {
	throttle time.Duration
	err      error
}

func newResponseError(headers http.Header, wrapped error) error {
	result := retryableError{err: wrapped}
	if value := headers.Get("Retry-After"); value != "" {
		result.throttle = retryAfterDuration(value)
	}
	return result
}

func retryAfterDuration(value string) time.Duration {
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil && seconds >= 0 {
		const maxRetryAfterSeconds = int64(1<<63-1) / int64(time.Second)
		if seconds > maxRetryAfterSeconds {
			return time.Duration(1<<63 - 1)
		}
		return time.Duration(seconds) * time.Second
	}
	if date, err := http.ParseTime(value); err == nil {
		return max(time.Until(date), 0)
	}
	return 0
}

func (e retryableError) Error() string {
	if e.err == nil {
		return "retry-able request failure"
	}
	return "retry-able request failure: " + e.err.Error()
}

func (e retryableError) Unwrap() error {
	return e.err
}

func evaluate(err error) (bool, time.Duration) {
	retryable, ok := err.(retryableError) //nolint:errorlint
	if !ok {
		return false, 0
	}
	return true, retryable.throttle
}
