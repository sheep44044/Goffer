package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// InitOTel 初始化 OpenTelemetry (包含 Trace 和 Metric)
func InitOTel(serviceName string, otelEndpoint string) (func(context.Context) error, error) {
	ctx := context.Background()

	// 1. 公共资源属性 (标识是哪个微服务发出的数据)
	res, err := resource.New(ctx, resource.WithAttributes(semconv.ServiceName(serviceName)))
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// ==========================================
	// 模块 A: 初始化 Trace (链路追踪 -> 发给 Jaeger)
	// ==========================================
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(otelEndpoint), otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp) // 设置全局 Tracer

	// ==========================================
	// 模块 B: 初始化 Metric (监控指标 -> 发给 Prometheus)
	// ==========================================
	metricExporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithEndpoint(otelEndpoint), otlpmetricgrpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}
	mp := metric.NewMeterProvider(
		// PeriodicReader 负责定时将内存中的指标打包发给 OTel Collector
		metric.WithReader(metric.NewPeriodicReader(metricExporter, metric.WithInterval(15*time.Second))),
		metric.WithResource(res),
	)
	otel.SetMeterProvider(mp) // 👈 设置全局 Meter

	// 3. 设置全局上下文传播器 (保证跨服务透传 TraceID)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	// 返回统一的 Shutdown 函数
	shutdown := func(c context.Context) error {
		err1 := tp.Shutdown(c)
		err2 := mp.Shutdown(c)
		if err1 != nil {
			return err1
		}
		return err2
	}

	return shutdown, nil
}
