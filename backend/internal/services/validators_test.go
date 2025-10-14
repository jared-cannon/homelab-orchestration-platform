package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateIPAddress(t *testing.T) {
	tests := []struct {
		name  string
		ip    string
		valid bool
	}{
		// Valid IPv4
		{"Valid IPv4 - standard", "192.168.1.1", true},
		{"Valid IPv4 - localhost", "127.0.0.1", true},
		{"Valid IPv4 - Tailscale range", "100.64.1.5", true},
		{"Valid IPv4 - zeros", "0.0.0.0", true},
		{"Valid IPv4 - max values", "255.255.255.255", true},

		// Invalid IPv4
		{"Invalid IPv4 - empty", "", false},
		{"Invalid IPv4 - out of range", "256.1.1.1", false},
		{"Invalid IPv4 - negative", "192.-1.1.1", false},
		{"Invalid IPv4 - missing octet", "192.168.1", false},
		{"Invalid IPv4 - too many octets", "192.168.1.1.1", false},
		{"Invalid IPv4 - letters", "192.168.1.abc", false},

		// Valid IPv6
		{"Valid IPv6 - full", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", true},
		{"Valid IPv6 - compressed", "2001:db8::1", true},
		{"Valid IPv6 - localhost", "::1", true},
		{"Valid IPv6 - all zeros", "::", true},

		// Not hostnames (should fail)
		{"Not IP - hostname", "server.local", false},
		{"Not IP - FQDN", "machine.wolf-bear.ts.net", false},
		{"Not IP - simple name", "myserver", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateIPAddress(tt.ip)
			assert.Equal(t, tt.valid, result, "ValidateIPAddress(%q) = %v, want %v", tt.ip, result, tt.valid)
		})
	}
}

func TestValidateHostname(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		valid    bool
	}{
		// Valid IP addresses (should pass)
		{"IPv4 address", "192.168.1.1", true},
		{"IPv4 - Tailscale", "100.64.1.5", true},
		{"IPv6 - localhost", "::1", true},

		// Valid simple hostnames
		{"Simple hostname", "server", true},
		{"Hostname with number", "server1", true},
		{"Hostname with hyphen", "my-server", true},
		{"Hostname with underscore", "my_server", true},

		// Valid FQDNs
		{"FQDN - standard", "server.example.com", true},
		{"FQDN - subdomain", "api.server.example.com", true},
		{"FQDN - local", "server.local", true},

		// Valid Tailscale hostnames (the key test cases!)
		{"Tailscale - standard format", "machine.wolf-bear.ts.net", true},
		{"Tailscale - different names", "myserver.red-blue.ts.net", true},
		{"Tailscale - three words", "api.happy-sunny-day.ts.net", true},
		{"Tailscale - single word tailnet", "device.mytailnet.ts.net", true},
		{"Tailscale - numbers in name", "server1.net-work2.ts.net", true},
		{"Tailscale - underscores", "my_device.tail_net.ts.net", true},

		// Edge cases - valid
		{"Single char", "a", true},
		{"Max label length (63 chars)", "a12345678901234567890123456789012345678901234567890123456789012", true},
		{"Multiple subdomains", "a.b.c.d.e.f.example.com", true},

		// Invalid cases
		{"Empty string", "", false},
		{"Starts with hyphen", "-server", false},
		{"Ends with hyphen", "server-", false},
		{"Starts with dot", ".server", false},
		{"Ends with dot", "server.", false},
		{"Double dot", "server..example.com", false},
		{"Label too long (64 chars)", "a1234567890123456789012345678901234567890123456789012345678901234", false},
		{"Total too long (254 chars)", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false},
		{"Special chars", "server@example.com", false},
		{"Spaces", "my server", false},
		{"Only dots", "...", false},
		{"Only hyphens", "---", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateHostname(tt.hostname)
			assert.Equal(t, tt.valid, result, "ValidateHostname(%q) = %v, want %v", tt.hostname, result, tt.valid)
		})
	}
}

func TestValidateMACAddress(t *testing.T) {
	tests := []struct {
		name  string
		mac   string
		valid bool
	}{
		// Valid formats
		{"Valid - colon separator", "00:11:22:33:44:55", true},
		{"Valid - hyphen separator", "00-11-22-33-44-55", true},
		{"Valid - uppercase", "AA:BB:CC:DD:EE:FF", true},
		{"Valid - mixed case", "Aa:Bb:Cc:Dd:Ee:Ff", true},

		// Invalid formats
		{"Invalid - empty", "", false},
		{"Invalid - too short", "00:11:22:33:44", false},
		{"Invalid - too long", "00:11:22:33:44:55:66", false},
		{"Invalid - no separator", "001122334455", false},
		{"Invalid - wrong separator", "00.11.22.33.44.55", false},
		{"Invalid - invalid chars", "00:11:22:33:44:GG", false},
		{"Invalid - single digit", "0:1:2:3:4:5", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateMACAddress(tt.mac)
			assert.Equal(t, tt.valid, result, "ValidateMACAddress(%q) = %v, want %v", tt.mac, result, tt.valid)
		})
	}
}
