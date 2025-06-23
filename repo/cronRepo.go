package repo

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	"ired.com/micuenta/models"
	"ired.com/micuenta/utils"
)

func CleanOldSessionsCron(db models.ConnDb, caller string) error {
	//show status of worker
	utils.ShowStatusWorker(db, "sinc_users", caller+"/begin")

	query := `WITH deleted_rows AS (
			DELETE FROM publico.cliente_session_store
			WHERE expires_at<NOW()
			RETURNING *
		)
		SELECT COUNT(*) AS num FROM deleted_rows;`

	var count string
	err := db.ConnPgsql.QueryRow(db.Ctx, query).Scan(&count)
	if err != nil {
		utils.Logline("error deleting old sessions", err)
		return err
	}

	utils.Logline("it was remove (" + count + ") records")

	//show status of worker
	utils.ShowStatusWorker(db, "sinc_users", caller+"/ending")

	return nil
}

type clientPasswdCron struct {
	id     string
	docid  string
	nombre string
}

func CreatePasswordsCron(db models.ConnDb, caller string) error {
	//show status of worker
	utils.ShowStatusWorker(db, "create_client_passwd", caller+"/begin")

	query := `SELECT id, docid, nombre FROM publico.cliente WHERE passwd IS NULL ORDER BY id ASC LIMIT 2000`
	rows, err := db.ConnPgsql.Query(db.Ctx, query)
	if err != nil {
		utils.Logline("error deleting old sessions", err)
		return err
	}
	defer rows.Close()

	var clientes []clientPasswdCron
	for rows.Next() {
		var cliente clientPasswdCron
		err = rows.Scan(&cliente.id, &cliente.docid, &cliente.nombre)
		if err != nil {
			utils.Logline("error scanning cliente for password change:", err)
			return fmt.Errorf("error scanning cliente for password change: %w", err)
		}
		clientes = append(clientes, cliente)
	}
	rows.Close()

	// Worker pool size (Adjust for optimal performance)
	var wg sync.WaitGroup
	const workerPoolSize = 10
	sem := make(chan struct{}, workerPoolSize) // Semaphore to limit concurrency

	var contador int
	for _, cliente := range clientes {
		wg.Add(1)
		sem <- struct{}{} // Limit concurrency

		go func(cliente clientPasswdCron) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			// create password hash based on numbers from docid
			password := utils.ExtractNumbers(cliente.docid)
			hash, _ := bcrypt.GenerateFromPassword([]byte(password), 10)

			//update cliente on db
			query = `UPDATE publico.cliente SET passwd=$1 WHERE id=$2`
			_, err := db.ConnPgsql.Exec(ctx, query, hash, cliente.id)
			if err != nil {
				utils.Logline("error updating user", cliente, err)
			} else {
				utils.Logline("password created sucessfully", cliente)
			}

			<-sem // Release semaphore
		}(cliente)

		contador++
	}

	wg.Wait()

	//show status of worker
	utils.Logline(fmt.Sprintf("there were (%d) passwords sincronized", contador))
	utils.ShowStatusWorker(db, "create_client_passwd", caller+"/ending")

	return nil
}
