package sf

import (
	"fmt"
	evbus "github.com/asaskevich/EventBus"
	"github.com/soffa-io/soffa-core-go/commons"
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

type MessageBroker struct {
	broker   messageBrokerPort
	queue    string
	exchange string
}

type messageBrokerPort interface {
	Ping() error
	Send(exchange string, routingKey string, event string, payload interface{}) error
	Broadcast(exchange string, event string, payload interface{}) error
	Listen(queueName string, exchange string, routingKeys []string, handler MessageHandler)
}

func (b MessageBroker) Broadcast(event string, payload interface{}) error {
	return b.broker.Send(fmt.Sprintf("%s.fanout", b.exchange), "", event, payload)
}

func (b MessageBroker) Send(routingKey string, event string, payload interface{}) error {
	return b.broker.Send(fmt.Sprintf("%s.topic", b.exchange), routingKey, event, payload)
}

func newMessageBroker(context *ApplicationContext, url string) (messageBrokerPort, error) {
	if url == "local" {
		return &InternalinternalMessageBroker{bus: evbus.New(), context: context}, nil
	} else if strings.HasPrefix(url, "amqp://") {
		publisher, _, err := rabbitmq.NewPublisher(url, amqp.Config{})
		if err != nil {
			return nil, err
		}
		return &RabbitMQ{url: url, publisher: publisher, context: context}, nil
	}
	return nil, fmt.Errorf("broker protocol not supported: %s", url)
}

// =========================================================================================================

type InternalinternalMessageBroker struct {
	messageBrokerPort
	bus evbus.Bus
	context *ApplicationContext
}

func (b *InternalinternalMessageBroker) Send(exchange string, routingKey string, event string, payload interface{}) error {
	data, err := prepareMessage(event, payload)
	if err != nil {
		return Capture("broker.message.encode", err)
	}
	b.bus.Publish(exchange, data)
	log.Infof("[broker] event %s sent to %s and %s", event, exchange, routingKey)
	return nil
}

func (b *InternalinternalMessageBroker) Listen(queueName string, exchange string, _ []string, handler MessageHandler) {
	err := b.bus.Subscribe(queueName, func(body []byte) {
		_ = handleBrokerMessage(b.context, body, handler)
	})
	log.FatalErr(Capture("internal.broker.subscribe", err))
	err = b.bus.Subscribe(fmt.Sprintf("%s.topic", exchange), func(body []byte) {
		_ = handleBrokerMessage(b.context, body, handler)
	})
	log.FatalErr(Capture("internal.broker.subscribe", err))
	err = b.bus.Subscribe(fmt.Sprintf("%s.fanout", exchange), func(body []byte) {
		_ = handleBrokerMessage(b.context, body, handler)
	})
	log.FatalErr(Capture("internal.broker.subscribe", err))
}

func (b *InternalinternalMessageBroker) Ping() error {
	return nil
}

// =========================================================================================================

type RabbitMQ struct {
	messageBrokerPort
	url       string
	publisher rabbitmq.Publisher
	context *ApplicationContext
}

func (b *RabbitMQ) Ping() error {
	conn, err := amqp.Dial(b.url)
	if err != nil {
		return err
	}
	defer func(conn *amqp.Connection) {
		_ = conn.Close()
	}(conn)
	return nil
}

func (b *RabbitMQ) Send(exchange string, routingKey string, event string, payload interface{}) error {
	data, err := prepareMessage(event, payload)
	if err != nil {
		return Capture("broker.message.encode", err)
	}
	err = Capture("amqp.message.publish", b.publisher.Publish(
		data,
		[]string{routingKey},
		rabbitmq.WithPublishOptionsContentType("text/plain"),
		rabbitmq.WithPublishOptionsMandatory,
		rabbitmq.WithPublishOptionsPersistentDelivery,
		rabbitmq.WithPublishOptionsExchange(exchange),
	))
	if err == nil {
		log.Infof("[rabbitmq] event %s sent to %s with routingKey  %s", event, exchange, routingKey)
	}
	return err
}

func (b *RabbitMQ) Listen(queueName string, exchange string, routingKeys []string, handler MessageHandler) {
	consumer, err := rabbitmq.NewConsumer(b.url, amqp.Config{})
	if err == nil {
		err = consumer.StartConsuming(
			func(d rabbitmq.Delivery) bool {
				return handleBrokerMessage(b.context, d.Body, handler)
			},
			queueName,
			routingKeys,
			rabbitmq.WithConsumeOptionsConcurrency(10),
			rabbitmq.WithConsumeOptionsQueueDurable,
			rabbitmq.WithConsumeOptionsQuorum,
			rabbitmq.WithConsumeOptionsBindingExchangeName(fmt.Sprintf("%s.topic", exchange)),
			rabbitmq.WithConsumeOptionsBindingExchangeKind("topic"),
			rabbitmq.WithConsumeOptionsBindingExchangeDurable,
		)
	}
	if err == nil {
		err = consumer.StartConsuming(
			func(d rabbitmq.Delivery) bool {
				return handleBrokerMessage(b.context, d.Body, handler)
			},
			queueName,
			[]string{""},
			rabbitmq.WithConsumeOptionsConcurrency(10),
			rabbitmq.WithConsumeOptionsQueueDurable,
			rabbitmq.WithConsumeOptionsQuorum,
			rabbitmq.WithConsumeOptionsBindingExchangeName(fmt.Sprintf("%s.fanout", exchange)),
			rabbitmq.WithConsumeOptionsBindingExchangeKind("fanout"),
			rabbitmq.WithConsumeOptionsBindingExchangeDurable,
		)
	}
	log.FatalErr(Capture("rabbitmq.connect", err))
}

// =========================================================================================================

func prepareMessage(event string, data interface{}) ([]byte, error) {
	message := Message{
		Event:   event,
		Payload: data,
	}
	return ToJson(message)
}

func handleBrokerMessage(context *ApplicationContext, body []byte, handler MessageHandler) bool {
	message, err := DecodeMessage(body)
	if Capture("amqp.message.decode", err) != nil {
		return true
	}
	log.Infof("[rabbitmq] event received to %s", message.Event)
	err = Capture("amqp.handle.message", handler(context, *message))
	if err != nil {
		return false
	}
	return true
}

func DecodeMessage(body []byte) (*Message, error) {
	if log.IsDebugEnabled() {
		log.Debugf("[message.inbound] -- %s", body)
	}

	var message *Message
	if err := FromJson(body, &message); err != nil {
		log.Errorf("Invalid message payload received\n%s", body)
		return nil, err
	}
	if commons.IsStrEmpty(message.Event) {
		return nil, fmt.Errorf("Invalid message payload received. No event provided\n%v", body)
	}
	return message, nil

}
