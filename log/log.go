package log

import (
	"github.com/soffa-io/soffa-core-go/errors"
	"go.uber.org/zap"
)

var (
	logger   *zap.SugaredLogger
	logLevel = "DEBUG"
)

func Infof(format string, args ...interface{}) {
	logger.Infof(format, args...)
}

func Info(args ...interface{}) {
	logger.Info(args...)
}

func Debugf(format string, args ...interface{}) {
	logger.Debugf(format, args...)
}

func Debug(args ...interface{}) {
	logger.Debug(args...)
}

func Warnf(format string, args ...interface{}) {
	logger.Warnf(format, args...)
}

func Warn(args ...interface{}) {
	logger.Warn(args...)
}

func Errorf(format string, args ...interface{}) {
	logger.Errorf(format, args...)
}
func ErrorIf(err error, format string, args ...interface{}) {
	if err != nil {
		logger.Errorf(format, args...)
		logger.Error(err)
	}
}

func Wrap(err error, message string) {
	logger.Error(errors.Wrap(err, message))
}
func Error(args ...interface{}) {
	logger.Error(args...)
}

func Fatalf(format string, args ...interface{}) {
	logger.Fatalf(format, args...)
}

func Fatal(args ...interface{}) {
	logger.Fatal(args...)
}

func FatalIf(err error) {
	if err != nil {
		logger.Fatal(err)
	}
}

func IsDebugEnabled() bool {
	return logLevel == "DEBUG"
}

func Init(level string) {
	logLevel = level
}

func init() {
	f, _ := zap.NewProduction()
	defer func(f *zap.Logger) {
		_ = f.Sync()
	}(f) // flushes buffer, if any
	logger = f.Sugar()
}

func CaptureSilent(operation string, err error) {
	_ = Capture(operation, err)
}

func Capture(operation string, err error) error {
	if err != nil {
		logger.Error("[capture] %s | %s", operation, err.Error())
	}
	return err
}
