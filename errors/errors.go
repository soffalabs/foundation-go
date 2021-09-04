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

func IsTechnicalErr(err error) bool {
	return Is(err, ErrTechnical{})
}

func IsFunctionalErr(err error) bool {
	return Is(err, ErrFunctional{})
}

func Is(err error, target error) bool {
	return e.Is(err, target)
}

func Unwrap(err error) error {
	res := e.Unwrap(err)
	if res == nil {
		return err
	}
	return res
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

func RaiseFunctional(msg string) {
	Raise(NewFunctionalError0(msg))
}

func RaiseErrNotFound(msg string) {
	Raise(NewFunctionalError(ErrNotFoundCode, msg))
}

func RaiseErrForbidden(msg string) {
	Raise(NewFunctionalError(ErrForbiddenCode, msg))
}

func RaiseValidationError(msg string) {
	Raise(NewFunctionalError("FVAL", msg))
}

func Raise(err ...error) {
	if err != nil {
		for _, r := range err {
			if r != nil {
				panic(r)
			}
		}
	}
}

func Raisef(err error, message string, args ...interface{}) {
	if err != nil {
		panic(Wrapf(err, message, args...))
	}
}

func RaiseNew(message string, args ...interface{}) {
	panic(Errorf(message, args...))
}

func Throwf(format string, a ...interface{}) {
	panic(Errorf(format, a...))
}

func GetAnyMessage(err error) *string {
	var message *string
	if err != nil {
		msg := err.Error()
		message = &msg
	}
	return message
}

func NewFunctionalError(code string, message string) error {
	return e.WithMessage(ErrFunctional{Code: code}, message)
}

func NewFunctionalError0(message string) error {
	return e.WithMessage(ErrFunctional{Code: "FERR"}, message)
}

func NewUnauthorizedError(message string) error {
	return e.WithMessage(ErrUnauthorized{}, message)
}

func NewTechnicalError(code string, message string) error {
	return e.WithMessage(ErrTechnical{Code: code}, message)
}

func NewTechnicalError0(message string) error {
	return e.WithMessage(ErrTechnical{Code: "TERR"}, message)
}

func AnyError(errors ...error) error {
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}
