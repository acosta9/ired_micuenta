package repo

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"ired.com/micuenta/models"
	"ired.com/micuenta/utils"
)

func SincTasaCambio(db models.ConnMysqlPgsql, caller string) error {
	//show status of worker
	utils.ShowStatusWorkerMysql(db, "sinc_tasa_cambio", caller+"/begin")

	//get fecha of last record on postgres
	var lastRecordDate string
	err := db.ConnPgsql.QueryRow(db.Ctx, "SELECT TO_CHAR(created_at, 'YYYY-MM-DD HH24:MI:SS') as fecha FROM publico.tasa_cambio WHERE moneda='bolivar' ORDER BY created_at DESC LIMIT 1").Scan(&lastRecordDate)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			utils.Logline("error getting fecha of last record", err)
			return err
		}
		lastRecordDate = "1970-01-01 00:00:00"
	}

	// get the last 100records from mysql that have a date greater than the last record of postgres
	query := `SELECT 
		CASE 
			WHEN q0.valor>1000 THEN q0.valor/1000000
				ELSE q0.valor
			END AS valor, q0.created_by, q0.created_at
		FROM (
			SELECT CAST(valor AS DECIMAL(15,4)) as valor, created_by, created_at FROM tasa_cambio WHERE created_at>? LIMIT 1000
		) as q0`
	rowsMysql, err := db.ConnMysql.QueryContext(db.Ctx, query, lastRecordDate)
	if err != nil {
		utils.Logline("error on getting tasa_cambio from mysql", err)
		return err
	}
	defer rowsMysql.Close()

	var contador int
	for rowsMysql.Next() {
		var tasaMonto float64
		var createdBy, createdAt string
		if err := rowsMysql.Scan(&tasaMonto, &createdBy, &createdAt); err != nil {
			utils.Logline("error scanning values of tasa cambio ", err)
			return err
		}

		query = `INSERT INTO publico.tasa_cambio (empresa_id, moneda, monto, created_at, created_by) 
			VALUES(1, 'bolivar', $1, $2, (SELECT id FROM publico.guard_user WHERE info->>'oldid'=$3))`
		_, err := db.ConnPgsql.Exec(db.Ctx, query, tasaMonto, createdAt, createdBy)
		if err != nil {
			utils.Logline("error inserting tasa_cambio", err)
			return err
		}
		contador++
	}
	rowsMysql.Close()

	//show status of worker
	utils.Logline(fmt.Sprintf("there were (%d) tasas_cambio records sincronized", contador))
	utils.ShowStatusWorkerMysql(db, "sinc_tasa_cambio", caller+"/ending")

	return nil
}
