package utils

import (
	"math"
	"regexp"
	"strconv"

	"ired.com/micuenta/models"
)

// return only the numbers inside a string
func ExtractNumbers(input string) string {
	re := regexp.MustCompile("[^0-9]+") // Matches everything except numbers
	return re.ReplaceAllString(input, "")
}

// RoundToTwoDecimalPlaces rounds a float64 to two decimal places
func RoundToTwoDecimalPlaces(value float64) float64 {
	return math.Round(value*100) / 100
}

// roundTo8Decimals rounds a float64 to 8 decimal places
func RoundTo8Decimals(value float64) float64 {
	return math.Round(value*1e8) / 1e8
}

func IsDigitsOnly(s string) bool {
	match, _ := regexp.MatchString(`^[0-9]+$`, s) // Ensures only digits 0-9
	return match
}

func RoundToFourDecimals(num float64) float64 {
	return math.Round(num*10000) / 10000
}

func TransformMonedaToArray(monto models.Moneda) *[]float64 {
	return &[]float64{monto.Dolar, monto.Bolivar}
}

func StringToInt64(num string) int64 {
	// Convert string to int64
	int64Value, err := strconv.ParseInt(num, 10, 64)
	if err != nil {
		Logline("Error converting string to int64:", err)
		return 0
	}

	return int64Value
}

func Int64ToString(num int64) string {
	// Convert int64 to string
	return strconv.FormatInt(num, 10)
}

func IntToString(num int) string {
	// Convert int64 to string
	return strconv.Itoa(num)
}

func ParseFloat(str string) (float64, error) {
	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		Logline("Error parsing float:", str, err)
		return 0, err
	}
	return val, nil
}

func FindIntInArray(slice []int, num int) bool {
	for _, v := range slice {
		if v == num {
			return true
		}
	}
	return false
}

func HasMaxDecimals(value float64, maxDecimals int) bool {
	factor := math.Pow(10, float64(maxDecimals)) // Dynamically set precision
	rounded := math.Round(value*factor) / factor
	return rounded == value
}
