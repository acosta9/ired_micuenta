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

func SincFacturaFiscal(db models.ConnMysqlPgsql, caller string) error {
	//show status of worker
	utils.ShowStatusWorkerMysql(db, "sinc_factura_fiscal", caller+"/begin")

	//get fecha of last record on postgres
	var lastRecordDate, endRecordDate string
	query := `SELECT TO_CHAR(updated_at, 'YYYY-MM-DD HH24:MI:SS') as fecha,
			TO_CHAR(updated_at + INTERVAL '5 MONTH', 'YYYY-MM-DD HH24:MI:SS') as fecha_end
		FROM venta.facturav 
		WHERE tipo IN ('fiscal_maquina', 'fiscal_talonario')
		ORDER BY updated_at DESC 
		LIMIT 1`
	err := db.ConnPgsql.QueryRow(db.Ctx, query).Scan(&lastRecordDate, &endRecordDate)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			utils.Logline("error getting fecha of last record", err)
			return err
		}
		lastRecordDate = "1970-01-01 00:00:00"
		endRecordDate = "2019-11-31 00:00:00"
	}

	// get the last 100records from mysql that have a date greater than the last record of postgres
	query = `SELECT f.id as factura_id, pf.id as pre_factura_id, pf.client_id, f.ncontrol, f.fecha, 0 as dias_credito, 
		CAST(f.subtotal AS DECIMAL(20,8)) as subtotal_dolar,
		CAST(f.subtotal2 AS DECIMAL(20,8)) as subtotal_bolivar,
		0 as desc_porc, 0 as desc_monto_dolar, 0 as desc_monto_bolivar,
		CAST(f.base_imponible AS DECIMAL(20,8)) as baseim_dolar,
		CAST(f.base_imponible2 AS DECIMAL(20,8)) as baseim_bolivar,
		CAST(f.iva AS DECIMAL(20,8)) as iva_porc,
		CAST(f.iva_monto AS DECIMAL(20,8)) as iva_dolar,
		CAST(f.iva_monto2 AS DECIMAL(20,8)) as iva_bolivar,
		CAST(f.igtf AS DECIMAL(20,8)) as igtf_porc,
		CAST(f.igtf_base AS DECIMAL(20,8)) as igtf_baseim_dolar,
		CAST(f.igtf_base2 AS DECIMAL(20,8)) as igtf_baseim_bolivar,
		CAST(f.igtf_monto AS DECIMAL(20,8)) as igtf_monto_dolar,
		CAST(f.igtf_monto2 AS DECIMAL(20,8)) as igtf_monto_bolivar,
		CAST(f.total AS DECIMAL(20,8)) as total_dolar,
		CAST(f.total2 AS DECIMAL(20,8)) as total_bolivar,
		CAST(f.tasa_cambio AS DECIMAL(20,8)) as tasa_cambio,
		CASE 
			WHEN f.num_fact_fiscal IS NOT NULL AND LENGTH(f.num_fact_fiscal)>1 THEN num_fact_fiscal
			ELSE f.num_factura
		END as nfactura,
		CASE 
			WHEN f.num_fact_fiscal IS NOT NULL AND LENGTH(f.num_fact_fiscal)>1 THEN 'fiscal_maquina'
			ELSE 'fiscal_talonario'
		END as tipo_factura,
		CASE 
			WHEN f.anulado = 1 THEN 'anulado'
			WHEN f.pagado = '1' THEN 'pagado'
			WHEN f.pagado = '0' AND (f.monto_pagado+0)=0 THEN 'pendiente'
			ELSE 'abonado'
		END as estatus,
		f.body_json as info,
		f.created_at, f.updated_at, f.created_by, f.updated_by,
		GROUP_CONCAT(
				pfd.id, 'üü', 
				pfd.pre_factura_id, 'üü', 
				COALESCE(pfd.contrato_det_id, ''), 'üü', 
				pfd.qty, 'üü', 
				CAST(pfd.price_unit AS DECIMAL(20,8)), 'üü',
				CAST(pfd.price_tot AS DECIMAL(20,8)), 'üü',
				CAST(pfd.price_unit_bs AS DECIMAL(20,8)), 'üü',
				CAST(pfd.price_tot_bs AS DECIMAL(20,8)), 'üü',
				LOWER(pfd.descripcion)
			ORDER BY pfd.id ASC SEPARATOR '||') as factura_det,
		LOWER(pf.razon_social) as rsocial, LOWER(pf.doc_id) as docid, pf.telf, LOWER(pf.direccion) as direccion, pf.concepto
		FROM factura as f
		LEFT JOIN pre_factura as pf ON pf.id=f.pre_factura_id
		LEFT JOIN pre_factura_det as pfd ON pfd.pre_factura_id=f.pre_factura_id
		WHERE f.updated_at>=? AND f.updated_at<=?
		GROUP BY f.id
		ORDER BY f.updated_at ASC
		LIMIT 4000
		`
	rowsMysql, err := db.ConnMysql.QueryContext(db.Ctx, query, lastRecordDate, endRecordDate)
	if err != nil {
		utils.Logline("error on getting facturas fiscales from mysql", err)
		return err
	}
	defer rowsMysql.Close()

	var facturaList []models.FacturaCron
	for rowsMysql.Next() {
		var factOldId, preFactOldId, conceptoPreFactura, detalleFactura string
		var infoFactura sql.NullString
		var factura models.FacturaCron
		if err := rowsMysql.Scan(&factOldId, &preFactOldId, &factura.ClienteOldid, &factura.NControl, &factura.Fecha, &factura.DiasCredito,
			&factura.SubTotal.Dolar, &factura.SubTotal.Bolivar,
			&factura.DescPorc, &factura.DescMonto.Dolar, &factura.DescMonto.Bolivar,
			&factura.BaseImponible.Dolar, &factura.BaseImponible.Bolivar,
			&factura.IvaPorc, &factura.IvaMonto.Dolar, &factura.IvaMonto.Bolivar,
			&factura.IgtfPorc, &factura.IgtfBase.Dolar, &factura.IgtfBase.Bolivar, &factura.IgtfMonto.Dolar, &factura.IgtfMonto.Bolivar,
			&factura.Total.Dolar, &factura.Total.Bolivar,
			&factura.TasaCambio, &factura.NFactura, &factura.TipoFactura, &factura.Estatus, &infoFactura,
			&factura.CreatedAt, &factura.UpdatedAt, &factura.CreatedByOldid, &factura.UpdatedByOldid, &detalleFactura,
			&factura.RazonSocial, &factura.DocId, &factura.Telefono, &factura.Direccion, &conceptoPreFactura); err != nil {
			utils.Logline("error scanning values of facturas fiscales ", err)
			return err
		}

		factura.Info = map[string]any{
			"origen": "from_scratch",
			"cliente_info": map[string]any{
				"razon_social": factura.RazonSocial,
				"docid":        factura.DocId,
				"telefono":     factura.Telefono,
				"direccion":    factura.Direccion,
			},
			"fact_oldid":      factOldId,
			"prefact_oldid":   preFactOldId,
			"json_maq_fiscal": infoFactura.String,
			"concepto":        conceptoPreFactura,
		}

		factura.DetalleFactura = detalleFactura

		facturaList = append(facturaList, factura)
	}
	rowsMysql.Close()

	contador := 0

	// Goroutine handling
	var wg sync.WaitGroup
	errChan := make(chan error, len(facturaList)) // Buffered channel to collect errors

	// Worker pool size (Adjust for optimal performance)
	const workerPoolSize = 10
	sem := make(chan struct{}, workerPoolSize) // Semaphore to limit concurrency

	for _, factura := range facturaList {
		wg.Add(1)
		sem <- struct{}{} // Limit concurrency

		go func(factura models.FacturaCron) {
			insertFactura(db, "factura", factura, &wg, errChan)
			<-sem // Release semaphore
		}(factura)

		contador++
	}
	wg.Wait()
	close(errChan)

	//show status of worker
	utils.Logline(fmt.Sprintf("there were (%d) factura_fiscales records sincronized", contador), lastRecordDate, endRecordDate)
	utils.ShowStatusWorkerMysql(db, "sinc_factura_fiscal", caller+"/ending")

	return nil
}

