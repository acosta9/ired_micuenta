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

func CronRoutes(r *gin.Engine) {
	cron := r.Group("/cron")
	{
		cron.GET("/clean-old-sessions", middlewares.BasicAuth(), cleanOldSessions)
		cron.GET("/create-clients-passwd", middlewares.BasicAuth(), createPasswords)
		cron.GET("/sinc-tasa-cambio", middlewares.BasicAuth(), sincTasaCambio)
		cron.GET("/sinc-factura-fiscal", middlewares.BasicAuth(), sincFacturaFiscal)
		cron.GET("/sinc-retencion", middlewares.BasicAuth(), sincRetenciones)
		cron.GET("/sinc-prefactura-anulada", middlewares.BasicAuth(), SincPreFacturaAnuladas)
		cron.GET("/sinc-prefactura-pagada", middlewares.BasicAuth(), SincPreFacturaPagadas)
		cron.GET("/sinc-recibov-anulado", middlewares.BasicAuth(), SincRecibovAnulado)
		cron.GET("/sinc-recibov-procesado", middlewares.BasicAuth(), SincRecibovProcesado)
	}
}

// @Summary 			Run the task sinc_users
// @Description 	cleaning old session of clientes from postgresDB
// @Tags 					Crons
// @Accept 				json
// @Produce 			json
// @Security 			BasicAuth
// @Success 			200 {object} models.SuccessResponse
// @Failure 			400 {object} models.ErrorResponse
// @Router 				/cron/clean-old-sessions [get]
func cleanOldSessions(c *gin.Context) {
	//set variables for handling pgsql and mysql conn
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	db := models.ConnDb{ConnPgsql: app.PoolPgsql, Ctx: ctx}

	if err := repo.CleanOldSessionsCron(db, "restApi"); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: err.Error()},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		models.SuccessResponse{Notice: ginI18n.MustGetMessage(c, "cronOK")},
	)
}

// @Summary 			Run the task create_client_passwd
// @Description 	search for all users with password null and set the password the same as the docid
// @Tags 					Crons
// @Accept 				json
// @Produce 			json
// @Security 			BasicAuth
// @Success 			200 {object} models.SuccessResponse
// @Failure 			400 {object} models.ErrorResponse
// @Router 				/cron/create-clients-passwd [get]
func createPasswords(c *gin.Context) {
	//set variables for handling pgsql and mysql conn
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db := models.ConnDb{ConnPgsql: app.PoolPgsql, Ctx: ctx}

	if err := repo.CreatePasswordsCron(db, "restApi"); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: err.Error()},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		models.SuccessResponse{Notice: ginI18n.MustGetMessage(c, "cronOK")},
	)
}

// @Summary 			Run the task sinc_tasa_cambio
// @Description 	busca registros nuevos en la bd de mysql y sincroniza la data a postgres
// @Tags 					Crons
// @Accept 				json
// @Produce 			json
// @Security 			BasicAuth
// @Success 			200 {object} models.SuccessResponse
// @Failure 			400 {object} models.ErrorResponse
// @Router 				/cron/sinc-tasa-cambio [get]
func sincTasaCambio(c *gin.Context) {
	//set variables for handling pgsql and mysql conn
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db := models.ConnMysqlPgsql{ConnPgsql: app.PoolPgsql, ConnMysql: app.PoolMysql, Ctx: ctx}

	if err := repo.SincTasaCambio(db, "restApi"); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: err.Error()},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		models.SuccessResponse{Notice: ginI18n.MustGetMessage(c, "cronOK")},
	)
}

// @Summary 			Run the task sinc_factura_fiscal
// @Description 	busca registros nuevos en la bd de mysql y sincroniza la data a postgres
// @Tags 					Crons
// @Accept 				json
// @Produce 			json
// @Security 			BasicAuth
// @Success 			200 {object} models.SuccessResponse
// @Failure 			400 {object} models.ErrorResponse
// @Router 				/cron/sinc-factura-fiscal [get]
func sincFacturaFiscal(c *gin.Context) {
	//set variables for handling pgsql and mysql conn
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	db := models.ConnMysqlPgsql{ConnPgsql: app.PoolPgsql, ConnMysql: app.PoolMysql, Ctx: ctx}

	if err := repo.SincFacturaFiscal(db, "restApi"); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: err.Error()},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		models.SuccessResponse{Notice: ginI18n.MustGetMessage(c, "cronOK")},
	)
}

