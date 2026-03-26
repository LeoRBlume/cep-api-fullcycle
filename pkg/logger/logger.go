package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"
)

// levelFatal é um nível customizado acima de Error para logs fatais.
const levelFatal = slog.Level(12)

// Logger encapsula o handler slog e a configuração de serviço.
type Logger struct {
	handler     slog.Handler
	serviceName string
	level       Level
	exitFunc    func(int)
}

var defaultLogger *Logger

func init() {
	defaultLogger = newLogger(Config{ServiceName: "app", Level: LevelInfo}, os.Stdout)
}

// Setup inicializa o logger global com a configuração fornecida.
func Setup(cfg Config) {
	defaultLogger = newLogger(cfg, os.Stdout)
}

func newLogger(cfg Config, w io.Writer) *Logger {
	if cfg.ServiceName == "" {
		cfg.ServiceName = "app"
	}
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: toSlogLevel(cfg.Level),
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.TimeKey:
				a.Key = "timestamp"
				if t, ok := a.Value.Any().(time.Time); ok {
					a.Value = slog.StringValue(t.Format(time.RFC3339))
				}
			case slog.MessageKey:
				a.Key = "message"
			case slog.LevelKey:
				if level, ok := a.Value.Any().(slog.Level); ok && level == levelFatal {
					a.Value = slog.StringValue("FATAL")
				}
			}
			return a
		},
	})
	return &Logger{
		handler:     h,
		serviceName: cfg.ServiceName,
		level:       cfg.Level,
		exitFunc:    os.Exit,
	}
}

func (l *Logger) log(ctx context.Context, level slog.Level, src, msg string, err error) {
	if !l.handler.Enabled(ctx, level) {
		return
	}

	traceID, userID := extractFromContext(ctx)

	attrs := []slog.Attr{
		slog.String("service", l.serviceName),
		slog.String("src", src),
	}
	if traceID != "" {
		attrs = append(attrs, slog.String("trace_id", traceID))
	}
	if userID != "" {
		attrs = append(attrs, slog.String("user_id", userID))
	}
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}

	r := slog.NewRecord(time.Now(), level, msg, 0)
	r.AddAttrs(attrs...)
	_ = l.handler.Handle(ctx, r)
}

// --- Métodos da struct Logger ---

func (l *Logger) Debug(ctx context.Context, src, msg string) {
	l.log(ctx, slog.LevelDebug, src, msg, nil)
}

func (l *Logger) Info(ctx context.Context, src, msg string) {
	l.log(ctx, slog.LevelInfo, src, msg, nil)
}

func (l *Logger) Warn(ctx context.Context, src, msg string) {
	l.log(ctx, slog.LevelWarn, src, msg, nil)
}

func (l *Logger) Error(ctx context.Context, src, msg string, err error) {
	l.log(ctx, slog.LevelError, src, msg, err)
}

func (l *Logger) Fatal(ctx context.Context, src, msg string, err error) {
	l.log(ctx, levelFatal, src, msg, err)
	l.exitFunc(1)
}

func (l *Logger) Debugf(ctx context.Context, src, msg string, args ...any) {
	l.log(ctx, slog.LevelDebug, src, fmt.Sprintf(msg, args...), nil)
}

func (l *Logger) Infof(ctx context.Context, src, msg string, args ...any) {
	l.log(ctx, slog.LevelInfo, src, fmt.Sprintf(msg, args...), nil)
}

func (l *Logger) Warnf(ctx context.Context, src, msg string, args ...any) {
	l.log(ctx, slog.LevelWarn, src, fmt.Sprintf(msg, args...), nil)
}

func (l *Logger) Errorf(ctx context.Context, src, msg string, err error, args ...any) {
	l.log(ctx, slog.LevelError, src, fmt.Sprintf(msg, args...), err)
}

func (l *Logger) Fatalf(ctx context.Context, src, msg string, err error, args ...any) {
	l.log(ctx, levelFatal, src, fmt.Sprintf(msg, args...), err)
	l.exitFunc(1)
}

// --- Funções globais (delegam ao defaultLogger) ---

func Debug(ctx context.Context, src, msg string) {
	defaultLogger.Debug(ctx, src, msg)
}

func Info(ctx context.Context, src, msg string) {
	defaultLogger.Info(ctx, src, msg)
}

func Warn(ctx context.Context, src, msg string) {
	defaultLogger.Warn(ctx, src, msg)
}

func Error(ctx context.Context, src, msg string, err error) {
	defaultLogger.Error(ctx, src, msg, err)
}

func Fatal(ctx context.Context, src, msg string, err error) {
	defaultLogger.Fatal(ctx, src, msg, err)
}

func Debugf(ctx context.Context, src, msg string, args ...any) {
	defaultLogger.Debugf(ctx, src, msg, args...)
}

func Infof(ctx context.Context, src, msg string, args ...any) {
	defaultLogger.Infof(ctx, src, msg, args...)
}

func Warnf(ctx context.Context, src, msg string, args ...any) {
	defaultLogger.Warnf(ctx, src, msg, args...)
}

func Errorf(ctx context.Context, src, msg string, err error, args ...any) {
	defaultLogger.Errorf(ctx, src, msg, err, args...)
}

func Fatalf(ctx context.Context, src, msg string, err error, args ...any) {
	defaultLogger.Fatalf(ctx, src, msg, err, args...)
}
