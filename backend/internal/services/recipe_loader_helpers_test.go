package services

import "testing"

func TestParseVariableName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Simple variable", "VAR_NAME", "VAR_NAME"},
		{"With whitespace", "  VAR_NAME  ", "VAR_NAME"},
		{"Default value with colon", "VAR_NAME:-default", "VAR_NAME"},
		{"Default value without colon", "VAR_NAME-default", "VAR_NAME"},
		{"Assign default with colon", "VAR_NAME:=default", "VAR_NAME"},
		{"Assign default without colon", "VAR_NAME=default", "VAR_NAME"},
		{"Error if unset with colon", "VAR_NAME:?error message", "VAR_NAME"},
		{"Error if unset without colon", "VAR_NAME?error message", "VAR_NAME"},
		{"Complex default", "PORT:-8080", "PORT"},
		{"Empty default", "ADMIN_TOKEN:-", "ADMIN_TOKEN"},
		{"Default with spaces", "VAR :- default value", "VAR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseVariableName(tt.input)
			if result != tt.expected {
				t.Errorf("parseVariableName(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsBuiltInVariable(t *testing.T) {
	tests := []struct {
		name     string
		varName  string
		expected bool
	}{
		// System variables
		{"DEPLOYMENT_ID", "DEPLOYMENT_ID", true},
		{"COMPOSE_PROJECT", "COMPOSE_PROJECT", true},
		{"DEVICE_IP", "DEVICE_IP", true},
		{"VERSION", "VERSION", true},

		// Database variables
		{"POSTGRES_HOST", "POSTGRES_HOST", true},
		{"POSTGRES_USER", "POSTGRES_USER", true},
		{"POSTGRES_PASSWORD", "POSTGRES_PASSWORD", true},
		{"POSTGRES_DB", "POSTGRES_DB", true},
		{"MYSQL_HOST", "MYSQL_HOST", true},
		{"MYSQL_USER", "MYSQL_USER", true},
		{"MARIADB_HOST", "MARIADB_HOST", true},

		// Cache variables
		{"REDIS_HOST", "REDIS_HOST", true},
		{"REDIS_PORT", "REDIS_PORT", true},
		{"MEMCACHED_HOST", "MEMCACHED_HOST", true},

		// Derived variables
		{"PASSWORD_HASH", "PASSWORD_HASH", true},
		{"ADMIN_PASSWORD_HASH", "ADMIN_PASSWORD_HASH", true},

		// User variables (not built-in)
		{"CUSTOM_VAR", "CUSTOM_VAR", false},
		{"PORT", "PORT", false},
		{"DOMAIN", "DOMAIN", false},
		{"APP_NAME", "APP_NAME", false},
		{"POSTGRES", "POSTGRES", false}, // Prefix must have underscore
		{"REDIS", "REDIS", false}, // Prefix must have underscore
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBuiltInVariable(tt.varName)
			if result != tt.expected {
				t.Errorf("isBuiltInVariable(%q) = %v, expected %v", tt.varName, result, tt.expected)
			}
		})
	}
}