func SincPreFactura(db models.ConnMysqlPgsql, caller string, tipo string) error {
	//show status of worker
	utils.ShowStatusWorkerMysql(db, "sinc_prefactura_"+tipo, caller+"/begin")

	var estatusPgsql, anuladoMysql, pagadoMysql, montoPagadoMysql string
	switch tipo {
	case "anulado":
		estatusPgsql = "'anulado'"
		anuladoMysql = "1"
		pagadoMysql = "0,1"
	case "pagado":
		estatusPgsql = "'pagado','abonado'"
		anuladoMysql = "0"
		pagadoMysql = "0,1"
		montoPagadoMysql = "AND (pf.monto_pagado+0)>0"
	}

	//get fecha of last record on postgres
	var lastRecordDate, endRecordDate string
	query := fmt.Sprintf(`SELECT TO_CHAR(updated_at, 'YYYY-MM-DD HH24:MI:SS') as fecha,
			TO_CHAR(updated_at + INTERVAL '5 MONTH', 'YYYY-MM-DD HH24:MI:SS') as fecha_end
		FROM venta.facturav 
		WHERE tipo='nota' AND estatus IN (%s)
		ORDER BY updated_at DESC 
		LIMIT 1`, estatusPgsql)
	err := db.ConnPgsql.QueryRow(db.Ctx, query).Scan(&lastRecordDate, &endRecordDate)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			utils.Logline("error getting fecha of last record", "sincPreFactura", tipo, err)
			return err
		}
		lastRecordDate = "1970-01-01 00:00:00"
		endRecordDate = "2019-11-31 00:00:00"
	}

	// get the last records from mysql that have a date greater than the last record of postgres
	query = fmt.Sprintf(`SELECT pf.id as pre_factura_id, pf.client_id, pf.fecha, 0 as dias_credito, 
		CAST(pf.subtotal AS DECIMAL(20,8)) as total_dolar, 
		0 as desc_porc, 0 as desc_monto_dolar, 0 as desc_monto_bolivar,
		COALESCE(pf.concepto, '') as concepto,
		'nota' as tipo_factura,
		CASE 
			WHEN pf.anulado = 1 THEN 'anulado'
			WHEN pf.pagado = '1' THEN 'pagado'
			WHEN pf.pagado = '0' AND (pf.monto_pagado+0)=0 THEN 'pendiente'
			ELSE 'abonado'
		END as estatus,
		pf.created_at, pf.updated_at, pf.created_by, pf.updated_by,
		GROUP_CONCAT(
				pfd.id, 'üü', 
				pfd.pre_factura_id, 'üü', 
				COALESCE(pfd.contrato_det_id, ''), 'üü', 
				pfd.qty, 'üü', 
				CAST(pfd.price_unit AS DECIMAL(20,8)), 'üü',
				CAST(pfd.price_tot AS DECIMAL(20,8)), 'üü',
				CAST(COALESCE(pfd.price_unit_bs, 0) AS DECIMAL(20,8)), 'üü',
				CAST(COALESCE(pfd.price_tot_bs, 0) AS DECIMAL(20,8)), 'üü',
				LOWER(pfd.descripcion)
			ORDER BY pfd.id ASC SEPARATOR '||'
		) as factura_det,
		LOWER(pf.razon_social) as rsocial, LOWER(pf.doc_id) as docid, pf.telf, LOWER(pf.direccion) as direccion
		FROM pre_factura as pf 
		LEFT JOIN factura as f ON f.pre_factura_id=pf.id
		LEFT JOIN pre_factura_det as pfd ON pfd.pre_factura_id=pf.id
		WHERE pf.updated_at>=? AND pf.updated_at<=? AND f.id IS NULL AND pf.anulado=? AND pf.pagado IN (%s) %s
		GROUP BY pf.id
		ORDER BY pf.updated_at ASC
		LIMIT 4000
		`, pagadoMysql, montoPagadoMysql)
	rowsMysql, err := db.ConnMysql.QueryContext(db.Ctx, query, lastRecordDate, endRecordDate, anuladoMysql)
	if err != nil {
		utils.Logline("error on getting pre_facturas from mysql", tipo, err)
		return err
	}
	defer rowsMysql.Close()

	var facturaList []models.FacturaCron
	for rowsMysql.Next() {
		var conceptoPreFactura, detalleFactura string
		var preFactOldId int
		var factura models.FacturaCron
		if err := rowsMysql.Scan(&preFactOldId, &factura.ClienteOldid, &factura.Fecha, &factura.DiasCredito,
			&factura.Total.Dolar, &factura.DescPorc, &factura.DescMonto.Dolar, &factura.DescMonto.Bolivar,
			&conceptoPreFactura, &factura.TipoFactura, &factura.Estatus,
			&factura.CreatedAt, &factura.UpdatedAt, &factura.CreatedByOldid, &factura.UpdatedByOldid, &detalleFactura,
			&factura.RazonSocial, &factura.DocId, &factura.Telefono, &factura.Direccion); err != nil {
			utils.Logline("error scanning values of pre_factura ", "sincPreFactura", tipo, preFactOldId, err)
			return err
		}

		createdAt := utils.StringToTime(factura.CreatedAt)
		if createdAt == nil {
			utils.Logline("error transforming created_at string to time time")
			return fmt.Errorf("error transforming created_at string to time time")
		}
		factura.NFactura = utils.GenerateNfacturaForPrefactura(*createdAt, factura.ClienteOldid, preFactOldId)

		factura.Info = map[string]any{
			"origen": "from_scratch",
			"cliente_info": map[string]any{
				"razon_social": factura.RazonSocial,
				"docid":        factura.DocId,
				"telefono":     factura.Telefono,
				"direccion":    factura.Direccion,
			},
			"prefact_oldid": preFactOldId,
			"concepto":      conceptoPreFactura,
		}

		factura.DetalleFactura = detalleFactura

		facturaList = append(facturaList, factura)
	}
	rowsMysql.Close()

	// Goroutine handling
	contador := 0
	var wg sync.WaitGroup
	errChan := make(chan error, len(facturaList)) // Buffered channel to collect errors

	// Worker pool size (Adjust for optimal performance)
	const workerPoolSize = 10
	sem := make(chan struct{}, workerPoolSize) // Semaphore to limit concurrency

	for _, factura := range facturaList {
		wg.Add(1)
		sem <- struct{}{} // Limit concurrency

		go func(factura models.FacturaCron) {
			insertFactura(db, "pre_factura", factura, &wg, errChan)
			<-sem // Release semaphore
		}(factura)

		contador++
	}
	wg.Wait()
	close(errChan)

	//show status of worker
	utils.Logline(fmt.Sprintf("there were (%d) pre_factura_%s records sincronized", contador, tipo), lastRecordDate, endRecordDate)
	utils.ShowStatusWorkerMysql(db, "sinc_prefactura_"+tipo, caller+"/ending")

	return nil
}

