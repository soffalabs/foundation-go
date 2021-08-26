package sf

type Credentials struct {
	Username string
	Password string
}

type Authentication struct {
	Username  string
	Guest bool
	Principal interface{}
}
