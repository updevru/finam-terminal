package ui

import (
	"strings"
	"strconv"
	"unicode"
)

// maskAccountID masks account ID for display
func maskAccountID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:4] + "****" + id[len(id)-4:]
}

// parseFloat parses a string to float64, handling commas as decimal separators
// and removing whitespace (including NBSP).
func parseFloat(s string) (float64, error) {
	// Remove all whitespace
	var sb strings.Builder
	for _, r := range s {
		if !unicode.IsSpace(r) {
			sb.WriteRune(r)
		}
	}
	s = sb.String()
	
	s = strings.ReplaceAll(s, ",", ".")
	return strconv.ParseFloat(s, 64)
}