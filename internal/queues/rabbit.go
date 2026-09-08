package queues

import (
	"fmt"
	"log"
	"rud-api/internal/config"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitQueue struct {
	Conn *amqp.Connection
	Ch   *amqp.Channel
}

func (r *RabbitQueue) ConnectRabbit(cfg config.RabbitConfig) error {
	url := fmt.Sprintf("amqp://%s:%s@%s:%d/", cfg.Username, cfg.Password, cfg.Host, cfg.Port)

	conn, errConn := amqp.DialConfig(url, amqp.Config{
		Vhost:     "/",
		Heartbeat: 10 * time.Second,
		Dial:      amqp.DefaultDial(30 * time.Second),
		Recovery: &amqp.Recovery{
			ReconnectionConfig: &amqp.ReconnectionConfig{
				RetryInterval: 6 * time.Second,
				MaxRetryCount: 10,
			},
			TopologyRecoveryMode: amqp.TopologyRecoveryAllEnabled,
			OnTopologyEntityError: func(conn *amqp.Connection, e amqp.TopologyRecoveryEntity) bool {
				log.Printf("❌ Falló la recuperación de un elemento en el cluster!")
				log.Printf("Tipo de entidad: %s", e.EntityType)
				log.Printf("Nombre: %s", e.EntityName)
				log.Printf("Error real de RabbitMQ: %v", e.Err)

				return true
			},
		},
		Properties: amqp.Table{
			"connection_name": "consumidor_reportes",
			"version":         "1.0.0",
		},
	})
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
