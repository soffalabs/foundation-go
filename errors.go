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

type UnauthorizedError struct {
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
