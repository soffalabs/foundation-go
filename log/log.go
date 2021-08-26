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
func ErrorIf(err error, format string, args ...interface{}) {
	if err != nil {
		logrus.Errorf(format, args...)
		logrus.Error(err)
	}
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

}
