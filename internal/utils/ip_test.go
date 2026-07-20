package utils

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetOutboundIP(t *testing.T) {
	ip := GetOutboundIP()
	if ip == "" {
		t.Skip("No outbound IP found (possibly no network connection)")
	}
	assert.NotEmpty(t, ip)
	assert.NotNil(t, net.ParseIP(ip))
}

func TestParseCIDR(t *testing.T) {
	tests := []struct {
		name    string
		cidr    string
		wantErr bool
	}{
		{
			name:    "Valid CIDR",
			cidr:    "192.168.0.0/24",
			wantErr: false,
		},
		{
			name:    "Valid IPv6 CIDR",
			cidr:    "2001:db8::/32",
			wantErr: false,
		},
		{
			name:    "Empty CIDR",
			cidr:    "",
			wantErr: false,
		},
		{
			name:    "Invalid CIDR",
			cidr:    "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ipNet, err := ParseCIDR(tt.cidr)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, ipNet)
			} else {
				assert.NoError(t, err)
				if tt.cidr != "" {
					assert.NotNil(t, ipNet)
				} else {
					assert.Nil(t, ipNet)
				}
			}
		})
	}
}

func TestIPInCIDR(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		cidr     string
		expected bool
		wantErr  bool
	}{
		{
			name:     "IP in CIDR",
			ip:       "192.168.1.1",
			cidr:     "192.168.0.0/16",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "IP not in CIDR",
			ip:       "10.0.0.1",
			cidr:     "192.168.0.0/24",
			expected: false,
			wantErr:  false,
		},
		{
			name:     "Empty CIDR - always true",
			ip:       "192.168.1.1",
			cidr:     "",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "Invalid IP",
			ip:       "invalid",
			cidr:     "192.168.0.0/24",
			expected: false,
			wantErr:  false,
		},
		{
			name:     "Invalid CIDR",
			ip:       "192.168.1.1",
			cidr:     "invalid",
			expected: false,
			wantErr:  true,
		},
		{
			name:     "IPv6 in CIDR",
			ip:       "2001:db8::1",
			cidr:     "2001:db8::/32",
			expected: true,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := IPInCIDR(tt.ip, tt.cidr)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGetRealIP(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string][]string
		expected string
	}{
		{
			name: "X-Real-IP present",
			headers: map[string][]string{
				"X-Real-Ip": {"192.168.1.1"},
			},
			expected: "192.168.1.1",
		},
		{
			name: "X-Forwarded-For present",
			headers: map[string][]string{
				"X-Forwarded-For": {"192.168.1.1, 10.0.0.1"},
			},
			expected: "192.168.1.1",
		},
		{
			name: "Both headers present - X-Real-IP takes precedence",
			headers: map[string][]string{
				"X-Real-Ip":       {"192.168.1.1"},
				"X-Forwarded-For": {"10.0.0.1"},
			},
			expected: "192.168.1.1",
		},
		{
			name:     "No headers",
			headers:  map[string][]string{},
			expected: "",
		},
		{
			name: "Empty X-Real-IP",
			headers: map[string][]string{
				"X-Real-Ip": {""},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetRealIP(tt.headers)
			assert.Equal(t, tt.expected, result)
		})
	}
}
