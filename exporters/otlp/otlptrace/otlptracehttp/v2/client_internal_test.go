// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptracehttp

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetryAfterDuration(t *testing.T) {
	t.Run("seconds", func(t *testing.T) {
		err := newResponseError(http.Header{"Retry-After": {"10"}}, nil)
		retry, delay := evaluate(err)
		assert.True(t, retry)
		assert.Equal(t, 10*time.Second, delay)
	})

	t.Run("HTTP date", func(t *testing.T) {
		date := time.Now().Add(time.Hour).Format(http.TimeFormat)
		err := newResponseError(http.Header{"Retry-After": {date}}, nil)
		retry, delay := evaluate(err)
		assert.True(t, retry)
		assert.Greater(t, delay, 59*time.Minute)
		assert.LessOrEqual(t, delay, time.Hour)
	})

	t.Run("overflow", func(t *testing.T) {
		err := newResponseError(http.Header{"Retry-After": {"9223372036854775807"}}, nil)
		retry, delay := evaluate(err)
		assert.True(t, retry)
		assert.Equal(t, time.Duration(1<<63-1), delay)
	})
}
