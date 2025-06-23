package models

import (
	"encoding/json"
	"errors"
	"log"
	"math"
	"regexp"
	"strconv"
	"strings"

	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	passwdValidator "github.com/wagslane/go-password-validator"
)

type errorMsgs struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func errorToJson(fieldError validator.FieldError, c *gin.Context) string {
	switch fieldError.Tag() {
	case "required":
		return ginI18n.MustGetMessage(c, "veRequired")
	case "number":
		return ginI18n.MustGetMessage(c, "veNumber")
	case "numeric":
		return ginI18n.MustGetMessage(c, "veNumeric")
	case "min":
		return ginI18n.MustGetMessage(c, "veMinChar") + " " + fieldError.Param() + " " + ginI18n.MustGetMessage(c, "veChar")
	case "max":
		return ginI18n.MustGetMessage(c, "veMaxChar") + " " + fieldError.Param() + " " + ginI18n.MustGetMessage(c, "veChar")
	case "gte":
		return ginI18n.MustGetMessage(c, "veGte") + " " + fieldError.Param()
	case "lte":
		return ginI18n.MustGetMessage(c, "veLte") + " " + fieldError.Param()
	case "notzero":
		return ginI18n.MustGetMessage(c, "veNotzero") + " " + fieldError.Param()
	case "alfanumspa":
		return ginI18n.MustGetMessage(c, "veAlphaNumSpa")
	case "gte_number":
		return ginI18n.MustGetMessage(c, "veGte") + " " + fieldError.Param()
	case "lte_number":
		return ginI18n.MustGetMessage(c, "veLte") + " " + fieldError.Param()
	case "decimals_number":
		return ginI18n.MustGetMessage(c, "veDecimals") + " " + fieldError.Param()
	case "boolean":
		return ginI18n.MustGetMessage(c, "veBoolean")
	case "passwd_strenght":
		return ginI18n.MustGetMessage(c, "vePasswordStrength")
	case "datetime":
		return ginI18n.MustGetMessage(c, "veDatetime")
	case "celular":
		return ginI18n.MustGetMessage(c, "veCelphone")
	case "email":
		return ginI18n.MustGetMessage(c, "veEmail")
	case "uuid":
		return ginI18n.MustGetMessage(c, "veUuid")
	}
	return fieldError.Error() // default error
}

func ParseError(err error, c *gin.Context) []errorMsgs {
	var validatorError validator.ValidationErrors
	if errors.As(err, &validatorError) {
		out := make([]errorMsgs, len(validatorError))
		for i, fieldError := range validatorError {
			out[i] = errorMsgs{strings.ToLower(fieldError.Field()), errorToJson(fieldError, c)}
		}
		return out
	}
	return nil
}

var alphaNumEs validator.Func = func(fl validator.FieldLevel) bool {
	hasWhitespace := strings.TrimSpace(fl.Field().String()) != fl.Field().String()
	if hasWhitespace {
		return false
	}
	regex := regexp.MustCompile(`^[a-z A-Z0-9ñÑáéíóúÁÉÍÓÚ]+$`)
	return regex.MatchString(fl.Field().String())
}

var gteNumber validator.Func = func(fl validator.FieldLevel) bool {
	tag := fl.Param() // Get the parameter from the tag
	minValue, err := strconv.ParseFloat(tag, 64)
	if err != nil {
		log.Println("error leyendo params in function gteNumber: %w", err)
		return false
	}

	num, err := fl.Field().Interface().(json.Number).Float64()
	if err != nil || num < minValue {
		return false
	}
	return true
}

var lteNumber validator.Func = func(fl validator.FieldLevel) bool {
	tag := fl.Param() // Get the parameter from the tag
	minValue, err := strconv.ParseFloat(tag, 64)
	if err != nil {
		log.Println("error leyendo params in function lteNumber: %w", err)
		return false
	}

	num, err := fl.Field().Interface().(json.Number).Float64()
	if err != nil || num > minValue {
		return false
	}
	return true
}

var passwdStrenght validator.Func = func(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	const minEntropyBits = 60
	if err := passwdValidator.Validate(password, minEntropyBits); err != nil {
		return false
	}

	return true
}

var celPhone validator.Func = func(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	matched, _ := regexp.MatchString(`^04(12|16|26|14|24)[0-9]{7}$`, phone)
	return matched
}

var decimalsNumber validator.Func = func(fl validator.FieldLevel) bool {
	tag := fl.Param() // Get the parameter from the tag
	maxDecimals, err := strconv.ParseFloat(tag, 64)
	if err != nil {
		log.Println("error leyendo params in function decimalsNumber: %w", err)
		return false
	}

	num := fl.Field().Float()

	return hasMaxDecimals(num, int(maxDecimals))
}

// var notZero validator.Func = func(fl validator.FieldLevel) bool {
// 	return fl.Field().Int() != 0
// }

func hasMaxDecimals(value float64, maxDecimals int) bool {
	factor := math.Pow(10, float64(maxDecimals)) // Dynamically set precision
	rounded := math.Round(value*factor) / factor
	return rounded == value
}
