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

func FacturaRoutes(r *gin.Engine) {
	susc := r.Group("/factura")
	{
		susc.GET("/list", middlewares.JwtAuth, listFacturas)
		susc.GET("/show", middlewares.JwtAuth, showFactura)
	}
}

// @Summary        detalle de una factura
// @Description    devuelve toda la data relacionada a una factura
// @Tags           Factura
// @Accept         json
// @Produce        json
// @Param          x-access-token header string true "Access Token"
// @Param 				 FacturaReqId query string true "retencionId (UUID), created_at(timestamptz)"
// @Failure 400    {object} models.ErrorResponse "Invalid Request or Incorrect Data"
// @Failure 401    {object} models.ErrorResponse "Unauthorized"
// @Success 			 200 {object} models.SuccessResponse{record=models.FacturaResponse}
// @Router         /factura/show [get]
func showFactura(c *gin.Context) {
	// Bind and Validate the data and the struct
	var factura models.FacturaReqId
	if err := c.ShouldBind(&factura); err != nil {
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
	facturaResponse, errType, err := repo.GetFactura(db, fmt.Sprintf("%s", userId), factura)
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
			Record: facturaResponse,
		},
	)
}

// @Summary 			Listado de facturas
// @Description 	Retrieve a list of facturas with pagination
// @Tags 					Factura
// @Accept 				json
// @Produce 			json
// @Param         x-access-token header string true "Access Token"
// @Param 				page query int false "Page number" default(1)
// @Param 				limit query int false "Number of recibos de pago per page" default(10)
// @Failure 400   {object} models.ErrorResponse "Invalid Request or Incorrect Data"
// @Failure 401   {object} models.ErrorResponse "Unauthorized"
// @Success 			200 {object} models.SuccessResponseWithMeta{record=[]models.FacturaList}
// @Router 				/factura/list [get]
func listFacturas(c *gin.Context) {
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
	facturasData, paginatorData, err := repo.FacturaList(db, userId, paginatorQuery)
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
			Record: facturasData,
		},
	)
}
