// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptracehttp

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp/v2/internal/envconfig"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp/v2/internal/retry"
	"go.opentelemetry.io/otel/internal/global"
)

const (
	defaultEndpoint       = "localhost:4318"
	defaultTracesPath     = "/v1/traces"
	defaultTimeout        = 10 * time.Second
	defaultMaxRequestSize = 64 * 1024 * 1024
)

// Compression describes the compression used for exported payloads.
type Compression int

const (
	// NoCompression sends payloads without compression.
	NoCompression Compression = iota
	// GzipCompression sends gzip-compressed payloads.
	GzipCompression
)

// Encoding describes the OTLP/HTTP payload encoding.
type Encoding int

const (
	// EncodingProtobuf sends protobuf-encoded payloads.
	EncodingProtobuf Encoding = iota
	// EncodingJSON sends JSON-encoded payloads.
	EncodingJSON
)

// HTTPTransportProxyFunc resolves which proxy URL to use for an HTTP request.
type HTTPTransportProxyFunc func(*http.Request) (*url.URL, error)

// RetryConfig configures retrying transient export failures.
type RetryConfig retry.Config

type config struct {
	endpoint       string
	urlPath        string
	insecure       bool
	tlsCfg         *tls.Config
	headers        map[string]string
	compression    Compression
	encoding       Encoding
	maxRequestSize int
	timeout        time.Duration
	retryCfg       retry.Config
	proxy          HTTPTransportProxyFunc
	httpClient     *http.Client
}

// Option configures an Exporter.
type Option interface {
	apply(config) config
}

type optionFunc func(config) config

func (o optionFunc) apply(cfg config) config {
	return o(cfg)
}

func newConfig(opts ...Option) config {
	cfg := config{
		endpoint:       defaultEndpoint,
		urlPath:        defaultTracesPath,
		compression:    NoCompression,
		encoding:       EncodingProtobuf,
		maxRequestSize: defaultMaxRequestSize,
		timeout:        defaultTimeout,
		retryCfg:       retry.DefaultConfig,
		proxy:          http.ProxyFromEnvironment,
	}
	cfg = applyEnv(cfg)
	for _, opt := range opts {
		cfg = opt.apply(cfg)
	}
	cfg.urlPath = cleanPath(cfg.urlPath)
	return cfg
}

func applyEnv(cfg config) config {
	reader := envconfig.EnvOptionsReader{
		GetEnv:    os.Getenv,
		ReadFile:  os.ReadFile,
		Namespace: "OTEL_EXPORTER_OTLP",
	}

	var tlsCfg tls.Config
	var hasTLSConfig bool

	reader.Apply(
		envconfig.WithURL("ENDPOINT", func(u *url.URL) {
			cfg.endpoint = u.Host
			cfg.urlPath = path.Join(u.Path, defaultTracesPath)
			cfg.insecure = isInsecureScheme(u.Scheme)
		}),
		envconfig.WithURL("TRACES_ENDPOINT", func(u *url.URL) {
			cfg.endpoint = u.Host
			cfg.urlPath = u.Path
			if cfg.urlPath == "" {
				cfg.urlPath = "/"
			}
			cfg.insecure = isInsecureScheme(u.Scheme)
		}),
		envconfig.WithBool("INSECURE", func(v bool) { cfg.insecure = v }),
		envconfig.WithBool("TRACES_INSECURE", func(v bool) { cfg.insecure = v }),
		envconfig.WithCertPool("CERTIFICATE", func(pool *x509.CertPool) {
			tlsCfg.RootCAs = pool
			hasTLSConfig = true
		}),
		envconfig.WithCertPool("TRACES_CERTIFICATE", func(pool *x509.CertPool) {
			tlsCfg.RootCAs = pool
			hasTLSConfig = true
		}),
		envconfig.WithClientCert("CLIENT_CERTIFICATE", "CLIENT_KEY", func(cert tls.Certificate) {
			tlsCfg.Certificates = []tls.Certificate{cert}
			hasTLSConfig = true
		}),
		envconfig.WithClientCert("TRACES_CLIENT_CERTIFICATE", "TRACES_CLIENT_KEY", func(cert tls.Certificate) {
			tlsCfg.Certificates = []tls.Certificate{cert}
			hasTLSConfig = true
		}),
		envconfig.WithHeaders("HEADERS", func(v map[string]string) { cfg.headers = v }),
		envconfig.WithHeaders("TRACES_HEADERS", func(v map[string]string) { cfg.headers = v }),
		envconfig.WithString("COMPRESSION", func(v string) { cfg.compression = envCompression(v) }),
		envconfig.WithString("TRACES_COMPRESSION", func(v string) { cfg.compression = envCompression(v) }),
		envconfig.WithString("PROTOCOL", func(v string) { cfg.encoding = envEncoding(v) }),
		envconfig.WithString("TRACES_PROTOCOL", func(v string) { cfg.encoding = envEncoding(v) }),
		envconfig.WithDuration("TIMEOUT", func(v time.Duration) { cfg.timeout = v }),
		envconfig.WithDuration("TRACES_TIMEOUT", func(v time.Duration) { cfg.timeout = v }),
	)
	if hasTLSConfig {
		cfg.tlsCfg = &tlsCfg
	}
	return cfg
}

