package services

import "strings"

// snakeToPascalCase converts snake_case to PascalCase
// Examples: dashboard_username -> DashboardUsername, enable_ssl -> EnableSsl
func snakeToPascalCase(input string) string {
	parts := strings.Split(input, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}
