package context

import (
	"net/http"
	"rud-api/internal/libs"
	"rud-api/internal/repositories"
	"rud-api/internal/types"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"golang.org/x/crypto/bcrypt"
)

type RegisterBody struct {
	Username     string `json:"username" binding:"required"`
	Email        string `json:"email" binding:"required"`
	Password     string `json:"password" binding:"required"`
	Nombre       string `json:"nombre" binding:"required"`
	Departamento string `json:"departamento"`
}

type LoginBody struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func RegisterHandler(repo *repositories.UsuarioRepositoriePg) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body RegisterBody
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

func LoginHandler(repo *repositories.UsuarioRepositoriePg, secretKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body LoginBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		user, err := repo.GetUsuarioByUsername(body.Username)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Credenciales inválidas"})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Credenciales inválidas"})
			return
		}

		signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.ES256, Key: []byte(secretKey)}, (&jose.SignerOptions{}).WithType("JWT"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al generar token"})
			return
		}

		cl := jwt.Claims{
			Expiry: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		}

		tokenStrinng := jwt.Signed(signer)
		token, err := tokenStrinng.Claims(cl).Claims(libs.PayloadCustom{
			IdUser:   user.ID,
			Username: user.Usernme,
		}).Serialize()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al generar token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token":    token,
			"username": user.Usernme,
			"nombre":   user.Nombre,
		})
	}
}
