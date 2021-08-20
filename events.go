package sf

import (
	"encoding/json"
	"fmt"
	evbus "github.com/asaskevich/EventBus"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"github.com/wagslane/go-rabbitmq"
	"strings"
)

type MessageBroker interface {
	Send(topic string, event string, payload interface{}) error
	Subscribe(topic string, broadcast bool, handler MessageHandler) error
	Unsubscribe(topic string, handler MessageHandler) error
}


func ConnectToBroker(url string) (MessageBroker, error) {
	if url == "local" {
		return &InternalMessageBroker{bus: evbus.New()}, nil
	}else if strings.HasPrefix( url, "amqp://") {
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
	message := Message{
		Event:   event,
		Payload: payload,
	}
	b.bus.Publish(channel, message)
	return nil
}

func (b *InternalMessageBroker) Subscribe(topic string, _ bool, handler MessageHandler) error {
	return Capture("internal.broker.subscribe", b.bus.Subscribe(topic, handler))
}

func (b *InternalMessageBroker) Unsubscribe(topic string, handler MessageHandler) error {
	return Capture("internal.brokern.unsubscribe", b.bus.Unsubscribe(topic, handler))
}

// =========================================================================================================


type RabbitMQ struct {
	MessageBroker
	url string
	publisher rabbitmq.Publisher
}

func (b *RabbitMQ) Send(channel string, event string, payload interface{}) error {
	message := Message{
		Event:   event,
		Payload: payload,
	}
	log.Debugf("[rabbitmq] sending message to %s", channel)
	data, err := EncodeMessage(message)
	if err != nil {
		return Capture("amqp.message.encode", err)
	}
	return Capture("amqp.message.publish", b.publisher.Publish(
		data,
		[]string{"default"},
		rabbitmq.WithPublishOptionsContentType("text/plain"),
		rabbitmq.WithPublishOptionsMandatory,
		rabbitmq.WithPublishOptionsPersistentDelivery,
		rabbitmq.WithPublishOptionsExchange(channel),
	))
}

func (b *RabbitMQ) Subscribe(topic string, broadcast bool, handler MessageHandler) error {
	kind := "topic"
	if broadcast {
		kind = "fanout"
	}
	consumer, err := rabbitmq.NewConsumer(b.url, amqp.Config{})
	if err != nil {
		return Capture("rabbitmq.connect", err)
	}
	return consumer.StartConsuming(
		func(d rabbitmq.Delivery) bool {
			message, err := DecodeMessage(d.Body)
			if Capture("amqp.message.decode", err) != nil {
				return true
			}
			if err = Capture("amqp.handle.message", handler(*message)); err != nil {
				return false
			}
			return true
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

func (b *RabbitMQ) Unsubscribe(_ string, _ MessageHandler) error {
	return nil
}

// var _ MessageBroker = (*InternalMessageBroker)(nil)

// =========================================================================================================

func EncodeMessage(message interface{}) ([]byte, error) {
	data, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func DecodeMessage(body []byte) (*Message, error) {
	if log.IsLevelEnabled(log.DebugLevel) {
		log.Debugf("[amqp.inbound] -- %s", body)
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
