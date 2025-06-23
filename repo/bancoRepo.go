package repo

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"ired.com/micuenta/models"
	"ired.com/micuenta/utils"
)

func FormasPagoList(c *gin.Context, db models.ConnDb) (*models.CuentasBanco, int, error) {
	userId, _ := c.Get("userId")

	query := `SELECT id, banco, info->>'web_nombre' as nombre, moneda, metodo_pago, info->>'web_detail' as detalle
		FROM publico.cuenta_banco
		WHERE (info->>'web_enable')::bool=true
		ORDER BY (info->>'web_order')::integer ASC`
	rows, err := db.ConnPgsql.Query(db.Ctx, query)
	if err != nil {
		utils.Logline("error on select cuenta_banco", err)
		return nil, http.StatusBadRequest, errors.New("errorGetData")
	}
	defer rows.Close()

	var formaPagoList []models.FormaPagoList
	for rows.Next() {
		var formaPago models.FormaPagoList
		var infoJson string
		err = rows.Scan(&formaPago.Id, &formaPago.Banco, &formaPago.Nombre, &formaPago.Moneda, &formaPago.MetodoPago, &infoJson)
		if err != nil {
			utils.Logline("error scanning cuenta_banco", userId, err)
			return nil, http.StatusBadRequest, errors.New("errorGetData")
		}

		formaPago.Detalle = getFormaPagoDetail(infoJson)

		formaPagoList = append(formaPagoList, formaPago)
	}
	rows.Close()

	query = `SELECT id, banco, info->>'web_nombre' as nombre, moneda
		FROM publico.cuenta_banco
		WHERE (info->>'web_banco_origen')::bool=true
		ORDER BY (info->>'web_order')::integer ASC`
	rows, err = db.ConnPgsql.Query(db.Ctx, query)
	if err != nil {
		utils.Logline("error on select cuenta_banco", err)
		return nil, http.StatusBadRequest, errors.New("errorGetData")
	}
	defer rows.Close()

	var bancoOrigenList []models.BancoOrigenList
	for rows.Next() {
		var bancoOrigen models.BancoOrigenList
		err = rows.Scan(&bancoOrigen.Id, &bancoOrigen.Banco, &bancoOrigen.Nombre, &bancoOrigen.Moneda)
		if err != nil {
			utils.Logline("error scanning cuenta_banco", userId, err)
			return nil, http.StatusBadRequest, errors.New("errorGetData")
		}
		bancoOrigenList = append(bancoOrigenList, bancoOrigen)
	}
	rows.Close()

	var tasaCambio float64
	err = db.ConnPgsql.QueryRow(db.Ctx, `SELECT monto FROM publico.tasa_cambio ORDER BY created_at DESC LIMIT 1`).Scan(&tasaCambio)
	if err != nil {
		utils.Logline("error on getting tasa_cambio", err)
		return nil, http.StatusBadRequest, errors.New("errorGetData")
	}

	cuentasBanco := models.CuentasBanco{FormaPagoList: formaPagoList, BancoOrigenList: bancoOrigenList, TasaCambio: tasaCambio}

	return &cuentasBanco, http.StatusOK, nil
}

func getFormaPagoDetail(jsonData string) any {
	// Create a map to hold only web_detail
	var rawData any

	// Unmarshal into the map
	if err := json.Unmarshal([]byte(jsonData), &rawData); err != nil {
		utils.Logline("Error decodig JSON", err)
		return nil
	}

	// Parse JSON into a map
	var data map[string]any
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		utils.Logline("Error decoding JSON:", err)
		return nil
	}

	// Preserve order by storing key-value pairs in a slice
	var orderedData []models.KeyValue
	for key, value := range data {
		orderedData = append(orderedData, models.KeyValue{Key: key, Value: value})
	}

	return orderedData
}
