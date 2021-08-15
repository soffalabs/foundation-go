package soffa

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"github.com/tidwall/gjson"
	"github.com/wagslane/go-rabbitmq"
)

type RabbitMQPublisher struct {
	MessagePublisher
	publisher rabbitmq.Publisher
}

func (r RabbitMQPublisher) Send(channel string, message Message) error {
	log.Debug("[rabbitmq] sending message to %s", channel)
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return r.publisher.Publish(
		data,
		[]string{"default"},
		rabbitmq.WithPublishOptionsContentType("application/json"),
		rabbitmq.WithPublishOptionsMandatory,
		rabbitmq.WithPublishOptionsPersistentDelivery,
		rabbitmq.WithPublishOptionsExchange(channel),
	)
}

func CreateMessagePublisher(url string) MessagePublisher {
	if url == FakeAmqpurl {
		log.Info("Using FakeMessagePublisherImpl")
		return FakeMessagePublisherImpl{}
	}
	publisher, _, err := rabbitmq.NewPublisher(url, amqp.Config{})
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Connected to RabbitMQ: %s", url)
	return RabbitMQPublisher{
		publisher: publisher,
	}
}

func CreateMessageListener(url string, channel string, handler MessageHandler) {

	if url == FakeAmqpurl {
		log.Info("Skipping message listener...")
		return
	}

	consumer, err := rabbitmq.NewConsumer(url, amqp.Config{})
	if err != nil {
		log.Fatal(err)
	}

	err = consumer.StartConsuming(
		func(d rabbitmq.Delivery) bool {
			payload := string(d.Body)
			event := gjson.Get(payload, "event")
			if event.Exists() {
				var message Message
				if err := json.Unmarshal(d.Body, &message); err != nil {
					log.Warnf("Invalid payload received (decoding failed)")
					return true
				}
				return handler.HandleMessage(message)
			} else {
				log.Warnf("Invalid payload received (missing  event)")
				return true
			}
		},
		channel,
		[]string{"default"},
		rabbitmq.WithConsumeOptionsConcurrency(10),
		rabbitmq.WithConsumeOptionsQueueDurable,
		rabbitmq.WithConsumeOptionsQuorum,
		rabbitmq.WithConsumeOptionsBindingExchangeName(channel),
		rabbitmq.WithConsumeOptionsBindingExchangeKind("topic"),
		rabbitmq.WithConsumeOptionsBindingExchangeDurable,
	)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Info("RabbitMQ listener started.")
	}

}
