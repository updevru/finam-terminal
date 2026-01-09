package ui

// maskAccountID masks account ID for display
func maskAccountID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:4] + "****" + id[len(id)-4:]
}
