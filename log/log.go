package log

import (
	"github.com/sirupsen/logrus"
	"os"
)

func Infof(format string, args ...interface{}) {
	logrus.Infof(format, args...)
}

func Info(args ...interface{}) {
	logrus.Info(args...)
}

func Debugf(format string, args ...interface{}) {
	logrus.Debugf(format, args...)
}

func Debug(args ...interface{}) {
	logrus.Debug(args...)
}

func Warnf(format string, args ...interface{}) {
	logrus.Warnf(format, args...)
}

func Warn(args ...interface{}) {
	logrus.Warn(args...)
}

func Errorf(format string, args ...interface{}) {
	logrus.Errorf(format, args...)
}

func Error(args ...interface{}) {
	logrus.Error(args...)
}

func Fatalf(format string, args ...interface{}) {
	logrus.Fatalf(format, args...)
}

func Fatal(args ...interface{}) {
	logrus.Fatal(args...)
}

func FatalErr(err error) {
	if err != nil {
		logrus.Fatal(err)
	}
}

func IsDebugEnabled() bool {
	return logrus.IsLevelEnabled(logrus.DebugLevel)
}

func init() {
	logrus.SetOutput(os.Stdout)
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "TRACE" {
		logrus.SetLevel(logrus.TraceLevel)
	} else if logLevel == "DEBUG" {
		logrus.SetLevel(logrus.DebugLevel)
	} else if logLevel == "WARN" {
		logrus.SetLevel(logrus.WarnLevel)
	} else if logLevel == "ERROR" {
		logrus.SetLevel(logrus.ErrorLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
}
