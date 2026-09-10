package main

import (
	"context"
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
	taskqueues "rud-api/internal/task_queues"
	"rud-api/internal/types"
	"strings"
	"sync"
	"syscall"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Connection struct {
	IP       string
	Username string
	Password string
	Sucursal string
}

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

	pgAvisorForConnections := &databases.PgDatabase{}
	pgAvisorForConnections.CreatePgDatabase(
		"192.168.5.193",
		5432,
		"aaron",
		"Wh0shotcupid+411",
		"avisor",
	)
	defer pgAvisorForConnections.Pool.Close()
	if err := pgAvisorForConnections.Pool.Ping(context.Background()); err != nil {
		log.Printf("No se ha podido establecer conexion")
		return
	}
	connections, errConnections := pgAvisorForConnections.Pool.Query(context.Background(), "select ip, username, password, sucursal from connections")
	if errConnections != nil {
		log.Printf("No se ha podido recolectar la informacion de conexion")
		return
	}
	defer connections.Close()

	moduloReporteRepo := &repositories.ModuloReporteRepositorioPg{Pool: pgDB.Pool}
	analisisRepo := []*repositories.AnalisisVentasRepositorie{}
	for connections.Next() {
		var conn Connection
		err := connections.Scan(&conn.IP, &conn.Username, &conn.Password, &conn.Sucursal)
		if err != nil {
			log.Printf("Error al escanear la fila de conexion: %v", err)
			continue
		}

		ipOnly, _, _ := strings.Cut(conn.IP, "\\")
		mssqlDB := &databases.MssqlDatabase{}
		mssqlDB.CreateMssqlDatabase(fmt.Sprintf(
			"sqlserver://%s:%s@%s?database=%s&connection+timeout=30",
			conn.Username,
			conn.Password,
			ipOnly,
			"VAD10",
		))

		analisisRepo = append(analisisRepo, &repositories.AnalisisVentasRepositorie{
			Sucursal: conn.Sucursal,
			Db:       mssqlDB.Db,
		})
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

			var bitacora strings.Builder
			bitacora.WriteString("Evento recibido para su procesamiento")
			var evento types.ReporteSolicitadoEvent

			if errUnmarshal := json.Unmarshal(message.Body, &evento); errUnmarshal != nil {
				bitacora.WriteString("\nError parseando el evento: ")
				bitacora.WriteString(errUnmarshal.Error())
				log.Printf("Error parseando el evento: %v", errUnmarshal)

				message.Ack(false)
				return
			}
			bitacora.WriteString("\nEvento: reporte ")
			bitacora.WriteString(evento.ReporteID)
			bitacora.WriteString(" del usuario ")
			bitacora.WriteString(evento.UsuarioID)

			hasAccess, errHas := moduloReporteRepo.HasModuloReporte(evento.UsuarioID, evento.ModuloReporteID)
			if errHas != nil {
				bitacora.WriteString("\nError validando permisos del usuario: ")
				bitacora.WriteString(errHas.Error())
				if errEstado := moduloReporteRepo.ActualizarEstadoReporte(evento.ReporteID, consts.EstadoReporteFallo, bitacora.String()); errEstado != nil {
					log.Printf("Error actualizando estado del reporte: %v", errEstado)
				}
				message.Ack(false)
				return
			}
			if !hasAccess {
				bitacora.WriteString("\nNo se encontraron permisos para generar el reporte")
				if errEstado := moduloReporteRepo.ActualizarEstadoReporte(evento.ReporteID, consts.EstadoReporteNoAutorizado, bitacora.String()); errEstado != nil {
					log.Printf("Error actualizando estado del reporte: %v", errEstado)
				}
				message.Ack(false)
				return
			}

			bitacora.WriteString("\nPermisos validados correctamente, iniciando construccion del reporte")
			if errEstado := moduloReporteRepo.ActualizarEstadoReporte(evento.ReporteID, consts.EstadoReporteProcesando, bitacora.String()); errEstado != nil {
				log.Printf("Error actualizando estado del reporte: %v", errEstado)
			}

			reporte_infor, errReporteInfo := moduloReporteRepo.GetModuloReporteByID(evento.ModuloReporteID)
			if errReporteInfo != nil {
				log.Printf("No se ha encontrado el modulo de reporte")
				message.Ack(false)
				return
			}

			switch reporte_infor.Nombre {
			case "AN_DEPARTAMENTO":
				{
					log.Printf("Analisis de ventas departamento")
					taskqueues.AnalisisDepartamentoQueueTask(
						msg,
						reporte_infor.QueryPrepare,
						moduloReporteRepo,
						analisisRepo,
						evento,
					)
				}
			case "AN_GRUPO":
				{
					log.Printf("Analisis de ventas grupo")
					taskqueues.AnalisisGrupoQueueTask(
						msg,
						reporte_infor.QueryPrepare,
						moduloReporteRepo,
						analisisRepo,
						evento,
					)
				}
			case "AN_SUBGRUPO":
				{
					log.Printf("Analisis de ventas subgrupo")
					taskqueues.AnalisisSubGrupoQueueTask(
						msg,
						reporte_infor.QueryPrepare,
						moduloReporteRepo,
						analisisRepo,
						evento,
					)
				}
			default:
				{
					bitacora.WriteString("\nNo se ha podido construir debido a que la funcion no esta definida en el builder debe notificar a sistemas de esto")
					if errEstado := moduloReporteRepo.ActualizarEstadoReporte(evento.ReporteID, consts.EstadoReporteProcesando, bitacora.String()); errEstado != nil {
						log.Printf("Error actualizando estado del reporte: %v", errEstado)
					}
					log.Printf("No se ha definido la funcion para ese reporte\n")
					message.Ack(false)
				}
			}
		}(message)
	}

	wg.Wait()
	log.Println("Worker finalizado correctamente")
}