func SincRetenciones(db models.ConnMysqlPgsql, caller string) error {
	//show status of worker
	utils.ShowStatusWorkerMysql(db, "sinc_retenciones", caller+"/begin")

	//get fecha of last record on postgres
	var lastRecordDate, endRecordDate string
	query := `SELECT TO_CHAR(updated_at, 'YYYY-MM-DD HH24:MI:SS') as fecha,
			TO_CHAR(updated_at + INTERVAL '5 MONTH', 'YYYY-MM-DD HH24:MI:SS') as fecha_end
		FROM venta.facturav_retencion
		ORDER BY updated_at DESC 
		LIMIT 1`
	err := db.ConnPgsql.QueryRow(db.Ctx, query).Scan(&lastRecordDate, &endRecordDate)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			utils.Logline("error getting fecha of last record", err)
			return err
		}
		lastRecordDate = "1970-01-01 00:00:00"
		endRecordDate = "2019-11-31 00:00:00"
	}

	// get the last 3000 records from mysql that have a date greater than the last record of postgres
	query = `SELECT q0.retencion_id, q0.factura_id, q0.factura_created_at, q0.fecha, q0.comprobante,
	 	q0.url_imagen, q0.descripcion,
		q0.monto_retenido_dolar, q0.monto_retenido_bolivar,
		CAST(q0.base_imponible_dolar AS DECIMAL(20,8)) as base_imponible_dolar,
		CAST(q0.base_imponible_bolivar AS DECIMAL(20,8)) as base_imponible_bolivar,
		CAST(q0.monto_retenido_bolivar*100/q0.base_imponible_bolivar AS DECIMAL(5,2)) as porcentaje_retencion,
		q0.tipo_retencion, q0.estatus, q0.created_at, q0.updated_at, q0.created_by, q0.updated_by
		FROM (
			SELECT r.id as retencion_id, r.factura_id as factura_id, f.created_at as factura_created_at,
				r.fecha, r.comprobante, r.url_imagen, LOWER(r.descripcion) as descripcion,
				CAST(r.monto AS DECIMAL(20,8)) as monto_retenido_bolivar,
				CAST(r.monto/f.tasa_cambio AS DECIMAL(20,8)) as monto_retenido_dolar,
				CASE
					WHEN r.tipo IN (1,2) THEN r.base_imponible
					ELSE r.iva_impuesto
				END as base_imponible_bolivar,
				CASE
					WHEN r.tipo IN (1,2) THEN r.base_imponible/f.tasa_cambio
					ELSE r.iva_impuesto/f.tasa_cambio
				END as base_imponible_dolar,
				CASE 
					WHEN r.tipo= 1 THEN 'islr'
					WHEN r.tipo= 2 THEN 'im'
					ELSE 'iva'
				END as tipo_retencion,
				CASE 
					WHEN r.anulado = 1 THEN 'anulado'
					ELSE 'procesado'
				END as estatus,
				r.created_at, r.updated_at, r.created_by, r.updated_by
			FROM retenciones as r
			LEFT JOIN factura as f ON f.id=r.factura_id
			WHERE r.updated_at>=? AND r.updated_at<=?
			ORDER BY r.updated_at ASC
			LIMIT 1500
		) as q0
		`
	rowsMysql, err := db.ConnMysql.QueryContext(db.Ctx, query, lastRecordDate, endRecordDate)
	if err != nil {
		utils.Logline("error on getting facturas fiscales from mysql", err)
		return err
	}
	defer rowsMysql.Close()

	var retencionList []models.RetencionCron
	for rowsMysql.Next() {
		var retencionOldId, factOldId, createdByMysql, updatedByMysql string
		var retencion models.RetencionCron
		if err := rowsMysql.Scan(&retencionOldId, &factOldId, &retencion.FacturavCreatedAt, &retencion.FechaRetencion,
			&retencion.NComprobante, &retencion.UrlFile, &retencion.Descripcion,
			&retencion.MontoRetenido.Dolar, &retencion.MontoRetenido.Bolivar,
			&retencion.BaseImponible.Dolar, &retencion.BaseImponible.Bolivar,
			&retencion.PorcentajeRetencion, &retencion.TipoRetencion, &retencion.Estatus,
			&retencion.CreatedAt, &retencion.UpdatedAt, &createdByMysql, &updatedByMysql); err != nil {
			utils.Logline("error scanning values of retenciones", err)
			return err
		}

		infoData := map[string]any{
			"oldid":       retencionOldId,
			"descripcion": retencion.Descripcion.String,
			"url_file":    retencion.UrlFile.String,
		}
		retencion.Info = infoData

		infoDataOld := map[string]any{
			"created_by":           createdByMysql,
			"updated_by":           updatedByMysql,
			"factura_id":           factOldId,
			"factura_created_at":   retencion.FacturavCreatedAt,
			"retencion_id":         retencionOldId,
			"retencion_created_at": retencion.CreatedAt,
		}
		retencion.InfoOld = infoDataOld

		retencionList = append(retencionList, retencion)
	}
	rowsMysql.Close()

	// Goroutine handling
	contador := 0
	var wg sync.WaitGroup
	errChan := make(chan error, len(retencionList)) // Buffered channel to collect errors

	// Worker pool size (Adjust for optimal performance)
	const workerPoolSize = 10
	sem := make(chan struct{}, workerPoolSize) // Semaphore to limit concurrency

	for _, retencion := range retencionList {
		wg.Add(1)
		sem <- struct{}{} // Limit concurrency

		go func(retencion models.RetencionCron) {
			insertRetencion(db, retencion, &wg, errChan)
			<-sem // Release semaphore
		}(retencion)

		contador++
	}
	wg.Wait()
	close(errChan)

	//show status of worker
	utils.Logline(fmt.Sprintf("there were (%d) retenciones records sincronized", contador), lastRecordDate, endRecordDate)
	utils.ShowStatusWorkerMysql(db, "sinc_retenciones", caller+"/ending")

	return nil
}

