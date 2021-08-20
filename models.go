package sf

import "fmt"

type HealthCheck struct {
	Status string `json:"status"`
}

type Message struct {
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
}

// H is a shortcut for map[string]interface{}
type H map[string]interface{}

type R struct {
	Error error
	Result interface{}
}

func (r R) HasError () bool {
	return r.Error != nil
}

func Result(result interface{}, err error) R {
	return R{Error: err, Result: result}
}

func Err(err error) R {
	return R{Error: err}
}

func Errf(format string, args... interface{}) R {
	return R{Error: fmt.Errorf(format, args...)}
}