package rpc

import (
	"github.com/soffa-io/soffa-core-go/errors"
	"github.com/soffa-io/soffa-core-go/h"
	"github.com/soffa-io/soffa-core-go/log"
)



type FakeRpcClient struct {
	Client
	subjects map[string]func([]byte) ([]byte, error)
}

func (n *FakeRpcClient) getFn(subj string) func([]byte) ([]byte, error) {
	for n, fn := range n.subjects {
		if subj == n || n == "*" {
			return fn
		}
	}
	return nil
}

//goland:noinspection GoDeferInLoop
func (n *FakeRpcClient) Publish(subj string, data []byte) error {
	fn := n.getFn(subj)
	if fn != nil {
		defer func() {
			_, _ = fn(data)
		}()
		return nil
	}
	return errors.Errorf("subject not found: %s", subj)
}

func (n *FakeRpcClient) Serve(op string, cb func(string, []byte) (interface{}, error)) {
	n.Subscribe(op, func(msg BinaryMessage) error {
		res, err := cb(op, msg.Data)
		if err != nil {
			msg.Reply(nil)
			return err
		} else {
			bytes, err := h.GetBytes(res)
			if err != nil {
				msg.Reply(nil)
				return err
			}
			msg.Reply(bytes)
			return nil
		}
	})
}

func (n *FakeRpcClient) ServeAll(subjs []string, cb func(string, []byte) (interface{}, error)) {
	for _, sub := range subjs {
		n.Serve(sub, cb)
	}
}

func (n *FakeRpcClient) Request(subj string, payload interface{}, dest interface{}) error {
	// bytes, err := prepareMessage(event, payload)
	bytes, err := h.GetBytes(payload)
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

func (n *FakeRpcClient) Subscribe(subject string, handler BinaryMessageHandler) {
	n.subjects[subject] = func(data []byte) ([]byte, error) {
		var reply []byte
		bmsg := BinaryMessage{
			Channel: subject,
			Data: data,
			Reply: func(data []byte) {
				reply = data
			},
		}
		err := handler(bmsg)
		log.ErrorIf(err, "message handling failed [%s] -- %v", subject, err)
		if err != nil {
			return nil, err
		}
		if h.IsNil(reply) {
			return nil, nil
		}
		return reply, nil
	}
}

func NewFakeRpcClient(_ string, name string) *FakeRpcClient {
	log.Infof("[fakerpc] %s is now ready", name)
	return &FakeRpcClient{
		subjects: map[string]func([]byte) ([]byte, error){},
	}
}
