package repositories

import (
	"errors"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitPublisherRepositorio struct {
	Ch         *amqp.Channel
	Exchange   string
	RoutingKey string
}

func (p *RabbitPublisherRepositorio) Publish(body []byte) error {
	if p.Ch == nil {
		return errors.New("canal de rabbitmq no disponible")
	}
	return p.Ch.Publish(
		p.Exchange,
		p.RoutingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}
