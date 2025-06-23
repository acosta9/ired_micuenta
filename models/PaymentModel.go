package models

import (
	"database/sql"
	"time"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

type PaymentReq struct {
	ProfileId      string             `json:"profile_id" binding:"required,number,min=1,max=15"`
	CuentaBancoId  string             `json:"cuenta_banco_id" binding:"required,number,min=1,max=15"`
	BancoClienteId string             `json:"banco_cliente_id" binding:"required,number,min=1,max=15"`
	Fecha          string             `json:"fecha" binding:"required,datetime=2006-01-02"`
	TasaCambio     float64            `json:"tasa_cambio" binding:"required,gte=0,decimals_number=4"`
	Referencia     string             `json:"referencia" binding:"required,alfanumspa,min=10,max=200"`
	Email          string             `json:"email" binding:"omitempty,email"`
	Telefono       string             `json:"telefono" binding:"omitempty,celular"`
	PaymentDetail  []PaymentReqDetail `json:"payment_detail" binding:"required,dive"`
}

type PaymentReqDetail struct {
	Monto         float64 `json:"monto" binding:"required,gte=0,decimals_number=2"`
	SuscripcionId string  `json:"suscripcion_id" binding:"required,number,min=1"`
}

type PaymentResponseDetail struct {
	Monto       Moneda `json:"monto"`
	Suscripcion any    `json:"suscripcion" binding:"omitempty"`
	Factura     any    `json:"factura" binding:"omitempty"`
}

type PaymentResponse struct {
	PaymentId     string                 `json:"payment_id"`
	Ncontrol      string                 `json:"ncontrol"`
	ProfileId     string                 `json:"profile_id"`
	Estatus       string                 `json:"estatus"`
	Fecha         string                 `json:"fecha"`
	MontoTotal    Moneda                 `json:"monto_total"`
	TasaCambio    float64                `json:"tasa_cambio"`
	Referencia    string                 `json:"referencia"`
	Email         string                 `json:"email"`
	Telefono      string                 `json:"telefono"`
	UrlFile       string                 `json:"url_file"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	MetodoPago    FormaPagoList          `json:"metodo_pago"`
	BancoCliente  BancoOrigenList        `json:"banco_cliente"`
	Suscripciones []SuscripcionShortInfo `json:"suscripciones" binding:"omitempty"`
	Facturas      []FacturaList          `json:"facturas" binding:"omitempty"`
	DatosBesser   FacturaDatosBesser     `json:"datos_besser"`
}

type PaymentReqId struct {
	PaymentId string `form:"payment_id" json:"payment_id" binding:"required,uuid"`
	CreatedAt string `form:"created_at" json:"created_at" binding:"required,datetime=2006-01-02T15:04:05-07:00"`
}

type PaymentList struct {
	PaymentId  string        `json:"payment_id"`
	Ncontrol   string        `json:"ncontrol"`
	Estatus    string        `json:"estatus"`
	Fecha      string        `json:"fecha"`
	Referencia string        `json:"referencia"`
	CreatedAt  time.Time     `json:"created_at"`
	MontoTotal Moneda        `json:"monto_total"`
	MetodoPago FormaPagoList `json:"metodo_pago"`
}

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("alfanumspa", alphaNumEs)
		v.RegisterValidation("celular", celPhone)
		v.RegisterValidation("decimals_number", decimalsNumber)
	}
}

type BalanceAvailable struct {
	Monto Moneda `json:"monto_disponible"`
}

type TransferReq struct {
	ProfileId         string  `json:"profile_id" binding:"required,number,min=1,max=15"`
	DestinatarioDocId string  `json:"destinatario_docid" binding:"required"`
	Descripcion       string  `json:"descripcion" binding:"required,alfanumspa,min=10,max=200"`
	Monto             float64 `json:"monto" binding:"required,gte=0,decimals_number=2"`
}

type TransferResponse struct {
	TransferId        string    `json:"transfer_id"`
	Ncontrol          string    `json:"ncontrol"`
	DestinatarioDocId string    `json:"destinatario_docid"`
	Monto             Moneda    `json:"monto"`
	Descripcion       string    `json:"descripcion"`
	CreatedAt         time.Time `json:"created_at"`
}

type TransferList struct {
	TransferId        string    `json:"transfer_id"`
	Ncontrol          string    `json:"ncontrol"`
	DestinatarioDocId string    `json:"destinatario_docid"`
	Monto             Moneda    `json:"monto"`
	Descripcion       string    `json:"descripcion"`
	CreatedAt         time.Time `json:"created_at"`
}

type TransferReqId struct {
	TransferId string `form:"transfer_id" json:"transfer_id" binding:"required,uuid"`
	CreatedAt  string `form:"created_at" json:"created_at" binding:"required,datetime=2006-01-02T15:04:05-07:00"`
}

type ReciboPagovCron struct {
	ClienteOldid    string         `json:"cliente_oldid"`
	Estatus         string         `json:"estatus"`
	Fecha           string         `json:"fecha"`
	Referencia      sql.NullString `json:"referencia"`
	PaymentMethod   string         `json:"payment_method"`
	PaymentDetail   sql.NullString `json:"payment_detail"`
	Monto           Moneda         `json:"monto"`
	TasaCambio      float64        `json:"tasa_cambio"`
	CreatedAt       string         `json:"created_at"`
	UpdatedAt       string         `json:"updated_at"`
	CreatedByOldid  string         `json:"created_by_oldid"`
	UpdatedByOldid  string         `json:"updated_by_oldid"`
	PreFacturaOldid sql.NullString `json:"pre_factura_oldid"`
	Info            map[string]any `json:"info"`
}

type PaymentResponseDetail2 struct {
	Monto   Moneda            `json:"monto"`
	Factura FacturaReciboCron `json:"factura"`
}

type FacturaReciboCron struct {
	Id        string `json:"id"`
	CreatedAt string `json:"created_at"`
}
