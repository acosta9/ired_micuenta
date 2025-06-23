package repo

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"ired.com/micuenta/models"
	"ired.com/micuenta/utils"
)

func SendPayment(db models.ConnDb, userId any, paymentReq models.PaymentReq) (*models.PaymentResponse, int, error) {

	if userId != paymentReq.ProfileId {
		utils.Logline("userId from JWT and recieve on json are not equal", userId, paymentReq)
		return nil, http.StatusBadRequest, errors.New("errorInternal")
	}

	//validar si al menos una suscripcion existe on json request
	if len(paymentReq.PaymentDetail) == 0 {
		return nil, http.StatusBadRequest, errors.New("vePaymentDetail")
	}

	//validar suscripciones
	if utils.PaymentReqAmountHasDupIds(paymentReq.PaymentDetail) {
		return nil, http.StatusBadRequest, errors.New("veSuscripcion")
	}

	//get cuentaBanco data from DB
	var err error
	var cuentaBanco models.FormaPagoList
	query := `SELECT id, metodo_pago, moneda
		FROM publico.cuenta_banco 
		WHERE id=$1 AND metodo_pago<>'otros' AND (info->>'web_enable')::bool=true`
	err = db.ConnPgsql.QueryRow(db.Ctx, query, paymentReq.CuentaBancoId).Scan(&cuentaBanco.Id, &cuentaBanco.MetodoPago, &cuentaBanco.Moneda)
	if err != nil {
		utils.Logline(fmt.Sprintf("cuenta_banco_id (%s) no encontrado", paymentReq.CuentaBancoId), paymentReq)
		return nil, http.StatusBadRequest, errors.New("veCuentaBancoId")
	}

	//validar que existen campos opciones para divisa y pago movil
	switch cuentaBanco.MetodoPago {
	case "divisa":
		if len(paymentReq.Email) <= 3 {
			return nil, http.StatusBadRequest, errors.New("veEmail")
		}
	case "pago_movil":
		if len(paymentReq.Telefono) <= 3 {
			return nil, http.StatusBadRequest, errors.New("veCelphone")
		}
	}

	//validar detalles de pago
	var paymentDetails []models.PaymentResponseDetail
	var montoTotal = []float64{0, 0}
	for _, detallePago := range paymentReq.PaymentDetail {
		var paymentDetail models.PaymentResponseDetail

		//fill paymentDetail struct
		if cuentaBanco.Moneda == "dolar" {
			paymentDetail.Monto.Dolar = detallePago.Monto
			paymentDetail.Monto.Bolivar = utils.RoundToFourDecimals(detallePago.Monto * paymentReq.TasaCambio)
		} else if cuentaBanco.Moneda == "bolivar" {
			paymentDetail.Monto.Bolivar = detallePago.Monto
			paymentDetail.Monto.Dolar = utils.RoundToFourDecimals(detallePago.Monto / paymentReq.TasaCambio)
		}

		//validar monto en relacion a moneda, maximo permitido en dolar es 3000
		montoTotal[0] += paymentDetail.Monto.Dolar
		montoTotal[1] += paymentDetail.Monto.Bolivar
		if paymentDetail.Monto.Dolar > 3000 {
			return nil, http.StatusBadRequest, errors.New("veMonto")
		}

		//validar si suscripcion existe
		suscDetalle, _, err := GetSuscripcion(db, fmt.Sprintf("%s", userId), detallePago.SuscripcionId)
		if err != nil {
			utils.Logline(fmt.Sprintf("error searching suscripcion_id: %s and cliente_id %s ", detallePago.SuscripcionId, userId), err, paymentReq)
			return nil, http.StatusBadRequest, errors.New("veSuscripcion")
		}

		paymentDetail.Suscripcion = *suscDetalle
		paymentDetails = append(paymentDetails, paymentDetail)
	}

	var bancoOrigen models.BancoOrigenList
	query = `SELECT id, moneda 
		FROM publico.cuenta_banco 
		WHERE id=$1 AND metodo_pago='otros' AND (info->>'web_banco_origen')::bool=true`
	err = db.ConnPgsql.QueryRow(db.Ctx, query, paymentReq.BancoClienteId).Scan(&bancoOrigen.Id, &bancoOrigen.Moneda)
	if err != nil {
		utils.Logline(fmt.Sprintf("cuenta_banco_id (%s) no encontrado", paymentReq.BancoClienteId), paymentReq)
		return nil, http.StatusBadRequest, errors.New("veBancoClienteId")
	}

	//validar que moneda del banco destino y origen coincidan
	if cuentaBanco.Moneda != bancoOrigen.Moneda {
		return nil, http.StatusBadRequest, errors.New("veBancoClienteId")
	}

	// handle json info column
	infoStruct := map[string]any{
		"email":          paymentReq.Email,
		"telefono":       paymentReq.Telefono,
		"payment_detail": paymentDetails,
		"url_file":       "",
	}

	var paymentId, paymentCreatedat string
	query = `SELECT id, (created_at)::text FROM venta.insert_recibo_pagov($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	err = db.ConnPgsql.QueryRow(db.Ctx, query, 1, paymentReq.ProfileId, paymentReq.CuentaBancoId, paymentReq.BancoClienteId, paymentReq.Fecha, strings.ToLower(paymentReq.Referencia),
		montoTotal, paymentReq.TasaCambio, "pendiente", 1, 1, infoStruct).Scan(&paymentId, &paymentCreatedat)
	if err != nil {
		fmt.Println(err)
		if strings.Contains(err.Error(), "ya existe") {
			return nil, http.StatusBadRequest, errors.New("veReferencia")
		}
		utils.Logline("error saving recibo_pagov", err, paymentReq)
		return nil, http.StatusBadRequest, errors.New("errorInsertRecord")
	}

	paymentReqId := models.PaymentReqId{
		PaymentId: paymentId,
		CreatedAt: paymentCreatedat,
	}

	paymentResponse, errType, err := GetPayment(db, paymentReq.ProfileId, paymentReqId)
	if err != nil {
		return nil, errType, err
	}

	return paymentResponse, http.StatusOK, nil
}

func ImageUpload(c *gin.Context, db models.ConnDb, file *multipart.FileHeader, paymentReq models.PaymentReqId) (int, error) {
	// Create uploads directory if it doesn't exist
	uploadDir := os.Getenv("PAYMENT_UPLOAD_FOLDER") + time.Now().Format("/2006/01/02")
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.Mkdir(uploadDir, 0755)
	}

	// Generate unique filename
	ext := filepath.Ext(file.Filename)

	//validate extensions allowed
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
		return http.StatusBadRequest, errors.New("veFileExtError")
	}

	//generate newFileName
	userId, _ := c.Get("userId")
	newFilename := fmt.Sprintf("%s_%s%s", userId, utils.GenerateUUID(), ext)
	filePath := filepath.Join(uploadDir, newFilename)

	// Save the file
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		return http.StatusBadRequest, errors.New("veFileError")
	}

	var reciboPagoId string
	query := `UPDATE venta.recibo_pagov
		SET info = jsonb_set(info, '{url_file}', to_jsonb($1::text))
		WHERE empresa_id = 1 AND cliente_id=$2 AND created_at=$3 AND id = $4 RETURNING id`
	if err := db.ConnPgsql.QueryRow(db.Ctx, query, filePath, userId, paymentReq.CreatedAt, paymentReq.PaymentId).Scan(&reciboPagoId); err != nil {
		os.Remove(filePath)
		utils.Logline("error updating recibo_pagov", err)
		return http.StatusBadRequest, errors.New("veFileError")
	}

	return http.StatusOK, nil
}

func GetPayment(db models.ConnDb, clienteId string, paymentReq models.PaymentReqId) (*models.PaymentResponse, int, error) {
	query := `SELECT rp.id, rp.cliente_id, rp.fecha::text, rp.tasa_cambio, ROUND(rp.monto[1],2) as tot_dolar, ROUND(rp.monto[2],2) as tot_bolivar, COALESCE(rp.referencia, '') as referencia, rp.estatus,
			COALESCE(rp.info->>'email', '') as email, COALESCE(rp.info->>'telefono', '') as telefono, rp.info->>'url_file' as url_file, rp.info->>'payment_detail' as pdetail,
			rp.created_at, rp.updated_at,
			mpago.id as mpago_id, mpago.banco as mpago_banco, mpago.metodo_pago as mpago_metodo, mpago.moneda as mpago_moneda, mpago.info->>'web_nombre' as mpago_nombre, mpago.info->>'web_detail' as mpago_detalle,
			COALESCE(mcliente.id, 0) as mcliente_id, COALESCE(mcliente.banco, '') as mcliente_banco, 
			COALESCE(mcliente.moneda::varchar, '') as mcliente_moneda, COALESCE(mcliente.info->>'web_nombre', '') as mcliente_nombre
		FROM venta.recibo_pagov as rp
		LEFT JOIN publico.cuenta_banco as mpago ON mpago.id=rp.metodo_pago_id
		LEFT JOIN publico.cuenta_banco as mcliente ON mcliente.id=rp.cuenta_cliente_id
		WHERE rp.cliente_id=$1 AND rp.created_at=$2 AND rp.id=$3`

	var payment models.PaymentResponse
	var paymentDetailJson string
	var metodoPagoInfo sql.NullString

	err := db.ConnPgsql.QueryRow(db.Ctx, query, clienteId, paymentReq.CreatedAt, paymentReq.PaymentId).Scan(&payment.PaymentId, &payment.ProfileId, &payment.Fecha, &payment.TasaCambio, &payment.MontoTotal.Dolar, &payment.MontoTotal.Bolivar,
		&payment.Referencia, &payment.Estatus, &payment.Email, &payment.Telefono, &payment.UrlFile, &paymentDetailJson,
		&payment.CreatedAt, &payment.UpdatedAt,
		&payment.MetodoPago.Id, &payment.MetodoPago.Banco, &payment.MetodoPago.MetodoPago, &payment.MetodoPago.Moneda, &payment.MetodoPago.Nombre, &metodoPagoInfo,
		&payment.BancoCliente.Id, &payment.BancoCliente.Banco, &payment.BancoCliente.Moneda, &payment.BancoCliente.Nombre)
	if err != nil {
		utils.Logline("error getting recibo_pagov", err, clienteId, paymentReq)
		return nil, http.StatusBadRequest, errors.New("errorGetData")
	}

	payment.Ncontrol = utils.GenerateNcontrolByUuid(payment.PaymentId)
	if metodoPagoInfo.Valid {
		payment.MetodoPago.Detalle = getFormaPagoDetail(metodoPagoInfo.String)
	}

	//iterate over payment detail to extract suscripciones pago
	var data []map[string]any
	if err := json.Unmarshal([]byte(paymentDetailJson), &data); err != nil {
		utils.Logline("error parsing json to struct", err, clienteId, paymentReq)
		return nil, http.StatusBadRequest, errors.New("errorGetData")
	}
	var suscripcionList []models.SuscripcionShortInfo
	for _, item := range data {
		if item["suscripcion"] != nil {
			montoSusc := item["monto"].(map[string]any)
			susc := item["suscripcion"].(map[string]any)

			suscripcion := models.SuscripcionShortInfo{
				DetalleRecibo: &models.Moneda{Dolar: montoSusc["dolar"].(float64), Bolivar: montoSusc["bolivar"].(float64)},
				Id:            int64(susc["id"].(float64)),
				Oldid:         susc["ncontrol"].(string),
				TipoConexion:  susc["tipo_conexion"].(string),
				TipoServicio:  susc["tipo_servicio"].(string),
				SpeedValue:    susc["speed_value"].(float64),
				SpeedUnit:     susc["speed_unit"].(string),
			}
			suscripcionList = append(suscripcionList, suscripcion)
		}
	}

	//get facturas asociadas al reciboPago
	facturaList, err := GetFacturaPayment(db, clienteId, paymentReq)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	payment.Suscripciones = suscripcionList
	payment.Facturas = *facturaList

	payment.DatosBesser = models.GetDatosBesser()

	return &payment, http.StatusOK, nil
}

func PaymentList(db models.ConnDb, userId any, pageQuery models.PaginatorQuery) (*[]models.PaymentList, *models.PaginatorData, error) {
	currentPage := pageQuery.Page
	limit := pageQuery.Limit
	offset := (currentPage - 1) * limit

	//get meta of paginator
	var totalCount int
	if err := db.ConnPgsql.QueryRow(db.Ctx, "SELECT COUNT(*) FROM venta.recibo_pagov WHERE cliente_id=$1", userId).Scan(&totalCount); err != nil {
		utils.Logline("error on query count", err)
		return nil, nil, errors.New("errorGetData")
	}
	paginatorData := models.GetPaginatorMeta(currentPage, limit, totalCount)

	//validate if current page is possible to offset
	if currentPage > paginatorData.TotalPages {
		return nil, nil, errors.New("errorPage")
	}

	query := `SELECT rp.id as payment_id, rp.estatus, rp.fecha::text as fecha, 
			COALESCE(rp.referencia, '') as referencia, rp.monto[1] as tot_dolar, rp.monto[2] as tot_bolivar, rp.created_at,
			mpago.id as mpago_id, mpago.banco as mpago_banco, mpago.metodo_pago as mpago_metodo, mpago.moneda as mpago_moneda, mpago.info->>'web_nombre' as mpago_nombre, mpago.info->>'web_detail' as mpago_detalle
		FROM venta.recibo_pagov as rp
		LEFT JOIN publico.cuenta_banco as mpago ON mpago.id=rp.metodo_pago_id
		WHERE rp.cliente_id=$1
		ORDER BY rp.created_at DESC 
		LIMIT $2 
		OFFSET $3`
	rows, err := db.ConnPgsql.Query(db.Ctx, query, userId, limit, offset)
	if err != nil {
		utils.Logline("error on select recibo_pagov", err)
		return nil, nil, errors.New("errorGetData")
	}
	defer rows.Close()

	var paymentList []models.PaymentList
	for rows.Next() {
		var payment models.PaymentList
		var metodoPagoInfo sql.NullString
		err = rows.Scan(&payment.PaymentId, &payment.Estatus, &payment.Fecha, &payment.Referencia, &payment.MontoTotal.Dolar, &payment.MontoTotal.Bolivar, &payment.CreatedAt,
			&payment.MetodoPago.Id, &payment.MetodoPago.Banco, &payment.MetodoPago.MetodoPago, &payment.MetodoPago.Moneda, &payment.MetodoPago.Nombre, &metodoPagoInfo)
		if err != nil {
			utils.Logline("error scanning recibo_pagov", err)
			return nil, nil, errors.New("errorGetData")
		}

		if metodoPagoInfo.Valid {
			payment.MetodoPago.Detalle = getFormaPagoDetail(metodoPagoInfo.String)
		}
		payment.Ncontrol = utils.GenerateNcontrolByUuid(payment.PaymentId)
		paymentList = append(paymentList, payment)
	}
	rows.Close()

	return &paymentList, &paginatorData, err
}

func BalanceAvailable(c *gin.Context, db models.ConnDb) (*models.BalanceAvailable, int, error) {
	userId, _ := c.Get("userId")
	var balanceAvailable models.BalanceAvailable
	query := `SELECT COALESCE(ROUND(SUM(q0.saldo_dolar),2),0) as saldo_dolar, COALESCE(ROUND(SUM(q0.saldo_bolivar),2),0) as saldo_bolivar
		FROM (
			SELECT suscripcion_id, saldo[1] as saldo_dolar, saldo[2] as saldo_bolivar FROM venta.get_saldo(1, $1, 'procesado') WHERE suscripcion_id IS NOT NULL
		) as q0`
	if err := db.ConnPgsql.QueryRow(db.Ctx, query, userId).Scan(&balanceAvailable.Monto.Dolar, &balanceAvailable.Monto.Bolivar); err != nil {
		utils.Logline("error on select saldo", err)
		return nil, http.StatusBadRequest, errors.New("errorGetData")
	}

	return &balanceAvailable, http.StatusOK, nil
}

func SendTransfer(db models.ConnDb, userId any, transferReq models.TransferReq) (*models.TransferResponse, int, error) {
	if userId != transferReq.ProfileId {
		utils.Logline("userId from JWT and recieve on json are not equal", userId, transferReq)
		return nil, http.StatusBadRequest, errors.New("errorInternal")
	}

	//validar si destinatarioDocId existe
	docIdReq := strings.ToLower(transferReq.DestinatarioDocId)
	var clienteDestinoId sql.NullString
	query := `SELECT id FROM publico.cliente WHERE docid=$1 LIMIT 1`
	if err := db.ConnPgsql.QueryRow(db.Ctx, query, docIdReq).Scan(&clienteDestinoId); err != nil {
		utils.Logline(fmt.Sprintf("docId (%s) no encontrado al momento de transferir", docIdReq), transferReq)
		return nil, http.StatusBadRequest, errors.New("errorInternal")
	}

	// handle json info column
	infoStruct := map[string]any{
		"destinatario_docid": transferReq.DestinatarioDocId,
		"descripcion":        transferReq.Descripcion,
	}

	var tasaCambio float64
	query = `SELECT publico.latest_tasa_cambio($1) as tasa_cambio`
	if err := db.ConnPgsql.QueryRow(db.Ctx, query, 1).Scan(&tasaCambio); err != nil {
		utils.Logline("tasa cambio no encontrada", err)
		return nil, http.StatusBadRequest, errors.New("errorInternal")
	}

	//arrayMonto
	montoArray := []float64{transferReq.Monto, transferReq.Monto * tasaCambio}

	var transferId, transferCreatedat string
	query = `SELECT id, (created_at)::text FROM venta.insert_transferenciav($1, $2, $3, $4, $5)`
	if err := db.ConnPgsql.QueryRow(db.Ctx, query, 1, transferReq.ProfileId, clienteDestinoId, montoArray, infoStruct).Scan(&transferId, &transferCreatedat); err != nil {
		utils.Logline("error saving transferenciav", err, transferReq)

		if strings.Contains(err.Error(), "cliente destino no posee una suscripcion_id") {
			return nil, http.StatusBadRequest, errors.New("errorInternal")
		}

		if strings.Contains(err.Error(), "balance insuficiente") {
			return nil, http.StatusBadRequest, errors.New("veAmmountInsufficient")
		}

		return nil, http.StatusBadRequest, errors.New("errorInsertRecord")
	}

	TransferReqId := models.TransferReqId{
		TransferId: transferId,
		CreatedAt:  transferCreatedat,
	}

	transferResponse, errType, err := GetTransfer(db, transferReq.ProfileId, TransferReqId)
	if err != nil {
		return nil, errType, err
	}

	return transferResponse, http.StatusOK, nil
}

func GetTransfer(db models.ConnDb, clienteId string, transferReq models.TransferReqId) (*models.TransferResponse, int, error) {
	query := `SELECT id, info->>'destinatario_docid' as destinatario_docid, ROUND(monto[1],2) as tot_dolar, ROUND(monto[2],2) as tot_bolivar, 
			info->>'descripcion' as descr, created_at
		FROM venta.transferenciav
		WHERE cliente_origen_id=$1 AND created_at=$2 AND id=$3`

	var transfer models.TransferResponse
	if err := db.ConnPgsql.QueryRow(db.Ctx, query, clienteId, transferReq.CreatedAt, transferReq.TransferId).Scan(&transfer.TransferId, &transfer.DestinatarioDocId,
		&transfer.Monto.Dolar, &transfer.Monto.Bolivar, &transfer.Descripcion, &transfer.CreatedAt); err != nil {
		utils.Logline("error getting venta.transferenciav", err, clienteId, transferReq)
		return nil, http.StatusBadRequest, errors.New("errorGetData")
	}

	transfer.Ncontrol = utils.GenerateNcontrolByUuid(transfer.TransferId)

	return &transfer, http.StatusOK, nil
}

func TransferList(db models.ConnDb, userId any, pageQuery models.PaginatorQuery) (*[]models.TransferList, *models.PaginatorData, error) {
	currentPage := pageQuery.Page
	limit := pageQuery.Limit
	offset := (currentPage - 1) * limit

	//get meta of paginator
	var totalCount int
	if err := db.ConnPgsql.QueryRow(db.Ctx, "SELECT COUNT(*) FROM venta.transferenciav WHERE cliente_origen_id=$1", userId).Scan(&totalCount); err != nil {
		utils.Logline("error on query count", err)
		return nil, nil, errors.New("errorGetData")
	}
	paginatorData := models.GetPaginatorMeta(currentPage, limit, totalCount)

	//validate if current page is possible to offset
	if currentPage > paginatorData.TotalPages {
		return nil, nil, errors.New("errorPage")
	}

	query := `SELECT id, info->>'destinatario_docid' as destinatario_docid, ROUND(monto[1],2) as tot_dolar, ROUND(monto[2],2) as tot_bolivar, 
			info->>'descripcion' as descr, created_at
		FROM venta.transferenciav
		WHERE cliente_origen_id=$1
		ORDER BY created_at DESC 
		LIMIT $2 
		OFFSET $3`
	rows, err := db.ConnPgsql.Query(db.Ctx, query, userId, limit, offset)
	if err != nil {
		utils.Logline("error on select venta.transferenciav", err)
		return nil, nil, errors.New("errorGetData")
	}
	defer rows.Close()

	var transferList []models.TransferList
	for rows.Next() {
		var transfer models.TransferList
		err = rows.Scan(&transfer.TransferId, &transfer.DestinatarioDocId, &transfer.Monto.Dolar, &transfer.Monto.Bolivar, &transfer.Descripcion, &transfer.CreatedAt)
		if err != nil {
			utils.Logline("error scanning venta.transferenciav", err)
			return nil, nil, errors.New("errorGetData")
		}

		transfer.Ncontrol = utils.GenerateNcontrolByUuid(transfer.TransferId)
		transferList = append(transferList, transfer)
	}
	rows.Close()

	return &transferList, &paginatorData, err
}

func GetPaymentFactura(db models.ConnDb, clienteId string, facturaReq models.FacturaReqId) (*[]models.PaymentList, error) {
	query := `SELECT rp.id as payment_id, rp.estatus, rp.fecha::text as fecha, 
			COALESCE(rp.referencia, '') as referencia, rp.monto[1] as tot_dolar, rp.monto[2] as tot_bolivar, rp.created_at,
			mpago.id as mpago_id, mpago.banco as mpago_banco, mpago.metodo_pago as mpago_metodo, mpago.moneda as mpago_moneda, mpago.info->>'web_nombre' as mpago_nombre, mpago.info->>'web_detail' as mpago_detalle
		FROM venta.recibo_pagov as rp
		LEFT JOIN publico.cuenta_banco as mpago ON mpago.id=rp.metodo_pago_id
		WHERE rp.cliente_id=$1 AND rp.info->'payment_detail'->0->'factura'->>'id'=$2
		ORDER BY rp.created_at DESC`

	rows, err := db.ConnPgsql.Query(db.Ctx, query, clienteId, facturaReq.Id)
	if err != nil {
		utils.Logline("error getting venta.recibo_pagov for a facturav", err, clienteId, facturaReq)
		return nil, errors.New("errorGetData")
	}
	defer rows.Close()

	var paymentList []models.PaymentList
	for rows.Next() {
		var payment models.PaymentList
		var metodoPagoInfo sql.NullString
		err = rows.Scan(&payment.PaymentId, &payment.Estatus, &payment.Fecha, &payment.Referencia, &payment.MontoTotal.Dolar, &payment.MontoTotal.Bolivar, &payment.CreatedAt,
			&payment.MetodoPago.Id, &payment.MetodoPago.Banco, &payment.MetodoPago.MetodoPago, &payment.MetodoPago.Moneda, &payment.MetodoPago.Nombre, &metodoPagoInfo)
		if err != nil {
			utils.Logline("error scanning recibo_pagov", err)
			return nil, errors.New("errorGetData")
		}

		if metodoPagoInfo.Valid {
			payment.MetodoPago.Detalle = getFormaPagoDetail(metodoPagoInfo.String)
		}
		payment.Ncontrol = utils.GenerateNcontrolByUuid(payment.PaymentId)
		paymentList = append(paymentList, payment)
	}
	rows.Close()

	return &paymentList, nil
}
