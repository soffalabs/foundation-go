package soffa

type HTTPError struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