// funciones para factura
func getFacturaInternoIds(db models.ConnMysqlPgsql, createdByOldId string, updatedByOldid string, clienteOldid string, facturaOldid string, facturaCreatedAt string) (*int, *int, *int, *sql.NullString, error) {
	query := `SELECT 
		(SELECT id FROM publico.guard_user WHERE info->>'oldid'=$1 LIMIT 1) as created_by, 
		(SELECT id FROM publico.guard_user WHERE info->>'oldid'=$2 LIMIT 1) as updated_by,
		(SELECT id FROM publico.cliente WHERE info->>'oldid'=$3 LIMIT 1) as cliente_id,
		(SELECT id FROM venta.facturav WHERE created_at=$4 AND info->>'fact_oldid'=$5 LIMIT 1) as factura_id`

	//get fecha of last record on postgres
	var createdBy, updatedBy, clienteId int
	var facturaId sql.NullString
	err := db.ConnPgsql.QueryRow(db.Ctx, query, createdByOldId, updatedByOldid, clienteOldid, facturaCreatedAt, facturaOldid).Scan(&createdBy, &updatedBy, &clienteId, &facturaId)
	if err != nil {
		utils.Logline(fmt.Sprintf("error getting ids (created_by:%s, updated_by:%s, cliente_id:%s) ", createdByOldId, updatedByOldid, clienteOldid), err)
		return nil, nil, nil, nil, err
	}

	return &createdBy, &updatedBy, &clienteId, &facturaId, nil
}
func parseFacturaDetalle(db models.ConnMysqlPgsql, groupConcatStr string) ([]map[string]any, error) {
	var detalles []map[string]any

	// Split entries by " | "
	entries := strings.Split(groupConcatStr, "||")
	for _, entry := range entries {
		parts := strings.Split(entry, "üü")
		if len(parts) == 9 {
			var err error
			var suscripcion *models.SuscripcionShortInfo
			if len(parts[2]) > 0 {
				suscripcion, err = getSuscripcionByOldid(db, parts[2])
				if err != nil {
					return nil, err
				}
			}

			concepto := utils.RemoveHTMLTags(parts[8])
			concepto = strings.ReplaceAll(concepto, "\"", " ")

			infoStruct := map[string]any{
				"oldid":         parts[0],
				"prefact_oldid": parts[1],
				"concepto":      concepto,
				"suscripcion":   suscripcion,
			}

			var qty, priceUnitDolar, priceUnitBs, priceTotDolar, priceTotBs float64
			if qty, err = utils.ParseFloat(parts[3]); err != nil {
				utils.Logline("error parsing qty to float", err)
				return nil, err
			}
			if priceUnitDolar, err = utils.ParseFloat(parts[4]); err != nil {
				utils.Logline("error parsing price_unit_dolar to float", err)
				return nil, err
			}
			if priceTotDolar, err = utils.ParseFloat(parts[5]); err != nil {
				utils.Logline("error parsing price_tot_dolar to float", err)
				return nil, err
			}
			if priceUnitBs, err = utils.ParseFloat(parts[6]); err != nil {
				utils.Logline("error parsing price_unit_bs to float", err)
				return nil, err
			}
			if priceTotBs, err = utils.ParseFloat(parts[7]); err != nil {
				utils.Logline("error parsing price_tot_bs to float", err)
				return nil, err
			}

			detalle := map[string]any{
				"tax_status": "gravable",
				"qty":        qty,
				"price_unit": fmt.Sprintf("%.8f,%.8f", priceUnitDolar, priceUnitBs),
				"price_tot":  fmt.Sprintf("%.8f,%.8f", priceTotDolar, priceTotBs),
				"info":       infoStruct,
			}

			detalles = append(detalles, detalle)
		} else {
			utils.Logline("error on len of parts", len(parts), entry)
			return nil, fmt.Errorf("error on len of parts")
		}
	}

	return detalles, nil
}
func insertFactura(db models.ConnMysqlPgsql, tipoFact string, factura models.FacturaCron, wg *sync.WaitGroup, errChan chan error) {
	defer wg.Done()

	// Parse details
	var createdBy, updatedBy, clienteId *int
	var facturaId *sql.NullString
	var facturaDetalles []map[string]any
	var err error
	if tipoFact == "factura" {
		if facturaDetalles, err = parseFacturaDetalle(db, factura.DetalleFactura); err != nil {
			utils.Logline("error parsing data for venta.facturav_det", "sincFactura", err, factura)
			errChan <- fmt.Errorf("error parsing data for venta.facturav_det")
			return
		}

		createdBy, updatedBy, clienteId, facturaId, err = getFacturaInternoIds(db, factura.CreatedByOldid, factura.UpdatedByOldid, factura.ClienteOldid, factura.Info["fact_oldid"].(string), factura.CreatedAt)
		if err != nil {
			utils.Logline("no se pudo insertar esta factura, no se consiguio userId of created_by", err)
			errChan <- err
			return
		}
		if facturaId.Valid {
			return
		}

	} else {
		if facturaDetalles, err = parsePreFactDetalle(db, factura.DetalleFactura, factura.TasaCambio); err != nil {
			utils.Logline("error parsing data for venta.facturav_det", "sincPreFactura", err, factura)
			errChan <- fmt.Errorf("error parsing data for venta.facturav_det")
			return
		}

		var tasaCambio float64
		createdBy, updatedBy, clienteId, tasaCambio, facturaId, err = getPreFactInternoIds(db, factura.CreatedByOldid, factura.UpdatedByOldid, factura.ClienteOldid, factura.CreatedAt, factura.Info["prefact_oldid"].(int))
		if err != nil {
			utils.Logline("no se pudo insertar esta pre_factura, no se consiguio userId of created_by", err)
			errChan <- err
			return
		}

		if facturaId.Valid {
			return
		}

		factura.TasaCambio = tasaCambio
		factura.Total.Bolivar = factura.Total.Dolar * tasaCambio
		factura.SubTotal.Dolar = factura.Total.Dolar / 1.16
		factura.SubTotal.Bolivar = factura.Total.Bolivar / 1.16
		factura.BaseImponible.Dolar = factura.SubTotal.Dolar
		factura.BaseImponible.Bolivar = factura.SubTotal.Bolivar
		factura.IvaMonto.Dolar = factura.BaseImponible.Dolar * 0.16
		factura.IvaMonto.Bolivar = factura.BaseImponible.Bolivar * 0.16
		factura.IvaPorc = 16
		factura.IgtfPorc = 0
		factura.IgtfBase.Dolar = 0
		factura.IgtfBase.Bolivar = 0
		factura.IgtfMonto.Dolar = 0
		factura.IgtfMonto.Bolivar = 0

	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Insert query
	query := `SELECT venta.insert_factura($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)`
	_, err = db.ConnPgsql.Exec(ctx, query,
		1, clienteId, factura.NControl, factura.NFactura, factura.Fecha, factura.TipoFactura, factura.Estatus, factura.DiasCredito,
		utils.TransformMonedaToArray(factura.SubTotal), factura.DescPorc, utils.TransformMonedaToArray(factura.DescMonto),
		utils.TransformMonedaToArray(factura.BaseImponible), factura.IvaPorc, utils.TransformMonedaToArray(factura.IvaMonto),
		factura.IgtfPorc, utils.TransformMonedaToArray(factura.IgtfBase), utils.TransformMonedaToArray(factura.IgtfMonto),
		utils.TransformMonedaToArray(factura.Total), factura.TasaCambio,
		factura.CreatedAt, factura.UpdatedAt, createdBy, updatedBy, factura.Info, facturaDetalles)
	if err != nil {
		utils.Logline("error inserting on venta.facturav", "sincPreFactura", err)
		errChan <- err
		return
	}
}

// funciones para prefactura
func getPreFactInternoIds(db models.ConnMysqlPgsql, createdByOldId string, updatedByOldid string, clienteOldid string, createdAt string, preFactOldid int) (*int, *int, *int, float64, *sql.NullString, error) {
	query := `SELECT 
		(SELECT id FROM publico.guard_user WHERE info->>'oldid'=$1 LIMIT 1) as created_by, 
		(SELECT id FROM publico.guard_user WHERE info->>'oldid'=$2 LIMIT 1) as updated_by,
		(SELECT id FROM publico.cliente WHERE info->>'oldid'=$3 LIMIT 1) as cliente_id,
		(SELECT monto FROM publico.tasa_cambio WHERE created_at<=$4 ORDER BY created_at DESC LIMIT 1) as tasa,
		(SELECT id FROM venta.facturav WHERE created_at=$4 AND info->>'prefact_oldid'=$5 LIMIT 1) as factura_id`

	//get fecha of last record on postgres
	var createdBy, updatedBy, clienteId int
	var tasaCambio float64
	var facturaId sql.NullString
	err := db.ConnPgsql.QueryRow(db.Ctx, query, createdByOldId, updatedByOldid, clienteOldid, createdAt, utils.IntToString(preFactOldid)).Scan(&createdBy, &updatedBy, &clienteId, &tasaCambio, &facturaId)
	if err != nil {
		utils.Logline(fmt.Sprintf("error getting ids (created_by:%s, updated_by:%s, cliente_id:%s, created_at:%s) ", createdByOldId, updatedByOldid, clienteOldid, createdAt), err)
		return nil, nil, nil, 0, nil, err
	}

	return &createdBy, &updatedBy, &clienteId, tasaCambio, &facturaId, nil
}
func parsePreFactDetalle(db models.ConnMysqlPgsql, groupConcatStr string, tasaCambio float64) ([]map[string]any, error) {
	var detalles []map[string]any

	// Split entries by " | "
	entries := strings.Split(groupConcatStr, "||")
	for _, entry := range entries {
		parts := strings.Split(entry, "üü")
		if len(parts) == 9 {
			var err error
			var suscripcion *models.SuscripcionShortInfo
			if len(parts[2]) > 0 {
				suscripcion, err = getSuscripcionByOldid(db, parts[2])
				if err != nil {
					return nil, err
				}
			}

			concepto := utils.RemoveHTMLTags(parts[8])
			concepto = strings.ReplaceAll(concepto, "\"", " ")

			infoStruct := map[string]any{
				"oldid":         parts[0],
				"prefact_oldid": parts[1],
				"concepto":      concepto,
				"suscripcion":   suscripcion,
			}

			var qty, priceUnit, priceTot float64
			if qty, err = utils.ParseFloat(parts[3]); err != nil {
				utils.Logline("error parsing qty to float", err)
				return nil, err
			}
			if priceUnit, err = utils.ParseFloat(parts[4]); err != nil {
				utils.Logline("error parsing price_unit_dolar to float", err)
				return nil, err
			}
			if priceTot, err = utils.ParseFloat(parts[5]); err != nil {
				utils.Logline("error parsing price_tot_dolar to float", err)
				return nil, err
			}

			detalle := map[string]any{
				"tax_status": "gravable",
				"qty":        qty,
				"price_unit": fmt.Sprintf("%.8f,%.8f", priceUnit/1.16, (priceUnit/1.16)*tasaCambio),
				"price_tot":  fmt.Sprintf("%.8f,%.8f", priceTot/1.16, (priceTot/1.16)*tasaCambio),
				"info":       infoStruct,
			}

			detalles = append(detalles, detalle)
		} else {
			utils.Logline("error on len of parts", len(parts), entry)
			return nil, fmt.Errorf("error on len of parts")
		}
	}

	return detalles, nil
}
func getSuscripcionByOldid(db models.ConnMysqlPgsql, suscripcionOldid string) (*models.SuscripcionShortInfo, error) {
	query := `SELECT s.id as id, s.info->>'oldid', regexp_replace(sv.nombre, '[^0-9]', '', 'g')::integer as speed_value, 'Mbps' as speed_unit,
			SPLIT_PART(st.nombre,'/',1) as zona, SPLIT_PART(st.nombre,'/',2) as tipo_conexion, SPLIT_PART(st.nombre,'/',3) as tipo_servicio
		FROM administracion.suscripcion as s
		LEFT JOIN administracion.servicio as sv ON sv.id=s.servicio_id
		LEFT JOIN administracion.servicio_tipo as st ON st.id=sv.servicio_tipo_id
		WHERE s.info->>'oldid' = $1
		LIMIT 1`

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var susc models.SuscripcionShortInfo
	if err := db.ConnPgsql.QueryRow(ctx, query, suscripcionOldid).Scan(&susc.Id, &susc.Oldid, &susc.SpeedValue, &susc.SpeedUnit, &susc.Zona, &susc.TipoConexion, &susc.TipoServicio); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		utils.Logline("error on select suscripcion from cliente", suscripcionOldid, err)
		return nil, err
	}

	return &susc, nil
}

