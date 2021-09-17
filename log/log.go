package log

import (
	"github.com/soffa-io/soffa-core-go/errors"
	"go.uber.org/zap"
	"strings"
)

var (
	Default     *Logger
	Application = ""
)

type Message struct {
	msg    string
	args   []interface{}
	error  error
	fields []interface{}
}

type Logger struct {
	log   *zap.SugaredLogger
	level string
}

func (m *Message) Err(err error) *Message {
	m.error = err
	return m
}

func (m *Message) F(key string, value interface{}) *Message {
	m.fields = append(m.fields, zap.Any(key, value))
	return m
}

/*
func (m *Message) Log() {
	l := logger.With(m.fields...)
	if m.error != nil {
		if m.args == nil || len(m.args) == 0 {
			l.Error(m.msg)
		} else {
			l.Errorf(m.msg, m.args...)
		}
	} else {
		if m.args == nil || len(m.args) == 0 {
			l.Info(m.msg)
		} else {
			l.Infof(m.msg, m.args...)
		}
	}
}


func (m *Message) Warn() {
	l := l.log.With(m.fields...)
	if m.args == nil || len(m.args) == 0 {
		l.Warn(m.msg)
	} else {
		l.Warnf(m.msg, m.args...)
	}
}

func (m *Message) Debug() {
	l := l.log.With(m.fields...)
	if m.args == nil || len(m.args) == 0 {
		l.Debug(m.msg)
	} else {
		l.Debugf(m.msg, m.args...)
	}
}


*/

func (l *Logger) M(msg string, args ...interface{}) *Message {
	return &Message{msg: msg, args: args, fields: []interface{}{}}
}
func (l *Logger) With(fields ...interface{}) *Logger {
	return &Logger{log: l.log.With(fields...), level: l.level}
}

func (l *Logger) Infof(format string, args ...interface{}) {
	if args == nil || len(args) == 0 {
		l.log.Info(format)
	} else {
		l.log.Infof(format, args...)
	}
}

func (l *Logger) Info(args ...interface{}) {
	l.log.Info(args...)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log.Debugf(format, args...)
}

func (l *Logger) Debug(args ...interface{}) {
	l.log.Debug(args...)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log.Warnf(format, args...)
}

func (l *Logger) Warn(args ...interface{}) {
	l.log.Warn(args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log.Errorf(format, args...)
}
func (l *Logger) ErrorIf(err error, format string, args ...interface{}) {
	if err != nil {
		l.log.Errorf(format, args...)
		l.log.Error(err)
	}
}

func (l *Logger) Wrap(err error, message string) {
	l.log.Error(errors.Wrap(err, message))
}

func (l *Logger) Wrapf(err error, message string, args ...interface{}) {
	l.log.Error(errors.Wrapf(err, message, args...))
}

func (l *Logger) Error(args ...interface{}) {
	l.log.Error(args...)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log.Fatalf(format, args...)
}

func (l *Logger) Fatal(args ...interface{}) {
	l.log.Fatal(args...)
}

func (l *Logger) FatalIf(err error) {
	if err != nil {
		l.log.Fatal(err)
	}
}

func (l *Logger) IsDebugEnabled() bool {
	return l.level == "DEBUG"
}

func (l *Logger) SetLevel(level string) {
	l.level = level
	if len(strings.TrimSpace(Application)) == 0 {
		l.Fatal("log.Application was not set")
	}
	l.log = l.log.With(zap.String("application", Application))
}

func init() {
	f, _ := zap.NewProduction()

	defer func(f *zap.Logger) {
		_ = f.Sync()
	}(f) // flushes buffer, if any
	Default = &Logger{log: f.Sugar(), level: "INFO"}
}

func (l *Logger) Capture(operation string, err error) error {
	if err != nil {
		l.log.Error("[capture] %s | %s", operation, err.Error())
	}
	return err
}
