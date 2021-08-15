package soffa

type MessagePublisher interface {
	Send(channel string, payload interface{}) error
}

type EntityManager interface {
	Create(model interface{}) error
	Transactional(callback func() error) error
	FindAll(dest interface{}, limit int) error
	FindBy(dest interface{}, where string, args ...interface{}) error
	ExistsBy(model interface{}, where string, args ...interface{}) (bool, error)
}
