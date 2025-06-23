package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"ired.com/micuenta/app"
	"ired.com/micuenta/middlewares"
	"ired.com/micuenta/models"
	"ired.com/micuenta/repo"
)

func PaymentRoutes(r *gin.Engine) {
	susc := r.Group("/payment")
	{
		susc.POST("/send", middlewares.JwtAuth, sendPayment)
		susc.POST("/image-upload", middlewares.JwtAuth, imageUpload)
		susc.GET("/show", middlewares.JwtAuth, showPayment)
		susc.GET("/list", middlewares.JwtAuth, listPayments)
		susc.GET("/transfer/balance", middlewares.JwtAuth, balanceAvailable)
		susc.POST("/transfer/send", middlewares.JwtAuth, sendTransfer)
		susc.GET("/transfer/list", middlewares.JwtAuth, listTransfers)
	}
}

// @Summary        endpoint para guardar formulario de pago
// @Description    procesa el formulario y valida el mismo, devuelve el id del recibo de pago
// @Tags           Payment
// @Accept         json
// @Produce        json
// @Param          x-access-token header string true "Access Token"
// @Param 				 payment body models.PaymentReq true "Payment Data"
// @Failure 400    {object} models.ErrorResponse "Invalid Request or Incorrect Data"
// @Failure 401    {object} models.ErrorResponse "Unauthorized"
// @Success 			200 {object} models.SuccessResponse{record=models.PaymentResponse}
// @Router         /payment/send [post]
func sendPayment(c *gin.Context) {
	// validate if body exist
	if c.Request.ContentLength == 0 {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, "errorFailedBody")},
		)
		return
	}

	// Bind and Validate the data and the struct
	var paymentReq models.PaymentReq
	if err := c.ShouldBindJSON(&paymentReq); err != nil {
		if strings.Contains(err.Error(), "invalid character") || strings.Contains(err.Error(), "unmarshal") {
			c.AbortWithStatusJSON(
				http.StatusBadRequest,
				models.ErrorResponse{Error: ginI18n.MustGetMessage(c, "invalidJson")},
			)
			return
		}

		errorFormJson := models.ParseError(err, c)
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: errorFormJson},
		)
		return
	}

	//set variables for handling pgsql conn
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db := models.ConnDb{ConnPgsql: app.PoolPgsql, Ctx: ctx}

	// process and check for errors
	userId, _ := c.Get("userId")
	paymentResponse, errType, err := repo.SendPayment(db, userId, paymentReq)

	// validar por errores
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
			Notice: ginI18n.MustGetMessage(c, "formOK"),
			Record: paymentResponse,
		},
	)
}

// @Summary					Upload image for payment
// @Description			Upload an image associated with a payment ID
// @Tags						Payment
// @Accept					multipart/form-data
// @Produce					json
// @Param           x-access-token header string true "Access Token"
// @Param						payment_id formData string true "paymentId (UUID), created_at(timestamptz)"
// @Param						image	formData file true "Image file to upload"
// @Failure 400    {object} models.ErrorResponse "Invalid Request or Incorrect Data"
// @Failure 401    {object} models.ErrorResponse "Unauthorized"
// @Success 				200 {object} models.SuccessResponse
// @Router 					/payment/image-upload [post]
func imageUpload(c *gin.Context) {
	// Bind and Validate the data and the struct
	var payment models.PaymentReqId
	if err := c.ShouldBind(&payment); err != nil {
		if strings.Contains(err.Error(), "invalid character") || strings.Contains(err.Error(), "unmarshal") {
			c.AbortWithStatusJSON(
				http.StatusBadRequest,
				models.ErrorResponse{Error: ginI18n.MustGetMessage(c, "invalidJson")},
			)
			return
		}

		errorFormJson := models.ParseError(err, c)
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: errorFormJson},
		)
		return
	}

	// validate that file exist
	file, err := c.FormFile("image")
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, "veImageRequired")},
		)
		return
	}

	//set variables for handling pgsql conn
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db := models.ConnDb{ConnPgsql: app.PoolPgsql, Ctx: ctx}

	//  process and check for errors
	if errType, err := repo.ImageUpload(c, db, file, payment); err != nil {
		c.JSON(
			errType,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, err.Error())},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		models.SuccessResponse{
			Notice: ginI18n.MustGetMessage(c, "formOK"),
		},
	)
}

