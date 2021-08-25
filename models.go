package sf

import "fmt"

type HealthCheck struct {
	Kind    string  `json:"kind,omitempty"`
	Name    string  `json:"name"`
	Status  string  `json:"status"`
	Message *string `json:"message,omitempty"`
}

type Message struct {
	Event   string                  `json:"event"`
	Payload interface{}             `json:"payload,omitempty"`
	Reply   func(interface{}) error `json:"-"`
}

// H is a shortcut for map[string]interface{}
type H map[string]interface{}

type R struct {
	Error  error
	Result interface{}
}

func (r R) HasError() bool {
	return r.Error != nil
}

func Result(result interface{}, err error) R {
	return R{Error: err, Result: result}
}

func Err(err error) R {
	return R{Error: err}
}

func Errf(format string, args ...interface{}) R {
	return R{Error: fmt.Errorf(format, args...)}
}

func (hc HealthCheck) get(err error) HealthCheck {
	if err != nil {
		hc.Status = "DOWN"
		msg := err.Error()
		hc.Message = &msg
	} else {
		hc.Status = "UP"
	}
	return hc
}
