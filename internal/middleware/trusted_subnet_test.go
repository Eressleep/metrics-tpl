package middleware

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestTrustedSubnetMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	// Парсим CIDR для тестов
	_, trustedIPNet, _ := net.ParseCIDR("192.168.0.0/16")
	_, trustedIPv6Net, _ := net.ParseCIDR("2001:db8::/32")

	tests := []struct {
		name           string
		ipNet          *net.IPNet
		xRealIP        string
		expectedStatus int
	}{
		{
			name:           "No trusted subnet - should pass",
			ipNet:          nil,
			xRealIP:        "192.168.1.1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "IP in trusted subnet - should pass",
			ipNet:          trustedIPNet,
			xRealIP:        "192.168.1.1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "IP not in trusted subnet - should fail",
			ipNet:          trustedIPNet,
			xRealIP:        "10.0.0.1",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Missing X-Real-IP header - should fail",
			ipNet:          trustedIPNet,
			xRealIP:        "",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Invalid IP in X-Real-IP - should fail",
			ipNet:          trustedIPNet,
			xRealIP:        "invalid-ip",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "IPv6 in trusted subnet - should pass",
			ipNet:          trustedIPv6Net,
			xRealIP:        "2001:db8::1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "IPv6 not in trusted subnet - should fail",
			ipNet:          trustedIPv6Net,
			xRealIP:        "2001:db8:1::1",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(TrustedSubnetMiddleware(tt.ipNet, logger))
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "ok")
			})

			req, _ := http.NewRequest("GET", "/test", nil)
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
