package soffa


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

func NewFunctionalError(message string, code string) error {
	return FunctionalError{
		Code:    code,
		Message: message,
	}
}

func NewTechnicalError(message string, code string) error {
	return FunctionalError{
		Code:    code,
		Message: message,
	}
}
