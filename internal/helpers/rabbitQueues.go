package helpers

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

func CreateQueue(ch *amqp.Channel, queueName string) error {
	_, errDeclare := ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if errDeclare != nil {
		return errDeclare
	}
	return nil
}

func BindQueueToExchange(ch *amqp.Channel, queueName string, exchange string, routingKey string) error {
	errBind := ch.QueueBind(
		queueName,
		routingKey,
		exchange,
		false,
		nil,
	)
	if errBind != nil {
		return errBind
	}
	return nil
}
