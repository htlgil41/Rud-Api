package queues

import (
	"fmt"
	"rud-api/internal/config"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitQueue struct {
	Conn *amqp.Connection
	Ch   *amqp.Channel
}

func (r *RabbitQueue) ConnectRabbit(cfg config.RabbitConfig) error {
	url := fmt.Sprintf("amqp://%s:%s@%s:%d/", cfg.Username, cfg.Password, cfg.Host, cfg.Port)

	conn, errConn := amqp.Dial(url)
	if errConn != nil {
		return errConn
	}
	r.Conn = conn

	ch, errCh := conn.Channel()
	if errCh != nil {
		r.Conn.Close()
		return errCh
	}
	r.Ch = ch

	errExchange := ch.ExchangeDeclare(
		cfg.Exchange,
		cfg.ExchangeType,
		true,
		false,
		false,
		false,
		nil,
	)
	if errExchange != nil {
		r.Conn.Close()
		return errExchange
	}

	return nil
}

func (r *RabbitQueue) CloseRabbit() {
	if r.Ch != nil {
		r.Ch.Close()
	}
	if r.Conn != nil {
		r.Conn.Close()
	}
}
