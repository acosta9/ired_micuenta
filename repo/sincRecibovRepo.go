package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"ired.com/micuenta/models"
	"ired.com/micuenta/utils"
)

func SincReciboVenta(db models.ConnMysqlPgsql, caller string, tipo string) error {
	//show status of worker
	utils.ShowStatusWorkerMysql(db, "sinc_recibo_pagov_"+tipo, caller+"/begin")

	var estatusPgsql, montoPendienteMysql string
	switch tipo {
	case "anulado":
		estatusPgsql = "anulado"
	case "procesado":
		estatusPgsql = "procesado"
		montoPendienteMysql = "AND (rp.pendiente_monto2+0)<=0"
	}

	//get fecha of last record on postgres
	var lastRecordDate, endRecordDate string
	query := `SELECT TO_CHAR(created_at, 'YYYY-MM-DD HH24:MI:SS') as fecha,
			TO_CHAR(created_at + INTERVAL '8 MONTH', 'YYYY-MM-DD HH24:MI:SS') as fecha_end
		FROM venta.recibo_pagov
		WHERE estatus=$1
		ORDER BY created_at DESC 
		LIMIT 1`
	err := db.ConnPgsql.QueryRow(db.Ctx, query, estatusPgsql).Scan(&lastRecordDate, &endRecordDate)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			utils.Logline("error getting fecha of last record", "sincReciboPago", tipo, err)
			return err
		}
		lastRecordDate = "1970-01-01 00:00:00"
		endRecordDate = "2019-11-31 00:00:00"
	}

	// get the last records from mysql that have a date greater than the last record of postgres
	query = fmt.Sprintf(`SELECT q0.*
			FROM (
				SELECT rp.id as recibo_pago_id, rpu.id as recibo_pago_user_id, rp.client_id as cliente_id, 
				CASE WHEN rp.anulado = 1 THEN 'anulado' ELSE 'procesado' END as estatus, 
				CASE WHEN rpu.id IS NOT NULL THEN rpu.fecha ELSE rp.fecha END as fecha,
				CASE WHEN rpu.id IS NOT NULL THEN rpu.num_recibo ELSE rp.num_recibo END as referencia,
				CASE rp.forma_pago
					WHEN 0 THEN 'besser_bolivar_efectivo_efectivo'
					WHEN 1 THEN 'besser_dolar_efectivo_efectivo'
					WHEN 2 THEN 'banesco_bolivar_transferencia_banesco'
					WHEN 3 THEN 'bank of america_dolar_divisa_transferencia bofa'
					WHEN 4 THEN 'bnc inter_dolar_divisa_transferencia bnc inter'
					WHEN 5 THEN 'bank of america_dolar_divisa_zelle'
					WHEN 6 THEN 'airtm_dolar_divisa_airtm'
					WHEN 7 THEN 'bod_bolivar_transferencia_bod'
					WHEN 8 THEN 'banesco_bolivar_pago.movil_banesco'
					WHEN 9 THEN 'mercantil_bolivar_punto.venta_mercantil'
					WHEN 10 THEN 'mercantil_bolivar_transferencia_mercantil'
					WHEN 11 THEN 'mercantil_bolivar_pago.movil_mercantil'
					WHEN 12 THEN 'venezuela_bolivar_biopago_biopago'
					WHEN 13 THEN 'banesco_bolivar_punto.venta_banesco'
					WHEN 14 THEN 'exterior_bolivar_transferencia_exterior'
					WHEN 15 THEN 'exterior_bolivar_pago.movil_exterior'
					WHEN 16 THEN 'exterior_bolivar_punto.venta_exterior'
					WHEN 17 THEN 'banplus_bolivar_transferencia_banplus'
					WHEN 18 THEN 'banplus_bolivar_pago.movil_banplus'
					WHEN 19 THEN 'banplus_bolivar_punto.venta_banplus'
					WHEN 20 THEN 'bancaribe_bolivar_transferencia_bancaribe'
					WHEN 21 THEN 'bancaribe_bolivar_pago.movil_bancaribe'
					WHEN 22 THEN 'venezuela_bolivar_transferencia_venezuela'
					ELSE 'unknown'
				END AS payment_method,
				CASE WHEN rpu.id IS NOT NULL THEN rpu.monto_bs ELSE rp.monto END as monto_bolivar,
				CASE WHEN rpu.id IS NOT NULL THEN rpu.monto ELSE rp.monto2 END as monto_dolar,
				CASE WHEN rpu.id IS NOT NULL THEN rpu.tasa_cambio ELSE rp.tasa_cambio END as tasa_cambio,
				CASE WHEN rpu.id IS NOT NULL THEN rpu.created_at ELSE rp.created_at END as created_at,
				CASE WHEN rpu.id IS NOT NULL THEN rpu.updated_at ELSE rp.updated_at END as updated_at,
				CASE WHEN cby.client_id IS NOT NULL THEN 1 ELSE rp.created_by END as created_by,
				CASE WHEN uby.client_id IS NOT NULL THEN 1 ELSE rp.updated_by END as updated_by,
				CASE WHEN rp.procesado='' OR rp.procesado IS NULL THEN rp.pre_factura_id ELSE rp.procesado END AS payment_detail,
				CASE WHEN rpu.id IS NOT NULL THEN rpu.url_imagen ELSE rp.url_imagen END as url_file,
				rp.pre_factura_id
			FROM recibo_pago as rp
			LEFT JOIN recibo_pago_user as rpu ON rpu.id=rp.recibo_pago_user_id
			LEFT JOIN sf_guard_user as cby ON cby.id=rp.created_by
			LEFT JOIN sf_guard_user as uby ON uby.id=rp.updated_by
			WHERE rp.created_at>=? AND rp.created_at<=? %s
			GROUP BY rp.id
			ORDER BY rp.created_at ASC
		) as q0
		WHERE q0.estatus=?
		LIMIT 3500
		`, montoPendienteMysql)

	rowsMysql, err := db.ConnMysql.QueryContext(db.Ctx, query, lastRecordDate, endRecordDate, estatusPgsql)
	if err != nil {
		utils.Logline("error on getting recibo_pago from mysql", tipo, err)
		return err
	}
	defer rowsMysql.Close()

	var reciboPagoList []models.ReciboPagovCron
	for rowsMysql.Next() {
		var reciboPago models.ReciboPagovCron
		var rpId, rpuId, urlFile sql.NullString
		if err := rowsMysql.Scan(&rpId, &rpuId, &reciboPago.ClienteOldid, &reciboPago.Estatus, &reciboPago.Fecha, &reciboPago.Referencia, &reciboPago.PaymentMethod,
			&reciboPago.Monto.Bolivar, &reciboPago.Monto.Dolar, &reciboPago.TasaCambio,
			&reciboPago.CreatedAt, &reciboPago.UpdatedAt, &reciboPago.CreatedByOldid, &reciboPago.UpdatedByOldid, &reciboPago.PaymentDetail, &urlFile, &reciboPago.PreFacturaOldid); err != nil {
			utils.Logline("error scanning values of recibo_pago ", "sincReciboPago", tipo, rpId, err)
			return err
		}

		reciboPago.Info = map[string]any{
			"url_file":            urlFile.String,
			"recibo_pago_id":      rpId.String,
			"recibo_pago_user_id": rpuId.String,
			"payment_detail":      "",
		}

		reciboPagoList = append(reciboPagoList, reciboPago)
	}
	rowsMysql.Close()

	// Goroutine handling
	contador := 0
	var wg sync.WaitGroup
	errChan := make(chan error, len(reciboPagoList)) // Buffered channel to collect errors

	// Worker pool size (Adjust for optimal performance)
	const workerPoolSize = 13
	sem := make(chan struct{}, workerPoolSize) // Semaphore to limit concurrency

	for _, reciboPago := range reciboPagoList {
		wg.Add(1)
		sem <- struct{}{} // Limit concurrency

		go func(reciboPago models.ReciboPagovCron) {
			insertReciboPago(db, reciboPago, &wg, errChan)
			<-sem // Release semaphore
		}(reciboPago)

		contador++
	}
	wg.Wait()
	close(errChan)

	//show status of worker
	utils.Logline(fmt.Sprintf("there were (%d) recibo_pago_%s records sincronized", contador, tipo), lastRecordDate, endRecordDate)
	utils.ShowStatusWorkerMysql(db, "sinc_recibo_pagov_"+tipo, caller+"/ending")

	return nil
}

