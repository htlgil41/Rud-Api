package context

import (
	"net/http"
	"rud-api/internal/libs"
	"rud-api/internal/repositories"
	"rud-api/internal/types"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type RegisterUsuarioBody struct {
	Username     string `json:"username" binding:"required"`
	Email        string `json:"email" binding:"required"`
	Password     string `json:"password" binding:"required"`
	Nombre       string `json:"nombre" binding:"required"`
	Departamento string `json:"departamento"`
}

type ActualizarPerfilBody struct {
	Username     string `json:"username" binding:"required"`
	Email        string `json:"email" binding:"required"`
	Departamento string `json:"departamento"`
}

type ResetearPasswordBody struct {
	Password string `json:"password" binding:"required"`
}

func RegisterUsuarioHandler(repo *repositories.UsuarioRepositoriePg) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body RegisterUsuarioBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al procesar contraseña"})
			return
		}

		newUser := types.UsuarioRespositorie{
			ID:           libs.GenerateUlid(),
			Usernme:      body.Username,
			Email:        body.Email,
			PasswordHash: string(hashedPassword),
			Nombre:       body.Nombre,
			Departamento: body.Departamento,
			Activo:       true,
		}

		created, err := repo.CreateUsuario(newUser)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"id":       created.ID,
			"username": created.Usernme,
			"email":    created.Email,
			"nombre":   created.Nombre,
		})
	}
}

func ActualizarPerfilUsuarioHandler(repo *repositories.UsuarioRepositoriePg) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no autenticado"})
			return
		}

		var body ActualizarPerfilBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		update := types.UsuarioRespositorie{
			ID:           userID,
			Usernme:      body.Username,
			Email:        body.Email,
			Departamento: body.Departamento,
		}

		updated, err := repo.UpdateDetailUsuario(update)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":           updated.ID,
			"username":     updated.Usernme,
			"email":        updated.Email,
			"departamento": updated.Departamento,
		})
	}
}

func ResetearPasswordHandler(repo *repositories.UsuarioRepositoriePg) gin.HandlerFunc {
	return func(c *gin.Context) {
		targetUserID := c.Param("usuario_id")
		if targetUserID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "usuario_id es requerido"})
			return
		}

		var body ResetearPasswordBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al procesar contraseña"})
			return
		}

		ok, err := repo.ChangePassowordUsuario(string(hashedPassword), targetUserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "No se pudo actualizar la contraseña"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Contraseña restablecida correctamente"})
	}
}
