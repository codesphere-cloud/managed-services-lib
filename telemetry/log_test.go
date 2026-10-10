// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package telemetry_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/codesphere-cloud/managed-services-lib/telemetry"
)

var _ = Describe("LogHandler", func() {
	var (
		buf    *bytes.Buffer
		logger *slog.Logger
	)

	BeforeEach(func() {
		buf = &bytes.Buffer{}
		logger = slog.New(telemetry.LogHandler(slog.NewJSONHandler(buf, nil))).With("component", "test")
	})

	record := func() map[string]any {
		var out map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &out)).To(Succeed())
		return out
	}

	It("adds the trace and span ID of the active span", func() {
		ctx, span := sdktrace.NewTracerProvider().Tracer("test").Start(context.Background(), "op")
		defer span.End()

		logger.InfoContext(ctx, "hello")
		Expect(record()).To(And(
			HaveKeyWithValue("trace_id", span.SpanContext().TraceID().String()),
			HaveKeyWithValue("span_id", span.SpanContext().SpanID().String()),
			HaveKeyWithValue("component", "test"),
		))
	})

	It("adds nothing without a span", func() {
		logger.InfoContext(context.Background(), "hello")
		Expect(record()).NotTo(HaveKey("trace_id"))
	})
})
