package utils

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"html"
	"math/big"
	"regexp"
	"time"
)

func HasDuplicatesInt(arr []int) bool {
	seen := make(map[int]bool) // Hash table for tracking seen elements

	for _, num := range arr {
		if seen[num] {
			return true // Found a duplicate
		}
		seen[num] = true
	}

	return false // No duplicates found
}

func HasDuplicateStrings(arr []string) bool {
	seen := make(map[string]bool) // Track seen strings

	for _, str := range arr {
		if seen[str] {
			return true // Found a duplicate
		}
		seen[str] = true
	}

	return false // No duplicates found
}

// Helper function to generate a simple UUID (in production, use github.com/google/uuid)
func GenerateUUID() string {
	// This is a simplified version - use a proper UUID library for production
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		Logline("error generando uuid", err)
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// helper to generate 10Char hash for prefactura to use in nfactura
func GenerateNfacturaForPrefactura(timestamp time.Time, clienteId string, preFactOldid int) string {
	// Combine timestamp + two integers into a string
	data := fmt.Sprintf("%d-%s-%d", timestamp.UnixNano(), clienteId, preFactOldid)

	// Create SHA256 hash
	hash := sha256.Sum256([]byte(data))

	// Encode to Base64 and trim to 10 characters
	return base64.URLEncoding.EncodeToString(hash[:])[:10]
}

func GenerateNcontrolByUuid(uuidStr string) string {
	// Create an MD5 hash of the UUID
	hash := md5.Sum([]byte(uuidStr))
	hexHash := hex.EncodeToString(hash[:])

	// Convert first part to a number
	num := new(big.Int)
	num.SetString(hexHash[:8], 16) // Use the first 8 hex chars

	// Convert number to Base36 for short representation
	return fmt.Sprintf("%08s", num.Text(36)) // 8-character padded base36
}

func RemoveHTMLTags(input string) string {
	re := regexp.MustCompile("<[^>]+>") // Matches everything inside <>
	return html.UnescapeString(re.ReplaceAllString(input, ""))
}

func StringToTime(input string) *time.Time {
	// Define the format corresponding to the input
	layout := "2006-01-02 15:04:05"

	// Parse the string into time.Time
	parsedTime, err := time.Parse(layout, input)
	if err != nil {
		Logline("Error parsing string to time:", err)
		return nil
	}

	return &parsedTime
}
