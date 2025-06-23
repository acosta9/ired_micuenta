package repo

import (
	"errors"
	"net/http"

	"ired.com/micuenta/models"
	"ired.com/micuenta/utils"
)

func GetRetencion(db models.ConnDb, clienteId string, retencionReq models.RetencionReqId) (*models.RetencionResponse, int, error) {
	query := `SELECT r.id, fv.id as factura_id, fv.created_at::text as factura_created_at, r.num_comprobante, r.tipo_retencion, r.estatus, r.fecha_retencion::text, r.created_at, r.updated_at,
			r.monto_retenido[1] as monto_retenido_dolar, r.monto_retenido[2] as monto_retenido_bolivar,
			r.base_imponible[1] as base_imponible_dolar, r.base_imponible[2] as base_imponible_bolivar,
			r.porcentaje_retencion, COALESCE(r.info->>'descripcion','') as descr
		FROM venta.facturav_retencion as r
		LEFT JOIN venta.facturav as fv ON fv.id=r.facturav_id AND fv.created_at=r.facturav_created_at
		WHERE fv.cliente_id=$1 AND DATE(r.created_at)=DATE($2) AND r.id=$3`

	var retencion models.RetencionResponse
	var facturaReq models.FacturaReqId

	err := db.ConnPgsql.QueryRow(db.Ctx, query, clienteId, retencionReq.CreatedAt, retencionReq.Id).Scan(&retencion.Id, &facturaReq.Id, &facturaReq.CreatedAt, &retencion.NComprobante,
		&retencion.TipoRetencion, &retencion.Estatus, &retencion.FechaRetencion, &retencion.CreatedAt, &retencion.UpdatedAt,
		&retencion.MontoRetenido.Dolar, &retencion.MontoRetenido.Bolivar, &retencion.BaseImponible.Dolar, &retencion.BaseImponible.Bolivar,
		&retencion.PorcentajeRetencion, &retencion.Descripcion)
	if err != nil {
		utils.Logline("error getting venta.facturav_retencion", err, clienteId, retencionReq)
		return nil, http.StatusBadRequest, errors.New("errorGetData")
	}

	factura, err := GetFacturaRetencion(db, clienteId, facturaReq)
	if err != nil {
		utils.Logline("error getting venta.facturav", err, clienteId, facturaReq)
		return nil, http.StatusBadRequest, errors.New("errorGetData")
	}

	retencion.Ncontrol = utils.GenerateNcontrolByUuid(retencion.Id)
	retencion.Factura = *factura

	return &retencion, http.StatusOK, nil
}

func RetencionList(db models.ConnDb, userId any, pageQuery models.PaginatorQuery) (*[]models.RetencionList, *models.PaginatorData, error) {
	currentPage := pageQuery.Page
	limit := pageQuery.Limit
	offset := (currentPage - 1) * limit

	//get meta of paginator
	var totalCount int
	query := `SELECT COUNT(*) 
		FROM venta.facturav_retencion as r
		LEFT JOIN venta.facturav as fv ON fv.id=r.facturav_id AND fv.created_at=r.facturav_created_at
		WHERE fv.cliente_id=$1`
	if err := db.ConnPgsql.QueryRow(db.Ctx, query, userId).Scan(&totalCount); err != nil {
		utils.Logline("error on query count", err)
		return nil, nil, errors.New("errorGetData")
	}
	paginatorData := models.GetPaginatorMeta(currentPage, limit, totalCount)

	//validate if current page is possible to offset
	if currentPage > paginatorData.TotalPages {
		return nil, nil, errors.New("errorPage")
	}

	query = `SELECT r.id, fv.nfactura, r.num_comprobante, r.tipo_retencion, r.estatus, r.fecha_retencion::text, r.created_at, 
			r.monto_retenido[1] as monto_retenido_dolar, r.monto_retenido[2] as monto_retenido_bolivar
		FROM venta.facturav_retencion as r
		LEFT JOIN venta.facturav as fv ON fv.id=r.facturav_id AND fv.created_at=r.facturav_created_at
		WHERE fv.cliente_id=$1
		ORDER BY r.created_at DESC 
		LIMIT $2 
		OFFSET $3`
	rows, err := db.ConnPgsql.Query(db.Ctx, query, userId, limit, offset)
	if err != nil {
		utils.Logline("error on select venta.facturav_retencion", err)
		return nil, nil, errors.New("errorGetData")
	}
	defer rows.Close()

	var retencionList []models.RetencionList
	for rows.Next() {
		var retencion models.RetencionList
		err = rows.Scan(&retencion.Id, &retencion.NFactura, &retencion.NComprobante, &retencion.TipoRetencion, &retencion.Estatus, &retencion.FechaRetencion, &retencion.CreatedAt,
			&retencion.MontoRetenido.Dolar, &retencion.MontoRetenido.Bolivar)
		if err != nil {
			utils.Logline("error scanning venta.facturav_retencion", err)
			return nil, nil, errors.New("errorGetData")
		}

		retencion.NControl = utils.GenerateNcontrolByUuid(retencion.Id)
		retencionList = append(retencionList, retencion)
	}
	rows.Close()

	return &retencionList, &paginatorData, err
}

func GetRetencionFactura(db models.ConnDb, facturaReq models.FacturaReqId) (*[]models.RetencionList, error) {
	query := `SELECT r.id, fv.nfactura, r.num_comprobante, r.tipo_retencion, r.estatus, r.fecha_retencion::text, r.created_at, 
			r.monto_retenido[1] as monto_retenido_dolar, r.monto_retenido[2] as monto_retenido_bolivar
		FROM venta.facturav_retencion as r
		LEFT JOIN venta.facturav as fv ON fv.id=r.facturav_id AND fv.created_at=r.facturav_created_at
		WHERE fv.created_at=$1 AND fv.id=$2
		ORDER BY r.created_at DESC`

	rows, err := db.ConnPgsql.Query(db.Ctx, query, facturaReq.CreatedAt, facturaReq.Id)
	if err != nil {
		utils.Logline("error getting venta.facturav_retencion for a facturav", err, facturaReq)
		return nil, errors.New("errorGetData")
	}
	defer rows.Close()

	var retencionList []models.RetencionList
	for rows.Next() {
		var retencion models.RetencionList
		err = rows.Scan(&retencion.Id, &retencion.NFactura, &retencion.NComprobante, &retencion.TipoRetencion, &retencion.Estatus, &retencion.FechaRetencion, &retencion.CreatedAt,
			&retencion.MontoRetenido.Dolar, &retencion.MontoRetenido.Bolivar)
		if err != nil {
			utils.Logline("error scanning venta.facturav_retencion for a facturav", err, facturaReq)
			return nil, errors.New("errorGetData")
		}

		retencion.NControl = utils.GenerateNcontrolByUuid(retencion.Id)
		retencionList = append(retencionList, retencion)
	}
	rows.Close()

	return &retencionList, nil
}
