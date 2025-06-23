package repo

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"ired.com/micuenta/models"
	"ired.com/micuenta/utils"
)

func SuscripcionList(c *gin.Context, db models.ConnDb) (*[]models.Suscripcion, int, error) {
	userId, _ := c.Get("userId")

	query := `SELECT s.id as id, TRIM(TO_CHAR((s.info->>'oldid')::integer, '000000')) as ncontrol, s.activo as estatus, 
			regexp_replace(sv.nombre, '[^0-9]', '', 'g')::integer as speed_value, 'Mbps' as speed_unit,
			SPLIT_PART(st.nombre,'/',1) as zona, SPLIT_PART(st.nombre,'/',2) as tipo_conexion, SPLIT_PART(st.nombre,'/',3) as tipo_servicio,
			e.info->>'gps' as gps, TO_CHAR((s.info->>'fecha_install')::date, 'DD') as dia_renovacion,
			s.precio[1] as costo, COALESCE((SELECT * FROM publico.latest_tasa_cambio(1)),1) as tasa_cambio, (s.info->>'convenio')::boolean as convenio, COALESCE(ROUND(saldo.saldo_dolar,2),0) as saldo_dolar
		FROM administracion.suscripcion as s
		LEFT JOIN (
			SELECT suscripcion_id, ROUND(saldo[1],2) as saldo_dolar FROM venta.get_saldo(1, $1, 'total') WHERE suscripcion_id IS NOT NULL
		) as saldo ON saldo.suscripcion_id=s.id
		LEFT JOIN administracion.servicio as sv ON sv.id=s.servicio_id
		LEFT JOIN administracion.servicio_tipo as st ON st.id=sv.servicio_tipo_id
		LEFT JOIN network.estacion as e ON e.suscripcion_id=s.id
		WHERE s.cliente_id=$1`
	rows, err := db.ConnPgsql.Query(db.Ctx, query, userId)
	if err != nil {
		utils.Logline("error on select suscripciones", err)
		return nil, http.StatusBadRequest, errors.New("errorGetData")
	}
	defer rows.Close()

	var suscripciones []models.Suscripcion
	for rows.Next() {
		var suscripcion models.Suscripcion
		var costoUsd, tasa, saldoDolar float64
		var tipo_servicio, gps, renewalDay string
		err = rows.Scan(&suscripcion.Id, &suscripcion.Ncontrol, &suscripcion.Estatus, &suscripcion.SpeedValue, &suscripcion.SpeedUnit,
			&suscripcion.Zona, &suscripcion.TipoConexion, &tipo_servicio, &gps, &renewalDay, &costoUsd, &tasa, &suscripcion.ExcluirPago, &saldoDolar)
		if err != nil {
			utils.Logline("error scanning suscripciones", userId, err)
			return nil, http.StatusBadRequest, errors.New("errorGetData")
		}

		suscripcion.TipoServicioAcronimo = utils.TipoServicioAcronimo(tipo_servicio)
		suscripcion.TipoServicio = utils.TipoServicioNombre(tipo_servicio)
		suscripcion.Gps = utils.ValidateGPS(gps)
		suscripcion.Renewal = renewalDay

		suscripcion.Costo = models.Moneda{Dolar: costoUsd, Bolivar: utils.RoundToTwoDecimalPlaces(costoUsd * tasa)}
		suscripcion.Saldo = models.Moneda{Dolar: saldoDolar, Bolivar: utils.RoundToTwoDecimalPlaces(saldoDolar * tasa)}

		suscripciones = append(suscripciones, suscripcion)
	}
	rows.Close()

	return &suscripciones, http.StatusOK, nil
}

func GetSuscripcion(db models.ConnDb, clienteId string, suscId string) (*models.Suscripcion, int, error) {
	var suscripcion models.Suscripcion
	var costoUsd, tasaCambio, saldoDolar float64

	query := `SELECT s.id as id, TRIM(TO_CHAR((s.info->>'oldid')::integer, '000000')) as ncontrol, s.activo as estatus, 
			regexp_replace(sv.nombre, '[^0-9]', '', 'g')::integer as speed_value, 'Mbps' as speed_unit,
			SPLIT_PART(st.nombre,'/',1) as zona, SPLIT_PART(st.nombre,'/',2) as tipo_conexion, SPLIT_PART(st.nombre,'/',3) as tipo_servicio,
			e.info->>'gps' as gps, TO_CHAR((s.info->>'fecha_install')::date, 'DD') as dia_renovacion,
			s.precio[1] as costo, COALESCE((SELECT * FROM publico.latest_tasa_cambio(1)),1) as tasa_cambio, COALESCE(ROUND(saldo.saldo_dolar, 2),0) as saldo_dolar
		FROM administracion.suscripcion as s
		LEFT JOIN administracion.servicio as sv ON sv.id=s.servicio_id
		LEFT JOIN administracion.servicio_tipo as st ON st.id=sv.servicio_tipo_id
		LEFT JOIN network.estacion as e ON e.suscripcion_id=s.id
		LEFT JOIN (
			SELECT suscripcion_id, ROUND(saldo[1],2) as saldo_dolar FROM venta.get_saldo(1, $1, 'total') WHERE suscripcion_id IS NOT NULL
		) as saldo ON saldo.suscripcion_id=s.id
		WHERE s.cliente_id=$1 AND s.id=$2`
	err := db.ConnPgsql.QueryRow(db.Ctx, query, clienteId, suscId).Scan(&suscripcion.Id, &suscripcion.Ncontrol, &suscripcion.Estatus,
		&suscripcion.SpeedValue, &suscripcion.SpeedUnit, &suscripcion.Zona, &suscripcion.TipoConexion, &suscripcion.TipoServicio,
		&suscripcion.Gps, &suscripcion.Renewal, &costoUsd, &tasaCambio, &saldoDolar)
	if err != nil {
		utils.Logline("error on select suscripcion", err)
		return nil, http.StatusBadRequest, errors.New("errorGetData")
	}

	suscripcion.Costo = models.Moneda{Dolar: costoUsd, Bolivar: utils.RoundToTwoDecimalPlaces(costoUsd * tasaCambio)}
	suscripcion.Saldo = models.Moneda{Dolar: saldoDolar, Bolivar: utils.RoundToTwoDecimalPlaces(saldoDolar * tasaCambio)}

	return &suscripcion, http.StatusOK, nil
}
