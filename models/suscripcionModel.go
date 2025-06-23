package models

type Suscripcion struct {
	Id                   int64   `json:"id"`
	Ncontrol             string  `json:"ncontrol"`
	Zona                 string  `json:"zona"`
	TipoConexion         string  `json:"tipo_conexion"`
	TipoServicio         string  `json:"tipo_servicio"`
	TipoServicioAcronimo string  `json:"tipo_servicio_acronimo"`
	SpeedValue           float64 `json:"speed_value"`
	SpeedUnit            string  `json:"speed_unit"`
	Gps                  string  `json:"coordenadas"`
	Estatus              bool    `json:"estatus"`
	ExcluirPago          bool    `json:"excluir_pago"`
	Renewal              string  `json:"renewal_day"`
	Saldo                Moneda  `json:"saldo" binding:"omitempty"`
	Costo                Moneda  `json:"costo"`
}

type SuscripcionShortInfo struct {
	DetalleRecibo *Moneda `json:"monto_pagado,omitempty"`
	Id            int64   `json:"id"`
	Oldid         string  `json:"ncontrol"`
	Zona          string  `json:"zona"`
	TipoConexion  string  `json:"tipo_conexion"`
	TipoServicio  string  `json:"tipo_servicio"`
	SpeedValue    float64 `json:"speed_value"`
	SpeedUnit     string  `json:"speed_unit"`
}
