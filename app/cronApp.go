package app

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/go-co-op/gocron/v2"
	"ired.com/micuenta/models"
	"ired.com/micuenta/repo"
	"ired.com/micuenta/utils"
)

// TaskConfig structure to hold the cron schedule and task name
type taskConfig struct {
	Schedule string `json:"schedule"`
	Task     string `json:"task"`
	Enabled  bool   `json:"enabled"`
}

// Load task configurations from file
func loadTasksConfig() ([]taskConfig, error) {
	// open file
	file, err := os.Open(".crontab")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// decode json data to struct
	var tasksConfig []taskConfig
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&tasksConfig)
	if err != nil {
		return nil, err
	}

	return tasksConfig, nil
}

func LoadCrontab() {

	// Load task configurations
	tasksConfig, err := loadTasksConfig()
	if err != nil {
		utils.Logline("Failed to load task configurations: %v", err)
		return
	}

	// Use America/Caracas time
	ccsLocation, _ := time.LoadLocation("America/Caracas")

	// Create a new scheduler
	scheduler, _ := gocron.NewScheduler(gocron.WithLocation(ccsLocation))

	// // Schedule tasks based on the configurations
	for _, taskConfig := range tasksConfig {
		if !taskConfig.Enabled {
			continue
		}
		var err error
		switch taskConfig.Task {
		case "clean_old_sessions":
			_, err = scheduler.NewJob(
				gocron.CronJob(taskConfig.Schedule, false),
				gocron.NewTask(cleanOldSessions),
				gocron.WithSingletonMode(gocron.LimitModeReschedule),
			)
		case "create_clients_passwd":
			_, err = scheduler.NewJob(
				gocron.CronJob(taskConfig.Schedule, false),
				gocron.NewTask(createClientPasswd),
				gocron.WithSingletonMode(gocron.LimitModeReschedule),
			)
		case "sinc_tasa_cambio":
			_, err = scheduler.NewJob(
				gocron.CronJob(taskConfig.Schedule, false),
				gocron.NewTask(sincTasaCambio),
				gocron.WithSingletonMode(gocron.LimitModeReschedule),
			)
		case "sinc_factura_fiscal":
			_, err = scheduler.NewJob(
				gocron.CronJob(taskConfig.Schedule, false),
				gocron.NewTask(sincFacturaFiscal),
				gocron.WithSingletonMode(gocron.LimitModeReschedule),
			)
		case "sinc_retenciones":
			_, err = scheduler.NewJob(
				gocron.CronJob(taskConfig.Schedule, false),
				gocron.NewTask(sincRetenciones),
				gocron.WithSingletonMode(gocron.LimitModeReschedule),
			)
		case "sinc_prefactura_anuladas":
			_, err = scheduler.NewJob(
				gocron.CronJob(taskConfig.Schedule, false),
				gocron.NewTask(sincPreFacturaAnuladas),
				gocron.WithSingletonMode(gocron.LimitModeReschedule),
			)
		case "sinc_prefactura_pagadas":
			_, err = scheduler.NewJob(
				gocron.CronJob(taskConfig.Schedule, false),
				gocron.NewTask(sincPreFacturaPagadas),
				gocron.WithSingletonMode(gocron.LimitModeReschedule),
			)
		case "sinc_recibopago_anulados":
			_, err = scheduler.NewJob(
				gocron.CronJob(taskConfig.Schedule, false),
				gocron.NewTask(sincReciboPagoAnulados),
				gocron.WithSingletonMode(gocron.LimitModeReschedule),
			)
		case "sinc_recibopago_procesados":
			_, err = scheduler.NewJob(
				gocron.CronJob(taskConfig.Schedule, false),
				gocron.NewTask(sincReciboPagoProcesados),
				gocron.WithSingletonMode(gocron.LimitModeReschedule),
			)
		default:
			utils.Logline("Unknown task", taskConfig.Task)
		}

		if err != nil {
			utils.Logline("Failed to schedule task", err)
		}
	}

	// Start the scheduler
	scheduler.Start()
}

// Define  task functions
func cleanOldSessions() {
	defer func() {
		if r := recover(); r != nil {
			utils.Logline("Recovered from panic <<clean_old_sessions>>: %v", r)
		}
	}()

	//set variables for handling pgsql and mysql conn
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	db := models.ConnDb{ConnPgsql: PoolPgsql, Ctx: ctx}

	// run actual task
	if err := repo.CleanOldSessionsCron(db, "cronJob"); err != nil {
		utils.Logline("Error on clean_old_sessions")
	}
}

func createClientPasswd() {
	defer func() {
		if r := recover(); r != nil {
			utils.Logline("Recovered from panic <<create_clients_passwd>>: %v", r)
		}
	}()

	//set variables for handling pgsql and mysql conn
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db := models.ConnDb{ConnPgsql: PoolPgsql, Ctx: ctx}

	// run actual task
	if err := repo.CreatePasswordsCron(db, "cronJob"); err != nil {
		utils.Logline("Error on create_clients_passwd")
	}
}

