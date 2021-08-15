package soffa

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
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

	if url == FakeMessagePublisherUrl {
		log.Info("Using FakeMessagePublisherImpl")
		return FakeMessagePublisherImpl{}
	}

	publisher, _, err := rabbitmq.NewPublisher(url, amqp.Config{})
	if err != nil {
		log.Fatal(err)
	}
	return RabbitMQPublisher{
		publisher: publisher,
	}
}

/*

//goland:noinspection ALL
func setupRabbitMQ() {
	consumer, err := rabbitmq.NewConsumer(os.Getenv("AMQP_URL"), amqp.Config{})
	if err != nil {
		if core.DevMode {
			log.Warn("Unable to connect to RabbitMQ, using internal messaging")
			return
		} else {
			log.Fatal(err)
		}
	}
	err = consumer.StartConsuming(
		func(d rabbitmq.Delivery) bool {
			payload := string(d.Body)
			event := gjson.Get(payload, "event")
			if event.Exists() {
				log.Debugf("New event received: [%s]", event.String())
				payload := gjson.Get(payload, "payload")
				return handleMessage(event.String(), payload.Raw)
			} else {
				log.Warnf("Invalid payload received (missing  event)")
				return true
			}
		},
		core.AppName,
		[]string{"default"},
		rabbitmq.WithConsumeOptionsConcurrency(10),
		rabbitmq.WithConsumeOptionsQueueDurable,
		rabbitmq.WithConsumeOptionsQuorum,
	)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Info("RabbitMQ listener started.")
	}

}

*/
