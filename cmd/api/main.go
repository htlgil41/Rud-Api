package main

import (
	"fmt"
	"log"
	"rud-api/internal/config"
	ctx "rud-api/internal/context"
	"rud-api/internal/databases"
	"rud-api/internal/libs"
	"rud-api/internal/repositories"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
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
	moduloReporteRepo := &repositories.ModuloReporteRepositorioPg{Pool: pgDB.Pool}

	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.ES256, Key: []byte(cfg.JWT.Secret)},
		(&jose.SignerOptions{}).WithType("JWT"),
	)
	if err != nil {
		log.Fatal("Error creating JWT signer:", err)
	}

	joseToken := &libs.JoseManagerToken{
		Signer:            signer,
		TokenSecretAccess: cfg.JWT.Secret,
		ClaisnAccess: jwt.Claims{
			Expiry: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}

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
		auth.POST("/login", ctx.LoginHandler(usuarioRepo, joseToken))
	}

	protected := router.Group("/api")
	protected.Use(ctx.AuthMiddleware(joseToken))
	{
		protected.POST("/usuarios", ctx.RequireModulo(usuarioRepo, "GESTIONAR_USUARIOS"), ctx.RegisterUsuarioHandler(usuarioRepo))
		protected.GET("/mis-modulos", ctx.GetMisModulosHandler(usuarioRepo))
		protected.GET("/reportes", ctx.GetMisReportesHandler(moduloReporteRepo))
		protected.POST("/reportes/solicitar", ctx.SolicitarReporteHandler(moduloReporteRepo))
	}

	serverAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("Starting Rud-Api server on %s\n", serverAddr)

	if err := router.Run(serverAddr); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