func sincTasaCambio() {
	defer func() {
		if r := recover(); r != nil {
			utils.Logline("Recovered from panic <<sinc_tasa_cambio>>: %v", r)
		}
	}()

	//set variables for handling pgsql and mysql conn
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db := models.ConnMysqlPgsql{ConnPgsql: PoolPgsql, ConnMysql: PoolMysql, Ctx: ctx}

	// run actual task
	if err := repo.SincTasaCambio(db, "cronJob"); err != nil {
		utils.Logline("Error on sinc_tasa_cambio")
	}
}

func sincFacturaFiscal() {
	defer func() {
		if r := recover(); r != nil {
			utils.Logline("Recovered from panic <<sinc_factura_fiscal>>: %v", r)
		}
	}()

	//set variables for handling pgsql and mysql conn
	ctx, cancel := context.WithTimeout(context.Background(), 55*time.Second)
	defer cancel()
	db := models.ConnMysqlPgsql{ConnPgsql: PoolPgsql, ConnMysql: PoolMysql, Ctx: ctx}

	// run actual task
	if err := repo.SincFacturaFiscal(db, "cronJob"); err != nil {
		utils.Logline("Error on sinc_factura_fiscal")
	}
}

func sincRetenciones() {
	defer func() {
		if r := recover(); r != nil {
			utils.Logline("Recovered from panic <<sinc_retenciones>>: %v", r)
		}
	}()

	//set variables for handling pgsql and mysql conn
	ctx, cancel := context.WithTimeout(context.Background(), 55*time.Second)
	defer cancel()
	db := models.ConnMysqlPgsql{ConnPgsql: PoolPgsql, ConnMysql: PoolMysql, Ctx: ctx}

	// run actual task
	if err := repo.SincRetenciones(db, "cronJob"); err != nil {
		utils.Logline("Error on sinc_retenciones")
	}
}

func sincPreFacturaAnuladas() {
	defer func() {
		if r := recover(); r != nil {
			utils.Logline("Recovered from panic <<sinc_prefactura_anuladas>>: %v", r)
		}
	}()

	//set variables for handling pgsql and mysql conn
	ctx, cancel := context.WithTimeout(context.Background(), 55*time.Second)
	defer cancel()
	db := models.ConnMysqlPgsql{ConnPgsql: PoolPgsql, ConnMysql: PoolMysql, Ctx: ctx}

	// run actual task
	if err := repo.SincPreFactura(db, "cronJob", "anulado"); err != nil {
		utils.Logline("Error on sinc_prefactura_anuladas")
	}
}

func sincPreFacturaPagadas() {
	defer func() {
		if r := recover(); r != nil {
			utils.Logline("Recovered from panic <<sinc_prefactura_pagadas>>: %v", r)
		}
	}()

	//set variables for handling pgsql and mysql conn
	ctx, cancel := context.WithTimeout(context.Background(), 55*time.Second)
	defer cancel()
	db := models.ConnMysqlPgsql{ConnPgsql: PoolPgsql, ConnMysql: PoolMysql, Ctx: ctx}

	// run actual task
	if err := repo.SincPreFactura(db, "cronJob", "pagado"); err != nil {
		utils.Logline("Error on sinc_prefactura_pagadas")
	}
}

func sincReciboPagoAnulados() {
	defer func() {
		if r := recover(); r != nil {
			utils.Logline("Recovered from panic <<sinc_recibopago_anulados>>: %v", r)
		}
	}()

	//set variables for handling pgsql and mysql conn
	ctx, cancel := context.WithTimeout(context.Background(), 55*time.Second)
	defer cancel()
	db := models.ConnMysqlPgsql{ConnPgsql: PoolPgsql, ConnMysql: PoolMysql, Ctx: ctx}

	// run actual task
	if err := repo.SincReciboVenta(db, "cronJob", "anulado"); err != nil {
		utils.Logline("Error on sinc_recibopago_anulados")
	}
}

func sincReciboPagoProcesados() {
	defer func() {
		if r := recover(); r != nil {
			utils.Logline("Recovered from panic <<sinc_recibopago_procesados>>: %v", r)
		}
	}()

	//set variables for handling pgsql and mysql conn
	ctx, cancel := context.WithTimeout(context.Background(), 55*time.Second)
	defer cancel()
	db := models.ConnMysqlPgsql{ConnPgsql: PoolPgsql, ConnMysql: PoolMysql, Ctx: ctx}

	// run actual task
	if err := repo.SincReciboVenta(db, "cronJob", "procesado"); err != nil {
		utils.Logline("Error on sinc_recibopago_procesados")
	}
}
