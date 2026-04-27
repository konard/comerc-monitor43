package logging

import (
	"context"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

// Logger обёртка над zerolog для удобного использования.
type Logger struct {
	logger *zerolog.Logger
}

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
}

// New создаёт новый logger.
func New(level, environment string) *Logger {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	var logger zerolog.Logger
	if environment == "development" {
		logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout}).Level(lvl)
	} else {
		logger = zerolog.New(os.Stdout).With().Timestamp().Logger().Level(lvl)
	}

	return &Logger{
		logger: &logger,
	}
}

// Info логирует информационное сообщение.
func (l *Logger) Info() *zerolog.Event {
	return l.logger.Info()
}

// Warn логирует предупреждение.
func (l *Logger) Warn() *zerolog.Event {
	return l.logger.Warn()
}

// Error логирует ошибку.
func (l *Logger) Error() *zerolog.Event {
	return l.logger.Error()
}

// Debug логирует отладочное сообщение.
func (l *Logger) Debug() *zerolog.Event {
	return l.logger.Debug()
}

// Fatal логирует фатальную ошибку и завершает программу.
func (l *Logger) Fatal() *zerolog.Event {
	return l.logger.Fatal()
}

// WithFields добавляет поля к логу.
func (l *Logger) WithFields(fields map[string]any) *zerolog.Logger {
	ctx := l.logger.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	logger := ctx.Logger()
	return &logger
}

// Warnf логирует предупреждение с форматированием.
func (l *Logger) Warnf(format string, args ...any) {
	l.logger.Warn().Msgf(format, args...)
}

// Errorf логирует ошибку с форматированием.
func (l *Logger) Errorf(format string, args ...any) {
	l.logger.Error().Msgf(format, args...)
}

// Infof логирует информационное сообщение с форматированием.
func (l *Logger) Infof(format string, args ...any) {
	l.logger.Info().Msgf(format, args...)
}

// ZLogger возвращает underlying zerolog.Logger.
func (l *Logger) ZLogger() *zerolog.Logger {
	return l.logger
}

// UnaryServerInterceptor возвращает gRPC interceptor для логирования.
func UnaryServerInterceptor(logger *Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := logger.logger.With().Str("method", info.FullMethod).Logger()

		start.Info().Interface("request", req).Msg("handling request")

		resp, err := handler(ctx, req)

		if err != nil {
			start.Error().Err(err).Msg("request failed")
		} else {
			start.Info().Interface("response", resp).Msg("request completed")
		}

		return resp, err
	}
}

// ContextWithLogger добавляет logger в context.
func ContextWithLogger(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// FromContext извлекает logger из context.
func FromContext(ctx context.Context) (*Logger, bool) {
	logger, ok := ctx.Value(loggerKey{}).(*Logger)
	return logger, ok
}

type loggerKey struct{}
