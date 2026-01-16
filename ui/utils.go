package ui

import (
	"strings"
	"strconv"
)

// maskAccountID masks account ID for display
func maskAccountID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:4] + "****" + id[len(id)-4:]
}

// parseFloat parses a string to float64, handling commas as decimal separators
func parseFloat(s string) (float64, error) {
	s = strings.ReplaceAll(s, ",", ".")
	return strconv.ParseFloat(s, 64)
}