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

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// Setup installs global meter and tracer providers that export metrics (including
// Go runtime metrics) and traces over OTLP/HTTP to OTEL_EXPORTER_OTLP_ENDPOINT.
// Without that variable it does nothing and all instrumentation stays a no-op.
// Call the returned function on shutdown to flush.
func Setup(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	noop := func(context.Context) error { return nil }
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		return noop, nil
	}

	// The pod name tells replicas apart.
	instance, _ := os.Hostname()
	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(serviceName), semconv.ServiceInstanceID(instance)),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return noop, err
	}

	metricExporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return noop, err
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
		sdkmetric.WithResource(res),
	)
	traceExporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return noop, err
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(traceExporter), sdktrace.WithResource(res))
	shutdown := func(ctx context.Context) error {
		return errors.Join(tp.Shutdown(ctx), mp.Shutdown(ctx))
	}

	otel.SetMeterProvider(mp)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	if err := runtime.Start(runtime.WithMeterProvider(mp)); err != nil {
		return shutdown, err
	}
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
