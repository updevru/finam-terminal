package ui

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"unicode"
)

// maskAccountID masks account ID for display
func maskAccountID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:4] + "****" + id[len(id)-4:]
}

// extractUserMessage extracts a user-friendly message from an error.
// It logs the full error and returns a cleaned up string, preferring Russian text if available.
func extractUserMessage(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	log.Printf("[ERROR] Full API error: %s", msg)

	// specific overrides
	if strings.Contains(msg, "PermissionDenied") {
		return "У вас не достаточно прав для выставления позиции"
	}

	// Look for Cyrillic characters
	for i, r := range msg {
		if unicode.Is(unicode.Cyrillic, r) {
			return strings.TrimSpace(msg[i:])
		}
	}

	// Fallback: try to clean up gRPC error
	if idx := strings.Index(msg, "desc = "); idx != -1 {
		clean := strings.TrimSpace(msg[idx+7:])
		// If it looks like the example [171]..., remove the code prefix if possible
		if bracketIdx := strings.Index(clean, "]"); bracketIdx != -1 {
			return strings.TrimSpace(clean[bracketIdx+1:])
		}
		return clean
	}

	return msg
}

// displayLots converts a raw quantity string to lot-based display.
// If lotSize > 0, divides qty by lotSize; otherwise returns rawQty as-is.
func displayLots(rawQty string, lotSize float64) string {
	if lotSize <= 0 {
		return rawQty
	}
	qty, err := parseFloat(rawQty)
	if err != nil {
		return rawQty
	}
	return fmt.Sprintf("%v", qty/lotSize)
}

// parseFloat parses a string to float64, handling commas as decimal separators
// and removing whitespace (including NBSP).
func parseFloat(s string) (float64, error) {
	// Remove all whitespace using runes to be safe
	var sb strings.Builder
	for _, r := range s {
		if !unicode.IsSpace(r) && r != '\u00A0' { // Handle regular space and NBSP
			sb.WriteRune(r)
		}
	}
	s = sb.String()

	s = strings.ReplaceAll(s, ",", ".")
	return strconv.ParseFloat(s, 64)
}
