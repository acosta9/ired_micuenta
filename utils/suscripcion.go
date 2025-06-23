package utils

import (
	"regexp"
	"strings"

	"ired.com/micuenta/models"
)

func TipoServicioAcronimo(input string) string {
	switch {
	case strings.Contains(input, "transporte"):
		return "transp"
	case strings.Contains(input, "datos"):
		return "transp"
	case strings.Contains(input, "res"):
		return "res"
	case strings.Contains(input, "emp"):
		return "emp"
	case strings.Contains(input, "sim"):
		return "sim"
	case strings.Contains(input, "conv"):
		return "res*"
	default:
		return "na"
	}
}

func TipoServicioNombre(input string) string {
	switch {
	case strings.Contains(input, "transporte"):
		return "transporte"
	case strings.Contains(input, "datos"):
		return "transporte"
	case strings.Contains(input, "res"):
		return "residencial"
	case strings.Contains(input, "emp"):
		return "empresarial"
	case strings.Contains(input, "sim"):
		return "simetrico"
	case strings.Contains(input, "conv"):
		return "residencial*"
	default:
		return "na"
	}
}

func ValidateGPS(input string) string {
	re := regexp.MustCompile(`^-?\d+(\.\d+)?,\-?\d+(\.\d+)?$`)
	if re.MatchString(input) {
		return input
	}
	return ""
}

func PaymentReqAmountHasDupIds(payments []models.PaymentReqDetail) bool {
	seen := make(map[string]bool) // Track seen IDs

	for _, payment := range payments {
		if seen[payment.SuscripcionId] {
			return true // Duplicate found
		}
		seen[payment.SuscripcionId] = true
	}

	return false // No duplicates found
}

// // GetNextRenewalDate calculates the next renewal date based on the installed date
// func NextRenewalDate(subscribedDate sql.NullTime) (time.Time, error) {
// 	if !subscribedDate.Valid {
// 		return time.Time{}, fmt.Errorf("invalid subscription date")
// 	}

// 	// Get current date
// 	now := time.Now()

// 	// Extract the day from the subscribed date
// 	renewalDay := subscribedDate.Time.Day()

// 	// Get next renewal month & year
// 	nextRenewal := time.Date(now.Year(), now.Month(), renewalDay, 0, 0, 0, 0, now.Location())

// 	// If today's date is past renewalDay, set renewal for the next month
// 	if now.Day() >= renewalDay {
// 		nextRenewal = nextRenewal.AddDate(0, 1, 0)
// 	}

// 	return nextRenewal, nil
// }
