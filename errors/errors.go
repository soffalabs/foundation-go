package errors

import (
	e "emperror.dev/errors"
)

func Wrap(err error, message string) error {
	return e.Wrap(err, message)
}

func Wrapf(err error, message string, a ...interface{}) error {
	return e.Wrapf(err, message, a...)
}

func New(message string) error {
	return e.New(message)
}

func Message(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func Errorf(format string, a ...interface{}) error {
	return e.Errorf(format, a...)
}

func Anyf(err error, format string, a ...interface{}) error {
	if err != nil {
		return err
	}
	return e.Errorf(format, a...)
}

func Any(err error, message string) error {
	if err != nil {
		return err
	}
	return e.New(message)
}