// @Summary 			Run the task sinc_retenciones
// @Description 	busca registros nuevos en la bd de mysql y sincroniza la data a postgres
// @Tags 					Crons
// @Accept 				json
// @Produce 			json
// @Security 			BasicAuth
// @Success 			200 {object} models.SuccessResponse
// @Failure 			400 {object} models.ErrorResponse
// @Router 				/cron/sinc-retencion [get]
func sincRetenciones(c *gin.Context) {
	//set variables for handling pgsql and mysql conn
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	db := models.ConnMysqlPgsql{ConnPgsql: app.PoolPgsql, ConnMysql: app.PoolMysql, Ctx: ctx}

	if err := repo.SincRetenciones(db, "restApi"); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: err.Error()},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		models.SuccessResponse{Notice: ginI18n.MustGetMessage(c, "cronOK")},
	)
}

// @Summary 			Run the task sinc_prefactura_anuladas
// @Description 	busca registros nuevos en la bd de mysql y sincroniza la data a postgres
// @Tags 					Crons
// @Accept 				json
// @Produce 			json
// @Security 			BasicAuth
// @Success 			200 {object} models.SuccessResponse
// @Failure 			400 {object} models.ErrorResponse
// @Router 				/cron/sinc-prefactura-anulada [get]
func SincPreFacturaAnuladas(c *gin.Context) {
	//set variables for handling pgsql and mysql conn
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	db := models.ConnMysqlPgsql{ConnPgsql: app.PoolPgsql, ConnMysql: app.PoolMysql, Ctx: ctx}

	if err := repo.SincPreFactura(db, "restApi", "anulado"); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: err.Error()},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		models.SuccessResponse{Notice: ginI18n.MustGetMessage(c, "cronOK")},
	)
}

// @Summary 			Run the task sinc_prefactura_pagadas
// @Description 	busca registros nuevos en la bd de mysql y sincroniza la data a postgres
// @Tags 					Crons
// @Accept 				json
// @Produce 			json
// @Security 			BasicAuth
// @Success 			200 {object} models.SuccessResponse
// @Failure 			400 {object} models.ErrorResponse
// @Router 				/cron/sinc-prefactura-pagadas [get]
func SincPreFacturaPagadas(c *gin.Context) {
	//set variables for handling pgsql and mysql conn
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	db := models.ConnMysqlPgsql{ConnPgsql: app.PoolPgsql, ConnMysql: app.PoolMysql, Ctx: ctx}

	if err := repo.SincPreFactura(db, "restApi", "pagado"); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: err.Error()},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		models.SuccessResponse{Notice: ginI18n.MustGetMessage(c, "cronOK")},
	)
}

// @Summary 			Run the task sinc_recibov_anulado
// @Description 	busca registros nuevos en la bd de mysql y sincroniza la data a postgres
// @Tags 					Crons
// @Accept 				json
// @Produce 			json
// @Security 			BasicAuth
// @Success 			200 {object} models.SuccessResponse
// @Failure 			400 {object} models.ErrorResponse
// @Router 				/cron/sinc-recibov-anulado [get]
func SincRecibovAnulado(c *gin.Context) {
	//set variables for handling pgsql and mysql conn
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	db := models.ConnMysqlPgsql{ConnPgsql: app.PoolPgsql, ConnMysql: app.PoolMysql, Ctx: ctx}

	if err := repo.SincReciboVenta(db, "restApi", "anulado"); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: err.Error()},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		models.SuccessResponse{Notice: ginI18n.MustGetMessage(c, "cronOK")},
	)
}

// @Summary 			Run the task sinc_recibov_procesado
// @Description 	busca registros nuevos en la bd de mysql y sincroniza la data a postgres
// @Tags 					Crons
// @Accept 				json
// @Produce 			json
// @Security 			BasicAuth
// @Success 			200 {object} models.SuccessResponse
// @Failure 			400 {object} models.ErrorResponse
// @Router 				/cron/sinc-recibov-procesado [get]
func SincRecibovProcesado(c *gin.Context) {
	//set variables for handling pgsql and mysql conn
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	db := models.ConnMysqlPgsql{ConnPgsql: app.PoolPgsql, ConnMysql: app.PoolMysql, Ctx: ctx}

	if err := repo.SincReciboVenta(db, "restApi", "procesado"); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: err.Error()},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		models.SuccessResponse{Notice: ginI18n.MustGetMessage(c, "cronOK")},
	)
}
