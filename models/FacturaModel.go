package models

import (
	"database/sql"
	"time"
)

type FacturaCron struct {
	Id             string         `json:"id"`
	RazonSocial    string         `json:"razon_social"`
	DocId          string         `json:"docid"`
	Telefono       string         `json:"telefono"`
	Direccion      string         `json:"direccion"`
	NControl       sql.NullString `json:"ncontrol"`
	NFactura       string         `json:"numero_factura"`
	Fecha          string         `json:"fecha"`
	DiasCredito    int            `json:"dias_credito"`
	SubTotal       Moneda         `json:"subtotal"`
	DescPorc       float64        `json:"desc_porcentaje"`
	DescMonto      Moneda         `json:"desc_monto_dolar"`
	BaseImponible  Moneda         `json:"base_imponible"`
	IvaPorc        float64        `json:"iva_porcentaje"`
	IvaMonto       Moneda         `json:"iva_monto"`
	IgtfPorc       float64        `json:"igtf_porcentaje"`
	IgtfBase       Moneda         `json:"igtf_base_imponible"`
	IgtfMonto      Moneda         `json:"igtf_monto"`
	Total          Moneda         `json:"total"`
	TasaCambio     float64        `json:"tasa_cambio"`
	TipoFactura    string         `json:"tipo_factura"`
	Estatus        string         `json:"estatus"`
	Info           map[string]any `json:"info"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at"`
	CreatedByOldid string         `json:"created_by_oldid"`
	UpdatedByOldid string         `json:"updated_by_oldid"`
	ClienteOldid   string         `json:"cliente_oldid"`
	DetalleFactura string         `json:"detalle_factura"`
}

type FacturaDetCron struct {
	TaxStatus string  `json:"tax_status"`
	Qty       float64 `json:"qty"`
	PriceUnit Moneda  `json:"price_unit"`
	PriceTot  Moneda  `json:"price_tot"`
	Info      any     `json:"info"`
}

type FacturaList struct {
	DetalleRecibo *Moneda   `json:"monto_pagado,omitempty"`
	Id            string    `json:"factura_id"`
	NumReferencia string    `json:"nreferencia"`
	Estatus       string    `json:"estatus"`
	CreatedAt     time.Time `json:"created_at"`
	Total         Moneda    `json:"monto_total"`
}

type FacturaReqId struct {
	Id        string `form:"factura_id" json:"factura_id" binding:"required,uuid"`
	CreatedAt string `form:"created_at" json:"created_at" binding:"required,datetime=2006-01-02T15:04:05-07:00"`
}

type FacturaResponse struct {
	Id            string               `json:"factura_id"`
	NumReferencia string               `json:"nreferencia"`
	RazonSocial   string               `json:"razon_social"`
	DocId         string               `json:"docid"`
	Telefono      string               `json:"telefono"`
	Direccion     string               `json:"direccion"`
	Estatus       string               `json:"estatus"`
	CreatedAt     time.Time            `json:"created_at"`
	SubTotal      Moneda               `json:"subtotal"`
	DescPorc      float64              `json:"desc_porcentaje"`
	DescMonto     Moneda               `json:"desc_monto_dolar"`
	BaseImponible Moneda               `json:"base_imponible"`
	IvaPorc       float64              `json:"iva_porcentaje"`
	IvaMonto      Moneda               `json:"iva_monto"`
	IgtfPorc      float64              `json:"igtf_porcentaje"`
	IgtfBase      Moneda               `json:"igtf_base_imponible"`
	IgtfMonto     Moneda               `json:"igtf_monto"`
	Total         Moneda               `json:"total"`
	DatosBesser   FacturaDatosBesser   `json:"datos_besser"`
	FacturaDet    []FacturaDetResponse `json:"factura_detalle"`
	Retenciones   []RetencionList      `json:"retenciones"`
	Payments      []PaymentList        `json:"pagos"`
}

type FacturaDetResponse struct {
	Qty         float64        `json:"cantidad"`
	Concepto    string         `json:"concepto"`
	PriceUnit   Moneda         `json:"precio_unitario"`
	PriceTot    Moneda         `json:"total"`
	TaxStatus   string         `json:"tax_status"`
	Suscripcion map[string]any `json:"suscripcion_detalle"`
}