func insertReciboPago(db models.ConnMysqlPgsql, reciboPago models.ReciboPagovCron, wg *sync.WaitGroup, errChan chan error) {
	defer wg.Done()

	createdBy, updatedBy, clienteId, metodoPagoId, rpOldid, err := getReciboInternoIds(db, reciboPago)
	if err != nil {
		utils.Logline("no se pudo insertar este recibo_pago, no se consiguio ids", "sincReciboPago", reciboPago, err)
		utils.Logline(fmt.Sprintf("error getting ids (created_by:%s, updated_by:%s, cliente_id:%s, metodo_pago_id:%s) ", reciboPago.CreatedByOldid, reciboPago.UpdatedByOldid, reciboPago.ClienteOldid, reciboPago.PaymentMethod), err)
		errChan <- err
		return
	}
	// check if id ya esta insertado en postgres
	if rpOldid.Valid {
		return
	}

	var paymentDetalles []models.PaymentResponseDetail2
	if reciboPago.Estatus == "procesado" {
		reciboPago.Estatus = "pendiente"

		if reciboPago.PreFacturaOldid.Valid && len(strings.Split(reciboPago.PaymentDetail.String, ";")) <= 2 {
			var paymentDetail models.PaymentResponseDetail2

			facturaId, facturaCreatedAt, err := getReciboFactura(db, reciboPago.PreFacturaOldid.String)
			if err != nil {
				utils.Logline("error getting factura_id", "sincReciboPago", reciboPago.Info["recibo_pago_id"])
				return
			}

			paymentDetail.Monto.Bolivar = utils.RoundTo8Decimals(reciboPago.Monto.Bolivar)
			paymentDetail.Monto.Dolar = utils.RoundTo8Decimals(reciboPago.Monto.Dolar)

			paymentDetail.Factura.Id = *facturaId
			paymentDetail.Factura.CreatedAt = *facturaCreatedAt

			paymentDetalles = append(paymentDetalles, paymentDetail)
		} else {
			if reciboPago.PaymentDetail.Valid {
				for _, item := range strings.Split(reciboPago.PaymentDetail.String, ";") {
					var paymentDetail models.PaymentResponseDetail2

					pago := strings.Split(item, "|")
					if len(pago) != 2 {
						continue
					}

					montoDolar, err := utils.ParseFloat(pago[1])
					if err != nil {
						utils.Logline("error parsing float ", err, reciboPago.Info["recibo_pago_id"])
						return
					}

					if montoDolar <= 0 {
						continue
					}

					facturaId, facturaCreatedAt, err := getReciboFactura(db, pago[0])
					if err != nil {
						utils.Logline("error getting factura_id", "sincReciboPago", reciboPago.Info["recibo_pago_id"])
						continue
					}

					montoBolivar := montoDolar * reciboPago.TasaCambio

					paymentDetail.Monto.Bolivar = utils.RoundTo8Decimals(montoBolivar)
					paymentDetail.Monto.Dolar = utils.RoundTo8Decimals(montoDolar)

					paymentDetail.Factura.Id = *facturaId
					paymentDetail.Factura.CreatedAt = *facturaCreatedAt

					paymentDetalles = append(paymentDetalles, paymentDetail)
				}
			}
		}

		// if len(paymentDetalles) <= 0 {
		// 	utils.Logline("error no hay detalles de a que prefactura se procesara el pago", reciboPago.Info["recibo_pago_id"], reciboPago.Info["recibo_pago_user_id"])
		// 	return
		// }

		reciboPago.Info["payment_detail"] = paymentDetalles
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Insert query
	var reciboId string
	query := `INSERT INTO venta.recibo_pagov (empresa_id, cliente_id, estatus, fecha, referencia, metodo_pago_id, monto, tasa_cambio, created_at, updated_at, created_by, updated_by, info)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) RETURNING id::text`
	err = db.ConnPgsql.QueryRow(ctx, query, 1, clienteId, reciboPago.Estatus, reciboPago.Fecha, reciboPago.Referencia, metodoPagoId,
		utils.TransformMonedaToArray(reciboPago.Monto), reciboPago.TasaCambio, reciboPago.CreatedAt, reciboPago.UpdatedAt, createdBy, updatedBy, reciboPago.Info).Scan(&reciboId)
	if err != nil {
		utils.Logline("error inserting on venta.recibo_pagov", "sincReciboPago", err, reciboPago.Info["recibo_pago_id"])
		errChan <- err
		return
	}

	if reciboPago.Estatus == "pendiente" && len(paymentDetalles) > 0 {
		if _, err := db.ConnPgsql.Exec(ctx, "UPDATE venta.recibo_pagov SET estatus='procesado' WHERE id=$1 AND created_at=$2", reciboId, reciboPago.CreatedAt); err != nil {
			utils.Logline("error updating to procesado venta.recibo_pagov", "sincReciboPago", reciboId, err, reciboPago.Info["recibo_pago_id"])
			errChan <- err
			return
		}
	}
}

// funciones para reciboPago
func getReciboInternoIds(db models.ConnMysqlPgsql, reciboPago models.ReciboPagovCron) (*int, *int, *int, *int, *sql.NullString, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	items := strings.Split(reciboPago.PaymentMethod, "_")
	if len(items) != 4 {
		return nil, nil, nil, nil, nil, errors.New("payment method incorrect " + reciboPago.PaymentMethod)
	}
	banco := items[0]
	moneda := items[1]
	metodoPago := strings.ReplaceAll(items[2], ".", "_")
	webNombre := items[3]

	query := `SELECT 
		(SELECT id FROM publico.guard_user WHERE info->>'oldid'=$1 LIMIT 1) as created_by, 
		(SELECT id FROM publico.guard_user WHERE info->>'oldid'=$2 LIMIT 1) as updated_by,
		(SELECT id FROM publico.cliente WHERE info->>'oldid'=$3 LIMIT 1) as cliente_id`
	//get fecha of last record on postgres
	var createdBy, updatedBy, clienteId int
	err := db.ConnPgsql.QueryRow(ctx, query, reciboPago.CreatedByOldid, reciboPago.UpdatedByOldid, reciboPago.ClienteOldid).Scan(&createdBy, &updatedBy, &clienteId)
	if err != nil {
		utils.Logline(fmt.Sprintf("error getting ids (created_by:%s, updated_by:%s, cliente_id:%s) ", reciboPago.CreatedByOldid, reciboPago.UpdatedByOldid, reciboPago.ClienteOldid), err)
		return nil, nil, nil, nil, nil, err
	}

	var metodoPagoId int
	query = `SELECT (SELECT id FROM publico.cuenta_banco WHERE banco=$1 AND moneda=$2 AND metodo_pago=$3 AND info->>'web_nombre'=$4 LIMIT 1) as metodo_pago_id`
	err = db.ConnPgsql.QueryRow(ctx, query, banco, moneda, metodoPago, webNombre).Scan(&metodoPagoId)
	if err != nil {
		utils.Logline(fmt.Sprintf("error getting ids (metodo_pago_id: %s) ", reciboPago.PaymentMethod), err)
		return nil, nil, nil, nil, nil, err
	}

	var rpOldid sql.NullString
	query = `SELECT id FROM venta.recibo_pagov WHERE created_at=$1 AND info->>'recibo_pago_id'=$2 LIMIT 1`
	db.ConnPgsql.QueryRow(ctx, query, reciboPago.CreatedAt, reciboPago.Info["recibo_pago_id"]).Scan(&rpOldid)

	return &createdBy, &updatedBy, &clienteId, &metodoPagoId, &rpOldid, nil
}
func getReciboFactura(db models.ConnMysqlPgsql, facturaOldId string) (*string, *string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	query := `SELECT id, created_at::varchar FROM venta.facturav WHERE info->>'prefact_oldid'=$1 LIMIT 1`
	//get fecha of last record on postgres
	var facturaId, facturaCreatedAt string
	err := db.ConnPgsql.QueryRow(ctx, query, facturaOldId).Scan(&facturaId, &facturaCreatedAt)
	if err != nil {
		utils.Logline(fmt.Sprintf("error getting factura_oldid: %s) ", facturaOldId), err)
		return nil, nil, err
	}

	return &facturaId, &facturaCreatedAt, nil
}
