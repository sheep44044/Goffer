package logger

import (
	"context"
	"sync"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/kitex/pkg/klog"
	hertz_zap "github.com/hertz-contrib/obs-opentelemetry/logging/zap"
	kitex_zap "github.com/kitex-contrib/obs-opentelemetry/logging/zap"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	globalLogger *zap.Logger
	initOnce     sync.Once
)

// Init 初始化全局 zap.Logger 并重定向 klog / hlog。
// level 取 "debug" / "info" / "warn" / "error"，默认 "info"。
func Init(level string) {
	initOnce.Do(func() {
		globalLogger = buildLogger(level)

		// 将 Kitex 框架的 klog 重定向到官方 zap 实现（自动从 ctx 提取 OTel TraceID）
		klog.SetLogger(kitex_zap.NewLogger())

		// 将 Hertz 框架的 hlog 重定向到官方 zap 实现（同上）
		hlog.SetLogger(hertz_zap.NewLogger())
	})
}

func buildLogger(level string) *zap.Logger {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(parseLevel(level)),
		Development:      false,     // 生产中通常设为 false
		Encoding:         "console", // 生产中建议改为 "json"
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	// 核心：AddCallerSkip(1) 保证打印的是调用 logger 业务代码的行号，而不是当前文件
	logger, err := cfg.Build(zap.AddCallerSkip(1))
	if err != nil {
		fallback := zap.NewExample()
		fallback.Error("logger init failed, using fallback", zap.Error(err))
		return fallback
	}
	return logger
}

func parseLevel(s string) zapcore.Level {
	switch s {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// L 返回底层 *zap.Logger，供需要直接使用 zap API 的高级场景。
// 若尚未 Init，返回一个 fallback Development Logger。
func L() *zap.Logger {
	if globalLogger == nil {
		fallback, _ := zap.NewDevelopment()
		return fallback
	}
	return globalLogger
}

// ===================== OTel TraceID 提取 =====================
// traceField 从 context 中提取 OpenTelemetry TraceID。
// 若 context 中无有效 Span，返回 zap.Skip() 避免输出空字段。
func traceField(ctx context.Context) zap.Field {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return zap.String("trace_id", span.SpanContext().TraceID().String())
	}
	return zap.Skip() // 返回一个无操作的 Field，不会在日志中产生多余的 key
}

func Debug(msg string, fields ...zap.Field) { L().Debug(msg, fields...) }
func Info(msg string, fields ...zap.Field)  { L().Info(msg, fields...) }
func Warn(msg string, fields ...zap.Field)  { L().Warn(msg, fields...) }
func Error(msg string, fields ...zap.Field) { L().Error(msg, fields...) }

func DebugCtx(ctx context.Context, msg string, fields ...zap.Field) {
	L().Debug(msg, append(fields, traceField(ctx))...)
}
func InfoCtx(ctx context.Context, msg string, fields ...zap.Field) {
	L().Info(msg, append(fields, traceField(ctx))...)
}
func WarnCtx(ctx context.Context, msg string, fields ...zap.Field) {
	L().Warn(msg, append(fields, traceField(ctx))...)
}
func ErrorCtx(ctx context.Context, msg string, fields ...zap.Field) {
	L().Error(msg, append(fields, traceField(ctx))...)
}

// Sync 刷新缓冲区，应在 main 函数的 defer 中调用。
func Sync() {
	if globalLogger != nil {
		_ = globalLogger.Sync()
	}
}
