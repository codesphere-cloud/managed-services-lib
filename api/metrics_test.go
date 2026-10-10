// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/codesphere-cloud/managed-services-lib/api"
	"github.com/codesphere-cloud/managed-services-lib/config"
	"github.com/codesphere-cloud/managed-services-lib/telemetry"
)

var _ = Describe("Server metrics", func() {
	var (
		reader  *sdkmetric.ManualReader
		handler http.Handler
	)

	BeforeEach(func() {
		prev := otel.GetMeterProvider()
		reader = sdkmetric.NewManualReader()
		otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
		DeferCleanup(func() { otel.SetMeterProvider(prev) })

		server, err := api.NewServer(&config.Config{Environment: "test"}, map[string]func(*gin.RouterGroup){
			"demo": func(g *gin.RouterGroup) {
				g.GET("/:id", func(c *gin.Context) { c.Status(http.StatusNoContent) })
			},
		})
		Expect(err).NotTo(HaveOccurred())
		handler = server.Handler()
	})

	serve := func(path string) {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, nil))
	}

	// routes returns the http.route of every recorded request duration.
	routes := func() []string {
		var rm metricdata.ResourceMetrics
		Expect(reader.Collect(context.Background(), &rm)).To(Succeed())
		var out []string
		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				if m.Name != "http.server.request.duration" {
					continue
				}
				for _, dp := range m.Data.(metricdata.Histogram[float64]).DataPoints {
					route, _ := dp.Attributes.Value(attribute.Key("http.route"))
					out = append(out, route.AsString())
				}
			}
		}
		return out
	}

	It("records request duration by route template", func() {
		serve("/api/v1/demo/abc")
		serve("/api/v1/demo/def")
		Expect(routes()).To(ConsistOf("/api/v1/demo/:id"))
	})

	It("skips health and readiness probes", func() {
		serve("/health")
		serve("/ready")
		Expect(routes()).To(BeEmpty())
	})
})

var _ = Describe("Server request logs", func() {
	It("carry the trace ID of the request", func() {
		prevTracer, prevLogger := otel.GetTracerProvider(), slog.Default()
		DeferCleanup(func() {
			otel.SetTracerProvider(prevTracer)
			slog.SetDefault(prevLogger)
		})
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
		buf := &bytes.Buffer{}
		slog.SetDefault(slog.New(telemetry.LogHandler(slog.NewJSONHandler(buf, nil))))

		server, err := api.NewServer(&config.Config{Environment: "test"}, nil)
		Expect(err).NotTo(HaveOccurred())
		server.Handler().ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/", nil))

		var line map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &line)).To(Succeed())
		Expect(line).To(HaveKeyWithValue("msg", "HTTP request"))
		Expect(line["trace_id"]).To(MatchRegexp("^[0-9a-f]{32}$"))
	})
})
