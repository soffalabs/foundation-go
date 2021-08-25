package sf

import "github.com/soffa-io/soffa-core-go/log"

type GenericError struct {
	Kind    string  `json:"string,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string  `json:"message,omitempty"`
}

type FunctionalError struct {
	error
	Code    string `json:"code"`
	Message string `json:"message"`
}

type TechnicalError struct {
	error
	Code    string `json:"code"`
	Message string `json:"message"`
}

type UnauthorizedError struct {
	error
	Code    string `json:"code"`
	Message string `json:"message"`
}

func CaptureSilent(operation string, err error)  {
	_ = Capture(operation, err)
}


func GetAnyMessage(err error) *string {
	var message *string
	if err != nil {
		msg := err.Error()
		message = &msg
	}
	return message
}

func Capture(operation string, err error) error {
	if err != nil {

		message := GenericError{}
		switch t := err.(type) {
		case TechnicalError:
			message.Kind = "Technical"
			message.Code = t.Code
			message.Message = t.Message
		case UnauthorizedError:
			message.Kind = "Unauthorized"
			message.Code = t.Code
			message.Message = t.Message
		case FunctionalError:
			message.Kind = "Functional"
			message.Code = t.Code
			message.Message = t.Message
		default:
			message.Kind = "Default"
			message.Message = err.Error()
		}

		log.Errorf("[capture]: %s | %s", operation, ToJsonStrSafe(message))
	}
	return err
}

func NewFunctionalError(message string, code string) error {
	return FunctionalError{
		Code:    code,
		Message: message,
	}
}

func NewFunctionalError0(message string) error {
	return FunctionalError{
		Code:    "FERR",
		Message: message,
	}
}
func NewUnauthorizedError(message string) error {
	return UnauthorizedError{
		Code:    "Unauthorized",
		Message: message,
	}
}

func NewTechnicalError(message string, code string) error {
	return FunctionalError{
		Code:    code,
		Message: message,
	}
}

func NewTechnicalError0(message string) error {
	return FunctionalError{
		Code:    "TERR",
		Message: message,
	}
}

func ThrowAny(err error) {
	if err != nil {
		panic(err)
	}
}

func AnyError(err1 error, err2 error) error {
	if err1 == nil {
		return err2
	}
	return err2
}

func Fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
