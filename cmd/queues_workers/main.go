package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"rud-api/internal/config"
	"rud-api/internal/consts"
	"rud-api/internal/databases"
	"rud-api/internal/helpers"
	"rud-api/internal/queues"
	"rud-api/internal/repositories"
	"rud-api/internal/types"
	"strings"
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

	pgDB := &databases.PgDatabase{}
	pgDB.CreatePgDatabase(
		cfg.DB.PGDBRud.Host,
		cfg.DB.PGDBRud.Port,
		cfg.DB.PGDBRud.Username,
		cfg.DB.PGDBRud.Password,
		cfg.DB.PGDBRud.DB,
	)

	moduloReporteRepo := &repositories.ModuloReporteRepositorioPg{Pool: pgDB.Pool}

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

			procesarMensaje(msg, moduloReporteRepo)
		}(message)
	}

	wg.Wait()
	log.Println("Worker finalizado correctamente")
}

func procesarMensaje(message amqp.Delivery, repo *repositories.ModuloReporteRepositorioPg) {
	var bitacora strings.Builder
	bitacora.WriteString("Evento recibido para su procesamiento")

	messageBody := string(message.Body)
	log.Printf("Mensaje recibido: %s", messageBody)

	var evento types.ReporteSolicitadoEvent
	if errUnmarshal := json.Unmarshal(message.Body, &evento); errUnmarshal != nil {
		bitacora.WriteString("\nError parseando el evento: " + errUnmarshal.Error())
		log.Printf("Error parseando el evento: %v", errUnmarshal)
		message.Nack(false, false)
		return
	}

	bitacora.WriteString("\nEvento: reporte " + evento.ReporteID + " del usuario " + evento.UsuarioID)

	hasAccess, errHas := repo.HasModuloReporte(evento.UsuarioID, evento.ModuloReporteID)
	if errHas != nil {
		bitacora.WriteString("\nError validando permisos del usuario: " + errHas.Error())
		if errEstado := repo.ActualizarEstadoReporte(evento.ReporteID, consts.EstadoReporteFallo, bitacora.String()); errEstado != nil {
			log.Printf("Error actualizando estado del reporte: %v", errEstado)
		}
		message.Nack(false, true)
		return
	}

	if !hasAccess {
		bitacora.WriteString("\nNo se encontraron permisos para generar el reporte")
		if errEstado := repo.ActualizarEstadoReporte(evento.ReporteID, consts.EstadoReporteNoAutorizado, bitacora.String()); errEstado != nil {
			log.Printf("Error actualizando estado del reporte: %v", errEstado)
		}
		message.Ack(false)
		return
	}

	bitacora.WriteString("\nPermisos validados correctamente, iniciando construccion del reporte")
	if errEstado := repo.ActualizarEstadoReporte(evento.ReporteID, consts.EstadoReporteProcesando, bitacora.String()); errEstado != nil {
		log.Printf("Error actualizando estado del reporte: %v", errEstado)
	}

	time.Sleep(20 * time.Second)

	bitacora.WriteString("\nReporte construido correctamente")
	if errEstado := repo.ActualizarEstadoReporte(evento.ReporteID, consts.EstadoReporteCompletado, bitacora.String()); errEstado != nil {
		log.Printf("Error actualizando estado del reporte: %v", errEstado)
	}

	if errAck := message.Ack(false); errAck != nil {
		log.Printf("Error confirmando el mensaje: %v", errAck)
		if errNack := message.Nack(false, true); errNack != nil {
			log.Printf("Error devolviendo el mensaje a la cola: %v", errNack)
		}
		return
	}

	fmt.Println("Process alredy")
}
