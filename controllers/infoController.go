package controllers

import (
	"context"
	"net/http"
	"strings"
	"time"

	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"ired.com/micuenta/app"
	"ired.com/micuenta/models"
	"ired.com/micuenta/repo"
)

func InfoRoutes(r *gin.Engine) {
	susc := r.Group("/info")
	{
		susc.GET("/oficinas", getOficinas)
		susc.GET("/faqs", getFaqs)
		susc.GET("/accesibilidad", getAccesibilidad)
		susc.GET("/legal/privacy_policy", getPrivacyPolicy)
		susc.GET("/legal/terminos_y_condiciones", getTermsAndConditions)
	}
}

// @Summary 			Listado de oficinas
// @Description 	Retrieve a list of oficinas de la empresa con su ubicacion
// @Tags 					Info
// @Accept 				json
// @Produce 			json
// @Param         tipo query string true "Tipo de plataforma" Enums(movil, web)
// @Failure 400   {object} models.ErrorResponse "Invalid Request or Incorrect Data"
// @Success 			200 {object} models.SuccessResponse{record=[]models.InfoLocation}
// @Router 				/info/oficinas [get]
func getOficinas(c *gin.Context) {
	// Bind and Validate the data and the struct
	var tipoReq models.InfoTipoReq
	if err := c.ShouldBind(&tipoReq); err != nil {
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

	// look for data
	dataOficinas, err := repo.GetOficinas(db, tipoReq.Tipo)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, err.Error())},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		models.SuccessResponse{
			Notice: ginI18n.MustGetMessage(c, "queryOK"),
			Record: dataOficinas,
		},
	)
}

// @Summary 			listado de preguntas frecuentes
// @Description 	Retrieve listado de preguntas frecuentes
// @Tags 					Info
// @Accept 				json
// @Produce 			json
// @Param         tipo query string true "Tipo de plataforma" Enums(movil, web)
// @Failure 400   {object} models.ErrorResponse "Invalid Request or Incorrect Data"
// @Success 			200 {object} models.SuccessResponse{record=[]models.InfoFaq}
// @Router 				/info/faqs [get]
func getFaqs(c *gin.Context) {
	// Bind and Validate the data and the struct
	var tipoReq models.InfoTipoReq
	if err := c.ShouldBind(&tipoReq); err != nil {
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

	// look for data
	dataFaqs, err := repo.GetFaqs(db, tipoReq.Tipo)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, err.Error())},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		models.SuccessResponse{
			Notice: ginI18n.MustGetMessage(c, "queryOK"),
			Record: dataFaqs,
		},
	)
}

// @Summary 			Texto con accesibilidad para la app movil
// @Description 	Retrieve Texto con accesibilidad
// @Tags 					Info
// @Accept 				json
// @Produce 			json
// @Param         tipo query string true "Tipo de plataforma" Enums(movil, web)
// @Failure 400   {object} models.ErrorResponse "Invalid Request or Incorrect Data"
// @Success 			200 {object} models.SuccessResponse
// @Router 				/info/accesibilidad [get]
func getAccesibilidad(c *gin.Context) {
	// Bind and Validate the data and the struct
	var tipoReq models.InfoTipoReq
	if err := c.ShouldBind(&tipoReq); err != nil {
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

	// look for data
	dataAccesibilidad, err := repo.GetAccesibilidad(db, tipoReq.Tipo)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, err.Error())},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		models.SuccessResponse{
			Notice: ginI18n.MustGetMessage(c, "queryOK"),
			Record: dataAccesibilidad,
		},
	)
}

// @Summary 			Texto con politicas de privacidad
// @Description 	Retrieve Texto con politicas de privacidad
// @Tags 					Info
// @Accept 				json
// @Produce 			json
// @Param         tipo query string true "Tipo de plataforma" Enums(movil, web)
// @Failure 400   {object} models.ErrorResponse "Invalid Request or Incorrect Data"
// @Success 			200 {object} models.SuccessResponse
// @Router 				/info/legal/privacy_policy [get]
func getPrivacyPolicy(c *gin.Context) {
	// Bind and Validate the data and the struct
	var tipoReq models.InfoTipoReq
	if err := c.ShouldBind(&tipoReq); err != nil {
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

	// look for data
	dataPrivacyPolicy, err := repo.GetPrivacyPolicy(db, tipoReq.Tipo)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, err.Error())},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		models.SuccessResponse{
			Notice: ginI18n.MustGetMessage(c, "queryOK"),
			Record: dataPrivacyPolicy,
		},
	)
}

// @Summary 			Texto con los terminos y condiciones
// @Description 	Retrieve Texto con los terminos y condiciones
// @Tags 					Info
// @Accept 				json
// @Produce 			json
// @Param         tipo query string true "Tipo de plataforma" Enums(movil, web)
// @Failure 400   {object} models.ErrorResponse "Invalid Request or Incorrect Data"
// @Success 			200 {object} models.SuccessResponse
// @Router 				/info/legal/terminos_y_condiciones [get]
func getTermsAndConditions(c *gin.Context) {
	// Bind and Validate the data and the struct
	var tipoReq models.InfoTipoReq
	if err := c.ShouldBind(&tipoReq); err != nil {
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

	// look for data
	dataTermsAndConditions, err := repo.GetTermsAndConditions(db, tipoReq.Tipo)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, err.Error())},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		models.SuccessResponse{
			Notice: ginI18n.MustGetMessage(c, "queryOK"),
			Record: dataTermsAndConditions,
		},
	)
}
