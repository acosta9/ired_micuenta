package models

import (
	"database/sql"
	"time"
)

type RetencionCron struct {
	Id                  string         `json:"id"`
	FacturavId          string         `json:"facturav_id"`
	FacturavCreatedAt   string         `json:"facturav_created_at"`
	NComprobante        string         `json:"num_comprobante"`
	TipoRetencion       string         `json:"tipo_retencion"`
	Estatus             string         `json:"estatus"`
	FechaRetencion      string         `json:"fecha_retencion"`
	BaseImponible       Moneda         `json:"base_imponible"`
	MontoRetenido       Moneda         `json:"monto_retenido"`
	PorcentajeRetencion float64        `json:"porcentaje_retencion"`
	UrlFile             sql.NullString `json:"url_file"`
	Descripcion         sql.NullString `json:"descripcion"`
	CreatedAt           string         `json:"created_at"`
	UpdatedAt           string         `json:"updated_at"`
	CreatedBy           int            `json:"created_by"`
	UpdatedBy           int            `json:"updated_by"`
	Info                map[string]any `json:"info"`
	InfoOld             map[string]any `json:"info_old"`
}

type RetencionList struct {
	Id             string    `json:"retencion_id"`
	NControl       string    `json:"ncontrol"`
	NFactura       string    `json:"numero_factura"`
	NComprobante   string    `json:"num_comprobante"`
	TipoRetencion  string    `json:"tipo_retencion"`
	Estatus        string    `json:"estatus"`
	FechaRetencion string    `json:"fecha_retencion"`
	MontoRetenido  Moneda    `json:"monto_retenido"`
	CreatedAt      time.Time `json:"created_at"`
}

type RetencionResponse struct {
	Id                  string      `json:"retencion_id"`
	Ncontrol            string      `json:"ncontrol"`
	NComprobante        string      `json:"num_comprobante"`
	TipoRetencion       string      `json:"tipo_retencion"`
	Estatus             string      `json:"estatus"`
	FechaRetencion      string      `json:"fecha_retencion"`
	BaseImponible       Moneda      `json:"base_imponible"`
	MontoRetenido       Moneda      `json:"monto_retenido"`
	Descripcion         string      `json:"descripcion"`
	PorcentajeRetencion float64     `json:"porcentaje_retencion"`
	CreatedAt           time.Time   `json:"created_at"`
	UpdatedAt           time.Time   `json:"updated_at"`
	Factura             FacturaList `json:"factura"`
}

type RetencionReqId struct {
	Id        string `form:"retencion_id" json:"retencion_id" binding:"required,uuid"`
	CreatedAt string `form:"created_at" json:"created_at" binding:"required,datetime=2006-01-02T15:04:05-07:00"`
}
