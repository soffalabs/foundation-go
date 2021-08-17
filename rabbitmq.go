package soffa

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"github.com/wagslane/go-rabbitmq"
)

type RabbitMQPublisher struct {
	MessagePublisher
	publisher rabbitmq.Publisher
}

func (r RabbitMQPublisher) Send(channel string, message Message) error {
	log.Debugf("[rabbitmq] sending message to %s", channel)
	data, err := EncodeMessage(message)
	if err != nil {
		return err
	}
	return r.publisher.Publish(
		data,
		[]string{"default"},
		rabbitmq.WithPublishOptionsContentType("text/plain"),
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
	if IsStrEmpty(url) {
		log.Fatalf("An empty amqurl was provided.")
	}
	log.Debugf("Connecting to RabbitMQ @ %s", url)
	publisher, _, err := rabbitmq.NewPublisher(url, amqp.Config{})
	if err != nil {
		if fallbackToFakePublisher {
			log.Info("Using FakeMessagePublisherImpl")
			return FakeMessagePublisherImpl{}
		}
		log.Fatal(err)
	}
	log.Infof("Connected to RabbitMQ: %s", url)
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

func EncodeMessage(message interface{}) ([]byte, error) {
	data, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func DecodeMessage(body []byte) (*Message, error) {
	sbody := string(body)
	if log.IsLevelEnabled(log.DebugLevel) {
		log.Debug("------- [amqp message received] -------")
		log.Debug(sbody)
		log.Debug("---------------------------------------")
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
			message, err := DecodeMessage(d.Body)
			if err != nil {
				return true
			}
			if err = handler.HandleMessage(*message); err != nil {
				return false
			}
			return true
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
