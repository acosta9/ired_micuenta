package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
	"ired.com/micuenta/app"
	"ired.com/micuenta/controllers"
	"ired.com/micuenta/middlewares"
	"ired.com/micuenta/utils"

	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
)

func init() {
	app.LoadEnvVariables()
	app.InitDbMysql()
	app.InitDbPgsql()
	app.LoadCrontab()

	gin.SetMode(os.Getenv("GIN_MODE"))

	// create log file and handle logrotate also
	logfile := &lumberjack.Logger{
		Filename:   "logs/main.log",
		MaxSize:    100,  // megabytes
		MaxBackups: 30,   // Keep logs for a month
		Compress:   true, // disabled by default
	}
	log.SetOutput(logfile)

	//manual function to rotate logs daily
	go func() {
		for range time.Tick(24 * time.Hour) {
			logfile.Rotate()
		}
	}()
}

// @Title								MiCuenta Service API
// @Version							1.0
// @Description 				service in Go using Gin framework
// @Host								127.0.0.1:7003
// @Contact.name   			Juan Acosta
// @Contact.url			    https://www.linkedin.com/in/juan-m-acosta-f-54219758/
// @Contact.email  			juan9acosta@gmail.com
// @securityDefinitions.basic BasicAuth
// @securityDefinitions.basic.description Basic Authentication
// @BasePath /
func main() {
	r := gin.Default()

	// aply startTimer middleware
	r.Use(middlewares.StartTimer())

	// apply custom logger middleware
	r.Use(middlewares.RequestLogger())

	// apply the security headers middleware globally - cors included here
	r.Use(middlewares.SecureHeaders())

	// apply i18n middleware
	r.Use(
		ginI18n.Localize(ginI18n.WithBundle(&ginI18n.BundleCfg{
			RootPath:         "./i18n",
			AcceptLanguage:   []language.Tag{language.English, language.Spanish},
			DefaultLanguage:  language.Spanish,
			UnmarshalFunc:    json.Unmarshal,
			FormatBundleFile: "json",
		})),
	)

	// recover if panic and log the fail
	r.Use(gin.RecoveryWithWriter(log.Writer()))

	// create store for session handlers
	store, err := app.NewPGStore(app.PoolPgsql, []byte(os.Getenv("SECRET")))
	if err != nil {
		utils.Fatalf("error creating store for sessions", err)
	}

	// middleware for sessions
	secureCookie, _ := strconv.ParseBool(os.Getenv("SECURE_COOKIE"))
	maxAgeCookie, _ := strconv.Atoi(os.Getenv("REFRESH_MAX_AGE"))
	store.Options(sessions.Options{MaxAge: maxAgeCookie, Path: "/", Secure: secureCookie, HttpOnly: true})
	r.Use(sessions.Sessions("auth_session", store))

	// load templates
	r.LoadHTMLGlob("templates/*")

	// load static files
	r.Static("/public", "./public")

	// manual routes
	controllers.AuthRoutes(r)
	controllers.SuscripcionRoutes(r)
	controllers.BancoRoutes(r)
	controllers.PaymentRoutes(r)
	controllers.FacturaRoutes(r)
	controllers.RetencionRoutes(r)
	controllers.InfoRoutes(r)
	controllers.CronRoutes(r)

	// load docs
	controllers.SwaggerRoutes(r)

	// handle 404 error with custom template
	r.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusNotFound, "404.tmpl", gin.H{
			"title": "Page Not Found",
		})
	})

	// Set a lower memory limit for multipart forms (default is 32 MiB)
	r.MaxMultipartMemory = 8 << 20

	// run server default port 8080 or lookup .env file
	r.Run()

	defer app.CloseDbMysql()
	defer app.CloseDbPgsql()
}
