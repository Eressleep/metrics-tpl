package middleware

import (
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TrustedSubnetMiddleware проверяет, что IP-адрес клиента входит в доверенную подсеть
func TrustedSubnetMiddleware(ipNet *net.IPNet, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ipNet == nil {
			c.Next()
			return
		}

		realIP := c.GetHeader("X-Real-IP")
		if realIP == "" {
			logger.Warn("X-Real-IP header is missing",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method))
			c.JSON(http.StatusForbidden, gin.H{
				"error": "X-Real-IP header is required",
			})
			c.Abort()
			return
		}

		ip := net.ParseIP(realIP)
		if ip == nil {
			logger.Warn("Invalid IP address in X-Real-IP header",
				zap.String("x-real-ip", realIP),
				zap.String("path", c.Request.URL.Path))
			c.JSON(http.StatusForbidden, gin.H{
				"error": "invalid IP address",
			})
			c.Abort()
			return
		}

		if !ipNet.Contains(ip) {
			logger.Warn("IP address not in trusted subnet",
				zap.String("ip", realIP),
				zap.String("path", c.Request.URL.Path))
			c.JSON(http.StatusForbidden, gin.H{
				"error": "IP address not in trusted subnet",
			})
			c.Abort()
			return
		}

		logger.Debug("IP address verified",
			zap.String("ip", realIP))

		c.Next()
	}
}
