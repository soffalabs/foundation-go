package rpc

import (
	"github.com/soffa-io/soffa-core-go/errors"
	"github.com/soffa-io/soffa-core-go/log"
	"strings"
)

type BinaryMessage struct {
	Channel string
	Data  []byte
	Reply func([]byte)
}

type BinaryMessageHandler = func(event BinaryMessage) error

type Result struct {
	Empty   bool
	Error   bool
	Message string
	Data    []byte
}

type Client  interface {
	Publish(subj string, data []byte) error
	Serve(op string, cb func(string, []byte) (interface{}, error))
	ServeAll(subjs []string, cb func(string, []byte) (interface{}, error))
	Request(subj string, payload interface{}, dest interface{}) error
	Subscribe(subject string, handler BinaryMessageHandler)
}

func NewClient(url string, name string) Client {
	if url == "local" {
		return NewFakeRpcClient(url, name)
	}else if strings.HasPrefix(url, "nats://") {
		return NewNatsClient(url, name)
	}
	log.Fatal(errors.Errorf("unsupported rpc url: %s", url))
	return nil
}