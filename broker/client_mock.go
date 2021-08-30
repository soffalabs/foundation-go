package broker


import (
	"github.com/nats-io/nats.go"
	"github.com/soffa-io/soffa-core-go/errors"
	"github.com/soffa-io/soffa-core-go/h"
	"github.com/soffa-io/soffa-core-go/log"
)


type FakeRpcClient struct {
	Client
	id   string
	conn *nats.Conn
	subjects map[string]func(interface{}) (interface{}, error)
}

func (n *FakeRpcClient) Start()  {
}

func (n *FakeRpcClient) Ping() error {
	return nil
}

func (n *FakeRpcClient) getFn(subj string) func(interface{}) (interface{}, error) {
	for n, fn := range n.subjects {
		if subj == n || n == "*" {
			return fn
		}
	}
	return nil
}

//goland:noinspection GoDeferInLoop
func (n *FakeRpcClient) Publish(subj string, data interface{}) error {
	fn := n.getFn(subj)
	if fn != nil {
		defer func() {
			_, _ = fn(data)
		}()
		return nil
	}
	return errors.Errorf("subject not found: %s", subj)
}

func (n *FakeRpcClient) Request(subj string, data interface{}, dest interface{}) error {
	// bytes, err := prepareMessage(event, payload)
	bytes, err := h.GetBytes(data)
	if err != nil {
		return errors.Wrapf(err, "[fake.rpc] bytes encoding failed -- %v", subj, err)
	}

	fn := n.getFn(subj)
	if fn == nil {
		return errors.Errorf("subject not found: %s", subj)
	}

	result, err := fn(bytes)
	if err != nil {
		return errors.Wrapf(err, "[fake.rpc] error sending message to %s -- %v", subj, err)
	}

	log.Infof("[fake.rpc] message sent to to %s", subj)

	if result == nil {
		return nil
	}
	return h.DecodeBytes(result, dest)
}

func (n *FakeRpcClient) Subscribe(subj string, handler Handler) {
	n.subjects[subj] = func(data interface{}) (interface{}, error) {
		defer func() {
			if err := recover(); err != nil {
				log.Errorf( "message handling failed [%s] -- %s", subj, err.(error).Error())
			}
		}()
		bytes, err := h.GetBytes(data)
		errors.Raise(err)
		bmsg := Message{Data: bytes}
		response := handler(bmsg)
		return h.Nil(response), nil
	}
}

func NewMockClient(name string) *FakeRpcClient {
	log.Infof("[fakerpc] %s is now ready", name)
	return &FakeRpcClient{
		subjects: map[string]func(interface{}) (interface{}, error){},
	}
}