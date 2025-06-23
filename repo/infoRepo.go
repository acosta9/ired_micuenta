package repo

import (
	"encoding/json"
	"errors"

	"ired.com/micuenta/models"
	"ired.com/micuenta/utils"
)

func GetOficinas(db models.ConnDb, tipoApp string) (*[]models.InfoLocation, error) {
	//get meta of paginator
	query := `SELECT q0.info->>'ciudad' as city, q0.info->>'nombre' as name, q0.info->>'horario' as hours, q0.info->>'direccion' as dir, q0.info->>'gps' as gps
		FROM (
			SELECT array_agg(dispositivos) as dispositivos, jsonb_array_elements(info->'locations') as info
			FROM publico.config_var, jsonb_array_elements_text(info->'dispositivos') as dispositivos
			WHERE tipo='app_micuenta' AND info->>'url'='/info/oficinas'
			GROUP BY 2
		) as q0
		WHERE (q0.info->>'enabled')::boolean = true AND q0.dispositivos @> ARRAY[$1]
		ORDER BY (q0.info->>'orden')::INT ASC`
	rows, err := db.ConnPgsql.Query(db.Ctx, query, tipoApp)
	if err != nil {
		utils.Logline("error on getting publico.config_var", err)
		return nil, errors.New("errorGetData")
	}
	defer rows.Close()

	var listOficinas []models.InfoLocation
	for rows.Next() {
		var oficina models.InfoLocation
		var horario string

		err = rows.Scan(&oficina.Ciudad, &oficina.Nombre, &horario, &oficina.Direccion, &oficina.Gps)
		if err != nil {
			utils.Logline("error scanning publico.config_var oficinas", err)
			return nil, errors.New("errorGetData")
		}

		if err := json.Unmarshal([]byte(horario), &oficina.Horario); err != nil {
			utils.Logline("Error marshaling horario:", err)
			return nil, errors.New("errorGetData")
		}

		listOficinas = append(listOficinas, oficina)
	}

	return &listOficinas, nil
}

func GetFaqs(db models.ConnDb, tipoApp string) (*[]models.InfoFaq, error) {
	//get meta of paginator
	query := `SELECT q0.info->>'pregunta' as pregunta, q0.info->>'respuesta' as respuesta
		FROM (
			SELECT array_agg(dispositivos) as dispositivos, jsonb_array_elements(info->'questions') as info
			FROM publico.config_var, jsonb_array_elements_text(info->'dispositivos') as dispositivos
			WHERE tipo='app_micuenta' AND info->>'url'='/info/faqs'
			GROUP BY 2
		) as q0
		WHERE (q0.info->>'enabled')::boolean = true AND q0.dispositivos @> ARRAY[$1]
		ORDER BY (q0.info->>'orden')::INT ASC`
	rows, err := db.ConnPgsql.Query(db.Ctx, query, tipoApp)
	if err != nil {
		utils.Logline("error on getting publico.config_var", err)
		return nil, errors.New("errorGetData")
	}
	defer rows.Close()

	var listFaqs []models.InfoFaq
	for rows.Next() {
		var faq models.InfoFaq

		err = rows.Scan(&faq.Question, &faq.Answer)
		if err != nil {
			utils.Logline("error scanning publico.config_var faqs", err)
			return nil, errors.New("errorGetData")
		}

		listFaqs = append(listFaqs, faq)
	}

	return &listFaqs, nil
}

func GetAccesibilidad(db models.ConnDb, tipoApp string) (*string, error) {
	//get meta of paginator
	query := `SELECT q0.info as texto
		FROM (
			SELECT array_agg(dispositivos) as dispositivos, info->>'texto' as info
			FROM publico.config_var, jsonb_array_elements_text(info->'dispositivos') as dispositivos
			WHERE tipo='app_micuenta' AND info->>'url'='/info/accesibilidad'
			GROUP BY 2
		) as q0
		WHERE q0.dispositivos @> ARRAY[$1]`
	var textoTermsAndConditions string
	if err := db.ConnPgsql.QueryRow(db.Ctx, query, tipoApp).Scan(&textoTermsAndConditions); err != nil {
		utils.Logline("error on getting publico.config_var", err)
		return nil, errors.New("errorGetData")
	}

	return &textoTermsAndConditions, nil
}

func GetPrivacyPolicy(db models.ConnDb, tipoApp string) (*string, error) {
	//get meta of paginator
	query := `SELECT q0.info as texto
		FROM (
			SELECT array_agg(dispositivos) as dispositivos, info->>'texto' as info
			FROM publico.config_var, jsonb_array_elements_text(info->'dispositivos') as dispositivos
			WHERE tipo='app_micuenta' AND info->>'url'='/info/legal/privacy_policy'
			GROUP BY 2
		) as q0
		WHERE q0.dispositivos @> ARRAY[$1]`
	var textoTermsAndConditions string
	if err := db.ConnPgsql.QueryRow(db.Ctx, query, tipoApp).Scan(&textoTermsAndConditions); err != nil {
		utils.Logline("error on getting publico.config_var", err)
		return nil, errors.New("errorGetData")
	}

	return &textoTermsAndConditions, nil
}

func GetTermsAndConditions(db models.ConnDb, tipoApp string) (*string, error) {
	//get meta of paginator
	query := `SELECT q0.info as texto
		FROM (
			SELECT array_agg(dispositivos) as dispositivos, info->>'texto' as info
			FROM publico.config_var, jsonb_array_elements_text(info->'dispositivos') as dispositivos
			WHERE tipo='app_micuenta' AND info->>'url'='/info/legal/terminos_y_condiciones'
			GROUP BY 2
		) as q0
		WHERE q0.dispositivos @> ARRAY[$1]`
	var textoTermsAndConditions string
	if err := db.ConnPgsql.QueryRow(db.Ctx, query, tipoApp).Scan(&textoTermsAndConditions); err != nil {
		utils.Logline("error on getting publico.config_var", err)
		return nil, errors.New("errorGetData")
	}

	return &textoTermsAndConditions, nil
}
