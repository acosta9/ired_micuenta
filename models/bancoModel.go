package models

type CuentasBanco struct {
	FormaPagoList   []FormaPagoList   `json:"metodos_pago_disponibles"`
	BancoOrigenList []BancoOrigenList `json:"bancos_cliente"`
	TasaCambio      float64           `json:"tasa_cambio"`
}

type FormaPagoList struct {
	Id         int64  `json:"id"`
	Banco      string `json:"banco"`
	MetodoPago string `json:"metodo_pago"`
	Moneda     string `json:"moneda"`
	Nombre     string `json:"nombre"`
	Detalle    any    `json:"detalle"`
}

type BancoOrigenList struct {
	Id     int64  `json:"id"`
	Banco  string `json:"banco"`
	Moneda string `json:"moneda"`
	Nombre string `json:"nombre"`
}

type Moneda struct {
	Bolivar float64 `json:"bolivar"`
	Dolar   float64 `json:"dolar"`
}
