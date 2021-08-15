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

func CreateMessagePublisher(url string, fallbackToFakePublisher bool) MessagePublisher {
	if url == FakeAmqpurl {
		log.Info("Using FakeMessagePublisherImpl")
		return FakeMessagePublisherImpl{}
	}
	publisher, _, err := rabbitmq.NewPublisher(url, amqp.Config{})
	if err != nil {
		if fallbackToFakePublisher {
			log.Info("Using FakeMessagePublisherImpl")
			return FakeMessagePublisherImpl{}
		}
		log.Fatal(err)
	}
	log.Info("Connected to RabbitMQ: %s", url)
	return RabbitMQPublisher{
		publisher: publisher,
	}
}

func CreateBroadcastMessageListener(url string, channel string, fallbackToFakePublisher bool, handler MessageHandler) {
	createTopicMessageListener(url, channel, "fanout", fallbackToFakePublisher, handler)
}

func CreateTopicMessageListener(url string, channel string, fallbackToFakePublisher bool, handler MessageHandler) {
	createTopicMessageListener(url, channel, "topic", fallbackToFakePublisher, handler)
}

func createTopicMessageListener(url string, channel string, king string, fallbackToFakePublisher bool, handler MessageHandler) {

	log.Infof("Creating messageListener: %s", url)

	if url == FakeAmqpurl {
		log.Info("Skipping message listener...")
		return
	}

	consumer, err := rabbitmq.NewConsumer(url, amqp.Config{})
	if err != nil {
		if fallbackToFakePublisher {
			log.Info("Skipping message listener...")
			return
		}
		log.Fatal(err)
	}

	err = consumer.StartConsuming(
		func(d rabbitmq.Delivery) bool {
			body := string(d.Body)
			if log.IsLevelEnabled(log.DebugLevel) {
				log.Debug("------- [amqp message received] -------")
				log.Debug(body)
				log.Debug("---------------------------------------")
			}
			event := gjson.Get(body, "event")
			payload := gjson.Get(body, "payload")
			if event.Exists() {
				var message = Message {
					Event: event.String(),
					Payload: payload.String(),
				}
				if err := handler.HandleMessage(message); err != nil {
					return false
				}
				return true
			} else {
				log.Warnf("Invalid payload received (missing  event)")
				return true
			}
		},
		channel,
		[]string{"default", ""},
		rabbitmq.WithConsumeOptionsConcurrency(10),
		rabbitmq.WithConsumeOptionsQueueDurable,
		rabbitmq.WithConsumeOptionsQuorum,
		rabbitmq.WithConsumeOptionsBindingExchangeName(channel),
		rabbitmq.WithConsumeOptionsBindingExchangeKind(king),
		rabbitmq.WithConsumeOptionsBindingExchangeDurable,
	)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Info("RabbitMQ listener started.")
	}

}
