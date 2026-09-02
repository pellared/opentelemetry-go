// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package transform

import (
	resourcepb "go.opentelemetry.io/proto/otlphttp/resource/v1"

	"go.opentelemetry.io/otel/sdk/resource"
)

// Resource transforms a Resource into an OTLP Resource.
func Resource(r *resource.Resource) *resourcepb.Resource {
	if r == nil {
		return nil
	}
	return &resourcepb.Resource{Attributes: ResourceAttributes(r)}
}