// funciones para retenciones
func getRetencionInternoIds(db models.ConnMysqlPgsql, createdByOldId string, updatedByOldid string, factOldId string, facturaCreatedAt string, retencionOldId string, retencionCreatedAt string) (*int, *int, *sql.NullString, *sql.NullString, error) {
	query := `SELECT 
		(SELECT id FROM publico.guard_user WHERE info->>'oldid'=$1 LIMIT 1) as created_by, 
		(SELECT id FROM publico.guard_user WHERE info->>'oldid'=$2 LIMIT 1) as updated_by,
		(SELECT id FROM venta.facturav WHERE created_at=$3 AND info->>'fact_oldid'=$4 LIMIT 1) as factura_id,
		(SELECT id FROM venta.facturav_retencion WHERE created_at=$5 AND info->>'oldid'=$6 LIMIT 1) as retencion_id`

	//get fecha of last record on postgres
	var createdBy, updatedBy int
	var facturaId sql.NullString
	var retencionId sql.NullString
	err := db.ConnPgsql.QueryRow(db.Ctx, query, createdByOldId, updatedByOldid, facturaCreatedAt, factOldId, retencionCreatedAt, retencionOldId).Scan(&createdBy, &updatedBy, &facturaId, &retencionId)
	if err != nil {
		utils.Logline("error getting ids ", err)
		return nil, nil, nil, nil, err
	}

	return &createdBy, &updatedBy, &facturaId, &retencionId, nil
}
func insertRetencion(db models.ConnMysqlPgsql, retencion models.RetencionCron, wg *sync.WaitGroup, errChan chan error) {
	defer wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// validar si retencion ya existe en postgresql asi como ids de postgres
	createdBy, updatedBy, facturaId, retencionId, err := getRetencionInternoIds(db, retencion.InfoOld["created_by"].(string), retencion.InfoOld["updated_by"].(string),
		retencion.InfoOld["factura_id"].(string), retencion.InfoOld["factura_created_at"].(string), retencion.InfoOld["retencion_id"].(string), retencion.InfoOld["retencion_created_at"].(string))
	if err != nil {
		utils.Logline("no se pudo insertar esta retencion, no se consiguio ids", "sincRetenciones", err)
		errChan <- err
	}
	if retencionId.Valid || !facturaId.Valid {
		return
	}

	retencion.CreatedBy = *createdBy
	retencion.UpdatedBy = *updatedBy
	retencion.FacturavId = facturaId.String

	queryInternal := `SELECT venta.insert_facturav_retencion($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`
	_, err = db.ConnPgsql.Exec(ctx, queryInternal,
		1, retencion.FacturavId, retencion.FacturavCreatedAt, retencion.TipoRetencion,
		utils.TransformMonedaToArray(retencion.MontoRetenido), utils.TransformMonedaToArray(retencion.BaseImponible),
		retencion.PorcentajeRetencion, retencion.FechaRetencion, retencion.NComprobante, retencion.Estatus,
		retencion.CreatedAt, retencion.UpdatedAt, retencion.CreatedBy, retencion.UpdatedBy, retencion.Info)
	if err != nil {
		utils.Logline("error inserting on venta.facturav_retencion", "sincRetenciones", err, retencion, retencion.PorcentajeRetencion)
		errChan <- err
		return
	}
}