// @Summary        detalle de un recibo de pago
// @Description    devuelve toda la data relacionada a un recibo de pago
// @Tags           Payment
// @Accept         json
// @Produce        json
// @Param          x-access-token header string true "Access Token"
// @Param 				 PaymentReqId query string true "paymentId (UUID), created_at(timestamptz)"
// @Failure 400    {object} models.ErrorResponse "Invalid Request or Incorrect Data"
// @Failure 401    {object} models.ErrorResponse "Unauthorized"
// @Success 			200 {object} models.SuccessResponse{record=models.PaymentResponse}
// @Router         /payment/show [get]
func showPayment(c *gin.Context) {
	// Bind and Validate the data and the struct
	var payment models.PaymentReqId
	if err := c.ShouldBind(&payment); err != nil {
		if strings.Contains(err.Error(), "invalid character") || strings.Contains(err.Error(), "unmarshal") {
			c.AbortWithStatusJSON(
				http.StatusBadRequest,
				models.ErrorResponse{Error: ginI18n.MustGetMessage(c, "invalidJson")},
			)
			return
		}

		errorFormJson := models.ParseError(err, c)
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: errorFormJson},
		)
		return
	}

	// set variables for handling dbs conns
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db := models.ConnDb{ConnPgsql: app.PoolPgsql, Ctx: ctx}

	userId, _ := c.Get("userId")
	paymentResponse, errType, err := repo.GetPayment(db, fmt.Sprintf("%s", userId), payment)
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
			Record: paymentResponse,
		},
	)
}

// @Summary 			Listado de recibos de pago
// @Description 	Retrieve a list of recibos de pagos with pagination
// @Tags 					Payment
// @Accept 				json
// @Produce 			json
// @Param         x-access-token header string true "Access Token"
// @Param 				page query int false "Page number" default(1)
// @Param 				limit query int false "Number of records per page" default(10)
// @Failure 400   {object} models.ErrorResponse "Invalid Request or Incorrect Data"
// @Failure 401   {object} models.ErrorResponse "Unauthorized"
// @Success 			200 {object} models.SuccessResponseWithMeta{record=[]models.PaymentList}
// @Router 				/payment/list [get]
func listPayments(c *gin.Context) {
	// Bind and Validate the data and the struct
	paginatorQueryUri := models.PaginatorQueryUri{Page: json.Number("1"), Limit: json.Number("10")}
	if err := c.ShouldBind(&paginatorQueryUri); err != nil {
		if strings.Contains(err.Error(), "invalid character") || strings.Contains(err.Error(), "unmarshal") {
			c.AbortWithStatusJSON(
				http.StatusBadRequest,
				models.ErrorResponse{Error: ginI18n.MustGetMessage(c, "invalidJson")},
			)
			return
		}

		errorFormJson := models.ParseError(err, c)
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: errorFormJson},
		)
		return
	}

	// trasnform uri into int struct paginator
	paginatorQuery := models.TransformPaginator(paginatorQueryUri)

	//set variables for handling pgsql conn
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db := models.ConnDb{ConnPgsql: app.PoolPgsql, Ctx: ctx}

	// look for data
	userId, _ := c.Get("userId")
	paymentsData, paginatorData, err := repo.PaymentList(db, userId, paginatorQuery)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, err.Error())},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		models.SuccessResponseWithMeta{
			Notice: ginI18n.MustGetMessage(c, "queryOK"),
			Meta:   paginatorData,
			Record: paymentsData,
		},
	)
}

// @Summary 			Saldo disponible para transferir en la cuenta
// @Description 	Retrieve the balance available to transfer
// @Tags 					Payment
// @Accept 				json
// @Produce 			json
// @Param         x-access-token header string true "Access Token"
// @Failure 400   {object} models.ErrorResponse "Invalid Request or Incorrect Data"
// @Failure 401   {object} models.ErrorResponse "Unauthorized"
// @Success 			200 {object} models.SuccessResponse{record=[]models.BalanceAvailable}
// @Router 				/payment/transfer/balance [get]
func balanceAvailable(c *gin.Context) {
	// set variables for handling dbs conns
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db := models.ConnDb{ConnPgsql: app.PoolPgsql, Ctx: ctx}

	balanceAvailable, errType, err := repo.BalanceAvailable(c, db)
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
			Record: balanceAvailable,
		},
	)
}