func isInsecureScheme(scheme string) bool {
	switch strings.ToLower(scheme) {
	case "http", "unix":
		return true
	default:
		return false
	}
}

func envCompression(value string) Compression {
	if strings.EqualFold(value, "gzip") {
		return GzipCompression
	}
	return NoCompression
}

func envEncoding(value string) Encoding {
	switch strings.ToLower(value) {
	case "http/json":
		return EncodingJSON
	case "grpc":
		global.Warn("grpc is not a valid protocol for OTLP/HTTP, defaulting to http/protobuf")
	}
	return EncodingProtobuf
}

func cleanPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "." {
		return defaultTracesPath
	}
	if !path.IsAbs(value) {
		return "/" + value
	}
	return value
}

// WithEndpoint sets the target endpoint as a host and optional port.
func WithEndpoint(endpoint string) Option {
	return optionFunc(func(cfg config) config {
		cfg.endpoint = endpoint
		return cfg
	})
}

// WithEndpointURL sets the complete target endpoint URL.
func WithEndpointURL(rawURL string) Option {
	u, err := url.Parse(rawURL)
	if err != nil {
		global.Error(err, "otlptrace: parse endpoint url", "url", rawURL)
		return optionFunc(func(cfg config) config { return cfg })
	}
	return optionFunc(func(cfg config) config {
		cfg.endpoint = u.Host
		cfg.urlPath = u.Path
		if cfg.urlPath == "" {
			cfg.urlPath = "/"
		}
		cfg.insecure = isInsecureScheme(u.Scheme)
		return cfg
	})
}

// WithCompression sets the compression strategy.
func WithCompression(compression Compression) Option {
	return optionFunc(func(cfg config) config {
		cfg.compression = compression
		return cfg
	})
}

// WithEncoding sets the OTLP/HTTP payload encoding.
func WithEncoding(encoding Encoding) Option {
	return optionFunc(func(cfg config) config {
		cfg.encoding = encoding
		return cfg
	})
}

// WithURLPath sets the URL path used for exports.
func WithURLPath(urlPath string) Option {
	return optionFunc(func(cfg config) config {
		cfg.urlPath = urlPath
		return cfg
	})
}

// WithTLSClientConfig sets the TLS configuration used for requests.
func WithTLSClientConfig(tlsCfg *tls.Config) Option {
	return optionFunc(func(cfg config) config {
		if tlsCfg == nil {
			cfg.tlsCfg = nil
		} else {
			cfg.tlsCfg = tlsCfg.Clone()
		}
		return cfg
	})
}

// WithInsecure disables client transport security.
func WithInsecure() Option {
	return optionFunc(func(cfg config) config {
		cfg.insecure = true
		return cfg
	})
}

// WithHeaders sets additional HTTP headers sent with each request.
func WithHeaders(headers map[string]string) Option {
	return optionFunc(func(cfg config) config {
		cfg.headers = headers
		return cfg
	})
}

// WithTimeout sets the maximum duration of an export request.
func WithTimeout(timeout time.Duration) Option {
	return optionFunc(func(cfg config) config {
		cfg.timeout = timeout
		return cfg
	})
}

// WithMaxRequestSize sets the maximum uncompressed request size in bytes.
// A non-positive size disables the limit.
func WithMaxRequestSize(size int) Option {
	return optionFunc(func(cfg config) config {
		cfg.maxRequestSize = size
		return cfg
	})
}

// WithRetry sets the retry policy for transient failures.
func WithRetry(retryConfig RetryConfig) Option {
	return optionFunc(func(cfg config) config {
		cfg.retryCfg = retry.Config(retryConfig)
		return cfg
	})
}

// WithProxy sets the HTTP proxy resolver.
func WithProxy(proxy HTTPTransportProxyFunc) Option {
	return optionFunc(func(cfg config) config {
		cfg.proxy = proxy
		return cfg
	})
}

// WithHTTPClient sets the HTTP client used for exports.
//
// This option takes precedence over [WithProxy], [WithTimeout], and
// [WithTLSClientConfig], as well as the corresponding certificate and timeout
// environment variables. The timeout and all other fields of client are left
// intact.
//
// Passing a client whose transport is already OpenTelemetry-instrumented can
// cause duplicate instrumentation or recursive exports.
func WithHTTPClient(client *http.Client) Option {
	return optionFunc(func(cfg config) config {
		cfg.httpClient = client
		return cfg
	})
}

// String provides a useful debugging representation of the endpoint.
func (cfg config) String() string {
	return fmt.Sprintf("%s%s", cfg.endpoint, cfg.urlPath)
}
