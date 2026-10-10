// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

// Package telemetry sets up OpenTelemetry for provider backends.
//
// The api server and the Kubernetes client are instrumented already; Setup decides
// where their telemetry goes. Backends add their own metrics with otel.Meter and wrap
// other HTTP clients with Transport.
package telemetry

import (
	"context"
	"errors"
	"net/http"
	"os"

	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// Setup installs global meter and tracer providers that export over OTLP to
// OTEL_EXPORTER_OTLP_ENDPOINT. Without an endpoint, or with OTEL_SDK_DISABLED=true,
// it does nothing and all instrumentation stays a no-op.
//
// Metrics (including Go runtime metrics) and traces are exported unless
// OTEL_METRICS_EXPORTER or OTEL_TRACES_EXPORTER is "none". The other standard OTEL_*
// variables apply (service name, resource attributes, protocol, export interval).
// Call the returned function on shutdown to flush.
func Setup(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	noop := func(context.Context) error { return nil }
	if os.Getenv("OTEL_SDK_DISABLED") == "true" ||
		(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" &&
			os.Getenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT") == "" &&
			os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") == "") {
		return noop, nil
	}

	// The pod name tells replicas apart. Later options win, so OTEL_SERVICE_NAME and
	// OTEL_RESOURCE_ATTRIBUTES override both.
	instance, _ := os.Hostname()
	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(serviceName), semconv.ServiceInstanceID(instance)),
		resource.WithTelemetrySDK(),
		resource.WithFromEnv(),
	)
	if err != nil {
		return noop, err
	}

	var shutdowns []func(context.Context) error
	shutdown := func(ctx context.Context) error {
		var errs []error
		for _, s := range shutdowns {
			errs = append(errs, s(ctx))
		}
		return errors.Join(errs...)
	}

	reader, err := autoexport.NewMetricReader(ctx)
	if err != nil {
		return noop, err
	}
	if !autoexport.IsNoneMetricReader(reader) {
		mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader), sdkmetric.WithResource(res))
		otel.SetMeterProvider(mp)
		shutdowns = append(shutdowns, mp.Shutdown)
		if err := runtime.Start(runtime.WithMeterProvider(mp)); err != nil {
			return shutdown, err
		}
	}

	exporter, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		return shutdown, err
	}
	if !autoexport.IsNoneSpanExporter(exporter) {
		tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter), sdktrace.WithResource(res))
		otel.SetTracerProvider(tp)
		shutdowns = append(shutdowns, tp.Shutdown)
	}
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return shutdown, nil
}

// Transport wraps base (http.DefaultTransport if nil) so that outgoing requests
// record http.client.* metrics and client spans.
func Transport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return otelhttp.NewTransport(base)
}