// @Summary        endpoint para guardar formulario de transferencia
// @Description    procesa el formulario y valida el mismo, devuelve el id de la transferencia
// @Tags           Payment
// @Accept         json
// @Produce        json
// @Param          x-access-token header string true "Access Token"
// @Param 				 transfer body models.TransferReq true "Transfer Data"
// @Failure 400    {object} models.ErrorResponse "Invalid Request or Incorrect Data"
// @Failure 401    {object} models.ErrorResponse "Unauthorized"
// @Success 			200 {object} models.SuccessResponse{record=models.TransferResponse}
// @Router         /payment/transfer/send [post]
func sendTransfer(c *gin.Context) {
	// validate if body exist
	if c.Request.ContentLength == 0 {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, "errorFailedBody")},
		)
		return
	}

	// Bind and Validate the data and the struct
	var transferReq models.TransferReq
	if err := c.ShouldBindJSON(&transferReq); err != nil {
		if strings.Contains(err.Error(), "invalid character") || strings.Contains(err.Error(), "unmarshal") {
			c.AbortWithStatusJSON(
				http.StatusBadRequest,
				models.ErrorResponse{Error: ginI18n.MustGetMessage(c, "invalidJson")},
			)
			return
		}

		errorFormJson := models.ParseError(err, c)
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: errorFormJson},
		)
		return
	}

	//set variables for handling pgsql conn
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db := models.ConnDb{ConnPgsql: app.PoolPgsql, Ctx: ctx}

	// process and check for errors
	userId, _ := c.Get("userId")
	transferResponse, errType, err := repo.SendTransfer(db, userId, transferReq)

	// validar por errores
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
			Notice: ginI18n.MustGetMessage(c, "formOK"),
			Record: transferResponse,
		},
	)
}

// @Summary 			Listado de transferencias
// @Description 	Retrieve a list of transferencias with pagination
// @Tags 					Payment
// @Accept 				json
// @Produce 			json
// @Param         x-access-token header string true "Access Token"
// @Param 				page query int false "Page number" default(1)
// @Param 				limit query int false "Number of records per page" default(10)
// @Failure 400   {object} models.ErrorResponse "Invalid Request or Incorrect Data"
// @Failure 401   {object} models.ErrorResponse "Unauthorized"
// @Success 			200 {object} models.SuccessResponseWithMeta{record=[]models.TransferList}
// @Router 				/payment/transfer/list [get]
func listTransfers(c *gin.Context) {
	// Bind and Validate the data and the struct
	paginatorQueryUri := models.PaginatorQueryUri{Page: json.Number("1"), Limit: json.Number("10")}
	if err := c.ShouldBind(&paginatorQueryUri); err != nil {
		if strings.Contains(err.Error(), "invalid character") || strings.Contains(err.Error(), "unmarshal") {
			c.AbortWithStatusJSON(
				http.StatusBadRequest,
				models.ErrorResponse{Error: ginI18n.MustGetMessage(c, "invalidJson")},
			)
			return
		}

		errorFormJson := models.ParseError(err, c)
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: errorFormJson},
		)
		return
	}

	// trasnform uri into int struct paginator
	paginatorQuery := models.TransformPaginator(paginatorQueryUri)

	//set variables for handling pgsql conn
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db := models.ConnDb{ConnPgsql: app.PoolPgsql, Ctx: ctx}

	// look for data
	userId, _ := c.Get("userId")
	paymentsData, paginatorData, err := repo.TransferList(db, userId, paginatorQuery)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, err.Error())},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		models.SuccessResponseWithMeta{
			Notice: ginI18n.MustGetMessage(c, "queryOK"),
			Meta:   paginatorData,
			Record: paymentsData,
		},
	)
}
