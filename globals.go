package sf

type Credentials struct {
	Username string
	Password string
}

type Authentication struct {
	Username  string
	Principal interface{}
}
