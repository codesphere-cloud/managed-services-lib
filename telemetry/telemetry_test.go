// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package telemetry_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opentelemetry.io/otel"

	"github.com/codesphere-cloud/managed-services-lib/telemetry"
)

func TestTelemetry(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Telemetry Suite")
}

var _ = Describe("Setup", func() {
	var paths chan string

	BeforeEach(func() {
		prevMeter, prevTracer := otel.GetMeterProvider(), otel.GetTracerProvider()
		DeferCleanup(func() {
			otel.SetMeterProvider(prevMeter)
			otel.SetTracerProvider(prevTracer)
		})
		for _, k := range []string{"OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "OTEL_TRACES_EXPORTER", "OTEL_METRICS_EXPORTER", "OTEL_SDK_DISABLED"} {
			GinkgoT().Setenv(k, "")
		}
	})

	// startCollector records the path of every OTLP export it receives.
	startCollector := func() {
		paths = make(chan string, 100)
		collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			paths <- r.URL.Path
			w.WriteHeader(http.StatusOK)
		}))
		DeferCleanup(collector.Close)
		GinkgoT().Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", collector.URL)
	}

	received := func() []string {
		var out []string
		for {
			select {
			case p := <-paths:
				out = append(out, p)
			default:
				return out
			}
		}
	}

	It("does nothing without an endpoint", func() {
		prev := otel.GetMeterProvider()

		shutdown, err := telemetry.Setup(context.Background(), "test")
		Expect(err).NotTo(HaveOccurred())
		Expect(shutdown(context.Background())).To(Succeed())
		Expect(otel.GetMeterProvider()).To(BeIdenticalTo(prev))
	})

	It("exports metrics but not traces by default", func() {
		startCollector()

		shutdown, err := telemetry.Setup(context.Background(), "test")
		Expect(err).NotTo(HaveOccurred())
		counter, err := otel.Meter("test").Int64Counter("test.events")
		Expect(err).NotTo(HaveOccurred())
		counter.Add(context.Background(), 1)
		_, span := otel.Tracer("test").Start(context.Background(), "op")
		span.End()

		Expect(shutdown(context.Background())).To(Succeed())
		got := received()
		Expect(got).To(ContainElement("/v1/metrics"))
		Expect(got).NotTo(ContainElement("/v1/traces"))
	})

	It("exports traces with OTEL_TRACES_EXPORTER=otlp", func() {
		startCollector()
		GinkgoT().Setenv("OTEL_TRACES_EXPORTER", "otlp")

		shutdown, err := telemetry.Setup(context.Background(), "test")
		Expect(err).NotTo(HaveOccurred())
		_, span := otel.Tracer("test").Start(context.Background(), "op")
		span.End()

		Expect(shutdown(context.Background())).To(Succeed())
		Expect(received()).To(ContainElement("/v1/traces"))
	})
})
