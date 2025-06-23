package repo

import (
	"database/sql"
	"encoding/json"
	"errors"
	"html"
	"net/http"

	"ired.com/micuenta/models"
	"ired.com/micuenta/utils"
)

func GetFactura(db models.ConnDb, clienteId string, facturaReq models.FacturaReqId) (*models.FacturaResponse, int, error) {
	query := `SELECT fv.id, fv.estatus, fv.created_at, 
			fv.subtotal[1] as subtotal_dolar, fv.subtotal[2] as subtotal_bolivar, 
			fv.desc_porc, fv.desc_monto[1] as desc_monto_dolar, fv.desc_monto[2] as desc_monto_bolivar, 
			fv.base_imp[1] as base_imp_dolar, fv.base_imp[2] as base_imp_bolivar, 
			fv.iva_porc, fv.iva_monto[1] as iva_monto_dolar, fv.iva_monto[2] as iva_monto_bolivar,
			fv.igtf_porc, fv.igtf_baseim[1] as igtf_base_dolar, fv.igtf_baseim[2] as igtf_base_bolivar, fv.igtf_monto[1] as igtf_monto_dolar, fv.igtf_monto[2] as igtf_monto_bolivar,
			fv.total[1] as total_dolar, fv.total[2] as total_bolivar,
			fv.info->'cliente_info'->>'docid' as docid, fv.info->'cliente_info'->>'razon_social' as razon_social, fv.info->'cliente_info'->>'direccion' as direccion, fv.info->'cliente_info'->>'telefono' as telefono
		FROM venta.facturav as fv
		WHERE fv.cliente_id=$1 AND DATE(fv.created_at)=DATE($2) AND fv.id=$3`

	var factura models.FacturaResponse

	err := db.ConnPgsql.QueryRow(db.Ctx, query, clienteId, facturaReq.CreatedAt, facturaReq.Id).Scan(&factura.Id, &factura.Estatus, &factura.CreatedAt,
		&factura.SubTotal.Dolar, &factura.SubTotal.Bolivar,
		&factura.DescPorc, &factura.DescMonto.Dolar, &factura.DescMonto.Bolivar,
		&factura.BaseImponible.Dolar, &factura.BaseImponible.Bolivar,
		&factura.IvaPorc, &factura.IvaMonto.Dolar, &factura.IvaMonto.Bolivar,
		&factura.IgtfPorc, &factura.IgtfBase.Dolar, &factura.IgtfBase.Bolivar, &factura.IgtfMonto.Dolar, &factura.IgtfMonto.Bolivar,
		&factura.Total.Dolar, &factura.Total.Bolivar,
		&factura.DocId, &factura.RazonSocial, &factura.Direccion, &factura.Telefono)
	if err != nil {
		utils.Logline("error getting venta.facturav", err, clienteId, facturaReq)
		return nil, http.StatusBadRequest, errors.New("errorGetData")
	}

	//get detalle de la factura
	facturaDets, err := GetFacturaDet(db, facturaReq)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	//get retenciones de la factura
	retencionList, err := GetRetencionFactura(db, facturaReq)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	//get recibos de la factura
	paymentList, err := GetPaymentFactura(db, clienteId, facturaReq)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	factura.NumReferencia = utils.GenerateNcontrolByUuid(factura.Id)
	factura.FacturaDet = *facturaDets
	factura.Retenciones = *retencionList
	factura.Payments = *paymentList
	factura.DatosBesser = models.GetDatosBesser()

	return &factura, http.StatusOK, nil
}

func FacturaList(db models.ConnDb, userId any, pageQuery models.PaginatorQuery) (*[]models.FacturaList, *models.PaginatorData, error) {
	currentPage := pageQuery.Page
	limit := pageQuery.Limit
	offset := (currentPage - 1) * limit

	//get meta of paginator
	var totalCount int
	if err := db.ConnPgsql.QueryRow(db.Ctx, "SELECT COUNT(*) FROM venta.facturav WHERE cliente_id=$1", userId).Scan(&totalCount); err != nil {
		utils.Logline("error on query count", err)
		return nil, nil, errors.New("errorGetData")
	}
	paginatorData := models.GetPaginatorMeta(currentPage, limit, totalCount)

	//validate if current page is possible to offset
	if currentPage > paginatorData.TotalPages {
		return nil, nil, errors.New("errorPage")
	}

	query := `SELECT fv.id as payment_id, fv.estatus, fv.total[1] as tot_dolar, fv.total[2] as tot_bolivar, fv.created_at
		FROM venta.facturav as fv
		WHERE fv.cliente_id=$1
		ORDER BY fv.created_at DESC 
		LIMIT $2 
		OFFSET $3`
	rows, err := db.ConnPgsql.Query(db.Ctx, query, userId, limit, offset)
	if err != nil {
		utils.Logline("error on select venta.facturav", err)
		return nil, nil, errors.New("errorGetData")
	}
	defer rows.Close()

	var facturaList []models.FacturaList
	for rows.Next() {
		var factura models.FacturaList
		err = rows.Scan(&factura.Id, &factura.Estatus, &factura.Total.Dolar, &factura.Total.Bolivar, &factura.CreatedAt)
		if err != nil {
			utils.Logline("error scanning venta.facturav", err)
			return nil, nil, errors.New("errorGetData")
		}

		factura.NumReferencia = utils.GenerateNcontrolByUuid(factura.Id)
		facturaList = append(facturaList, factura)
	}
	rows.Close()

	return &facturaList, &paginatorData, err
}

