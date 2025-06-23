package models

// Struct with validation rules
type InfoTipoReq struct {
	Tipo string `json:"tipo" form:"tipo" binding:"required,oneof=movil web"`
}

type InfoLocation struct {
	Ciudad    string            `json:"ciudad"`
	Nombre    string            `json:"nombre"`
	Horario   map[string]string `json:"horario"`
	Direccion string            `json:"direccion"`
	Gps       string            `json:"coordenadas"`
}

type InfoFaq struct {
	Question string `json:"pregunta"`
	Answer   string `json:"respuesta"`
}
