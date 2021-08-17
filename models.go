package soffa

type HealthCheck struct {
	Status string `json:"status"`
}

type Message struct {
	Context interface{}
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
}

// H is a shortcut for map[string]interface{}
type H map[string]interface{}