func GetFacturaDet(db models.ConnDb, facturaReqId models.FacturaReqId) (*[]models.FacturaDetResponse, error) {
	query := `SELECT qty, price_unit[1] as price_unit_dolar, price_unit[2] as price_unit_bolivar, price_tot[1] as price_tot_dolar, price_tot[2] as price_tot_bolivar,
			tax_status, info->>'concepto' as concepto, info->>'suscripcion' as suscripcion
		FROM venta.facturav_det
		WHERE created_at=$1 AND facturav_id=$2`

	rows, err := db.ConnPgsql.Query(db.Ctx, query, facturaReqId.CreatedAt, facturaReqId.Id)
	if err != nil {
		utils.Logline("error on select venta.facturav_det", err, facturaReqId)
		return nil, errors.New("errorGetData")
	}
	defer rows.Close()

	var facturaDets []models.FacturaDetResponse
	for rows.Next() {
		var facturaDet models.FacturaDetResponse
		var concepto string
		var detalleJson sql.NullString
		err = rows.Scan(&facturaDet.Qty, &facturaDet.PriceUnit.Dolar, &facturaDet.PriceUnit.Bolivar, &facturaDet.PriceTot.Dolar, &facturaDet.PriceTot.Bolivar,
			&facturaDet.TaxStatus, &concepto, &detalleJson)
		if err != nil {
			utils.Logline("error scanning venta.facturav_det", err)
			return nil, errors.New("errorGetData")
		}

		if detalleJson.Valid {
			if err := json.Unmarshal([]byte(detalleJson.String), &facturaDet.Suscripcion); err != nil {
				utils.Logline("error parsing json to struct", err, facturaReqId)
				return nil, errors.New("errorGetData")
			}
		}

		facturaDet.Concepto = html.UnescapeString(concepto)
		facturaDets = append(facturaDets, facturaDet)
	}

	return &facturaDets, nil
}

func GetFacturaPayment(db models.ConnDb, clienteId string, paymentReq models.PaymentReqId) (*[]models.FacturaList, error) {
	query := `WITH det AS (
	  SELECT (payment->'factura'->>'id')::UUID as facturav_id, (payment->'factura'->>'created_at')::TIMESTAMPTZ as facturav_createdat, 
			(payment->'monto'->>'dolar')::DECIMAL(20,8) as monto_dolar, (payment->'monto'->>'bolivar')::DECIMAL(20,8) as monto_bolivar
		FROM venta.recibo_pagov, jsonb_array_elements(info->'payment_detail') AS payment
		WHERE id=$1 and created_at=$2
	)
	
	SELECT fv.id as payment_id, fv.estatus, fv.total[1] as tot_dolar, fv.total[2] as tot_bolivar, fv.created_at, det.monto_dolar, det.monto_bolivar
		FROM venta.facturav as fv
		LEFT JOIN det ON det.facturav_id=fv.id AND det.facturav_createdat=fv.created_at
		WHERE fv.cliente_id=$3 AND det.facturav_id IS NOT NULL
		ORDER BY fv.created_at DESC `
	rows, err := db.ConnPgsql.Query(db.Ctx, query, paymentReq.PaymentId, paymentReq.CreatedAt, clienteId)
	if err != nil {
		utils.Logline("error on select venta.facturav", err)
		return nil, errors.New("errorGetData")
	}
	defer rows.Close()

	var facturaList []models.FacturaList
	for rows.Next() {
		var factura models.FacturaList
		var detalleRecibo models.Moneda
		err = rows.Scan(&factura.Id, &factura.Estatus, &factura.Total.Dolar, &factura.Total.Bolivar, &factura.CreatedAt, &detalleRecibo.Dolar, &detalleRecibo.Bolivar)
		if err != nil {
			utils.Logline("error scanning venta.facturav", err)
			return nil, errors.New("errorGetData")
		}

		factura.DetalleRecibo = &detalleRecibo
		factura.NumReferencia = utils.GenerateNcontrolByUuid(factura.Id)
		facturaList = append(facturaList, factura)
	}
	rows.Close()

	return &facturaList, nil
}

func GetFacturaRetencion(db models.ConnDb, clienteId string, facturaReq models.FacturaReqId) (*models.FacturaList, error) {
	query := `SELECT fv.id as payment_id, fv.estatus, fv.total[1] as tot_dolar, fv.total[2] as tot_bolivar, fv.created_at
		FROM venta.facturav as fv
		WHERE fv.id=$1 AND fv.created_at=$2 AND fv.cliente_id=$3
		LIMIT 1`

	var factura models.FacturaList
	err := db.ConnPgsql.QueryRow(db.Ctx, query, facturaReq.Id, facturaReq.CreatedAt, clienteId).Scan(&factura.Id, &factura.Estatus, &factura.Total.Dolar, &factura.Total.Bolivar, &factura.CreatedAt)
	if err != nil {
		utils.Logline("error on select venta.facturav", err)
		return nil, errors.New("errorGetData")
	}

	factura.NumReferencia = utils.GenerateNcontrolByUuid(factura.Id)

	return &factura, nil
}
