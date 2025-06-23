package middlewares

import (
	"encoding/base64"
	"net/http"
	"os"
	"strings"

	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"ired.com/micuenta/models"
)

func JwtAuth(c *gin.Context) {
	// get the cookie off req
	authToken := c.GetHeader("Authorization")

	if len(authToken) < 8 || authToken[:7] != "Bearer " {
		c.AbortWithStatusJSON(
			http.StatusUnauthorized,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, "userNotAuth")},
		)
		return
	}
	authToken = authToken[7:]

	// decode/validate it
	claims := &models.Claims{}
	token, err := jwt.ParseWithClaims(authToken, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("SECRET")), nil
	})
	if err != nil || !token.Valid {
		c.AbortWithStatusJSON(
			http.StatusUnauthorized,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, "userNotAuth")},
		)
		return
	}

	if claims.ChangePasswd {
		c.AbortWithStatusJSON(
			http.StatusUnauthorized,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, "passwdChangeRequired")},
		)
		return
	}

	// attach to req
	c.Set("userId", claims.UserId)

	c.Next()
}

func JwtPasswdAuth(c *gin.Context) {
	// get the token from the header
	authToken := c.GetHeader("Authorization")

	if len(authToken) < 8 || authToken[:7] != "Bearer " {
		c.AbortWithStatusJSON(
			http.StatusUnauthorized,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, "userNotAuth")},
		)
		return
	}
	authToken = authToken[7:]

	// decode/validate it
	claims := &models.Claims{}
	token, err := jwt.ParseWithClaims(authToken, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("SECRET")), nil
	})
	if err != nil || !token.Valid {
		c.AbortWithStatusJSON(
			http.StatusUnauthorized,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, "userNotAuth")},
		)
		return
	}

	// attach to req
	c.Set("userId", claims.UserId)

	c.Next()
}

func BasicAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Header("WWW-Authenticate", `Basic realm="Restricted"`)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		token := strings.Split(authHeader, "Basic ")
		if len(token) != 2 {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		decoded, err := base64.StdEncoding.DecodeString(token[1])
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		credentials := strings.Split(string(decoded), ":")
		if len(credentials) != 2 || credentials[0] != os.Getenv("DOC_USER") || credentials[1] != os.Getenv("DOC_PASSWD") {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Next()
	}
}
