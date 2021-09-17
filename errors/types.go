package errors

const (
	ErrNotFoundCode  = "F404"
	ErrForbiddenCode = "F403"
	ErrUnauthorizedCode = "F401"
)

type ErrFunctional struct {
	Code string `json:"code"`
}

func (e ErrFunctional) Error() string {
	return e.Code
}

type ErrTechnical struct {
	Code string `json:"code"`
}

func (e ErrTechnical) Error() string {
	return e.Code
}

type ErrUnauthorized struct {
}

func (e ErrUnauthorized) Error() string {
	return "Unauthorized"
}
