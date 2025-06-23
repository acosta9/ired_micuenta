package controllers

import (
	"context"
	"net/http"
	"time"

	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"ired.com/micuenta/app"
	"ired.com/micuenta/middlewares"
	"ired.com/micuenta/models"
	"ired.com/micuenta/repo"
)

func BancoRoutes(r *gin.Engine) {
	susc := r.Group("/banco")
	{
		susc.GET("/formas-de-pago", middlewares.JwtAuth, formaPagoList)
	}
}

// @Summary        listado formas de pago
// @Description    devuelve un listado de las formas de pago disponibles para el cliente
// @Tags           Banco
// @Accept         json
// @Produce        json
// @Param          x-access-token header string true "Access Token"
// @Failure 400    {object} models.ErrorResponse "Invalid Request or Incorrect Data"
// @Failure 401    {object} models.ErrorResponse "Unauthorized"
// @Success 			200 {object} models.SuccessResponse{record=models.CuentasBanco}
// @Router         /banco/formas-de-pago [get]
func formaPagoList(c *gin.Context) {
	// set variables for handling dbs conns
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db := models.ConnDb{ConnPgsql: app.PoolPgsql, Ctx: ctx}

	formaPagoList, errType, err := repo.FormasPagoList(c, db)
	if err != nil {
		c.JSON(
			errType,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, err.Error())},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		models.SuccessResponse{
			Notice: ginI18n.MustGetMessage(c, "queryOK"),
			Record: formaPagoList,
		},
	)
}
