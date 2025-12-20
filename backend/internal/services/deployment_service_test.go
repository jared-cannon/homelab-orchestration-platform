package services

import (
	"strings"
	"testing"
)

func TestIsValidHostname(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		want     bool
	}{
		// Valid hostnames
		{
			name:     "simple domain",
			hostname: "example.com",
			want:     true,
		},
		{
			name:     "subdomain",
			hostname: "app.example.com",
			want:     true,
		},
		{
			name:     "multi-level subdomain",
			hostname: "my-app.staging.example.com",
			want:     true,
		},
		{
			name:     "hyphens in label",
			hostname: "my-cool-app.example.com",
			want:     true,
		},
		{
			name:     "numbers in label",
			hostname: "app123.example.com",
			want:     true,
		},
		{
			name:     "uppercase (should be normalized)",
			hostname: "App.Example.COM",
			want:     true,
		},
		{
			name:     "single character labels",
			hostname: "a.b.c",
			want:     true,
		},
		{
			name:     "max label length (63 chars)",
			hostname: strings.Repeat("a", 63) + ".example.com",
			want:     true,
		},
		{
			name:     "total max length (253 chars)",
			hostname: strings.Repeat("a", 60) + "." + strings.Repeat("b", 60) + "." + strings.Repeat("c", 60) + "." + strings.Repeat("d", 60) + ".com",
			want:     true,
		},

		// Invalid hostnames
		{
			name:     "empty string",
			hostname: "",
			want:     false,
		},
		{
			name:     "starts with hyphen",
			hostname: "-example.com",
			want:     false,
		},
		{
			name:     "ends with hyphen",
			hostname: "example-.com",
			want:     false,
		},
		{
			name:     "starts with dot",
			hostname: ".example.com",
			want:     false,
		},
		{
			name:     "ends with dot",
			hostname: "example.com.",
			want:     false,
		},
		{
			name:     "double dot",
			hostname: "example..com",
			want:     false,
		},
		{
			name:     "space in hostname",
			hostname: "my app.example.com",
			want:     false,
		},
		{
			name:     "underscore in hostname",
			hostname: "my_app.example.com",
			want:     false,
		},
		{
			name:     "special characters",
			hostname: "app@example.com",
			want:     false,
		},
		{
			name:     "label too long (64 chars)",
			hostname: strings.Repeat("a", 64) + ".example.com",
			want:     false,
		},
		{
			name:     "total length exactly 254 chars (1 over RFC 1035 limit)",
			hostname: strings.Repeat("a", 250) + ".com", // 250 + 4 = 254
			want:     false,
		},
		{
			name:     "hyphen only label",
			hostname: "-.example.com",
			want:     false,
		},
		{
			name:     "single label too short (empty after split)",
			hostname: "example.",
			want:     false,
		},
		{
			name:     "single label hostname (no dots)",
			hostname: "localhost",
			want:     false, // Reject: homelab needs FQDN for Traefik/Let's Encrypt
		},
		{
			name:     "single label with numbers",
			hostname: "server1",
			want:     false, // Reject: need proper domain suffix
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidHostname(tt.hostname)
			if got != tt.want {
				t.Errorf("isValidHostname(%q) = %v, want %v", tt.hostname, got, tt.want)
			}
		})
	}
}
