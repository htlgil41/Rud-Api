package main

import (
	"fmt"
	"log"
	"rud-api/internal/config"
	ctx "rud-api/internal/context"
	"rud-api/internal/databases"
	"rud-api/internal/repositories"

	"github.com/gin-gonic/gin"
)

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

	usuarioRepo := &repositories.UsuarioRepositoriePg{Pool: pgDB.Pool}

	if cfg.Server.Mode == "DEV" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.GET("/healthcheck", ctx.HealthCheckServer())
	auth := router.Group("/auth")
	{
		auth.POST("/register", ctx.RegisterHandler(usuarioRepo))
		auth.POST("/login", ctx.LoginHandler(usuarioRepo, cfg.JWT.Secret))
	}

	serverAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("Starting Rud-Api server on %s\n", serverAddr)

	if err := router.Run(serverAddr); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
