package middleware

import (
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

	tests := []struct {
		name           string
		trustedSubnet  string
		xRealIP        string
		expectedStatus int
	}{
		{
			name:           "No trusted subnet - should pass",
			trustedSubnet:  "",
			xRealIP:        "192.168.1.1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "IP in trusted subnet - should pass",
			trustedSubnet:  "192.168.0.0/16",
			xRealIP:        "192.168.1.1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "IP not in trusted subnet - should fail",
			trustedSubnet:  "192.168.0.0/24",
			xRealIP:        "192.168.1.1",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Missing X-Real-IP header - should fail",
			trustedSubnet:  "192.168.0.0/24",
			xRealIP:        "",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Invalid IP in X-Real-IP - should fail",
			trustedSubnet:  "192.168.0.0/24",
			xRealIP:        "invalid-ip",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Invalid CIDR - should fail",
			trustedSubnet:  "invalid-cidr",
			xRealIP:        "192.168.1.1",
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "IPv6 in trusted subnet - should pass",
			trustedSubnet:  "2001:db8::/32",
			xRealIP:        "2001:db8::1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "IPv6 not in trusted subnet - should fail",
			trustedSubnet:  "2001:db8::/48",
			xRealIP:        "2001:db8:1::1",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(TrustedSubnetMiddleware(tt.trustedSubnet, logger))
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

func TestTrustedSubnetMiddlewareWithRealIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	router := gin.New()
	router.Use(TrustedSubnetMiddleware("192.168.0.0/16", logger))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Real-IP", "192.168.1.100")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	req, _ = http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Real-IP", "10.0.0.1")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}
