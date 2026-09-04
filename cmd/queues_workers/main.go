package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"rud-api/internal/config"
	"rud-api/internal/helpers"
	"rud-api/internal/queues"
	"sync"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const numeroWorkers = 5

func main() {
	cfg := config.LoadConfigWithVyper()
	if cfg == nil {
		log.Fatal("Error loading configuration")
	}

	rabbit := &queues.RabbitQueue{}
	if errRabbit := rabbit.ConnectRabbit(cfg.Rabbit); errRabbit != nil {
		log.Fatal("Error conectando a RabbitMQ:", errRabbit)
	}
	defer rabbit.CloseRabbit()

	if errNewQueue := helpers.CreateQueue(rabbit.Ch, cfg.Rabbit.Consumer); errNewQueue != nil {
		log.Fatal("Error creando la cola:", errNewQueue)
	}

	if errBind := helpers.BindQueueToExchange(
		rabbit.Ch,
		cfg.Rabbit.Consumer,
		cfg.Rabbit.Exchange,
		cfg.Reportes.RoutingKey,
	); errBind != nil {
		log.Fatal("Error vinculando la cola al exchange:", errBind)
	}

	fmt.Println("Cola creada y vinculada al exchange correctamente")

	messages, errConsume := rabbit.Ch.Consume(
		cfg.Rabbit.Consumer,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if errConsume != nil {
		log.Fatal("Error iniciando el consumidor:", errConsume)
	}

	fmt.Println("Worker escuchando mensajes de la cola:", cfg.Rabbit.Consumer)
	semaforo := make(chan struct{}, numeroWorkers)
	var wg sync.WaitGroup

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Señal recibida, cerrando consumidor...")
		rabbit.CloseRabbit()
	}()

	for message := range messages {
		semaforo <- struct{}{}

		wg.Add(1)
		go func(msg amqp.Delivery) {
			defer wg.Done()
			defer func() { <-semaforo }()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Panic procesando mensaje: %v", r)
					msg.Nack(false, true)
				}
			}()

			procesarMensaje(msg)
		}(message)
	}

	wg.Wait()
	log.Println("Worker finalizado correctamente")
}

func procesarMensaje(message amqp.Delivery) {
	messageBody := string(message.Body)
	log.Printf("Mensaje recibido: %s", messageBody)

	time.Sleep(20 * time.Second)

	if errAck := message.Ack(false); errAck != nil {
		log.Printf("Error confirmando el mensaje: %v", errAck)
		if errNack := message.Nack(false, true); errNack != nil {
			log.Printf("Error devolviendo el mensaje a la cola: %v", errNack)
		}
		return
	}

	fmt.Println("Process alredy")
}
