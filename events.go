package sf

import (
	"encoding/json"
	"fmt"
	evbus "github.com/asaskevich/EventBus"
	"github.com/soffa-io/soffa-core-go/log"
	"github.com/streadway/amqp"
	"github.com/wagslane/go-rabbitmq"
	"strings"
)

type Subscription struct {
	Topic     string
	Broadcast bool
	Handler   MessageHandler
}

type MessageBroker interface {
	Ping() error
	Send(topic string, event string, payload interface{}) error
	Subscribe(topic string, broadcast bool, handler MessageHandler)
	SubscribeAll(sub Subscription) error
	Unsubscribe(topic string, handler MessageHandler) error
}

func ConnectToBroker(url string) (MessageBroker, error) {
	if url == "local" {
		return &InternalMessageBroker{bus: evbus.New()}, nil
	} else if strings.HasPrefix(url, "amqp://") {
		publisher, _, err := rabbitmq.NewPublisher(url, amqp.Config{})
		if err != nil {
			return nil, err
		}
		return &RabbitMQ{url: url, publisher: publisher}, nil
	}
	return nil, fmt.Errorf("broker protocol not supported: %s", url)
}

// =========================================================================================================

type InternalMessageBroker struct {
	MessageBroker
	bus evbus.Bus
}

func (b *InternalMessageBroker) Send(channel string, event string, payload interface{}) error {
	data, err := prepareMessage(event, payload)
	if err != nil {
		return Capture("broker.message.encode", err)
	}
	b.bus.Publish(channel, data)
	log.Infof("[broker] event %s sent to %s", event, channel)
	return nil
}

func (b *InternalMessageBroker) Subscribe(topic string, _ bool, handler MessageHandler) {
	err := b.bus.Subscribe(topic, func(body []byte) {
		_ = handleBrokerMessage(body, handler)
	})
	log.FatalErr(Capture("internal.broker.subscribe", err))
}

func (b *InternalMessageBroker) Unsubscribe(topic string, handler MessageHandler) error {
	return Capture("internal.brokern.unsubscribe", b.bus.Unsubscribe(topic, handler))
}

func (b *InternalMessageBroker) Ping() error {
	return nil
}

// =========================================================================================================

type RabbitMQ struct {
	MessageBroker
	url       string
	publisher rabbitmq.Publisher
}

func (b *RabbitMQ) Ping() error {
	return b.Send("ping", "ping", "PING")
}

func (b *RabbitMQ) Send(channel string, event string, payload interface{}) error {
	log.Debug("[rabbitmq] sending message to %s", channel)
	data, err := prepareMessage(event, payload)
	if err != nil {
		return Capture("broker.message.encode", err)
	}
	err = Capture("amqp.message.publish", b.publisher.Publish(
		data,
		[]string{"default"},
		rabbitmq.WithPublishOptionsContentType("text/plain"),
		rabbitmq.WithPublishOptionsMandatory,
		rabbitmq.WithPublishOptionsPersistentDelivery,
		rabbitmq.WithPublishOptionsExchange(channel),
	))
	if err == nil {
		log.Infof("[rabbitmq] event %s sent to %s", event, channel)
	}
	return err
}

func (b *RabbitMQ) Subscribe(topic string, broadcast bool, handler MessageHandler) {
	kind := "topic"
	if broadcast {
		kind = "fanout"
	}
	consumer, err := rabbitmq.NewConsumer(b.url, amqp.Config{})
	if err == nil {
		err = consumer.StartConsuming(
			func(d rabbitmq.Delivery) bool {
				return handleBrokerMessage(d.Body, handler)
			},
			topic,
			[]string{"default", ""},
			rabbitmq.WithConsumeOptionsConcurrency(10),
			rabbitmq.WithConsumeOptionsQueueDurable,
			rabbitmq.WithConsumeOptionsQuorum,
			rabbitmq.WithConsumeOptionsBindingExchangeName(topic),
			rabbitmq.WithConsumeOptionsBindingExchangeKind(kind),
			rabbitmq.WithConsumeOptionsBindingExchangeDurable,
		)
	}
	log.FatalErr(Capture("rabbitmq.connect", err))
}

func (b *RabbitMQ) Unsubscribe(_ string, _ MessageHandler) error {
	return nil
}

// var _ MessageBroker = (*InternalMessageBroker)(nil)

// =========================================================================================================

func prepareMessage(event string, payload interface{}) ([]byte, error) {
	data, err := EncodeMessage(payload)
	if err != nil {
		return nil, err
	}
	message := Message{
		Event:   event,
		Payload: data,
	}
	return EncodeMessage(message)
}

func handleBrokerMessage(body []byte, handler MessageHandler) bool {
	message, err := DecodeMessage(body)
	if Capture("amqp.message.decode", err) != nil {
		return true
	}
	err = Capture("amqp.handle.message", handler(*message))
	if err != nil {
		return false
	}
	return true
}

func EncodeMessage(message interface{}) ([]byte, error) {
	data, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func DecodeMessage(body []byte) (*Message, error) {
	if log.IsDebugEnabled() {
		log.Debug("[amqp.inbound] -- %s", body)
	}

	var message *Message
	if err := json.Unmarshal(body, &message); err != nil {
		log.Error("Invalid RabbitMQ payload received\n%v", body)
		return nil, err
	}
	if IsStrEmpty(message.Event) {
		return nil, fmt.Errorf("Invalid message payload received. No event provided\n%v", body)
	}
	return message, nil

}
