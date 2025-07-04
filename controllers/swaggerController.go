package controllers

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "ired.com/micuenta/docs"
	"ired.com/micuenta/middlewares"
)

func SwaggerRoutes(r *gin.Engine) {
	r.GET("/docs/*any", middlewares.BasicAuth(), ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.DefaultModelsExpandDepth(0)))
}
